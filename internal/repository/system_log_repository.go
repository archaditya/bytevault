package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/archaditya/bytevault/internal/model"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SystemLogRepository struct {
	db *pgxpool.Pool
}

func NewSystemLogRepository(db *pgxpool.Pool) *SystemLogRepository {
	return &SystemLogRepository{db: db}
}

// UpsertLocalLog registers or updates a local daily log record.
func (r *SystemLogRepository) UpsertLocalLog(ctx context.Context, logDate time.Time, fileName, localPath string, fileSize int64) error {
	query := `
		INSERT INTO system_log_archives (log_date, file_name, storage_location, local_path, file_size_bytes, updated_at)
		VALUES ($1, $2, 'local', $3, $4, NOW())
		ON CONFLICT (log_date) DO UPDATE SET
			file_size_bytes = EXCLUDED.file_size_bytes,
			local_path = EXCLUDED.local_path,
			updated_at = NOW()
		WHERE system_log_archives.storage_location = 'local'
	`
	_, err := r.db.Exec(ctx, query, logDate.Format("2006-01-02"), fileName, localPath, fileSize)
	if err != nil {
		return fmt.Errorf("upsert local log archive: %w", err)
	}
	return nil
}

// MarkArchivedToR2 records that the file was compressed and uploaded to Cloudflare R2.
func (r *SystemLogRepository) MarkArchivedToR2(ctx context.Context, logDate time.Time, storageKey string, compressedSize int64) error {
	query := `
		UPDATE system_log_archives SET
			storage_location = 'r2',
			storage_key = $2,
			compressed_size_bytes = $3,
			archived_to_r2_at = NOW(),
			updated_at = NOW()
		WHERE log_date = $1
	`
	res, err := r.db.Exec(ctx, query, logDate.Format("2006-01-02"), storageKey, compressedSize)
	if err != nil {
		return fmt.Errorf("mark log archived to r2: %w", err)
	}
	if res.RowsAffected() == 0 {
		// Insert if not previously recorded
		insertQuery := `
			INSERT INTO system_log_archives (log_date, file_name, storage_location, storage_key, compressed_size_bytes, archived_to_r2_at, updated_at)
			VALUES ($1, $2, 'r2', $3, $4, NOW(), NOW())
			ON CONFLICT (log_date) DO UPDATE SET
				storage_location = 'r2',
				storage_key = EXCLUDED.storage_key,
				compressed_size_bytes = EXCLUDED.compressed_size_bytes,
				archived_to_r2_at = NOW(),
				updated_at = NOW()
		`
		fileName := fmt.Sprintf("%s.app.log", logDate.Format("02-01-2006"))
		_, err = r.db.Exec(ctx, insertQuery, logDate.Format("2006-01-02"), fileName, storageKey, compressedSize)
		return err
	}
	return nil
}

// ListExpiredR2 returns all log archives stored on R2 that are older than the cutoff (6 months).
func (r *SystemLogRepository) ListExpiredR2(ctx context.Context, cutoff time.Time) ([]*model.SystemLogArchive, error) {
	query := `
		SELECT id, log_date::text, file_name, storage_location, storage_key, local_path,
		       file_size_bytes, compressed_size_bytes, archived_to_r2_at, purged_at, created_at, updated_at
		FROM system_log_archives
		WHERE storage_location = 'r2' AND log_date < $1
	`
	rows, err := r.db.Query(ctx, query, cutoff.Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("list expired r2 logs: %w", err)
	}
	defer rows.Close()

	var logs []*model.SystemLogArchive
	for rows.Next() {
		var a model.SystemLogArchive
		if err := rows.Scan(
			&a.ID, &a.LogDate, &a.FileName, &a.StorageLocation, &a.StorageKey, &a.LocalPath,
			&a.FileSizeBytes, &a.CompressedSizeBytes, &a.ArchivedToR2At, &a.PurgedAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, &a)
	}
	return logs, nil
}

// MarkPurged marks a log as permanently purged from R2.
func (r *SystemLogRepository) MarkPurged(ctx context.Context, logDate string) error {
	query := `
		UPDATE system_log_archives SET
			storage_location = 'purged',
			purged_at = NOW(),
			updated_at = NOW()
		WHERE log_date = $1
	`
	_, err := r.db.Exec(ctx, query, logDate)
	if err != nil {
		return fmt.Errorf("mark log purged: %w", err)
	}
	return nil
}

// ListAll returns all tracked log archives for the Admin UI dashboard.
func (r *SystemLogRepository) ListAll(ctx context.Context, limit, offset int) ([]*model.SystemLogArchive, int, error) {
	var total int
	if err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM system_log_archives").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count system log archives: %w", err)
	}

	query := `
		SELECT id, log_date::text, file_name, storage_location, storage_key, local_path,
		       file_size_bytes, compressed_size_bytes, archived_to_r2_at, purged_at, created_at, updated_at
		FROM system_log_archives
		ORDER BY log_date DESC
		LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list system log archives: %w", err)
	}
	defer rows.Close()

	var list []*model.SystemLogArchive
	for rows.Next() {
		var a model.SystemLogArchive
		if err := rows.Scan(
			&a.ID, &a.LogDate, &a.FileName, &a.StorageLocation, &a.StorageKey, &a.LocalPath,
			&a.FileSizeBytes, &a.CompressedSizeBytes, &a.ArchivedToR2At, &a.PurgedAt, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		list = append(list, &a)
	}
	return list, total, nil
}
