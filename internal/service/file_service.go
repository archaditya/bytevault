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

// CreateUploadSession creates a file record in UPLOADING state and generates a presigned URL
func (s *FileService) CreateUploadSession(ctx context.Context, userID, filename string, size int64, contentType string) (*model.File, string, error) {
	storageKey := s.generateStorageKey(userID, filename)

	// 1. Generate presigned PUT upload URL (expires in 15 mins)
	uploadURL, err := s.storage.GeneratePresignedUploadURL(ctx, storageKey, contentType, 15*time.Minute)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	// 2. Insert metadata record in UPLOADING state
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
	}

	if err := s.repo.Create(ctx, fileMeta); err != nil {
		return nil, "", err
	}

	return fileMeta, uploadURL, nil
}

// CompleteUpload marks the file record as READY
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

	// Move status from UPLOADING to READY
	return s.repo.UpdateStatus(ctx, fileID, "READY")
}

// Fallback upload (Multipart direct through backend)
func (s *FileService) Upload(ctx context.Context, userID, filename string, size int64, contentType string, content io.Reader) (*model.File, error) {
	storageKey := s.generateStorageKey(userID, filename)

	_, err := s.storage.Upload(ctx, storageKey, content, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed upload in storage: %w", err)
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

	if !file.IsPublic && file.UserID != userID {
		return nil, nil, fmt.Errorf("unauthorized to download this file")
	}

	stream, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to acquire stream: %w", err)
	}

	return stream, file, nil
}

func (s *FileService) DownloadPublic(ctx context.Context, fileID string) (io.ReadCloser, *model.File, error) {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return nil, nil, err
	}
	if file == nil || !file.IsPublic {
		return nil, nil, fmt.Errorf("file not found or private")
	}

	stream, err := s.storage.Download(ctx, file.StorageKey)
	if err != nil {
		return nil, nil, err
	}

	return stream, file, nil
}

func (s *FileService) ListUserFiles(ctx context.Context, userID string) ([]*model.File, error) {
	return s.repo.ListByUserID(ctx, userID)
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

func (s *FileService) Delete(ctx context.Context, fileID, userID string) error {
	file, err := s.repo.FindByID(ctx, fileID)
	if err != nil {
		return err
	}
	if file == nil || file.UserID != userID {
		return fmt.Errorf("file not found or unauthorized")
	}

	if err := s.storage.Delete(ctx, file.StorageKey); err != nil {
		return err
	}

	return s.repo.SoftDelete(ctx, fileID)
}
