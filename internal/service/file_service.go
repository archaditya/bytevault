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
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/storage"
)

const (
	// DefaultQuotaBytes defines 1GB of storage quota per user
	DefaultQuotaBytes = 1 * 1024 * 1024 * 1024
	// MaxFileSizeLimit defines a 100MB limit for a single file upload
	MaxFileSizeLimit = 100 * 1024 * 1024
)

// Whitelisted allowed MIME types for storage
var AllowedMimeTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/gif":  true,
	"image/webp": true,
	"image/svg+xml": true,
	"application/pdf": true,
	"application/msword": true,
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document":   true,
	"application/vnd.ms-excel":                                                 true,
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet":         true,
	"application/vnd.ms-powerpoint":                                            true,
	"application/vnd.openxmlformats-officedocument.presentationml.presentation": true,
	"text/plain":                   true,
	"text/csv":                     true,
	"text/markdown":                true,
	"audio/mpeg":                   true,
	"audio/wav":                    true,
	"audio/ogg":                    true,
	"video/mp4":                    true,
	"video/mpeg":                   true,
	"video/quicktime":              true,
	"video/webm":                   true,
	"application/zip":              true,
	"application/x-tar":            true,
	"application/x-rar-compressed": true,
	"application/x-7z-compressed":  true,
}

type FileService struct {
	repo            *repository.FileRepository
	storage         storage.StorageProvider
	storageProvider string
	bucket          *string
}

func NewFileService(repo *repository.FileRepository, storage storage.StorageProvider, provider string, bucket string) *FileService {
	var bPtr *string
	if bucket != "" {
		bPtr = &bucket
	}
	return &FileService{
		repo:            repo,
		storage:         storage,
		storageProvider: provider,
		bucket:          bPtr,
	}
}

func (s *FileService) generateStorageKey(userID, filename string) string {
	return fmt.Sprintf("user/%s/docs/%s", userID, filepath.Base(filename))
}

func (s *FileService) validateFile(ctx context.Context, userID string, size int64, contentType string) error {
	// 1. Max File Size Validation
	if size > MaxFileSizeLimit {
		return fmt.Errorf("file size (%d bytes) exceeds the maximum allowed limit of 100MB", size)
	}

	// 2. MIME Type Validation
	if !AllowedMimeTypes[contentType] {
		return fmt.Errorf("unsupported file type: %s", contentType)
	}

	// 3. User Storage Quota Check
	used, err := s.repo.GetUserStorageUsed(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to fetch user storage usage: %w", err)
	}
	if used+size > DefaultQuotaBytes {
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

	// fetch first 512 bytes from storage for signature validation
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

	// verify magic bytes /signature compatibility
	if err := ValidateMagicBytes(detectedMime, file.ContentType, file.Filename); err != nil {
		_ = s.storage.Delete(ctx, file.StorageKey)
		_ = s.repo.UpdateStatus(ctx, fileID, "FAILED")
		return fmt.Errorf("upload rejected: %w", err)
	}

	return s.repo.UpdateStatus(ctx, fileID, "READY")
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
	if err := ValidateMagicBytes(detectedMime, contentType, filename); err != nil {
		return nil, err
	}

	// reconstruct stream using MuliReder to ensure all bytes are uploaded
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
		Status:          "READY",
		FolderID:        folderID,
	}

	if err := s.repo.Create(ctx, fileMeta); err != nil {
		_ = s.storage.Delete(ctx, storageKey)
		return nil, err
	}

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

	url, err := s.storage.GeneratePresignedDownloadURL(ctx, file.StorageKey, 30*time.Second, file.Filename, inline)

	if err != nil {
		return  "", nil, fmt.Errorf("failed to generate download URL: %w", err)
	}

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
	return s.repo.ListByUserID(ctx, params)
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

	return s.repo.MoveFile(ctx, fileID, folderID)
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


// ValidateMagicBytes verifies that the adcual bytes of the files matched the alloed types
func ValidateMagicBytes(detectedType, declaredType, fileName string) error {
	detectedType = strings.ToLower(strings.Split(detectedType, ";")[0])
	declaredType = strings.ToLower(strings.Split(declaredType, ";")[0])
	ext := strings.ToLower(filepath.Ext(fileName))

	isOfficeDoc := (declaredType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document" ||
		declaredType == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" ||
		declaredType == "application/vnd.openxmlformats-officedocument.presentationml.presentation")
	
	actualAllowedType := detectedType
	if detectedType == "application/zip" && isOfficeDoc {
		actualAllowedType = declaredType
	}

	if !AllowedMimeTypes[actualAllowedType] {
		return fmt.Errorf("detected MIME type %s is not permitted in ByteVault", detectedType)
	}

	if !areTypesCompatible(detectedType, declaredType, ext) {
		return fmt.Errorf("extention spoofing detected: declared content type %s does not align with actual content tyoe %s", declaredType, detectedType)
	}

	return nil
}

func areTypesCompatible(detected, declared, ext string) bool {
	if detected == declared {
		return true
	}

	if detected == "application/zip" {
		if declared == "application/zip" {
			return true
		}
		if ext == ".docx" || ext == ".xlsx" || ext == ".pptx" || ext == ".doc" || ext == ".xls" || ext == ".ppt" {
			return true
		}
	}

	if strings.HasPrefix(detected, "text/") && strings.HasPrefix(declared, "text/") {
		return true
	}
	if detected == "text/plain" && (declared == "text/markdown" || declared == "text/csv") {
		return true
	}

	if detected == "application/octet-stream" {
		blockedExts:= map[string]bool{
			".exe": true,
			".bat": true, 
			".sh": true, 
			".dll": true,
			".com": true, 
			".cmd": true, 
			".msi": true, 
			".scr": true,
		}

		return !blockedExts[ext]
	}

	return false
}