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
	sizeErr    error
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
	if m.sizeErr != nil {
		return 0, m.sizeErr
	}
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

func TestEphemeralService_SingleUsePreviewAllowed(t *testing.T) {
	ctx := context.Background()

	// 1. Single-use share (MaxDownloads = 1) CAN generate PreviewURL for streaming before download
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
	if retrievedSingle.PreviewURL == nil {
		t.Fatal("expected PreviewURL to be available for single-use share preview streaming")
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

func TestEphemeralService_SizeErrorFailClosed(t *testing.T) {
	ctx := context.Background()

	uploadID := "upload-id"
	share := &model.EphemeralShare{
		ID:           "ephem-size-err",
		Token:        "tok-size-err",
		StorageKey:   "ephemeral/tok-size-err/payload.bin",
		Filename:     "payload.bin",
		FileSize:     1024,
		ContentType:  "application/octet-stream",
		Status:       "UPLOADING",
		UploadID:     &uploadID,
		MaxDownloads: 1,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	mockStore := &mockEphemeralStore{share: share}
	// Storage provider returns an error on GetObjectSize (e.g. S3 timeout)
	mockStorage := &mockStorageProvider{
		sizeErr: errors.New("s3 connection reset by peer"),
	}
	svc := service.NewEphemeralService(mockStore, nil, mockStorage)

	// 1. CompleteMultipartUpload on storage error: must fail-closed (return error) but be non-destructive (no delete, no burn)
	err := svc.CompleteMultipartUpload(ctx, "tok-size-err", "upload-id", []model.UploadPart{{PartNumber: 1, ETag: "etag"}})
	if err == nil {
		t.Fatal("expected error on size lookup failure, got nil")
	}
	if mockStorage.deletedKey != "" {
		t.Fatalf("expected storage object NOT to be deleted on transient storage error, got '%s'", mockStorage.deletedKey)
	}
	if mockStore.updatedStatus == "BURNED" {
		t.Fatal("expected share status NOT to be BURNED on transient storage error")
	}

	// 2. RequestDownload on storage error: must fail-closed (return error) but be non-destructive (no delete, no burn)
	activeShare := &model.EphemeralShare{
		ID:           "ephem-download-err",
		Token:        "tok-download-err",
		StorageKey:   "ephemeral/tok-download-err/payload.bin",
		Filename:     "payload.bin",
		FileSize:     1024,
		ContentType:  "application/octet-stream",
		Status:       "ACTIVE",
		MaxDownloads: 1,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	mockStoreActive := &mockEphemeralStore{share: activeShare}
	mockStorageActive := &mockStorageProvider{
		sizeErr: errors.New("timeout querying storage"),
	}
	svcActive := service.NewEphemeralService(mockStoreActive, nil, mockStorageActive)

	_, dlErr := svcActive.RequestDownload(ctx, "tok-download-err", nil)
	if dlErr == nil {
		t.Fatal("expected RequestDownload to fail on size lookup error, got nil")
	}
	if mockStorageActive.deletedKey != "" {
		t.Fatalf("expected storage object NOT to be deleted on download transient size error, got '%s'", mockStorageActive.deletedKey)
	}
	if mockStoreActive.updatedStatus == "BURNED" {
		t.Fatal("expected download share status NOT to be BURNED on transient size error")
	}

	// 3. Confirmed size violation: actual size > declared size MUST delete object and burn share
	mockStoreOversized := &mockEphemeralStore{share: activeShare}
	mockStorageOversized := &mockStorageProvider{
		objectSize: 5000, // declared is 1024
	}
	svcOversized := service.NewEphemeralService(mockStoreOversized, nil, mockStorageOversized)
	_, dlErrOver := svcOversized.RequestDownload(ctx, "tok-download-err", nil)
	if dlErrOver == nil {
		t.Fatal("expected RequestDownload to fail on oversized file, got nil")
	}
	if mockStorageOversized.deletedKey != activeShare.StorageKey {
		t.Fatalf("expected oversized object '%s' to be deleted, got '%s'", activeShare.StorageKey, mockStorageOversized.deletedKey)
	}
	if mockStoreOversized.updatedStatus != "BURNED" {
		t.Fatalf("expected oversized share status to be BURNED, got '%s'", mockStoreOversized.updatedStatus)
	}
}

func TestEphemeralService_GetMetadata_PreviewURL(t *testing.T) {
	ctx := context.Background()

	// 1. Single-use MP4 file <= 500MB MUST generate PreviewURL
	mp4Share := &model.EphemeralShare{
		Token:        "tok-video",
		Filename:     "vacation.mp4",
		ContentType:  "video/mp4",
		FileSize:     150 * 1024 * 1024, // 150MB
		Status:       "ACTIVE",
		MaxDownloads: 1, // Single-use!
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	mockStore := &mockEphemeralStore{share: mp4Share}
	mockStorage := &mockStorageProvider{}
	svc := service.NewEphemeralService(mockStore, nil, mockStorage)

	res, err := svc.GetMetadata(ctx, "tok-video")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.PreviewURL == nil || *res.PreviewURL != "https://mock.storage/download" {
		t.Fatalf("expected PreviewURL to be generated for MP4 video, got %v", res.PreviewURL)
	}

	// 2. Large image (e.g. 25MB) MUST generate PreviewURL
	imgShare := &model.EphemeralShare{
		Token:        "tok-img",
		Filename:     "poster.png",
		ContentType:  "image/png",
		FileSize:     25 * 1024 * 1024,
		Status:       "ACTIVE",
		MaxDownloads: 1,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	mockStore.share = imgShare
	resImg, err := svc.GetMetadata(ctx, "tok-img")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resImg.PreviewURL == nil {
		t.Fatal("expected PreviewURL for 25MB image")
	}

	// 3. Password-protected share MUST NOT generate PreviewURL
	pwd := "hashed"
	pwdShare := &model.EphemeralShare{
		Token:        "tok-pwd",
		Filename:     "secret.mp4",
		ContentType:  "video/mp4",
		FileSize:     1024,
		Status:       "ACTIVE",
		PasswordHash: &pwd,
		ExpiresAt:    time.Now().Add(1 * time.Hour),
	}
	mockStore.share = pwdShare
	resPwd, err := svc.GetMetadata(ctx, "tok-pwd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resPwd.PreviewURL != nil {
		t.Fatalf("expected nil PreviewURL for password-protected share, got %v", resPwd.PreviewURL)
	}
}

func TestEphemeralService_TransitionState_Validation(t *testing.T) {
	ctx := context.Background()

	share := &model.EphemeralShare{
		Token:  "tok-fsm",
		Status: string(model.EphemeralStateInitiated),
	}
	mockStore := &mockEphemeralStore{share: share}
	svc := service.NewEphemeralService(mockStore, nil, &mockStorageProvider{})

	// 1. Legal transition: INITIATED -> UPLOADING
	if err := svc.TransitionState(ctx, "tok-fsm", model.EphemeralStateUploading); err != nil {
		t.Fatalf("expected legal transition INITIATED -> UPLOADING to succeed, got: %v", err)
	}
	if share.Status != string(model.EphemeralStateUploading) {
		t.Fatalf("expected status UPLOADING, got: %s", share.Status)
	}

	// 2. Legal transition: UPLOADING -> ACTIVE
	if err := svc.TransitionState(ctx, "tok-fsm", model.EphemeralStateActive); err != nil {
		t.Fatalf("expected legal transition UPLOADING -> ACTIVE to succeed, got: %v", err)
	}

	// 3. Legal transition: ACTIVE -> BURNED
	if err := svc.TransitionState(ctx, "tok-fsm", model.EphemeralStateBurned); err != nil {
		t.Fatalf("expected legal transition ACTIVE -> BURNED to succeed, got: %v", err)
	}

	// 4. Illegal transition: BURNED -> ACTIVE (cannot un-burn)
	if err := svc.TransitionState(ctx, "tok-fsm", model.EphemeralStateActive); err == nil {
		t.Fatal("expected illegal transition BURNED -> ACTIVE to fail, got nil")
	}

	// 5. Illegal transition: BURNED -> UPLOADING
	if err := svc.TransitionState(ctx, "tok-fsm", model.EphemeralStateUploading); err == nil {
		t.Fatal("expected illegal transition BURNED -> UPLOADING to fail, got nil")
	}

	// 6. Legal transition: BURNED -> PURGED (terminal)
	if err := svc.TransitionState(ctx, "tok-fsm", model.EphemeralStatePurged); err != nil {
		t.Fatalf("expected legal transition BURNED -> PURGED to succeed, got: %v", err)
	}

	// 7. Illegal transition: PURGED -> anything
	if err := svc.TransitionState(ctx, "tok-fsm", model.EphemeralStateInitiated); err == nil {
		t.Fatal("expected illegal transition PURGED -> INITIATED to fail, got nil")
	}
}
