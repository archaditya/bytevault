package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type LiveProgress struct {
	ProgressPct   int
	UploadedBytes int64
	UpdatedAt     time.Time
}

type EphemeralShareStore interface {
	CountUploadsByIPInWindow(ctx context.Context, ip string) (int, error)
	Create(ctx context.Context, share *model.EphemeralShare) error
	FindByToken(ctx context.Context, token string) (*model.EphemeralShare, error)
	UpdateStatus(ctx context.Context, token string, status string) error
	DeleteByToken(ctx context.Context, token string) error
	IncrementDownloadAndCheckBurn(ctx context.Context, id string) (int, bool, error)
	MarkBurned(ctx context.Context, id string) error
	ListAllForAdmin(ctx context.Context, limit, offset int) ([]*model.EphemeralShare, int, error)
}

type EphemeralService struct {
	repo          EphemeralShareStore
	settingRepo   *repository.EphemeralSettingRepository
	storage       storage.StorageProvider
	bandwidthRepo *repository.BandwidthRepository
	progressMu    sync.RWMutex
	progressMap   map[string]LiveProgress
}

func (s *EphemeralService) SetBandwidthRepo(repo *repository.BandwidthRepository) {
	s.bandwidthRepo = repo
}

func NewEphemeralService(repo EphemeralShareStore, settingRepo *repository.EphemeralSettingRepository, storage storage.StorageProvider) *EphemeralService {
	return &EphemeralService{
		repo:        repo,
		settingRepo: settingRepo,
		storage:     storage,
		progressMap: make(map[string]LiveProgress),
	}
}

func (s *EphemeralService) GetSettings(ctx context.Context) (*repository.EphemeralSettings, error) {
	return s.settingRepo.GetSettings(ctx)
}

func (s *EphemeralService) UpdateSettings(ctx context.Context, cfg *repository.EphemeralSettings) error {
	return s.settingRepo.UpdateSettings(ctx, cfg)
}

func (s *EphemeralService) CreateUploadSession(ctx context.Context, filename string, size int64, contentType string, password *string, ip, ua *string) (*model.EphemeralShare, string, error) {
	// Read current system governance settings from DB
	settings, err := s.settingRepo.GetSettings(ctx)
	if err != nil || settings == nil {
		settings = &repository.EphemeralSettings{MaxFileSizeGb: 2, MaxDownloads: 1, RateLimit24h: 50, ExpiryMinutes: 60}
	}

	maxSizeBytes := int64(settings.MaxFileSizeGb * 1024 * 1024 * 1024)
	if size > maxSizeBytes {
		return nil, "", fmt.Errorf("file size exceeds current limit of %.1f GB", settings.MaxFileSizeGb)
	}

	if ip != nil && *ip != "" {
		count, err := s.repo.CountUploadsByIPInWindow(ctx, *ip)
		if err == nil && count >= settings.RateLimit24h {
			return nil, "", fmt.Errorf("rate limit reached: maximum %d uploads per 24 hours per IP address", settings.RateLimit24h)
		}
	}

	ttl := time.Duration(settings.ExpiryMinutes) * time.Minute

	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	storageKey := fmt.Sprintf("ephemeral/%s/%s", token, filepath.Base(filename))

	var passwordHash *string
	if password != nil && *password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), 10)
		if err == nil {
			hStr := string(hash)
			passwordHash = &hStr
		}
	}

	uploadURL, err := s.storage.GeneratePresignedUploadURL(ctx, storageKey, contentType, 60*time.Minute)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	share := &model.EphemeralShare{
		Token:        token,
		StorageKey:   storageKey,
		Filename:     filename,
		FileSize:     size,
		ContentType:  contentType,
		Status:       "ACTIVE",
		MaxDownloads: settings.MaxDownloads,
		IPAddress:    ip,
		UserAgent:    ua,
		PasswordHash: passwordHash,
		ExpiresAt:    time.Now().Add(ttl),
		IsLive:       false,
	}

	if err := s.repo.Create(ctx, share); err != nil {
		return nil, "", err
	}

	return share, uploadURL, nil
}

// CreateMultipartUploadSession initiates a multi-part upload for large ephemeral files.
func (s *EphemeralService) CreateMultipartUploadSession(
	ctx context.Context,
	filename string,
	size int64,
	contentType string,
	password *string,
	ip, ua *string,
	partCount int,
	isLive bool,
) (*model.EphemeralShare, string, []map[string]any, error) {
	settings, err := s.GetSettings(ctx)
	if err != nil {
		settings = &repository.EphemeralSettings{MaxFileSizeGb: 2, MaxDownloads: 1, RateLimit24h: 50, ExpiryMinutes: 60}
	}

	maxBytes := int64(settings.MaxFileSizeGb) * 1024 * 1024 * 1024
	if size > maxBytes {
		return nil, "", nil, fmt.Errorf("file size exceeds maximum allowed guest limit of %d GB", int(settings.MaxFileSizeGb))
	}

	if ip != nil && *ip != "" {
		count, err := s.repo.CountUploadsByIPInWindow(ctx, *ip)
		if err == nil && count >= settings.RateLimit24h {
			return nil, "", nil, fmt.Errorf("rate limit reached: maximum %d uploads per 24 hours per IP address", settings.RateLimit24h)
		}
	}

	// Validate partCount against maximum upper bound to prevent DoS via excessive presigned URL generation
	const MaxPartsAllowed = 2000
	if partCount <= 0 || partCount > MaxPartsAllowed {
		return nil, "", nil, fmt.Errorf("part_count must be between 1 and %d", MaxPartsAllowed)
	}

	ttl := time.Duration(settings.ExpiryMinutes) * time.Minute

	tokenBytes := make([]byte, 16)
	_, _ = rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	cleanFilename := filepath.Base(filename)
	storageKey := fmt.Sprintf("ephemeral/%s/%s", token, cleanFilename)

	var passwordHash *string
	if password != nil && *password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(*password), 10)
		if err == nil {
			hStr := string(hash)
			passwordHash = &hStr
		}
	}

	uploadID, err := s.storage.InitiateMultipartUpload(ctx, storageKey, contentType)
	if err != nil {
		return nil, "", nil, fmt.Errorf("failed to initiate multipart upload: %w", err)
	}

	var partURLs []map[string]any
	for i := 1; i <= partCount; i++ {
		url, err := s.storage.GeneratePresignedUploadPartURL(ctx, storageKey, uploadID, int32(i), 60*time.Minute)
		if err != nil {
			_ = s.storage.AbortMultipartUpload(ctx, storageKey, uploadID)
			return nil, "", nil, fmt.Errorf("failed to generate presigned URL for part %d: %w", i, err)
		}
		partURLs = append(partURLs, map[string]any{
			"part_number": i,
			"url":         url,
		})
	}

	share := &model.EphemeralShare{
		Token:        token,
		StorageKey:   storageKey,
		UploadID:     &uploadID,
		Filename:     filename,
		FileSize:     size,
		ContentType:  contentType,
		Status:       "UPLOADING",
		MaxDownloads: settings.MaxDownloads,
		IPAddress:    ip,
		UserAgent:    ua,
		PasswordHash: passwordHash,
		ExpiresAt:    time.Now().Add(ttl),
		IsLive:       isLive,
	}

	if err := s.repo.Create(ctx, share); err != nil {
		_ = s.storage.AbortMultipartUpload(ctx, storageKey, uploadID)
		return nil, "", nil, err
	}

	return share, uploadID, partURLs, nil
}

// RefreshMultipartPartURLs generates fresh presigned URLs for pending parts of an active ephemeral upload.
func (s *EphemeralService) RefreshMultipartPartURLs(ctx context.Context, token string, uploadID string, partNumbers []int32) ([]map[string]any, error) {
	share, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}

	if share.Status != "UPLOADING" || time.Now().After(share.ExpiresAt) {
		return nil, fmt.Errorf("upload session is not active or has expired")
	}
	if share.UploadID == nil || *share.UploadID == "" || *share.UploadID != uploadID {
		return nil, fmt.Errorf("invalid upload session")
	}

	const MaxPartsAllowed = 2000
	if len(partNumbers) == 0 || len(partNumbers) > MaxPartsAllowed {
		return nil, fmt.Errorf("part_numbers count must be between 1 and %d", MaxPartsAllowed)
	}

	var refreshedURLs []map[string]any
	seen := make(map[int32]struct{}, len(partNumbers))
	for _, partNum := range partNumbers {
		if partNum <= 0 || int(partNum) > MaxPartsAllowed {
			return nil, fmt.Errorf("invalid part number: %d", partNum)
		}
		if _, duplicate := seen[partNum]; duplicate {
			continue // Deduplicate redundant part requests
		}
		seen[partNum] = struct{}{}

		url, err := s.storage.GeneratePresignedUploadPartURL(ctx, share.StorageKey, uploadID, partNum, 60*time.Minute)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh presigned URL for part %d: %w", partNum, err)
		}
		refreshedURLs = append(refreshedURLs, map[string]any{
			"part_number": partNum,
			"url":         url,
		})
	}

	return refreshedURLs, nil
}

// UpdateLiveProgress records the streaming upload progress from the sender in-memory.
func (s *EphemeralService) UpdateLiveProgress(ctx context.Context, token string, pct int, bytes int64) error {
	share, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return err
	}

	if share.Status != "UPLOADING" || time.Now().After(share.ExpiresAt) {
		return fmt.Errorf("upload session is not active")
	}

	if pct < 0 {
		pct = 0
	} else if pct > 100 {
		pct = 100
	}
	if bytes < 0 {
		bytes = 0
	} else if bytes > share.FileSize {
		bytes = share.FileSize
	}

	s.progressMu.Lock()
	s.progressMap[token] = LiveProgress{
		ProgressPct:   pct,
		UploadedBytes: bytes,
		UpdatedAt:     time.Now(),
	}
	s.progressMu.Unlock()
	return nil
}

func (s *EphemeralService) CompleteMultipartUpload(ctx context.Context, token string, uploadID string, parts []model.UploadPart) error {
	share, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return err
	}

	// Guard against re-completing or completing burned/expired shares
	if share.Status != "UPLOADING" || time.Now().After(share.ExpiresAt) {
		return fmt.Errorf("upload session is not active or has expired")
	}
	if share.UploadID == nil || *share.UploadID == "" || *share.UploadID != uploadID {
		return fmt.Errorf("invalid upload session")
	}

	_, err = s.storage.CompleteMultipartUpload(ctx, share.StorageKey, uploadID, parts)
	if err != nil {
		return fmt.Errorf("failed to complete multipart upload: %w", err)
	}

	// Security: Verify actual assembled object size in storage against claimed size and 2 GB limit (fail-closed, non-destructive on storage error)
	actualSize, sizeErr := s.storage.GetObjectSize(ctx, share.StorageKey)
	if sizeErr != nil {
		// Transient storage error — do NOT delete object or burn share; let sender retry
		return fmt.Errorf("failed to verify uploaded file size: %w", sizeErr)
	}
	const maxAllowedLimit int64 = 2 * 1024 * 1024 * 1024
	if actualSize > share.FileSize || actualSize > maxAllowedLimit {
		_ = s.storage.Delete(ctx, share.StorageKey)
		_ = s.repo.UpdateStatus(ctx, token, "BURNED")
		s.progressMu.Lock()
		delete(s.progressMap, token)
		s.progressMu.Unlock()
		return fmt.Errorf("uploaded file size (%d bytes) exceeds declared size (%d bytes) or maximum limit", actualSize, share.FileSize)
	}

	s.progressMu.Lock()
	delete(s.progressMap, token)
	s.progressMu.Unlock()

	return s.repo.UpdateStatus(ctx, token, "ACTIVE")
}

func (s *EphemeralService) AbortMultipartUpload(ctx context.Context, token string, uploadID string) error {
	share, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return err
	}

	// Security: Only allow aborting sessions that are actively uploading.
	// Prevents malicious clients from deleting already completed ACTIVE shares.
	if share.Status != "UPLOADING" {
		return fmt.Errorf("cannot abort session: upload is not active")
	}
	if share.UploadID == nil || *share.UploadID == "" || *share.UploadID != uploadID {
		return fmt.Errorf("invalid upload session")
	}

	_ = s.storage.AbortMultipartUpload(ctx, share.StorageKey, uploadID)

	s.progressMu.Lock()
	delete(s.progressMap, token)
	s.progressMu.Unlock()

	return s.repo.DeleteByToken(ctx, token)
}

// isSupportedEphemeralPreview checks if the content type or file extension is safe and capable of in-browser preview streaming.
func isSupportedEphemeralPreview(contentType, filename string) bool {
	ct := strings.ToLower(contentType)
	ext := strings.ToLower(filepath.Ext(filename))

	// Video streaming (MP4, WebM, QuickTime)
	if strings.HasPrefix(ct, "video/mp4") || strings.HasPrefix(ct, "video/webm") || strings.HasPrefix(ct, "video/quicktime") ||
		ext == ".mp4" || ext == ".webm" || ext == ".mov" {
		return true
	}

	// Image rendering (JPEG, PNG, WebP, GIF, SVG)
	if strings.HasPrefix(ct, "image/jpeg") || strings.HasPrefix(ct, "image/png") || strings.HasPrefix(ct, "image/webp") ||
		strings.HasPrefix(ct, "image/gif") || strings.HasPrefix(ct, "image/svg+xml") ||
		ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" || ext == ".gif" || ext == ".svg" {
		return true
	}

	// Audio playback (MP3, WAV, OGG)
	if strings.HasPrefix(ct, "audio/mpeg") || strings.HasPrefix(ct, "audio/mp3") || strings.HasPrefix(ct, "audio/wav") || strings.HasPrefix(ct, "audio/ogg") ||
		ext == ".mp3" || ext == ".wav" || ext == ".ogg" {
		return true
	}

	// Document viewing (PDF)
	if strings.HasPrefix(ct, "application/pdf") || ext == ".pdf" {
		return true
	}

	return false
}

func (s *EphemeralService) GetMetadata(ctx context.Context, token string) (*model.EphemeralShare, error) {
	share, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if share.Status == "BURNED" || time.Now().After(share.ExpiresAt) {
		return nil, repository.ErrEphemeralNotFound
	}
	if share.Status == "UPLOADING" {
		s.progressMu.RLock()
		if p, ok := s.progressMap[token]; ok {
			share.ProgressPct = p.ProgressPct
			share.UploadedBytes = p.UploadedBytes
		}
		s.progressMu.RUnlock()
	}
	// Burn-safe preview URLs: generate preview for active, unpassworded shares of supported media types <= 500MB.
	// Both single-use (max_downloads = 1) and multi-use shares can preview before claiming/burning.
	if share.Status == "ACTIVE" && (share.PasswordHash == nil || *share.PasswordHash == "") {
		if isSupportedEphemeralPreview(share.ContentType, share.Filename) && share.FileSize <= 500*1024*1024 {
			url, err := s.storage.GeneratePresignedDownloadURL(ctx, share.StorageKey, 15*time.Minute, "", true)
			if err == nil {
				share.PreviewURL = &url
			}
		}
	}
	return share, nil
}

func (s *EphemeralService) RequestDownload(ctx context.Context, token string, password *string) (string, error) {
	share, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return "", err
	}

	if share.Status == "BURNED" || time.Now().After(share.ExpiresAt) {
		return "", repository.ErrEphemeralNotFound
	}

	if share.Status != "ACTIVE" {
		return "", fmt.Errorf("file is still being uploaded by the sender. Please wait for completion.")
	}

	if share.PasswordHash != nil && *share.PasswordHash != "" {
		if password == nil || bcrypt.CompareHashAndPassword([]byte(*share.PasswordHash), []byte(*password)) != nil {
			return "", fmt.Errorf("invalid password")
		}
	}

	// Security: Verify actual object size in storage before generating download link (fail-closed, non-destructive on storage error)
	actualSize, sizeErr := s.storage.GetObjectSize(ctx, share.StorageKey)
	if sizeErr != nil {
		// Transient storage error — do NOT delete object or burn share; let recipient retry
		return "", fmt.Errorf("file size verification failed: %w", sizeErr)
	}
	const maxAllowedLimit int64 = 2 * 1024 * 1024 * 1024
	if actualSize > share.FileSize || actualSize > maxAllowedLimit {
		_ = s.storage.Delete(ctx, share.StorageKey)
		_ = s.repo.UpdateStatus(ctx, token, "BURNED")
		return "", fmt.Errorf("file size verification failed: object was removed")
	}

	_, shouldBurn, err := s.repo.IncrementDownloadAndCheckBurn(ctx, share.ID)
	if err != nil {
		return "", fmt.Errorf("failed to process download: %w", err)
	}

	// Generate 30-minute active presigned URL for browser stream and resumable HTTP Range
	downloadURL, err := s.storage.GeneratePresignedDownloadURL(ctx, share.StorageKey, 30*time.Minute, share.Filename, false)
	if err != nil {
		return "", fmt.Errorf("failed to generate download link: %w", err)
	}

	if shouldBurn {
		// Mark BURNED in DB immediately so no SECOND request can generate a presigned URL
		_ = s.repo.MarkBurned(ctx, share.ID)
		// Delay physical R2 storage deletion by 30 minutes so active/resumed download streams cleanly
		go func(storageKey string) {
			time.Sleep(30 * time.Minute)
			bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = s.storage.Delete(bgCtx, storageKey)
		}(share.StorageKey)
	}

	if s.bandwidthRepo != nil {
		go func(sID string, size int64) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = s.bandwidthRepo.RecordEgress(bgCtx, nil, &sID, size, "instant_share", "")
		}(share.ID, share.FileSize)
	}

	return downloadURL, nil
}

func (s *EphemeralService) ListAllForAdmin(ctx context.Context, limit, offset int) ([]*model.EphemeralShare, int, error) {
	return s.repo.ListAllForAdmin(ctx, limit, offset)
}

// AdminBurnShare allows an administrator to immediately burn and purge an ephemeral share.
func (s *EphemeralService) AdminBurnShare(ctx context.Context, token string) error {
	share, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return err
	}
	if share == nil {
		return fmt.Errorf("ephemeral share not found")
	}

	if share.StorageKey != "" {
		_ = s.storage.Delete(ctx, share.StorageKey)
	}

	s.progressMu.Lock()
	delete(s.progressMap, token)
	s.progressMu.Unlock()

	return s.repo.UpdateStatus(ctx, token, "BURNED")
}

