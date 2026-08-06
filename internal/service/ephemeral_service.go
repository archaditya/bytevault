package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
	"github.com/archaditya/bytevault/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

type EphemeralService struct {
	repo        *repository.EphemeralShareRepository
	settingRepo *repository.EphemeralSettingRepository
	storage     storage.StorageProvider
}

func NewEphemeralService(repo *repository.EphemeralShareRepository, settingRepo *repository.EphemeralSettingRepository, storage storage.StorageProvider) *EphemeralService {
	return &EphemeralService{
		repo:        repo,
		settingRepo: settingRepo,
		storage:     storage,
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
		settings = &repository.EphemeralSettings{MaxFileSizeGb: 2, MaxDownloads: 1, RateLimit24h: 2, ExpiryMinutes: 60}
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

	uploadURL, err := s.storage.GeneratePresignedUploadURL(ctx, storageKey, contentType, 15*time.Minute)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	share := &model.EphemeralShare{
		Token:        token,
		StorageKey:   storageKey,
		Filename:     filename,
		FileSize:     size,
		ContentType:  contentType,
		Status:       "UPLOADING",
		MaxDownloads: settings.MaxDownloads,
		IPAddress:    ip,
		UserAgent:    ua,
		PasswordHash: passwordHash,
		ExpiresAt:    time.Now().Add(ttl),
	}

	if err := s.repo.Create(ctx, share); err != nil {
		return nil, "", err
	}

	return share, uploadURL, nil
}

func (s *EphemeralService) GetMetadata(ctx context.Context, token string) (*model.EphemeralShare, error) {
	share, err := s.repo.FindByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if share.Status == "BURNED" || time.Now().After(share.ExpiresAt) {
		return nil, repository.ErrEphemeralNotFound
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

	if share.PasswordHash != nil && *share.PasswordHash != "" {
		if password == nil || bcrypt.CompareHashAndPassword([]byte(*share.PasswordHash), []byte(*password)) != nil {
			return "", fmt.Errorf("invalid password")
		}
	}

	_, shouldBurn, err := s.repo.IncrementDownloadAndCheckBurn(ctx, share.ID)
	if err != nil {
		return "", fmt.Errorf("failed to process download: %w", err)
	}

	// Generate 5-minute active presigned URL for browser stream
	downloadURL, err := s.storage.GeneratePresignedDownloadURL(ctx, share.StorageKey, 5*time.Minute, share.Filename, false)
	if err != nil {
		return "", fmt.Errorf("failed to generate download link: %w", err)
	}

	if shouldBurn {
		// Mark BURNED in DB immediately so no SECOND request can generate a presigned URL
		_ = s.repo.MarkBurned(ctx, share.ID)
		// Delay physical R2 storage deletion by 5 minutes so active download streams cleanly
		go func(storageKey string) {
			time.Sleep(5 * time.Minute)
			bgCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			_ = s.storage.Delete(bgCtx, storageKey)
		}(share.StorageKey)
	}

	return downloadURL, nil
}

func (s *EphemeralService) ListAllForAdmin(ctx context.Context, limit, offset int) ([]*model.EphemeralShare, int, error) {
	return s.repo.ListAllForAdmin(ctx, limit, offset)
}
