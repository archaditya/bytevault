package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/archaditya/bytevault/internal/model"
)

type FileRepository struct {
	db *pgxpool.Pool
}

func NewFileRepository(db *pgxpool.Pool) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(ctx context.Context, file *model.File) error {
	query := `
		INSERT INTO files (user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW(), NOW())
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
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create file record: %w", err)
	}
	return nil
}

func (r *FileRepository) FindByID(ctx context.Context, id string) (*model.File, error) {
	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, created_at, updated_at
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

func (r *FileRepository) ListByUserID(ctx context.Context, userID string) ([]*model.File, error) {
	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, created_at, updated_at
		FROM files
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
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
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, nil
}

func (r *FileRepository) UpdatePublicStatus(ctx context.Context, id string, isPublic bool) error {
	query := `UPDATE files SET is_public = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, isPublic, id)
	return err
}

func (r *FileRepository) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE files SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
