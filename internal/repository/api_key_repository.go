package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrAPIKeyNotFound = errors.New("api key not found")
)

type APIKeyRepository struct {
	db *pgxpool.Pool
}

func NewAPIKeyRepository(db *pgxpool.Pool) *APIKeyRepository {
	return &APIKeyRepository{db: db}
}

func (r *APIKeyRepository) Create(ctx context.Context, apiKey *model.APIKey) error {
	query := `
		INSERT INTO api_keys (user_id, name, key_prefix, key_hash, rotation_count, rate_limit_per_min)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		apiKey.UserID,
		apiKey.Name,
		apiKey.KeyPrefix,
		apiKey.KeyHash,
		apiKey.RotationCount,
		apiKey.RateLimitPerMin,
	).Scan(&apiKey.ID, &apiKey.CreatedAt, &apiKey.UpdatedAt)
}

func (r *APIKeyRepository) GetByHash(ctx context.Context, keyHash string) (*model.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, rotation_count, rate_limit_per_min, last_used_at, created_at, updated_at
		FROM api_keys
		WHERE key_hash = $1
	`
	var k model.APIKey
	err := r.db.QueryRow(ctx, query, keyHash).Scan(
		&k.ID,
		&k.UserID,
		&k.Name,
		&k.KeyPrefix,
		&k.KeyHash,
		&k.RotationCount,
		&k.RateLimitPerMin,
		&k.LastUsedAt,
		&k.CreatedAt,
		&k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to query api key by hash: %w", err)
	}
	k.MaxRotations = 3
	return &k, nil
}

func (r *APIKeyRepository) GetByID(ctx context.Context, id, userID string) (*model.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, rotation_count, rate_limit_per_min, last_used_at, created_at, updated_at
		FROM api_keys
		WHERE id = $1 AND user_id = $2
	`
	var k model.APIKey
	err := r.db.QueryRow(ctx, query, id, userID).Scan(
		&k.ID,
		&k.UserID,
		&k.Name,
		&k.KeyPrefix,
		&k.KeyHash,
		&k.RotationCount,
		&k.RateLimitPerMin,
		&k.LastUsedAt,
		&k.CreatedAt,
		&k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAPIKeyNotFound
		}
		return nil, fmt.Errorf("failed to query api key: %w", err)
	}
	k.MaxRotations = 3
	return &k, nil
}

func (r *APIKeyRepository) ListByUserID(ctx context.Context, userID string) ([]*model.APIKey, error) {
	query := `
		SELECT id, user_id, name, key_prefix, key_hash, rotation_count, rate_limit_per_min, last_used_at, created_at, updated_at
		FROM api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	defer rows.Close()

	keys := make([]*model.APIKey, 0)
	for rows.Next() {
		var k model.APIKey
		if err := rows.Scan(
			&k.ID,
			&k.UserID,
			&k.Name,
			&k.KeyPrefix,
			&k.KeyHash,
			&k.RotationCount,
			&k.RateLimitPerMin,
			&k.LastUsedAt,
			&k.CreatedAt,
			&k.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		k.MaxRotations = 3
		keys = append(keys, &k)
	}
	return keys, nil
}

func (r *APIKeyRepository) CountByUserID(ctx context.Context, userID string) (int, error) {
	query := `SELECT COUNT(*) FROM api_keys WHERE user_id = $1`
	var count int
	err := r.db.QueryRow(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count user api keys: %w", err)
	}
	return count, nil
}

func (r *APIKeyRepository) UpdateKeyHashAndRotation(ctx context.Context, id, userID, newKeyPrefix, newKeyHash string, newRotationCount int) error {
	query := `
		UPDATE api_keys
		SET key_prefix = $1, key_hash = $2, rotation_count = $3, updated_at = NOW()
		WHERE id = $4 AND user_id = $5
	`
	cmdTag, err := r.db.Exec(ctx, query, newKeyPrefix, newKeyHash, newRotationCount, id, userID)
	if err != nil {
		return fmt.Errorf("failed to rotate api key: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

func (r *APIKeyRepository) UpdateLastUsed(ctx context.Context, id string) error {
	query := `UPDATE api_keys SET last_used_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *APIKeyRepository) Delete(ctx context.Context, id, userID string) error {
	query := `DELETE FROM api_keys WHERE id = $1 AND user_id = $2`
	cmdTag, err := r.db.Exec(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete api key: %w", err)
	}
	if cmdTag.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}
