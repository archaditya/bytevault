package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
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

	return s.repo.UpdateStatus(ctx, fileID, "READY")
}

func (s *FileService) Upload(ctx context.Context, userID, filename string, size int64, contentType string, content io.Reader, folderID *string) (*model.File, error) {
	if err := s.validateFile(ctx, userID, size, contentType); err != nil {
		return nil, err
	}

	storageKey := s.generateStorageKey(userID, filename)

	_, err := s.storage.Upload(ctx, storageKey, content, size, contentType)
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

func (s *FileService) Download(ctx context.Context, fileID, userID string) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, fmt.Errorf("file not found")
	}
	if file.UserID != userID {
		return nil, nil, fmt.Errorf("unauthorized")
	}

	stream, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download from storage: %w", err)
	}

	return stream, file, nil
}

func (s *FileService) DownloadPublic(ctx context.Context, fileID string) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if file == nil {
		return nil, nil, fmt.Errorf("file not found")
	}
	if !file.IsPublic {
		return nil, nil, fmt.Errorf("unauthorized")
	}

	stream, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to download from storage: %w", err)
	}

	return stream, file, nil
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

	if err := s.repo.SoftDelete(ctx, fileID); err != nil {
		return err
	}

	_ = s.storage.Delete(ctx, file.StorageKey)
	return nil
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
