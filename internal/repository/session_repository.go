package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSessionNotFound = errors.New("session not found")

type Session struct {
	ID               string
	UserID           string
	RefreshTokenHash string
	UserAgent        *string
	IPAddress        *string
	ExpiresAt        time.Time
	CreatedAt        time.Time
	LastUsedAt       *time.Time
}

type SessionRepository struct {
	db *pgxpool.Pool
}

func NewSessionRepository(db *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{db: db}
}

func (r *SessionRepository) Create(ctx context.Context, session *Session) error {
	query := `
		INSERT INTO sessions (user_id, refresh_token_hash, user_agent, ip_address, expires_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(ctx, query, session.UserID, session.RefreshTokenHash, session.UserAgent, session.IPAddress, session.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	return nil
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, hash string) (*Session, error) {
	query := `
		SELECT id, user_id, refresh_token_hash, user_agent, ip_address, expires_at, created_at, last_used_at
		FROM sessions WHERE refresh_token_hash = $1
	`

	var s Session
	err := r.db.QueryRow(ctx, query, hash).Scan(
		&s.ID, &s.UserID, &s.RefreshTokenHash,
		&s.UserAgent, &s.IPAddress,
		&s.ExpiresAt, &s.CreatedAt, &s.LastUsedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("failed to find session: %w", err)
	}
	return &s, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, hash string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM sessions WHERE refresh_token_hash = $1", hash)
	return err
}

func (r *SessionRepository) DeleteAllByUserID(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, "DELETE FROM sessions WHERE user_id = $1", userID)
	return err
}
