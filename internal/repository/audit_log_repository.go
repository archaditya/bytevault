package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

type AuditLogRepository struct {
	db *pgxpool.Pool
}

func NewAuditLogRepository(db *pgxpool.Pool) *AuditLogRepository {
	return &AuditLogRepository{db: db}
}

func (r *AuditLogRepository) Create(ctx context.Context, log *model.AuditLog) error {
	query := `
		INSERT INTO audit_logs (
			tenant_id, actor_user_id, actor_type, action, resource_type, resource_id,
			status, request_id, ip_address, user_agent, metadata, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, COALESCE($12, NOW())
		)
		RETURNING id, created_at
	`

	var createdAt time.Time
	if !log.CreatedAt.IsZero() {
		createdAt = log.CreatedAt
	} else {
		createdAt = time.Now()
	}

	metadata := log.Metadata
	if len(metadata) == 0 {
		metadata = []byte("{}")
	}

	actorType := log.ActorType
	if actorType == "" {
		actorType = "user"
	}

	status := log.Status
	if status == "" {
		status = "success"
	}

	return r.db.QueryRow(
		ctx,
		query,
		log.TenantID,
		log.ActorUserID,
		actorType,
		log.Action,
		log.ResourceType,
		log.ResourceID,
		status,
		log.RequestID,
		log.IPAddress,
		log.UserAgent,
		metadata,
		createdAt,
	).Scan(&log.ID, &log.CreatedAt)
}

func (r *AuditLogRepository) List(ctx context.Context, filter model.AuditLogFilter) ([]model.AuditLog, int64, error) {
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	if filter.TenantID != nil && *filter.TenantID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("tenant_id = $%d", argIdx))
		args = append(args, *filter.TenantID)
		argIdx++
	}

	if filter.ActorUserID != nil && *filter.ActorUserID != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("actor_user_id = $%d", argIdx))
		args = append(args, *filter.ActorUserID)
		argIdx++
	}

	if filter.Action != nil && *filter.Action != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("action = $%d", argIdx))
		args = append(args, *filter.Action)
		argIdx++
	}

	if filter.ResourceType != nil && *filter.ResourceType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("resource_type = $%d", argIdx))
		args = append(args, *filter.ResourceType)
		argIdx++
	}

	if filter.Status != nil && *filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIdx))
		args = append(args, *filter.Status)
		argIdx++
	}

	if filter.StartDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argIdx))
		args = append(args, *filter.StartDate)
		argIdx++
	}

	if filter.EndDate != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argIdx))
		args = append(args, *filter.EndDate)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Total count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs WHERE %s", whereSQL)
	var total int64
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`
		SELECT id, tenant_id, actor_user_id, actor_type, action, resource_type, resource_id,
		       status, request_id, ip_address, user_agent, metadata, created_at
		FROM audit_logs
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereSQL, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []model.AuditLog
	for rows.Next() {
		var l model.AuditLog
		err := rows.Scan(
			&l.ID,
			&l.TenantID,
			&l.ActorUserID,
			&l.ActorType,
			&l.Action,
			&l.ResourceType,
			&l.ResourceID,
			&l.Status,
			&l.RequestID,
			&l.IPAddress,
			&l.UserAgent,
			&l.Metadata,
			&l.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan audit log row: %w", err)
		}
		logs = append(logs, l)
	}

	return logs, total, nil
}
