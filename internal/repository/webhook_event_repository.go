package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// WebhookEventRepository tracks processed Razorpay webhook events for idempotency.
type WebhookEventRepository struct {
	db *pgxpool.Pool
}

func NewWebhookEventRepository(db *pgxpool.Pool) *WebhookEventRepository {
	return &WebhookEventRepository{db: db}
}

// Exists checks if a webhook event has already been processed.
func (r *WebhookEventRepository) Exists(ctx context.Context, eventID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM processed_webhook_events WHERE event_id = $1`, eventID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check webhook event: %w", err)
	}
	return count > 0, nil
}

// Record stores a processed webhook event ID for future idempotency checks.
func (r *WebhookEventRepository) Record(ctx context.Context, eventID, eventType, subscriptionID string) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO processed_webhook_events (event_id, event_type, razorpay_subscription_id, processed_at)
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (event_id) DO NOTHING`,
		eventID, eventType, subscriptionID,
	)
	if err != nil {
		return fmt.Errorf("record webhook event: %w", err)
	}
	return nil
}

// CleanupOlderThan removes processed events older than the given time to prevent unbounded growth.
func (r *WebhookEventRepository) CleanupOlderThan(ctx context.Context, before time.Time) (int64, error) {
	res, err := r.db.Exec(ctx, `DELETE FROM processed_webhook_events WHERE processed_at < $1`, before)
	if err != nil {
		return 0, fmt.Errorf("cleanup webhook events: %w", err)
	}
	return res.RowsAffected(), nil
}
