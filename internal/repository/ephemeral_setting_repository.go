package repository

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EphemeralSettings struct {
	MaxFileSizeGb float64 `json:"max_file_size_gb"`
	MaxDownloads  int     `json:"max_downloads_cap"`
	RateLimit24h  int     `json:"rate_limit_24h"`
	ExpiryMinutes int     `json:"expiry_minutes"`
}

type EphemeralSettingRepository struct {
	db *pgxpool.Pool
}

func NewEphemeralSettingRepository(db *pgxpool.Pool) *EphemeralSettingRepository {
	return &EphemeralSettingRepository{db: db}
}

func (r *EphemeralSettingRepository) GetSettings(ctx context.Context) (*EphemeralSettings, error) {
	query := `SELECT max_file_size_gb, max_downloads_cap, rate_limit_24h, expiry_minutes FROM ephemeral_settings WHERE id = 1`
	var s EphemeralSettings
	err := r.db.QueryRow(ctx, query).Scan(&s.MaxFileSizeGb, &s.MaxDownloads, &s.RateLimit24h, &s.ExpiryMinutes)
	if err != nil {
		return &EphemeralSettings{MaxFileSizeGb: 2, MaxDownloads: 1, RateLimit24h: 2, ExpiryMinutes: 60}, nil
	}
	return &s, nil
}

func (r *EphemeralSettingRepository) UpdateSettings(ctx context.Context, s *EphemeralSettings) error {
	query := `
		INSERT INTO ephemeral_settings (id, max_file_size_gb, max_downloads_cap, rate_limit_24h, expiry_minutes, updated_at)
		VALUES (1, $1, $2, $3, $4, NOW())
		ON CONFLICT (id) DO UPDATE
		SET max_file_size_gb = EXCLUDED.max_file_size_gb,
		    max_downloads_cap = EXCLUDED.max_downloads_cap,
		    rate_limit_24h = EXCLUDED.rate_limit_24h,
		    expiry_minutes = EXCLUDED.expiry_minutes,
		    updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, s.MaxFileSizeGb, s.MaxDownloads, s.RateLimit24h, s.ExpiryMinutes)
	return err
}
