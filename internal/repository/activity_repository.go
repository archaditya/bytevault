package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

type ActivityRepository struct {
	db *pgxpool.Pool
}

func NewActivityRepository(db *pgxpool.Pool) *ActivityRepository {
	return &ActivityRepository{db: db}
}

// Log records an activity. Call this from services whenever something important happens.
func (r *ActivityRepository) Log(ctx context.Context, log *model.ActivityLog) error {
	// Convert metadata map to JSON bytes for JSONB column
	metaJSON, err := json.Marshal(log.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}

	query := `
		INSERT INTO activity_logs (user_id, action, resource_type, resource_id, metadata, ip_address, user_agent)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err = r.db.Exec(ctx, query,
		log.UserID, log.Action, log.ResourceType, log.ResourceID,
		metaJSON, log.IPAddress, log.UserAgent,
	)
	if err != nil {
		return fmt.Errorf("failed to log activity: %w", err)
	}
	return nil
}

// ListAll returns paginated activity logs (newest first)
// offset = skip N rows, limit = return N rows
func (r *ActivityRepository) ListAll(ctx context.Context, limit, offset int) ([]model.ActivityLog, int, error) {
	// Get total count for pagination
	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM activity_logs").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count activity logs: %w", err)
	}

	query := `
		SELECT id, user_id, action, resource_type, resource_id, metadata, ip_address, user_agent, created_at
		FROM activity_logs ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list activity logs: %w", err)
	}
	defer rows.Close()

	var logs []model.ActivityLog
	for rows.Next() {
		var l model.ActivityLog
		var metaJSON []byte

		if err := rows.Scan(
			&l.ID, &l.UserID, &l.Action, &l.ResourceType, &l.ResourceID,
			&metaJSON, &l.IPAddress, &l.UserAgent, &l.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan activity log: %w", err)
		}

		if metaJSON != nil {
			json.Unmarshal(metaJSON, &l.Metadata)
		}

		logs = append(logs, l)
	}

	return logs, total, nil
}

// ListByUserID returns activity logs for a specific user
func (r *ActivityRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]model.ActivityLog, error) {
	query := `
		SELECT id, user_id, action, resource_type, resource_id, metadata, ip_address, user_agent, created_at
		FROM activity_logs WHERE user_id = $1
		ORDER BY created_at DESC LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list user activity: %w", err)
	}
	defer rows.Close()

	var logs []model.ActivityLog
	for rows.Next() {
		var l model.ActivityLog
		var metaJSON []byte

		if err := rows.Scan(
			&l.ID, &l.UserID, &l.Action, &l.ResourceType, &l.ResourceID,
			&metaJSON, &l.IPAddress, &l.UserAgent, &l.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan activity log: %w", err)
		}

		if metaJSON != nil {
			json.Unmarshal(metaJSON, &l.Metadata)
		}

		logs = append(logs, l)
	}

	return logs, nil
}