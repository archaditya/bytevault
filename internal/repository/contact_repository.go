package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/archaditya/bytevault/internal/model"
)

type ContactRepository struct {
	db *pgxpool.Pool
}

func NewContactRepository(db *pgxpool.Pool) *ContactRepository {
	return &ContactRepository{db: db}
}

func (r *ContactRepository) Create(ctx context.Context, q *model.ContactQuery) error {
	query := `
		INSERT INTO contact_queries (name, email, subject, message, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, 'pending', NOW(), NOW())
		RETURNING id, status, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query, q.Name, q.Email, q.Subject, q.Message).
		Scan(&q.ID, &q.Status, &q.CreatedAt, &q.UpdatedAt)
}

func (r *ContactRepository) ListAll(ctx context.Context, limit, offset int) ([]*model.ContactQuery, int, error) {
	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM contact_queries").Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT q.id, q.name, q.email, q.subject, q.message, q.reply, q.replied_at, q.replied_by, q.status, q.created_at, q.updated_at,
		       COALESCE(u.first_name || ' ' || u.last_name, '') as replier_name,
		       COALESCE(u.email, '') as replier_email
		FROM contact_queries q
		LEFT JOIN users u ON q.replied_by = u.id
		ORDER BY q.created_at DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var queries []*model.ContactQuery
	for rows.Next() {
		var q model.ContactQuery
		err := rows.Scan(
			&q.ID, &q.Name, &q.Email, &q.Subject, &q.Message, &q.Reply, &q.RepliedAt, &q.RepliedBy, &q.Status, &q.CreatedAt, &q.UpdatedAt,
			&q.ReplierName, &q.ReplierEmail,
		)
		if err != nil {
			return nil, 0, err
		}
		queries = append(queries, &q)
	}
	return queries, total, nil
}

func (r *ContactRepository) FindByID(ctx context.Context, id string) (*model.ContactQuery, error) {
	query := `
		SELECT q.id, q.name, q.email, q.subject, q.message, q.reply, q.replied_at, q.replied_by, q.status, q.created_at, q.updated_at,
		       COALESCE(u.first_name || ' ' || u.last_name, '') as replier_name,
		       COALESCE(u.email, '') as replier_email
		FROM contact_queries q
		LEFT JOIN users u ON q.replied_by = u.id
		WHERE q.id = $1
	`
	var q model.ContactQuery
	err := r.db.QueryRow(ctx, query, id).Scan(
		&q.ID, &q.Name, &q.Email, &q.Subject, &q.Message, &q.Reply, &q.RepliedAt, &q.RepliedBy, &q.Status, &q.CreatedAt, &q.UpdatedAt,
		&q.ReplierName, &q.ReplierEmail,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find contact query: %w", err)
	}
	return &q, nil
}

func (r *ContactRepository) UpdateReply(ctx context.Context, id string, reply string, repliedBy string) error {
	query := `
		UPDATE contact_queries
		SET reply = $1, replied_by = $2, replied_at = NOW(), status = 'replied', updated_at = NOW()
		WHERE id = $3
	`
	_, err := r.db.Exec(ctx, query, reply, repliedBy, id)
	return err
}
