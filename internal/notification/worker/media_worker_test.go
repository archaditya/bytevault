package worker

import (
	"bytes"
	"context"
	"image/jpeg"
	"io"
	"testing"
	"time"

	"github.com/archaditya/bytevault/internal/model"
)

type mockWorkerStorage struct {
	content []byte
}

func (m *mockWorkerStorage) Upload(ctx context.Context, storageKey string, content io.Reader, size int64, contentType string) (string, error) {
	return storageKey, nil
}
func (m *mockWorkerStorage) Download(ctx context.Context, storageKey string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.content)), nil
}
func (m *mockWorkerStorage) Delete(ctx context.Context, storageKey string) error {
	return nil
}
func (m *mockWorkerStorage) GetObjectSize(ctx context.Context, storageKey string) (int64, error) {
	return int64(len(m.content)), nil
}
func (m *mockWorkerStorage) GeneratePresignedUploadURL(ctx context.Context, storageKey string, contentType string, expiry time.Duration) (string, error) {
	return "", nil
}
func (m *mockWorkerStorage) GeneratePresignedDownloadURL(ctx context.Context, storageKey string, expiry time.Duration, filename string, inline bool) (string, error) {
	return "", nil
}
func (m *mockWorkerStorage) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}
func (m *mockWorkerStorage) InitiateMultipartUpload(ctx context.Context, storageKey string, contentType string) (string, error) {
	return "", nil
}
func (m *mockWorkerStorage) GeneratePresignedUploadPartURL(ctx context.Context, storageKey string, uploadID string, partNumber int32, expiry time.Duration) (string, error) {
	return "", nil
}
func (m *mockWorkerStorage) CompleteMultipartUpload(ctx context.Context, storageKey string, uploadID string, parts []model.UploadPart) (string, error) {
	return "", nil
}
func (m *mockWorkerStorage) AbortMultipartUpload(ctx context.Context, storageKey string, uploadID string) error {
	return nil
}
func (m *mockWorkerStorage) AbortOrphanMultipartUploads(ctx context.Context, prefix string, olderThan time.Duration) error {
	return nil
}

func TestMediaWorker_ProcessVideo_MalformedAndNonVideoFallback(t *testing.T) {
	ctx := context.Background()

	// 1. Non-video extension (bypass attempt or incorrect handler call)
	mockStorageNonVideo := &mockWorkerStorage{content: []byte("sample text content")}
	wNonVideo := &MediaWorker{storage: mockStorageNonVideo}
	fileNonVideo := &model.File{
		ID:         "file-txt-1",
		Filename:   "document.txt",
		StorageKey: "files/document.txt",
	}

	poster, err := wNonVideo.processVideo(ctx, fileNonVideo)
	if err != nil {
		t.Fatalf("unexpected error on non-video file: %v", err)
	}
	if len(poster) == 0 {
		t.Fatal("expected non-empty fallback poster bytes")
	}
	cfg, err := jpeg.DecodeConfig(bytes.NewReader(poster))
	if err != nil {
		t.Fatalf("expected valid JPEG poster from document: %v", err)
	}
	if cfg.Width == 0 || cfg.Height == 0 {
		t.Fatalf("expected valid image dimensions, got %dx%d", cfg.Width, cfg.Height)
	}

	// 2. Malformed / HLS pseudo-mp4 containing potential SSRF or local file references
	hlsExploitPayload := []byte("#EXTM3U\n#EXT-X-VERSION:3\n#EXT-X-TARGETDURATION:10\n#EXTINF:10,0\nhttp://169.254.169.254/latest/meta-data/\n")
	mockStorageHLS := &mockWorkerStorage{content: hlsExploitPayload}
	wHLS := &MediaWorker{storage: mockStorageHLS}
	fileHLS := &model.File{
		ID:         "file-hls-exploit",
		Filename:   "exploit.mp4",
		StorageKey: "files/exploit.mp4",
	}

	hlsPoster, err := wHLS.processVideo(ctx, fileHLS)
	if err != nil {
		t.Fatalf("unexpected error when processing malformed video: %v", err)
	}
	if len(hlsPoster) == 0 {
		t.Fatal("expected fallback poster for malformed video")
	}
	cfgHLS, err := jpeg.DecodeConfig(bytes.NewReader(hlsPoster))
	if err != nil {
		t.Fatalf("expected valid JPEG poster fallback for malformed video: %v", err)
	}
	if cfgHLS.Width == 0 || cfgHLS.Height == 0 {
		t.Fatalf("expected valid fallback poster dimensions, got %dx%d", cfgHLS.Width, cfgHLS.Height)
	}

	// 3. Random garbage binary data with .mp4 extension
	garbageBytes := []byte{0x00, 0xFF, 0xFE, 0x41, 0x42, 0x43, 0x12, 0x34, 0x56}
	mockStorageGarbage := &mockWorkerStorage{content: garbageBytes}
	wGarbage := &MediaWorker{storage: mockStorageGarbage}
	fileGarbage := &model.File{
		ID:         "file-garbage",
		Filename:   "corrupt.mp4",
		StorageKey: "files/corrupt.mp4",
	}

	garbagePoster, err := wGarbage.processVideo(ctx, fileGarbage)
	if err != nil {
		t.Fatalf("unexpected error processing garbage video: %v", err)
	}
	if len(garbagePoster) == 0 {
		t.Fatal("expected fallback poster for garbage video")
	}
	cfgGarbage, err := jpeg.DecodeConfig(bytes.NewReader(garbagePoster))
	if err != nil {
		t.Fatalf("expected valid JPEG poster fallback for garbage video: %v", err)
	}
	if cfgGarbage.Width == 0 || cfgGarbage.Height == 0 {
		t.Fatalf("expected valid fallback poster dimensions, got %dx%d", cfgGarbage.Width, cfgGarbage.Height)
	}
}
