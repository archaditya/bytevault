package local

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

type LocalStorage struct {
	baseDir string
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create base upload directory: %w", err)
	}
	return &LocalStorage{baseDir: baseDir}, nil
}

func (l *LocalStorage) getAbsPath(storageKey string) string {
	return filepath.Join(l.baseDir, storageKey)
}

func (l *LocalStorage) Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error) {
	fullPath := l.getAbsPath(storageKey)
	
	// Ensure directory nesting exists
	if err := os.MkdirAll(filepath.Dir(fullPath), os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create subdirectories: %w", err)
	}

	dest, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, content); err != nil {
		return "", fmt.Errorf("failed to write local file: %w", err)
	}

	return fullPath, nil
}

func (l *LocalStorage) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	fullPath := l.getAbsPath(storageKey)
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open local file: %w", err)
	}
	return file, nil
}

func (l *LocalStorage) Delete(ctx context.Context, storageKey string) error {
	fullPath := l.getAbsPath(storageKey)
	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to delete local file: %w", err)
	}
	return nil
}

func (l *LocalStorage) GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error) {
	// Dev environment shortcut: direct local upload route
	return fmt.Sprintf("/api/v1/files/upload/direct?key=%s", storageKey), nil
}

func (l *LocalStorage) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration) (string, error) {
	// Dev environment shortcut: direct file access route
	return fmt.Sprintf("/api/v1/files/download/direct?key=%s", storageKey), nil
}