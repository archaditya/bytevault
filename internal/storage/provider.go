package storage

import (
	"context"
	"io"
	"time"
)

type StorageProvider interface {
	// Upload saves the file content and returns a public URL or identifier path
	Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error)
	// Download retrieves the binary data stream for the given storage key
	Download(ctx context.Context, storageKey string) (io.ReadCloser, error)
	// Delete removes the file physically
	Delete(ctx context.Context, storageKey string) error
	// GeneratePresignedUploadURL creates a temporary URL allowing clients to PUT/POST files directly
	GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error)
	// GeneratePresignedDownloadURL creates a secure temporary URL for downloading private assets
	GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration) (string, error)
}