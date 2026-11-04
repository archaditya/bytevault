package service_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/service"
	"github.com/archaditya/bytevault/internal/storage"
)

// mockStorageProvider implements storage.StorageProvider for unit testing.
type mockStorageProvider struct {
	objectSize int64
	deletedKey string
}

func (m *mockStorageProvider) Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error) {
	return storageKey, nil
}
func (m *mockStorageProvider) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	return nil, nil
}
func (m *mockStorageProvider) Delete(ctx context.Context, storageKey string) error {
	m.deletedKey = storageKey
	return nil
}
func (m *mockStorageProvider) GetObjectSize(ctx context.Context, storageKey string) (int64, error) {
	return m.objectSize, nil
}
func (m *mockStorageProvider) GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error) {
	return "https://mock.storage/upload", nil
}
func (m *mockStorageProvider) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration, filename string, inline bool) (string, error) {
	return "https://mock.storage/download", nil
}
func (m *mockStorageProvider) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}
func (m *mockStorageProvider) InitiateMultipartUpload(ctx context.Context, storageKey string, contentType string) (string, error) {
	return "upload-123", nil
}
func (m *mockStorageProvider) GeneratePresignedUploadPartURL(ctx context.Context, storageKey string, uploadID string, partNumber int32, expiry time.Duration) (string, error) {
	return "https://mock.storage/part", nil
}
func (m *mockStorageProvider) CompleteMultipartUpload(ctx context.Context, storageKey string, uploadID string, parts []model.UploadPart) (string, error) {
	return "https://mock.storage/completed", nil
}
func (m *mockStorageProvider) AbortMultipartUpload(ctx context.Context, storageKey string, uploadID string) error {
	return nil
}
func (m *mockStorageProvider) AbortOrphanMultipartUploads(ctx context.Context, prefix string, olderThan time.Duration) error {
	return nil
}

func TestEphemeralMetadataDTO_Sanitization(t *testing.T) {
	senderIP := "198.51.100.25"
	senderUA := "Mozilla/5.0 SecretBrowser"
	passwordHash := "hashed-secret"
	uploadID := "upload-xyz"

	share := &model.EphemeralShare{
		ID:            "uuid-1234-5678",
		Token:         "tok-abcdef",
		StorageKey:    "ephemeral/tok-abcdef/private_doc.pdf",
		UploadID:      &uploadID,
		Filename:      "private_doc.pdf",
		FileSize:      2048,
		ContentType:   "application/pdf",
		Status:        "ACTIVE",
		MaxDownloads:  1,
		DownloadCount: 0,
		IPAddress:     &senderIP,
		UserAgent:     &senderUA,
		HasPassword:   true,
		PasswordHash:  &passwordHash,
		ExpiresAt:     time.Now().Add(1 * time.Hour),
		CreatedAt:     time.Now(),
		IsLive:        false,
	}

	dto := share.ToMetadataDTO()
	if dto == nil {
		t.Fatal("expected non-nil DTO")
	}

	// Verify public fields are preserved
	if dto.Token != share.Token || dto.Filename != share.Filename || dto.FileSize != share.FileSize {
		t.Errorf("public fields not preserved in DTO: %+v", dto)
	}

	// Ensure the DTO does not leak internal DB ID, StorageKey, IP, or UA
	// (verified statically by DTO struct definition not possessing those fields)
}

func TestStorageProvider_PrefixedAndMock(t *testing.T) {
	mock := &mockStorageProvider{objectSize: 4096}
	prefixed := storage.NewPrefixedStorageProvider("test-prefix", mock)

	size, err := prefixed.GetObjectSize(context.Background(), "file.bin")
	if err != nil {
		t.Fatalf("unexpected error from prefixed GetObjectSize: %v", err)
	}
	if size != 4096 {
		t.Fatalf("expected size 4096, got %d", size)
	}
}

func TestEphemeralService_NilSafe(t *testing.T) {
	var nilShare *model.EphemeralShare
	dto := nilShare.ToMetadataDTO()
	if dto != nil {
		t.Errorf("expected nil DTO for nil share, got %+v", dto)
	}
	_ = errors.New("check")
}

type mockEphemeralStore struct {
	share         *model.EphemeralShare
	updatedStatus string
}

func (m *mockEphemeralStore) CountUploadsByIPInWindow(ctx context.Context, ip string) (int, error) {
	return 0, nil
}
func (m *mockEphemeralStore) Create(ctx context.Context, share *model.EphemeralShare) error {
	return nil
}
func (m *mockEphemeralStore) FindByToken(ctx context.Context, token string) (*model.EphemeralShare, error) {
	if m.share == nil {
		return nil, repository.ErrEphemeralNotFound
	}
	return m.share, nil
}
func (m *mockEphemeralStore) UpdateStatus(ctx context.Context, token string, status string) error {
	m.updatedStatus = status
	if m.share != nil {
		m.share.Status = status
	}
	return nil
}
func (m *mockEphemeralStore) DeleteByToken(ctx context.Context, token string) error {
	return nil
}
func (m *mockEphemeralStore) IncrementDownloadAndCheckBurn(ctx context.Context, id string) (int, bool, error) {
	if m.share != nil {
		m.share.DownloadCount++
		if m.share.MaxDownloads > 0 && m.share.DownloadCount >= m.share.MaxDownloads {
			return m.share.DownloadCount, true, nil
		}
		return m.share.DownloadCount, false, nil
	}
	return 0, false, nil
}
func (m *mockEphemeralStore) MarkBurned(ctx context.Context, id string) error {
	if m.share != nil {
		m.share.Status = "BURNED"
	}
	return nil
}
func (m *mockEphemeralStore) ListAllForAdmin(ctx context.Context, limit, offset int) ([]*model.EphemeralShare, int, error) {
	return nil, 0, nil
}

func TestEphemeralService_OversizedUploadRejectionAndPurge(t *testing.T) {
	ctx := context.Background()
	uploadID := "upload-test-123"

	// Attacker claims 1-byte file size when starting upload
	share := &model.EphemeralShare{
		ID:          "ephem-1",
		Token:       "tok-attack-1",
		StorageKey:  "ephemeral/tok-attack-1/bomb.bin",
		UploadID:    &uploadID,
		Filename:    "bomb.bin",
		FileSize:    1, // declared 1 byte!
		ContentType: "application/octet-stream",
		Status:      "UPLOADING",
		ExpiresAt:   time.Now().Add(1 * time.Hour),
	}

	mockStore := &mockEphemeralStore{share: share}
	// Storage provider returns real object size: 10 MB (violating claimed 1 byte)
	mockStorage := &mockStorageProvider{objectSize: 10 * 1024 * 1024}

	svc := service.NewEphemeralService(mockStore, nil, mockStorage)

	err := svc.CompleteMultipartUpload(ctx, "tok-attack-1", uploadID, nil)
	if err == nil {
		t.Fatal("expected CompleteMultipartUpload to reject oversized upload, got nil error")
	}
	if !strings.Contains(err.Error(), "exceeds declared size") {
		t.Fatalf("expected error mentioning 'exceeds declared size', got: %v", err)
	}

	// Verify defense: uploaded object in storage was purged
	if mockStorage.deletedKey != share.StorageKey {
		t.Fatalf("expected storage object '%s' to be deleted, got '%s'", share.StorageKey, mockStorage.deletedKey)
	}

	// Verify defense: database status was updated to BURNED
	if mockStore.updatedStatus != "BURNED" {
		t.Fatalf("expected share status to be set to 'BURNED', got '%s'", mockStore.updatedStatus)
	}
}

func TestEphemeralService_SingleUsePreviewSuppression(t *testing.T) {
	ctx := context.Background()

	// 1. Single-use share (MaxDownloads = 1) MUST NOT have PreviewURL
	singleUseShare := &model.EphemeralShare{
		ID:           "ephem-single",
		Token:        "tok-single",
		StorageKey:   "ephemeral/tok-single/photo.png",
		Filename:     "photo.png",
		FileSize:     1024,
		ContentType:  "image/png",
		Status:       "ACTIVE",
		MaxDownloads: 1, // Single-use!
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	mockStoreSingle := &mockEphemeralStore{share: singleUseShare}
	mockStorage := &mockStorageProvider{objectSize: 1024}
	svcSingle := service.NewEphemeralService(mockStoreSingle, nil, mockStorage)

	retrievedSingle, err := svcSingle.GetMetadata(ctx, "tok-single")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrievedSingle.PreviewURL != nil {
		t.Fatalf("expected PreviewURL to be suppressed for single-use share, got: %s", *retrievedSingle.PreviewURL)
	}

	// 2. Multi-use share (MaxDownloads = 5) CAN generate PreviewURL
	multiUseShare := &model.EphemeralShare{
		ID:           "ephem-multi",
		Token:        "tok-multi",
		StorageKey:   "ephemeral/tok-multi/photo.png",
		Filename:     "photo.png",
		FileSize:     1024,
		ContentType:  "image/png",
		Status:       "ACTIVE",
		MaxDownloads: 5, // Multi-use
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	mockStoreMulti := &mockEphemeralStore{share: multiUseShare}
	svcMulti := service.NewEphemeralService(mockStoreMulti, nil, mockStorage)

	retrievedMulti, err := svcMulti.GetMetadata(ctx, "tok-multi")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if retrievedMulti.PreviewURL == nil {
		t.Fatal("expected PreviewURL to be generated for multi-use share, got nil")
	}
}
