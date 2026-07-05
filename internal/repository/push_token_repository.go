package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

type PushTokenRepository struct {
	db *pgxpool.Pool
}

func NewPushTokenRepository(db *pgxpool.Pool) *PushTokenRepository {
	return &PushTokenRepository{db: db}
}

// Upsert inserts or updates a push token for a user.
func (r *PushTokenRepository) Upsert(ctx context.Context, t *model.PushToken) error {
	query := `
		INSERT INTO push_tokens (user_id, token, device_type)
		VALUES ($1, $2, $3)
		ON CONFLICT (token)
		DO UPDATE SET user_id = $1, device_type = $3, is_active = true, updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, t.UserID, t.Token, t.DeviceType).Scan(&t.ID, &t.CreatedAt, &t.UpdatedAt)
}

// GetActiveByUser returns all active push tokens for a user.
func (r *PushTokenRepository) GetActiveByUser(ctx context.Context, userID string) ([]model.PushToken, error) {
	query := `
		SELECT id, user_id, token, device_type, is_active, created_at, updated_at
		FROM push_tokens
		WHERE user_id = $1 AND is_active = true
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get push tokens: %w", err)
	}
	defer rows.Close()

	var tokens []model.PushToken
	for rows.Next() {
		var t model.PushToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Token, &t.DeviceType, &t.IsActive, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

// Deactivate marks a token as inactive (e.g. when Firebase reports it invalid).
func (r *PushTokenRepository) Deactivate(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE push_tokens SET is_active = false, updated_at = NOW() WHERE token = $1`, token,
	)
	return err
}

// DeleteByUser removes all push tokens for a user (e.g. on logout from all devices).
func (r *PushTokenRepository) DeleteByUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM push_tokens WHERE user_id = $1`, userID)
	return err
}
