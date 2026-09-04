package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/archaditya/bytevault/internal/model"
)

var ErrEphemeralNotFound = errors.New("ephemeral share not found or self-destructed")

type EphemeralShareRepository struct {
	db *pgxpool.Pool
}

func NewEphemeralShareRepository(db *pgxpool.Pool) *EphemeralShareRepository {
	return &EphemeralShareRepository{db: db}
}

func (r *EphemeralShareRepository) Create(ctx context.Context, share *model.EphemeralShare) error {
	query := `
		INSERT INTO ephemeral_shares (token, storage_key, filename, file_size, content_type, status, max_downloads, download_count, ip_address, user_agent, password_hash, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		share.Token,
		share.StorageKey,
		share.Filename,
		share.FileSize,
		share.ContentType,
		share.Status,
		share.MaxDownloads,
		share.DownloadCount,
		share.IPAddress,
		share.UserAgent,
		share.PasswordHash,
		share.ExpiresAt,
	).Scan(&share.ID, &share.CreatedAt)
}

func (r *EphemeralShareRepository) FindByToken(ctx context.Context, token string) (*model.EphemeralShare, error) {
	query := `
		SELECT id, token, storage_key, filename, file_size, content_type, status, max_downloads, download_count, ip_address, user_agent, password_hash, expires_at, created_at, burned_at
		FROM ephemeral_shares
		WHERE token = $1
	`
	var s model.EphemeralShare
	err := r.db.QueryRow(ctx, query, token).Scan(
		&s.ID, &s.Token, &s.StorageKey, &s.Filename, &s.FileSize, &s.ContentType, &s.Status,
		&s.MaxDownloads, &s.DownloadCount, &s.IPAddress, &s.UserAgent, &s.PasswordHash,
		&s.ExpiresAt, &s.CreatedAt, &s.BurnedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrEphemeralNotFound
		}
		return nil, err
	}
	s.HasPassword = s.PasswordHash != nil && *s.PasswordHash != ""
	return &s, nil
}

func (r *EphemeralShareRepository) IncrementDownloadAndCheckBurn(ctx context.Context, id string) (int, bool, error) {
	query := `
		UPDATE ephemeral_shares
		SET download_count = download_count + 1
		WHERE id = $1
		RETURNING download_count, max_downloads
	`
	var count, max int
	err := r.db.QueryRow(ctx, query, id).Scan(&count, &max)
	if err != nil {
		return 0, false, err
	}
	shouldBurn := count >= max
	return count, shouldBurn, nil
}

func (r *EphemeralShareRepository) MarkBurned(ctx context.Context, id string) error {
	query := `UPDATE ephemeral_shares SET status = 'BURNED', burned_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *EphemeralShareRepository) UpdateStatus(ctx context.Context, token string, status string) error {
	query := `UPDATE ephemeral_shares SET status = $1 WHERE token = $2`
	_, err := r.db.Exec(ctx, query, status, token)
	return err
}

func (r *EphemeralShareRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM ephemeral_shares WHERE token = $1`
	_, err := r.db.Exec(ctx, query, token)
	return err
}

func (r *EphemeralShareRepository) ListAllForAdmin(ctx context.Context, limit, offset int) ([]*model.EphemeralShare, int, error) {
	var total int
	r.db.QueryRow(ctx, `SELECT COUNT(*) FROM ephemeral_shares`).Scan(&total)

	query := `
		SELECT id, token, storage_key, filename, file_size, content_type, status, max_downloads, download_count, ip_address, user_agent, expires_at, created_at, burned_at
		FROM ephemeral_shares
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var shares []*model.EphemeralShare
	for rows.Next() {
		var s model.EphemeralShare
		if err := rows.Scan(
			&s.ID, &s.Token, &s.StorageKey, &s.Filename, &s.FileSize, &s.ContentType, &s.Status,
			&s.MaxDownloads, &s.DownloadCount, &s.IPAddress, &s.UserAgent, &s.ExpiresAt, &s.CreatedAt, &s.BurnedAt,
		); err != nil {
			return nil, 0, err
		}
		shares = append(shares, &s)
	}
	return shares, total, nil
}

func (r *EphemeralShareRepository) PurgeExpiredAndBurned(ctx context.Context) ([]string, error) {
	query := `
		DELETE FROM ephemeral_shares
		WHERE (status = 'READY' AND expires_at < NOW()) OR status = 'BURNED'
		RETURNING storage_key
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err == nil {
			keys = append(keys, key)
		}
	}
	return keys, nil
}

// CountUploadsByIPInWindow counts guest uploads from an IP in the last 24 hours
func (r *EphemeralShareRepository) CountUploadsByIPInWindow(ctx context.Context, ip string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM ephemeral_shares
		WHERE ip_address = $1 AND created_at > NOW() - INTERVAL '24 hours'
	`
	var count int
	err := r.db.QueryRow(ctx, query, ip).Scan(&count)
	return count, err
}

