package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/archaditya/bytevault/internal/model"
)

type ListFilesParams struct {
	UserID   string
	FolderID *string
	Search   string
	SortBy   string // name, size, date
	SortDir  string // asc, desc
	Limit    int
	Cursor   string // RFC3339 timestamp
}

type FileRepository struct {
	db *pgxpool.Pool
}

func NewFileRepository(db *pgxpool.Pool) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(ctx context.Context, file *model.File) error {
	query := `
		INSERT INTO files (user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, status, folder_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	err := r.db.QueryRow(ctx, query,
		file.UserID,
		file.Filename,
		file.StorageProvider,
		file.Bucket,
		file.StorageKey,
		file.FileSize,
		file.ContentType,
		file.IsPublic,
		file.Status,
		file.FolderID,
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create file record: %w", err)
	}
	return nil
}

func (r *FileRepository) FindByID(ctx context.Context, id string) (*model.File, error) {
	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, status, folder_id, created_at, updated_at
		FROM files
		WHERE id = $1 AND deleted_at IS NULL
	`
	var file model.File
	err := r.db.QueryRow(ctx, query, id).Scan(
		&file.ID,
		&file.UserID,
		&file.Filename,
		&file.StorageProvider,
		&file.Bucket,
		&file.StorageKey,
		&file.FileSize,
		&file.ContentType,
		&file.IsPublic,
		&file.Status,
		&file.FolderID,
		&file.CreatedAt,
		&file.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find file metadata: %w", err)
	}
	return &file, nil
}

func (r *FileRepository) ListByUserID(ctx context.Context, params ListFilesParams) ([]*model.File, string, error) {
	var conditions []string
	var args []any
	argIndex := 1

	conditions = append(conditions, fmt.Sprintf("user_id = $%d", argIndex))
	args = append(args, params.UserID)
	argIndex++

	conditions = append(conditions, "deleted_at IS NULL")
	conditions = append(conditions, "status = 'READY'")

	// Folder or Global Search filter
	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("filename ILIKE $%d", argIndex))
		args = append(args, "%"+params.Search+"%")
		argIndex++
	} else {
		if params.FolderID != nil && *params.FolderID != "" {
			conditions = append(conditions, fmt.Sprintf("folder_id = $%d", argIndex))
			args = append(args, *params.FolderID)
			argIndex++
		} else {
			conditions = append(conditions, "folder_id IS NULL")
		}
	}

	// Cursor Pagination filter
	if params.Cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, params.Cursor)
		if err == nil {
			if params.SortDir == "asc" {
				conditions = append(conditions, fmt.Sprintf("created_at > $%d", argIndex))
			} else {
				conditions = append(conditions, fmt.Sprintf("created_at < $%d", argIndex))
			}
			args = append(args, cursorTime)
			argIndex++
		}
	}

	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, status, folder_id, created_at, updated_at
		FROM files
		WHERE ` + strings.Join(conditions, " AND ")

	// Order By Field
	orderBy := "created_at"
	if params.SortBy == "name" {
		orderBy = "filename"
	} else if params.SortBy == "size" {
		orderBy = "file_size"
	}

	// Order By Direction
	orderDir := "DESC"
	if params.SortDir == "asc" {
		orderDir = "ASC"
	}

	query += fmt.Sprintf(" ORDER BY %s %s", orderBy, orderDir)

	// Limit
	limitVal := 20
	if params.Limit > 0 {
		limitVal = params.Limit
	}
	query += fmt.Sprintf(" LIMIT %d", limitVal)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		var f model.File
		err := rows.Scan(
			&f.ID,
			&f.UserID,
			&f.Filename,
			&f.StorageProvider,
			&f.Bucket,
			&f.StorageKey,
			&f.FileSize,
			&f.ContentType,
			&f.IsPublic,
			&f.Status,
			&f.FolderID,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, "", err
		}
		files = append(files, &f)
	}

	// Calculate Next Cursor
	nextCursor := ""
	if len(files) == limitVal && len(files) > 0 {
		nextCursor = files[len(files)-1].CreatedAt.Format(time.RFC3339)
	}

	return files, nextCursor, nil
}

func (r *FileRepository) UpdatePublicStatus(ctx context.Context, id string, isPublic bool) error {
	query := `UPDATE files SET is_public = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, isPublic, id)
	return err
}

func (r *FileRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	query := `UPDATE files SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, status, id)
	return err
}

func (r *FileRepository) MoveFile(ctx context.Context, id string, folderID *string) error {
	query := `UPDATE files SET folder_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, folderID, id)
	return err
}

func (r *FileRepository) GetUserStorageUsed(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COALESCE(SUM(file_size), 0) FROM files WHERE user_id = $1 AND deleted_at IS NULL AND status = 'READY'`
	var total int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}

func (r *FileRepository) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE files SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
