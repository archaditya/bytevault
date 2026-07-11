package storage

import (
	"context"
	"io"
	"time"

	"github.com/archaditya/bytevault/internal/model"
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
	GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration, filename string, inline bool) (string, error)
	// List returns a list of all storage keys matching the given prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Multipart Upload
	InitiateMultipartUpload(ctx context.Context, storageKey string, contentType string) (string, error)
	GeneratePresignedUploadPartURL(ctx context.Context, storageKey string, uploadID string, partNumber int32, expiry time.Duration) (string, error)
	CompleteMultipartUpload(ctx context.Context, storageKey string, uploadID string, parts []model.UploadPart) (string, error)
	AbortMultipartUpload(ctx context.Context, storagekey string, uploadId string) error
}
