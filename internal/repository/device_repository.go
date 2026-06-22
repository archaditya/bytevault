package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

type DeviceRepository struct {
	db *pgxpool.Pool
}

func NewDeviceRepository(db *pgxpool.Pool) *DeviceRepository {
	return &DeviceRepository{db: db}
}

// Upsert registers or updates a device token.
// If the fcm_token already exists, update the user and mark active.
func (r *DeviceRepository) Upsert(ctx context.Context, device *model.UserDevice) error {
	query := `
		INSERT INTO user_devices (user_id, fcm_token, device_type, device_id)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (fcm_token) DO UPDATE SET
			user_id = $1, device_type = $3, device_id = $4,
			is_active = true, updated_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, device.UserID, device.FcmToken, device.DeviceType, device.DeviceID)
	if err != nil {
		return fmt.Errorf("failed to upsert device: %w", err)
	}
	return nil
}

// FindByUserID returns all active devices for a user
func (r *DeviceRepository) FindByUserID(ctx context.Context, userID string) ([]model.UserDevice, error) {
	query := `
		SELECT id, user_id, fcm_token, device_type, device_id, is_active, last_used_at, created_at, updated_at
		FROM user_devices WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to find devices: %w", err)
	}
	defer rows.Close()

	var devices []model.UserDevice
	for rows.Next() {
		var d model.UserDevice
		if err := rows.Scan(
			&d.ID, &d.UserID, &d.FcmToken, &d.DeviceType, &d.DeviceID,
			&d.IsActive, &d.LastUsedAt, &d.CreatedAt, &d.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan device: %w", err)
		}
		devices = append(devices, d)
	}
	return devices, nil
}

// Deactivate marks a device as inactive (soft remove)
func (r *DeviceRepository) Deactivate(ctx context.Context, id, userID string) error {
	query := `UPDATE user_devices SET is_active = false, updated_at = NOW()
	          WHERE id = $1 AND user_id = $2`
	_, err := r.db.Exec(ctx, query, id, userID)
	return err
}