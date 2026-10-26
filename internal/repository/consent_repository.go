package repository

import (
	"context"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ConsentRepository struct {
	db *pgxpool.Pool
}

func NewConsentRepository(db *pgxpool.Pool) *ConsentRepository {
	return &ConsentRepository{db: db}
}

// RecordConsent logs a new affirmative grant in the audit trail.
func (r *ConsentRepository) RecordConsent(ctx context.Context, c *model.UserConsent) error {
	query := `
		INSERT INTO user_consents (user_id, consent_type, version, status, ip_address, user_agent, granted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`
	grantedAt := c.GrantedAt
	if grantedAt.IsZero() {
		grantedAt = time.Now()
	}
	return r.db.QueryRow(ctx, query,
		c.UserID,
		c.ConsentType,
		c.Version,
		c.Status,
		c.IPAddress,
		c.UserAgent,
		grantedAt,
	).Scan(&c.ID, &c.CreatedAt, &c.UpdatedAt)
}

// GetConsentsByUserID retrieves all consent audit records for a given user.
func (r *ConsentRepository) GetConsentsByUserID(ctx context.Context, userID string) ([]*model.UserConsent, error) {
	query := `
		SELECT id, user_id, consent_type, version, status, ip_address, user_agent, granted_at, revoked_at, created_at, updated_at
		FROM user_consents
		WHERE user_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.UserConsent
	for rows.Next() {
		var c model.UserConsent
		if err := rows.Scan(
			&c.ID,
			&c.UserID,
			&c.ConsentType,
			&c.Version,
			&c.Status,
			&c.IPAddress,
			&c.UserAgent,
			&c.GrantedAt,
			&c.RevokedAt,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, &c)
	}
	return records, rows.Err()
}

// RevokeConsent updates the latest grant of a specific type to revoked status.
func (r *ConsentRepository) RevokeConsent(ctx context.Context, userID, consentType string) error {
	query := `
		UPDATE user_consents
		SET status = 'revoked', revoked_at = NOW(), updated_at = NOW()
		WHERE user_id = $1 AND consent_type = $2 AND status = 'granted'
	`
	_, err := r.db.Exec(ctx, query, userID, consentType)
	return err
}
