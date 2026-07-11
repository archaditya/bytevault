package storage

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/archaditya/bytevault/internal/model"
)

type PrefixedStorageProvider struct {
	prefix string
	parent StorageProvider
}

// NewPrefixedStorageProvider wraps an existing StorageProvider and prefixes all storage keys.
func NewPrefixedStorageProvider(prefix string, parent StorageProvider) StorageProvider {
	if prefix == "" {
		return parent
	}
	// Ensure the prefix ends with a slash
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return &PrefixedStorageProvider{
		prefix: prefix,
		parent: parent,
	}
}

func (p *PrefixedStorageProvider) Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error) {
	return p.parent.Upload(ctx, p.prefix+storageKey, content, size, contentType)
}

func (p *PrefixedStorageProvider) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	return p.parent.Download(ctx, p.prefix+storageKey)
}

func (p *PrefixedStorageProvider) Delete(ctx context.Context, storageKey string) error {
	return p.parent.Delete(ctx, p.prefix+storageKey)
}

func (p *PrefixedStorageProvider) GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error) {
	return p.parent.GeneratePresignedUploadURL(ctx, p.prefix+storageKey, contentType, expiry)
}

func (p *PrefixedStorageProvider) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration, filename string, inline bool) (string, error) {
	return p.parent.GeneratePresignedDownloadURL(ctx, p.prefix+storageKey, expiry, filename, inline)
}

func (p *PrefixedStorageProvider) List(ctx context.Context, prefixArg string) ([]string, error) {
	keys, err := p.parent.List(ctx, p.prefix+prefixArg)
	if err != nil {
		return nil, err
	}
	// Strip the prefix from keys returned to the scheduler/caller
	stripped := make([]string, 0, len(keys))
	for _, key := range keys {
		if strings.HasPrefix(key, p.prefix) {
			stripped = append(stripped, key[len(p.prefix):])
		} else {
			stripped = append(stripped, key)
		}
	}
	return stripped, nil
}

func (p *PrefixedStorageProvider) InitiateMultipartUpload(ctx context.Context, storageKey string, contentType string) (string, error) {
	return p.parent.InitiateMultipartUpload(ctx, p.prefix+storageKey, contentType)
}

func (p *PrefixedStorageProvider) GeneratePresignedUploadPartURL(ctx context.Context, storageKey string, uploadID string, partNumber int32, expiry time.Duration) (string, error) {
	return p.parent.GeneratePresignedUploadPartURL(ctx, p.prefix+storageKey, uploadID, partNumber, expiry)
}

func (p *PrefixedStorageProvider) CompleteMultipartUpload(ctx context.Context, storageKey string, uploadID string, parts []model.UploadPart) (string, error) {
	return p.parent.CompleteMultipartUpload(ctx, p.prefix+storageKey, uploadID, parts)
}

func (p *PrefixedStorageProvider) AbortMultipartUpload(ctx context.Context, storageKey string, uploadID string) error {
	return p.parent.AbortMultipartUpload(ctx, p.prefix+storageKey, uploadID)
}
