package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

type SubscriptionAuditRepository struct {
	db *pgxpool.Pool
}

func NewSubscriptionAuditRepository(db *pgxpool.Pool) *SubscriptionAuditRepository {
	return &SubscriptionAuditRepository{db: db}
}

func (r *SubscriptionAuditRepository) Log(ctx context.Context, entry *model.SubscriptionAuditLog) error {
	payloadJSON, _ := json.Marshal(entry.Payload)
	if len(payloadJSON) == 0 {
		payloadJSON = []byte("{}")
	}

	query := `
		INSERT INTO subscription_audit_logs (
			user_id, subscription_id, transaction_id, event_type, event_source,
			payload, status, error_message, ip_address, user_agent, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id, created_at
	`
	err := r.db.QueryRow(ctx, query,
		entry.UserID, entry.SubscriptionID, entry.TransactionID, entry.EventType, entry.EventSource,
		payloadJSON, entry.Status, entry.ErrorMessage, entry.IPAddress, entry.UserAgent,
	).Scan(&entry.ID, &entry.CreatedAt)

	if err != nil {
		return fmt.Errorf("insert subscription audit log: %w", err)
	}
	return nil
}

func (r *SubscriptionAuditRepository) ListAll(ctx context.Context, limit, offset int) ([]*model.SubscriptionAuditLog, int, error) {
	countQuery := `SELECT COUNT(*) FROM subscription_audit_logs`
	var total int
	if err := r.db.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count subscription audit logs: %w", err)
	}

	query := `
		SELECT id, user_id, subscription_id, transaction_id, event_type, event_source,
		       payload, status, error_message, ip_address, user_agent, created_at
		FROM subscription_audit_logs
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list subscription audit logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.SubscriptionAuditLog
	for rows.Next() {
		var l model.SubscriptionAuditLog
		var payloadJSON []byte
		err := rows.Scan(
			&l.ID, &l.UserID, &l.SubscriptionID, &l.TransactionID, &l.EventType, &l.EventSource,
			&payloadJSON, &l.Status, &l.ErrorMessage, &l.IPAddress, &l.UserAgent, &l.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scan subscription audit log: %w", err)
		}
		if len(payloadJSON) > 0 {
			_ = json.Unmarshal(payloadJSON, &l.Payload)
		}
		logs = append(logs, &l)
	}
	return logs, total, nil
}

func (r *SubscriptionAuditRepository) ExportDailyLogs(ctx context.Context, date time.Time) ([]*model.SubscriptionAuditLog, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	endOfDay := startOfDay.Add(24 * time.Hour)

	query := `
		SELECT id, user_id, subscription_id, transaction_id, event_type, event_source,
		       payload, status, error_message, ip_address, user_agent, created_at
		FROM subscription_audit_logs
		WHERE created_at >= $1 AND created_at < $2
		ORDER BY created_at ASC
	`
	rows, err := r.db.Query(ctx, query, startOfDay, endOfDay)
	if err != nil {
		return nil, fmt.Errorf("export daily logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.SubscriptionAuditLog
	for rows.Next() {
		var l model.SubscriptionAuditLog
		var payloadJSON []byte
		err := rows.Scan(
			&l.ID, &l.UserID, &l.SubscriptionID, &l.TransactionID, &l.EventType, &l.EventSource,
			&payloadJSON, &l.Status, &l.ErrorMessage, &l.IPAddress, &l.UserAgent, &l.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan daily audit log: %w", err)
		}
		if len(payloadJSON) > 0 {
			_ = json.Unmarshal(payloadJSON, &l.Payload)
		}
		logs = append(logs, &l)
	}
	return logs, nil
}
