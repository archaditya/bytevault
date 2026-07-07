package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

type NotificationRepository struct {
	db *pgxpool.Pool
}

func NewNotificationRepository(db *pgxpool.Pool) *NotificationRepository {
	return &NotificationRepository{db: db}
}

// Create inserts a new in-app notification.
func (r *NotificationRepository) Create(ctx context.Context, n *model.Notification) error {
	query := `
		INSERT INTO notifications (user_id, type, title, body, metadata, channel)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at
	`
	return r.db.QueryRow(ctx, query,
		n.UserID, n.Type, n.Title, n.Body, n.Metadata, n.Channel,
	).Scan(&n.ID, &n.CreatedAt)
}

// ListByUser returns paginated notifications for a user (newest first).
func (r *NotificationRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]model.Notification, int, error) {
	var total int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1`, userID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	query := `
		SELECT id, user_id, type, title, body, metadata, channel, is_read, read_at, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body,
			&n.Metadata, &n.Channel, &n.IsRead, &n.ReadAt, &n.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}

	return notifications, total, nil
}

// UnreadCount returns the number of unread notifications for a user.
func (r *NotificationRepository) UnreadCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx,
		`SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`, userID,
	).Scan(&count)
	return count, err
}

// MarkAsRead marks a single notification as read.
func (r *NotificationRepository) MarkAsRead(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET is_read = true, read_at = NOW() WHERE id = $1 AND user_id = $2`,
		id, userID,
	)
	return err
}

// MarkAllAsRead marks all of a user's notifications as read.
func (r *NotificationRepository) MarkAllAsRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE notifications SET is_read = true, read_at = NOW() WHERE user_id = $1 AND is_read = false`,
		userID,
	)
	return err
}

// CleanupOld deletes notifications older than the given time.
func (r *NotificationRepository) CleanupOld(ctx context.Context, before time.Time) (int64, error) {
	tag, err := r.db.Exec(ctx, `DELETE FROM notifications WHERE created_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListAll lists and paginates all notifications for admin logs.
func (r *NotificationRepository) ListAll(ctx context.Context, limit, offset int) ([]model.Notification, int, error) {
	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM notifications").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	query := `
		SELECT id, user_id, type, title, body, metadata, channel, is_read, read_at, created_at
		FROM notifications
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query notifications: %w", err)
	}
	defer rows.Close()

	var notifications []model.Notification
	for rows.Next() {
		var n model.Notification
		if err := rows.Scan(
			&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body,
			&n.Metadata, &n.Channel, &n.IsRead, &n.ReadAt, &n.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}

	return notifications, total, nil
}
