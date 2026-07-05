package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

var ErrVerificationNotFound = errors.New("verification record not found")

type EmailVerificationRepository struct {
	db *pgxpool.Pool
}

func NewEmailVerificationRepository(db *pgxpool.Pool) *EmailVerificationRepository {
	return &EmailVerificationRepository{db: db}
}

// Create stores a new OTP verification record.
func (r *EmailVerificationRepository) Create(ctx context.Context, v *model.EmailVerification) error {
	query := `
		INSERT INTO email_verifications (user_id, email, otp_code, purpose, max_attempts, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		v.UserID, v.Email, v.OTPCode, v.Purpose, v.MaxAttempts, v.ExpiresAt,
	).Scan(&v.ID, &v.CreatedAt)
}

// FindLatestByEmail retrieves the most recent unused OTP for a given email+purpose.
func (r *EmailVerificationRepository) FindLatestByEmail(ctx context.Context, email, purpose string) (*model.EmailVerification, error) {
	query := `
		SELECT id, user_id, email, otp_code, purpose, attempts, max_attempts, is_used, expires_at, created_at
		FROM email_verifications
		WHERE email = $1 AND purpose = $2 AND is_used = false
		ORDER BY created_at DESC
		LIMIT 1
	`
	var v model.EmailVerification
	err := r.db.QueryRow(ctx, query, email, purpose).Scan(
		&v.ID, &v.UserID, &v.Email, &v.OTPCode, &v.Purpose,
		&v.Attempts, &v.MaxAttempts, &v.IsUsed, &v.ExpiresAt, &v.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrVerificationNotFound
		}
		return nil, fmt.Errorf("failed to find verification: %w", err)
	}
	return &v, nil
}

// IncrementAttempts bumps the attempt counter by 1.
func (r *EmailVerificationRepository) IncrementAttempts(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE email_verifications SET attempts = attempts + 1 WHERE id = $1`, id)
	return err
}

// MarkUsed marks an OTP as consumed.
func (r *EmailVerificationRepository) MarkUsed(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `UPDATE email_verifications SET is_used = true WHERE id = $1`, id)
	return err
}

// InvalidateAllForUser marks all pending OTPs for a user as used (e.g. after successful verify).
func (r *EmailVerificationRepository) InvalidateAllForUser(ctx context.Context, userID, purpose string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE email_verifications SET is_used = true WHERE user_id = $1 AND purpose = $2 AND is_used = false`,
		userID, purpose,
	)
	return err
}

// CleanupExpired deletes OTPs older than the given time (used by scheduler).
func (r *EmailVerificationRepository) CleanupExpired(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM email_verifications WHERE expires_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
