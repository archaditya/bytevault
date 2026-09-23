package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/archaditya/bytevault/internal/repository"
)

const (
	MaxKeysPerUser         = 2
	MaxRotationsPerKey     = 3
	DefaultRateLimitPerMin = 60
)

var (
	ErrMaxKeysExceeded     = fmt.Errorf("maximum limit of %d API keys reached. Delete an existing key to generate a new one", MaxKeysPerUser)
	ErrMaxRotationsReached = fmt.Errorf("maximum rotation limit (%d times) reached for this API key. Please delete it and create a new key", MaxRotationsPerKey)
	ErrInvalidAPIKey       = errors.New("invalid or revoked API key")
)

type APIKeyService struct {
	repo *repository.APIKeyRepository
}

func NewAPIKeyService(repo *repository.APIKeyRepository) *APIKeyService {
	return &APIKeyService{repo: repo}
}

func (s *APIKeyService) generateKey() (string, string, string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	rawKey := "ppv_live_" + hex.EncodeToString(bytes)
	prefix := rawKey[:12] + "..." + rawKey[len(rawKey)-4:]

	hash := sha256.Sum256([]byte(rawKey))
	hashHex := hex.EncodeToString(hash[:])

	return rawKey, prefix, hashHex, nil
}

func (s *APIKeyService) CreateKey(ctx context.Context, userID, name string) (*model.APIKey, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "Production API Key"
	}
	if len(name) > 100 {
		name = name[:100]
	}

	count, err := s.repo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing API key count: %w", err)
	}
	if count >= MaxKeysPerUser {
		return nil, ErrMaxKeysExceeded
	}

	rawKey, prefix, hashHex, err := s.generateKey()
	if err != nil {
		return nil, err
	}

	apiKey := &model.APIKey{
		UserID:          userID,
		Name:            name,
		KeyPrefix:       prefix,
		KeyHash:         hashHex,
		RotationCount:   0,
		MaxRotations:    MaxRotationsPerKey,
		RateLimitPerMin: DefaultRateLimitPerMin,
	}

	if err := s.repo.Create(ctx, apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key in database: %w", err)
	}

	// Populate plaintext key for the ONE-TIME return to user
	apiKey.PlainKey = rawKey
	apiKey.MaxRotations = MaxRotationsPerKey
	return apiKey, nil
}

func (s *APIKeyService) RotateKey(ctx context.Context, id, userID string) (*model.APIKey, error) {
	existing, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	if existing.RotationCount >= MaxRotationsPerKey {
		return nil, ErrMaxRotationsReached
	}

	rawKey, prefix, hashHex, err := s.generateKey()
	if err != nil {
		return nil, err
	}

	newRotationCount := existing.RotationCount + 1
	if err := s.repo.UpdateKeyHashAndRotation(ctx, id, userID, prefix, hashHex, newRotationCount); err != nil {
		return nil, err
	}

	updated, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}

	// Populate plaintext key for the ONE-TIME return to user
	updated.PlainKey = rawKey
	updated.MaxRotations = MaxRotationsPerKey
	return updated, nil
}

func (s *APIKeyService) ValidateKey(ctx context.Context, rawKey string) (*model.APIKey, error) {
	hash := sha256.Sum256([]byte(rawKey))
	hashHex := hex.EncodeToString(hash[:])

	apiKey, err := s.repo.GetByHash(ctx, hashHex)
	if err != nil {
		if errors.Is(err, repository.ErrAPIKeyNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}

	// Asynchronously record last usage timestamp
	go func(keyID string) {
		_ = s.repo.UpdateLastUsed(context.Background(), keyID)
	}(apiKey.ID)

	return apiKey, nil
}

func (s *APIKeyService) ListKeys(ctx context.Context, userID string) ([]*model.APIKey, error) {
	return s.repo.ListByUserID(ctx, userID)
}

func (s *APIKeyService) DeleteKey(ctx context.Context, id, userID string) error {
	return s.repo.Delete(ctx, id, userID)
}
