package repository

import (
	"context"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemSettings struct {
	MaxUploadMB       int  `json:"max_upload_mb"`
	RateLimitPerHour  int  `json:"rate_limit_per_hour"`
	AllowPublicShares bool `json:"allow_public_shares"`
	MaintenanceMode   bool `json:"maintenance_mode"`
}

type SystemSettingRepository struct {
	db         *pgxpool.Pool
	mu         sync.RWMutex
	cached     *SystemSettings
	lastCached time.Time
}

func NewSystemSettingRepository(db *pgxpool.Pool) *SystemSettingRepository {
	return &SystemSettingRepository{db: db}
}

func (r *SystemSettingRepository) GetSettings(ctx context.Context) (*SystemSettings, error) {
	r.mu.RLock()
	if r.cached != nil && time.Since(r.lastCached) < 3*time.Second {
		cp := *r.cached
		r.mu.RUnlock()
		return &cp, nil
	}
	r.mu.RUnlock()

	query := `SELECT max_upload_mb, rate_limit_per_hour, allow_public_shares, maintenance_mode FROM system_settings WHERE id = 1`
	var s SystemSettings
	err := r.db.QueryRow(ctx, query).Scan(&s.MaxUploadMB, &s.RateLimitPerHour, &s.AllowPublicShares, &s.MaintenanceMode)
	if err != nil {
		fallback := &SystemSettings{
			MaxUploadMB:       100,
			RateLimitPerHour:  1000,
			AllowPublicShares: true,
			MaintenanceMode:   false,
		}
		return fallback, nil
	}

	r.mu.Lock()
	r.cached = &s
	r.lastCached = time.Now()
	r.mu.Unlock()

	return &s, nil
}

func (r *SystemSettingRepository) IsMaintenanceMode(ctx context.Context) bool {
	s, err := r.GetSettings(ctx)
	if err != nil || s == nil {
		return false
	}
	return s.MaintenanceMode
}

func (r *SystemSettingRepository) UpdateSettings(ctx context.Context, s *SystemSettings) error {
	query := `
		INSERT INTO system_settings (id, max_upload_mb, rate_limit_per_hour, allow_public_shares, maintenance_mode, updated_at)
		VALUES (1, $1, $2, $3, $4, NOW())
		ON CONFLICT (id) DO UPDATE
		SET max_upload_mb = EXCLUDED.max_upload_mb,
		    rate_limit_per_hour = EXCLUDED.rate_limit_per_hour,
		    allow_public_shares = EXCLUDED.allow_public_shares,
		    maintenance_mode = EXCLUDED.maintenance_mode,
		    updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, s.MaxUploadMB, s.RateLimitPerHour, s.AllowPublicShares, s.MaintenanceMode)
	if err == nil {
		r.mu.Lock()
		cp := *s
		r.cached = &cp
		r.lastCached = time.Now()
		r.mu.Unlock()
	}
	return err
}

