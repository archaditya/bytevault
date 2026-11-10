package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/archaditya/bytevault/internal/model"
)

type AbuseRepository struct {
	db *pgxpool.Pool
}

func NewAbuseRepository(db *pgxpool.Pool) *AbuseRepository {
	return &AbuseRepository{db: db}
}

func (r *AbuseRepository) Create(ctx context.Context, report *model.AbuseReport) error {
	query := `
		INSERT INTO abuse_reports (
			ticket_number, file_id, target_url, category,
			reporter_name, reporter_email, description, evidence_url,
			status, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending', NOW(), NOW())
		RETURNING id, status, created_at, updated_at
	`
	return r.db.QueryRow(
		ctx, query,
		report.TicketNumber, report.FileID, report.TargetURL, report.Category,
		report.ReporterName, report.ReporterEmail, report.Description, report.EvidenceURL,
	).Scan(&report.ID, &report.Status, &report.CreatedAt, &report.UpdatedAt)
}

func (r *AbuseRepository) FindByID(ctx context.Context, id string) (*model.AbuseReport, error) {
	query := `
		SELECT 
			ar.id, ar.ticket_number, ar.file_id, ar.target_url, ar.category,
			ar.reporter_name, ar.reporter_email, ar.description, ar.evidence_url,
			ar.status, ar.admin_notes, ar.reviewed_by, ar.reviewed_at, ar.created_at, ar.updated_at,
			f.filename, f.file_size, f.content_type, f.status,
			u.id, u.email, u.nsfw_strikes,
			rev.email
		FROM abuse_reports ar
		LEFT JOIN files f ON ar.file_id = f.id
		LEFT JOIN users u ON f.user_id = u.id
		LEFT JOIN users rev ON ar.reviewed_by = rev.id
		WHERE ar.id = $1
	`
	var rep model.AbuseReport
	err := r.db.QueryRow(ctx, query, id).Scan(
		&rep.ID, &rep.TicketNumber, &rep.FileID, &rep.TargetURL, &rep.Category,
		&rep.ReporterName, &rep.ReporterEmail, &rep.Description, &rep.EvidenceURL,
		&rep.Status, &rep.AdminNotes, &rep.ReviewedBy, &rep.ReviewedAt, &rep.CreatedAt, &rep.UpdatedAt,
		&rep.FileName, &rep.FileSize, &rep.FileContentType, &rep.FileStatus,
		&rep.UploaderID, &rep.UploaderEmail, &rep.UploaderStrikes,
		&rep.ReviewerEmail,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find abuse report: %w", err)
	}
	return &rep, nil
}

func (r *AbuseRepository) FindByTicket(ctx context.Context, ticket string) (*model.AbuseReport, error) {
	query := `
		SELECT 
			ar.id, ar.ticket_number, ar.file_id, ar.target_url, ar.category,
			ar.reporter_name, ar.reporter_email, ar.description, ar.evidence_url,
			ar.status, ar.admin_notes, ar.reviewed_by, ar.reviewed_at, ar.created_at, ar.updated_at,
			f.filename, f.file_size, f.content_type, f.status,
			u.id, u.email, u.nsfw_strikes,
			rev.email
		FROM abuse_reports ar
		LEFT JOIN files f ON ar.file_id = f.id
		LEFT JOIN users u ON f.user_id = u.id
		LEFT JOIN users rev ON ar.reviewed_by = rev.id
		WHERE ar.ticket_number = $1
	`
	var rep model.AbuseReport
	err := r.db.QueryRow(ctx, query, ticket).Scan(
		&rep.ID, &rep.TicketNumber, &rep.FileID, &rep.TargetURL, &rep.Category,
		&rep.ReporterName, &rep.ReporterEmail, &rep.Description, &rep.EvidenceURL,
		&rep.Status, &rep.AdminNotes, &rep.ReviewedBy, &rep.ReviewedAt, &rep.CreatedAt, &rep.UpdatedAt,
		&rep.FileName, &rep.FileSize, &rep.FileContentType, &rep.FileStatus,
		&rep.UploaderID, &rep.UploaderEmail, &rep.UploaderStrikes,
		&rep.ReviewerEmail,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find abuse report by ticket: %w", err)
	}
	return &rep, nil
}

func (r *AbuseRepository) ListAll(ctx context.Context, status string, limit, offset int) ([]*model.AbuseReport, int, error) {
	countQuery := `SELECT COUNT(*) FROM abuse_reports WHERE ($1 = '' OR status = $1)`
	var total int
	err := r.db.QueryRow(ctx, countQuery, status).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	query := `
		SELECT 
			ar.id, ar.ticket_number, ar.file_id, ar.target_url, ar.category,
			ar.reporter_name, ar.reporter_email, ar.description, ar.evidence_url,
			ar.status, ar.admin_notes, ar.reviewed_by, ar.reviewed_at, ar.created_at, ar.updated_at,
			f.filename, f.file_size, f.content_type, f.status,
			u.id, u.email, u.nsfw_strikes,
			rev.email
		FROM abuse_reports ar
		LEFT JOIN files f ON ar.file_id = f.id
		LEFT JOIN users u ON f.user_id = u.id
		LEFT JOIN users rev ON ar.reviewed_by = rev.id
		WHERE ($1 = '' OR ar.status = $1)
		ORDER BY ar.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := r.db.Query(ctx, query, status, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var reports []*model.AbuseReport
	for rows.Next() {
		var rep model.AbuseReport
		err := rows.Scan(
			&rep.ID, &rep.TicketNumber, &rep.FileID, &rep.TargetURL, &rep.Category,
			&rep.ReporterName, &rep.ReporterEmail, &rep.Description, &rep.EvidenceURL,
			&rep.Status, &rep.AdminNotes, &rep.ReviewedBy, &rep.ReviewedAt, &rep.CreatedAt, &rep.UpdatedAt,
			&rep.FileName, &rep.FileSize, &rep.FileContentType, &rep.FileStatus,
			&rep.UploaderID, &rep.UploaderEmail, &rep.UploaderStrikes,
			&rep.ReviewerEmail,
		)
		if err != nil {
			return nil, 0, err
		}
		reports = append(reports, &rep)
	}
	return reports, total, nil
}

func (r *AbuseRepository) UpdateStatusAndReview(ctx context.Context, id string, status string, adminNotes *string, reviewedBy string) error {
	query := `
		UPDATE abuse_reports
		SET status = $1, admin_notes = $2, reviewed_by = $3, reviewed_at = NOW(), updated_at = NOW()
		WHERE id = $4
	`
	_, err := r.db.Exec(ctx, query, status, adminNotes, reviewedBy, id)
	return err
}
