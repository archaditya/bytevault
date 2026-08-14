package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/notification/queue"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/security"
	"github.com/archaditya/bytevault/internal/storage"
)

const (
	// DefaultQuotaBytes defines 1GB of storage quota per user
	DefaultQuotaBytes = 1 * 1024 * 1024 * 1024
	// MaxFileSizeLimit defines a 100MB limit for a single file upload
	MaxFileSizeLimit = 100 * 1024 * 1024
)

// Whitelisted allowed MIME types for storage (all developer, media, document, apple, and data formats)
var AllowedMimeTypes = map[string]bool{
	// Images & Apple Formats
	"image/jpeg":          true,
	"image/png":           true,
	"image/gif":           true,
	"image/webp":          true,
	"image/svg+xml":       true,
	"image/heic":          true,
	"image/heif":          true,
	"image/heic-sequence": true,
	"image/heif-sequence": true,
	"image/avif":          true,
	"image/bmp":           true,
	"image/tiff":          true,
	"image/x-icon":        true,

	// Documents & Office
	"application/pdf":                                                           true,
	"application/msword":                                                        true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.ms-excel":                                                 true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                            true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"application/rtf":                                                           true,
	"application/epub+zip":                                                      true,

	// Apple & macOS Formats
	"application/x-iwork-pages-sffpages":     true,
	"application/x-iwork-numbers-sffnumbers": true,
	"application/x-iwork-keynote-sffkey":     true,
	"application/vnd.apple.pages":            true,
	"application/vnd.apple.numbers":          true,
	"application/vnd.apple.keynote":          true,
	"application/x-apple-diskimage":          true,
	"application/x-plist":                    true,

	// Developer, Code & Data formats
	"application/json":          true,
	"application/ld+json":       true,
	"application/x-ndjson":      true,
	"application/javascript":    true,
	"text/javascript":           true,
	"application/x-javascript":  true,
	"application/typescript":    true,
	"text/typescript":           true,
	"text/plain":                true,
	"text/csv":                  true,
	"text/markdown":             true,
	"text/html":                 true,
	"text/css":                  true,
	"text/xml":                  true,
	"application/xml":           true,
	"application/xhtml+xml":     true,
	"text/yaml":                 true,
	"text/x-yaml":               true,
	"application/yaml":          true,
	"application/x-yaml":        true,
	"text/x-python":             true,
	"application/x-python-code": true,
	"text/x-go":                 true,
	"text/x-c":                  true,
	"text/x-c++":                true,
	"text/x-java-source":        true,
	"text/x-rust":               true,
	"text/x-ruby":               true,
	"text/x-php":                true,
	"text/x-shellscript":        true,
	"text/x-sh":                 true,
	"text/x-sql":                true,
	"application/sql":           true,
	"application/x-sql":         true,
	"application/wasm":          true,
	"application/graphql":       true,
	"application/x-protobuf":    true,
	"application/toml":          true,
	"text/x-toml":               true,

	// Audio formats
	"audio/mpeg":   true,
	"audio/wav":    true,
	"audio/ogg":    true,
	"audio/x-m4a":  true,
	"audio/mp4":    true,
	"audio/aac":    true,
	"audio/flac":   true,
	"audio/aiff":   true,
	"audio/x-aiff": true,
	"audio/webm":   true,

	// Video formats
	"video/mp4":        true,
	"video/mpeg":       true,
	"video/quicktime":  true,
	"video/webm":       true,
	"video/x-m4v":      true,
	"video/x-matroska": true,

	// Archives & Datasets
	"application/zip":                true,
	"application/x-tar":              true,
	"application/x-rar-compressed":   true,
	"application/x-7z-compressed":    true,
	"application/gzip":               true,
	"application/x-gzip":             true,
	"application/x-bzip2":            true,
	"application/x-xz":               true,
	"application/vnd.apache.parquet": true,
	"application/x-parquet":          true,
	"application/x-sqlite3":          true,
	"application/vnd.sqlite3":        true,
	"application/octet-stream":       true,
}

type FileService struct {
	repo            *repository.FileRepository
	userRepo        *repository.UserRepository
	storage         storage.StorageProvider
	storageProvider string
	bucket          *string
	redisQueue      *queue.RedisQueue
	malwareScanner  security.MalwareScanner
	activityRepo    *repository.ActivityRepository
}

func NewFileService(
	repo *repository.FileRepository,
	userRepo *repository.UserRepository,
	storage storage.StorageProvider,
	provider string,
	bucket string,
	redisQueue *queue.RedisQueue,
	activityRepo *repository.ActivityRepository,
) *FileService {
	var bPtr *string
	if bucket != "" {
		bPtr = &bucket
	}
	return &FileService{
		repo:            repo,
		userRepo:        userRepo,
		storage:         storage,
		storageProvider: provider,
		bucket:          bPtr,
		redisQueue:      redisQueue,
		malwareScanner:  security.NewDefaultMalwareScanner(),
		activityRepo:    activityRepo,
	}
}

func (s *FileService) logActivity(ctx context.Context, userID, action, resourceType, resourceID string, meta map[string]any) {
	if s.activityRepo == nil {
		return
	}
	_ = s.activityRepo.Log(ctx, &model.ActivityLog{
		UserID:       &userID,
		Action:       action,
		ResourceType: &resourceType,
		ResourceID:   &resourceID,
		Metadata:     meta,
	})
}

// enqueueMediaProcessingJob safely triggers asynchronous background thumbnail generation
func (s *FileService) enqueueMediaProcessingJob(ctx context.Context, file *model.File) {
	if s.redisQueue == nil || file == nil {
		return
	}
	go func() {
		_ = s.redisQueue.EnqueueMediaJob(context.Background(), file.ID, file.UserID, file.StorageKey, file.ContentType)
	}()
}


func (s *FileService) generateStorageKey(userID, filename string) string {
	return fmt.Sprintf("user/%s/docs/%s", userID, filepath.Base(filename))
}

func (s *FileService) validateFile(ctx context.Context, userID string, size int64, contentType string) error {
	// 1. Fetch user for dynamic limits
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch user details: %w", err)
	}

	// 2. Per-user max file size validation
	maxFileSize := int64(MaxFileSizeLimit)
	if user.MaxFileSizeBytes != nil {
		maxFileSize = *user.MaxFileSizeBytes
	}
	if size > maxFileSize {
		return fmt.Errorf("file size (%d bytes) exceeds your maximum allowed file size limit of %d bytes", size, maxFileSize)
	}

	// Sanitize content type string
	cleanContentType := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))

	// 3. MIME Type Validation
	if !AllowedMimeTypes[cleanContentType] && cleanContentType != "application/octet-stream" {
		return fmt.Errorf("unsupported file type: %s", contentType)
	}

	// 4. User Storage Quota Check
	used, err := s.repo.GetUserStorageUsed(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch user storage usage: %w", err)
	}

	totalLimit := int64(DefaultQuotaBytes)
	if user.StorageLimitBytes != nil {
		totalLimit = *user.StorageLimitBytes
	}

	if used+size > totalLimit {
		return fmt.Errorf("insufficient storage. Uploading this file will exceed your remaining storage quota")
	}

	return nil
}

func (s *FileService) CreateUploadSession(ctx context.Context, userID, filename string, size int64, contentType string, folderID *string) (*model.File, string, error) {
	if err := s.validateFile(ctx, userID, size, contentType); err != nil {
		return nil, "", err
	}

	storageKey := s.generateStorageKey(userID, filename)

	uploadURL, err := s.storage.GeneratePresignedUploadURL(ctx, storageKey, contentType, 15*time.Minute)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	if folderID != nil && *folderID == "" {
		folderID = nil
	}

	fileMeta := &model.File{
		UserID:          userID,
		Filename:        filename,
		StorageProvider: s.storageProvider,
		Bucket:          s.bucket,
		StorageKey:      storageKey,
		FileSize:        size,
		ContentType:     contentType,
		IsPublic:        false,
		Status:          "UPLOADING",
		FolderID:        folderID,
	}

	if err := s.repo.Create(ctx, fileMeta); err != nil {
		return nil, "", err
	}

	s.logActivity(ctx, userID, "file.upload_session", "file", fileMeta.ID, map[string]any{
		"filename":  filename,
		"file_size": size,
	})

	return fileMeta, uploadURL, nil
}

func (s *FileService) CompleteUpload(ctx context.Context, fileID, userID string) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil {
		return fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Fetch first 512 bytes from storage for signature validation
	stream, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, fileID, "FAILED")
		return fmt.Errorf("failed to download file from storage for validation: %w", err)
	}

	header := make([]byte, 512)
	n, readErr := io.ReadFull(stream, header)
	_ = stream.Close()

	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		_ = s.storage.Delete(ctx, file.StorageKey)
		_ = s.repo.UpdateStatus(ctx, fileID, "FAILED")
		return fmt.Errorf("failed to read file header from storage: %w", readErr)
	}
	headerBytes := header[:n]

	// Sniff actual content-type via magic bytes
	detectedMime := http.DetectContentType(headerBytes)

	// Check for HEIC magic bytes signature
	if DetectHEICMagicBytes(headerBytes) {
		detectedMime = "image/heic"
	}

	// Verify magic bytes / signature compatibility
	if err := ValidateMagicBytes(detectedMime, file.ContentType, file.Filename); err != nil {
		_ = s.storage.Delete(ctx, file.StorageKey)
		_ = s.repo.UpdateStatus(ctx, fileID, "FAILED")
		return fmt.Errorf("upload rejected: %w", err)
	}

	// Set status to PENDING_SCAN and enqueue background worker for malware scan + thumbnailing
	if err := s.repo.UpdateStatus(ctx, fileID, "PENDING_SCAN"); err != nil {
		return err
	}
	s.enqueueMediaProcessingJob(ctx, file)

	s.logActivity(ctx, userID, "file.upload_completed", "file", file.ID, map[string]any{
		"filename":  file.Filename,
		"file_size": file.FileSize,
	})

	return nil
}

func (s *FileService) Upload(ctx context.Context, userID, filename string, size int64, contentType string, content io.Reader, folderID *string) (*model.File, error) {
	if err := s.validateFile(ctx, userID, size, contentType); err != nil {
		return nil, err
	}

	// Read first 512 bytes of file
	header := make([]byte, 512)
	n, readErr := io.ReadFull(content, header)
	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		return nil, fmt.Errorf("failed to read file header for validation: %w", readErr)
	}
	headerBytes := header[:n]

	// Validate magic bytes
	detectedMime := http.DetectContentType(headerBytes)
	if DetectHEICMagicBytes(headerBytes) {
		detectedMime = "image/heic"
	}

	if err := ValidateMagicBytes(detectedMime, contentType, filename); err != nil {
		return nil, err
	}

	// reconstruct stream using MultiReader to ensure all bytes are uploaded
	fullContent := io.MultiReader(bytes.NewReader(headerBytes), content)

	storageKey := s.generateStorageKey(userID, filename)

	_, err := s.storage.Upload(ctx, storageKey, fullContent, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed upload in storage: %w", err)
	}

	if folderID != nil && *folderID == "" {
		folderID = nil
	}

	fileMeta := &model.File{
		UserID:          userID,
		Filename:        filename,
		StorageProvider: s.storageProvider,
		Bucket:          s.bucket,
		StorageKey:      storageKey,
		FileSize:        size,
		ContentType:     contentType,
		IsPublic:        false,
		Status:          "PENDING_SCAN",
		FolderID:        folderID,
	}

	if err := s.repo.Create(ctx, fileMeta); err != nil {
		_ = s.storage.Delete(ctx, storageKey)
		return nil, err
	}

	s.enqueueMediaProcessingJob(ctx, fileMeta)

	return fileMeta, nil
}

func (s *FileService) Download(ctx context.Context, fileID, userID string, inline bool) (string, *model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return "", nil, err
	}
	if file == nil {
		return "", nil, fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return "", nil, fmt.Errorf("unauthorized")
	}

	if file.Status == "PENDING_SCAN" {
		return "", nil, fmt.Errorf("file is undergoing background security scan")
	}
	if file.Status == "BLOCKED_MALWARE" {
		return "", nil, fmt.Errorf("file access blocked: security threat detected")
	}

	url, err := s.storage.GeneratePresignedDownloadURL(ctx, file.StorageKey, 30*time.Second, file.Filename, inline)

	if err != nil {
		return "", nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

	s.logActivity(ctx, userID, "file.download", "file", file.ID, map[string]any{
		"filename":  file.Filename,
		"file_size": file.FileSize,
	})

	return url, file, nil
}

func (s *FileService) DownloadPublic(ctx context.Context, fileID string, inline bool) (string, *model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return "", nil, err
	}
	if file == nil {
		return "", nil, fmt.Errorf("file not found")
	}
	if !file.IsPublic {
		return "", nil, fmt.Errorf("unauthorized")
	}

	if file.Status == "PENDING_SCAN" {
		return "", nil, fmt.Errorf("file is undergoing background security scan")
	}
	if file.Status == "BLOCKED_MALWARE" {
		return "", nil, fmt.Errorf("file access blocked: security threat detected")
	}

	url, err := s.storage.GeneratePresignedDownloadURL(ctx, file.StorageKey, 30*time.Second, file.Filename, inline)
	if err != nil {
		return "", nil, fmt.Errorf("failed to download from storage: %w", err)
	}

	// Increment downloads count asynchronously ONLY for public shared downloads
	go func() {
		_ = s.repo.IncrementDownloads(context.Background(), file.ID)
	}()

	return url, file, nil
}

func (s *FileService) ListUserFiles(ctx context.Context, params repository.ListFilesParams) ([]*model.File, string, error) {
	files, nextCursor, err := s.repo.ListByUserID(ctx, params)
	if err != nil {
		return nil, "", err
	}

	// Enrich files with direct presigned thumbnail URLs in the JSON response
	for _, f := range files {
		if f.ThumbnailKey != nil && *f.ThumbnailKey != "" {
			url, err := s.storage.GeneratePresignedDownloadURL(ctx, *f.ThumbnailKey, 30*time.Minute, f.Filename, true)
			if err == nil {
				f.ThumbnailURL = &url
			}
		}
	}

	return files, nextCursor, nil
}

func (s *FileService) ToggleShareStatus(ctx context.Context, fileID, userID string, isPublic bool) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil || file.UserID != userID {
		return fmt.Errorf("file not found or unauthorized")
	}

	return s.repo.UpdatePublicStatus(ctx, fileID, isPublic)
}

func (s *FileService) MoveFile(ctx context.Context, fileID, userID string, folderID *string) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil || file.UserID != userID {
		return fmt.Errorf("file not found or unauthorized")
	}

	if folderID != nil && *folderID == "" {
		folderID = nil
	}

	s.logActivity(ctx, userID, "file.move", "file", file.ID, map[string]any{
		"from_folder": file.FolderID,
		"to_folder":   folderID,
	})

	return s.repo.MoveFile(ctx, fileID, folderID)
}

func (s *FileService) RenameFile(ctx context.Context, fileID, userID, newFilename string) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil || file.UserID != userID {
		return fmt.Errorf("file not found or unauthorized")
	}

	newFilename = strings.TrimSpace(newFilename)
	if newFilename == "" {
		return fmt.Errorf("filename cannot be empty")
	}

	s.logActivity(ctx, userID, "file.rename", "file", file.ID, map[string]any{
		"old_filename": file.Filename,
		"new_filename": newFilename,
	})

	return s.repo.RenameFile(ctx, fileID, newFilename)
}

func (s *FileService) Delete(ctx context.Context, fileID, userID string) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil {
		return fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	s.logActivity(ctx, userID, "file.delete", "file", file.ID, map[string]any{
		"filename":  file.Filename,
		"file_size": file.FileSize,
	})

	// Soft delete only — R2 cleanup happens via scheduler after 30-day cooldown
	return s.repo.SoftDelete(ctx, fileID)
}

func (s *FileService) GetFileDetails(ctx context.Context, fileID, userID string) (*model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}
	return file, nil
}

func (s *FileService) GetPublicMetadata(ctx context.Context, fileID string) (*model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("file not found")
	}
	if !file.IsPublic {
		return nil, fmt.Errorf("unauthorized")
	}
	return file, nil
}

// --- Multipart Upload Methods ---

func (s *FileService) CreateMultipartUploadSession(ctx context.Context, userID, filename string, size int64, contentType string, folderID *string, partCount int) (*model.File, string, []map[string]interface{}, error) {
	if err := s.validateFile(ctx, userID, size, contentType); err != nil {
		return nil, "", nil, err
	}

	storageKey := s.generateStorageKey(userID, filename)

	// Initialize Multipart upload
	uploadID, err := s.storage.InitiateMultipartUpload(ctx, storageKey, contentType)
	if err != nil {
		return nil, "", nil, err
	}

	// Generate Presigned URLs for each part
	var partsURLs []map[string]interface{}
	for i := 1; i <= partCount; i++ {
		url, err := s.storage.GeneratePresignedUploadPartURL(ctx, storageKey, uploadID, int32(i), 30*time.Minute)
		if err != nil {
			//Abort session 
			_ = s.storage.AbortMultipartUpload(ctx, storageKey, uploadID)
			return nil, "", nil, fmt.Errorf("failed to generate presigned URL for part %d: %w", i, err)
		}
		partsURLs = append(partsURLs, map[string]interface{}{
			"part_number": i,
			"url":         url,
		})
	}

	if folderID != nil && *folderID == "" {
		folderID = nil
	}

	fileMeta := &model.File{
		UserID:          userID,
		Filename:        filename,
		StorageProvider: s.storageProvider,
		Bucket:          s.bucket,
		StorageKey:      storageKey,
		FileSize:        size,
		ContentType:     contentType,
		IsPublic:        false,
		Status:          "UPLOADING",
		FolderID:        folderID,
	}

	// create file record in db
	if err := s.repo.Create(ctx, fileMeta); err != nil {
		_ = s.storage.AbortMultipartUpload(ctx, storageKey, uploadID)
		return nil, "", nil, err
	}

	s.logActivity(ctx, userID, "file.upload_session", "file", fileMeta.ID, map[string]any{
		"filename":  filename,
		"file_size": size,
	})

	return fileMeta, uploadID, partsURLs, nil
}

func (s *FileService) CompleteMultipartUpload(ctx context.Context, fileID, userID string, uploadID string, parts []model.UploadPart) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil {
		return fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	// Assemble all parts and complete multipart upload
	_, err = s.storage.CompleteMultipartUpload(ctx, file.StorageKey, uploadID, parts)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, fileID, "FAILED")
		return fmt.Errorf("failed to complete multipart upload assembly: %w", err)
	}

	// signature validation
	stream, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		_ = s.repo.UpdateStatus(ctx, fileID, "FAILED")
		return fmt.Errorf("failed to download file from storage for validation: %w", err)
	}

	header := make([]byte, 512)
	n, readErr := io.ReadFull(stream, header)
	_ = stream.Close()

	if readErr != nil && readErr != io.ErrUnexpectedEOF && readErr != io.EOF {
		_ = s.storage.Delete(ctx, file.StorageKey)
		_ = s.repo.UpdateStatus(ctx, fileID, "FAILED")
		return fmt.Errorf("failed to read file header from storage: %w", readErr)
	}
	headerBytes := header[:n]

	// sniff actual content-type via magic bytes
	detectedMime := http.DetectContentType(headerBytes)
	if DetectHEICMagicBytes(headerBytes) {
		detectedMime = "image/heic"
	}

	// verify magic bytes / signature validation
	if err := ValidateMagicBytes(detectedMime, file.ContentType, file.Filename); err != nil {
		_ = s.storage.Delete(ctx, file.StorageKey)
		_ = s.repo.UpdateStatus(ctx, fileID, "FAILED")
		return fmt.Errorf("upload rejected: %w", err)
	}

	if err := s.repo.UpdateStatus(ctx, fileID, "PENDING_SCAN"); err != nil {
		return err
	}

	s.enqueueMediaProcessingJob(ctx, file)

	s.logActivity(ctx, userID, "file.upload_completed", "file", file.ID, map[string]any{
		"filename":  file.Filename,
		"file_size": file.FileSize,
	})

	return nil
}

func (s *FileService) AbortMultipartUpload(ctx context.Context, fileID, userID string, uploadID string) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil {
		return fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return fmt.Errorf("unauthorized")
	}

	_ = s.storage.AbortMultipartUpload(ctx, file.StorageKey, uploadID)

	s.logActivity(ctx, userID, "file.upload_aborted", "file", file.ID, map[string]any{
		"filename":  file.Filename,
		"file_size": file.FileSize,
	})

	return s.repo.UpdateStatus(ctx, fileID, "FAILED")
}

// RefreshMultipartPartURLs generates fresh presigned URLs for pending parts of an existing multipart upload.
// This is called when the client resumes after presigned URLs have expired.
func (s *FileService) RefreshMultipartPartURLs(ctx context.Context, fileID, userID, uploadID string, partNumbers []int32) ([]map[string]interface{}, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return nil, err
	}
	if file == nil {
		return nil, fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return nil, fmt.Errorf("unauthorized")
	}
	if file.Status != "UPLOADING" {
		return nil, fmt.Errorf("file is not in UPLOADING state")
	}

	var refreshedURLs []map[string]interface{}
	for _, partNum := range partNumbers {
		url, err := s.storage.GeneratePresignedUploadPartURL(ctx, file.StorageKey, uploadID, partNum, 30*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh presigned URL for part %d: %w", partNum, err)
		}
		refreshedURLs = append(refreshedURLs, map[string]interface{}{
			"part_number": partNum,
			"url":         url,
		})
	}

	s.logActivity(ctx, userID, "file.refresh_presigned_urls", "file", file.ID, map[string]any{
		"filename":  file.Filename,
		"file_size": file.FileSize,
	})

	return refreshedURLs, nil
}

// --- Validation Helpers ---

func DetectHEICMagicBytes(header []byte) bool {
	if len(header) < 12 {
		return false
	}
	if string(header[4:8]) != "ftyp" {
		return false
	}
	majorBrand := string(header[8:12])
	switch majorBrand {
	case "heic", "heix", "hevc", "hevx", "mif1", "msf1":
		return true
	default:
		return false
	}
}

func ValidateMagicBytes(detectedType, declaredType, fileName string) error {
	detectedType = strings.ToLower(strings.Split(detectedType, ";")[0])
	declaredType = strings.ToLower(strings.Split(declaredType, ";")[0])
	ext := strings.ToLower(filepath.Ext(fileName))

	if ext == ".heic" || ext == ".heif" || declaredType == "image/heic" || declaredType == "image/heif" {
		if detectedType == "application/octet-stream" || detectedType == "image/heic" || detectedType == "image/heif" {
			return nil
		}
	}

	// Block only dangerous executable payloads
	blockedExts := map[string]bool{
		".exe": true, ".bat": true, ".cmd": true, ".com": true,
		".msi": true, ".scr": true, ".pif": true, ".vbs": true, ".wsf": true,
	}
	if blockedExts[ext] {
		return fmt.Errorf("executable file type %s is not permitted in ByteVault", ext)
	}

	if !areTypesCompatible(detectedType, declaredType, ext) {
		return fmt.Errorf("extension spoofing detected: declared content type %s does not align with actual content type %s", declaredType, detectedType)
	}

	return nil
}

func areTypesCompatible(detected, declared, ext string) bool {
	if detected == declared {
		return true
	}

	// Apple HEIC / HEIF / Live Photo
	if (ext == ".heic" || ext == ".heif") && (declared == "image/heic" || declared == "image/heif" || detected == "image/heic" || detected == "image/heif") {
		return true
	}

	// Zip containers (DOCX, XLSX, PPTX, Pages, Numbers, Keynote, EPUB, JAR, etc.)
	if detected == "application/zip" {
		zipExts := map[string]bool{
			".docx": true, ".xlsx": true, ".pptx": true,
			".doc": true, ".xls": true, ".ppt": true,
			".pages": true, ".numbers": true, ".key": true, ".keynote": true,
			".epub": true, ".jar": true, ".apk": true, ".zip": true,
		}
		if zipExts[ext] || strings.Contains(declared, "zip") || strings.Contains(declared, "officedocument") || strings.Contains(declared, "iwork") || strings.Contains(declared, "apple") {
			return true
		}
	}

	// Text / Code / JSON / YAML / XML / Script / Developer files (http.DetectContentType sniffs text files as text/plain, text/html, or text/xml)
	isTextDetected := strings.HasPrefix(detected, "text/") || detected == "text/plain" || detected == "application/octet-stream"
	isTextOrCodeDeclared := strings.HasPrefix(declared, "text/") ||
		strings.Contains(declared, "json") ||
		strings.Contains(declared, "javascript") ||
		strings.Contains(declared, "typescript") ||
		strings.Contains(declared, "yaml") ||
		strings.Contains(declared, "xml") ||
		strings.Contains(declared, "sql") ||
		strings.Contains(declared, "graphql") ||
		strings.Contains(declared, "wasm") ||
		strings.Contains(declared, "toml") ||
		strings.Contains(declared, "protobuf")

	if isTextDetected && isTextOrCodeDeclared {
		return true
	}

	// Generic binary / octet-stream / data / media
	if detected == "application/octet-stream" || declared == "application/octet-stream" {
		blockedExts := map[string]bool{
			".exe": true, ".bat": true, ".cmd": true, ".com": true,
			".msi": true, ".scr": true, ".pif": true, ".vbs": true, ".wsf": true,
		}
		return !blockedExts[ext]
	}

	// Audio & Video container variations
	if (strings.HasPrefix(detected, "audio/") || strings.HasPrefix(detected, "video/")) &&
		(strings.HasPrefix(declared, "audio/") || strings.HasPrefix(declared, "video/")) {
		return true
	}

	return false
}

func (s *FileService) GetThumbnail(ctx context.Context, fileID, userID string) (string, *model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil || file == nil {
		return "", nil, fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return "", nil, fmt.Errorf("unauthorized")
	}

	if file.ThumbnailKey == nil || *file.ThumbnailKey == "" {
		return "", nil, fmt.Errorf("thumbnail not available")
	}

	url, err := s.storage.GeneratePresignedDownloadURL(ctx, *file.ThumbnailKey, 5*time.Minute, file.Filename, true)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate thumbnail presigned URL: %w", err)
	}

	return url, file, nil
}
