package service

import (
	"context"
	"fmt"
	"io"
	"path/filepath"

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

// generateStorageKey builds a structural key: user/{userID}/docs/{filename}
func (s *FileService) generateStorageKey(userID, filename string) string {
	return fmt.Sprintf("user/%s/docs/%s", userID, filepath.Base(filename))
}

func (s *FileService) Upload(ctx context.Context, userID, filename string, size int64, contentType string, content io.Reader) (*model.File, error) {
	storageKey := s.generateStorageKey(userID, filename)

	// 1. Upload file contents (discarding URL return value, error check only)
	_, err := s.storage.Upload(ctx, storageKey, content, size, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed upload in storage: %w", err)
	}

	// 2. Save metadata record to DB
	fileMeta := &model.File{
		UserID:          userID,
		Filename:        filename,
		StorageProvider: s.storageProvider,
		Bucket:          s.bucket,
		StorageKey:      storageKey,
		FileSize:        size,
		ContentType:     contentType,
		IsPublic:        false,
	}

	if err := s.repo.Create(ctx, fileMeta); err != nil {
		// Cleanup storage on DB failure
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
