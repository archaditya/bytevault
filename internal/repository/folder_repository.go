package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/archaditya/bytevault/internal/model"
)

type FolderRepository struct {
	db *pgxpool.Pool
}

func NewFolderRepository(db *pgxpool.Pool) *FolderRepository {
	return &FolderRepository{db: db}
}

func (r *FolderRepository) Create(ctx context.Context, folder *model.Folder) error {
	query := `
		INSERT INTO folders (user_id, name, parent_id, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		folder.UserID,
		folder.Name,
		folder.ParentID,
		folder.IsPublic,
	).Scan(&folder.ID, &folder.CreatedAt, &folder.UpdatedAt)
}

func (r *FolderRepository) FindByID(ctx context.Context, id string) (*model.Folder, error) {
	query := `
		SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
		FROM folders
		WHERE id = $1 AND deleted_at IS NULL
	`
	var folder model.Folder
	err := r.db.QueryRow(ctx, query, id).Scan(
		&folder.ID,
		&folder.UserID,
		&folder.Name,
		&folder.ParentID,
		&folder.IsPublic,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find folder: %w", err)
	}
	return &folder, nil
}

func (r *FolderRepository) FindByIDPublic(ctx context.Context, id string) (*model.Folder, error) {
	query := `
		WITH RECURSIVE folder_hierarchy AS (
			SELECT id, user_id, name, parent_id, is_public, created_at, updated_at, 1 as depth
			FROM folders
			WHERE id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT p.id, p.user_id, p.name, p.parent_id, p.is_public, p.created_at, p.updated_at, fh.depth + 1
			FROM folders p
			INNER JOIN folder_hierarchy fh ON p.id = fh.parent_id
			WHERE p.deleted_at IS NULL AND fh.depth < 10
		)
		SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
		FROM folders
		WHERE id = $1 
		  AND deleted_at IS NULL 
		  AND EXISTS (SELECT 1 FROM folder_hierarchy WHERE is_public = true)
	`
	var folder model.Folder
	err := r.db.QueryRow(ctx, query, id).Scan(
		&folder.ID,
		&folder.UserID,
		&folder.Name,
		&folder.ParentID,
		&folder.IsPublic,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find public folder: %w", err)
	}
	return &folder, nil
}

func (r *FolderRepository) GetPublicBreadcrumbs(ctx context.Context, folderID string) ([]*model.Folder, error) {
	query := `
		WITH RECURSIVE folder_path AS (
			SELECT id, user_id, name, parent_id, is_public, created_at, updated_at, 1 as depth
			FROM folders
			WHERE id = $1 AND deleted_at IS NULL
			UNION ALL
			SELECT p.id, p.user_id, p.name, p.parent_id, p.is_public, p.created_at, p.updated_at, fp.depth + 1
			FROM folders p
			INNER JOIN folder_path fp ON p.id = fp.parent_id
			WHERE p.deleted_at IS NULL AND fp.depth < 10
		)
		SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
		FROM folder_path
		ORDER BY depth DESC
	`
	rows, err := r.db.Query(ctx, query, folderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var crumbs []*model.Folder
	for rows.Next() {
		var f model.Folder
		if err := rows.Scan(
			&f.ID,
			&f.UserID,
			&f.Name,
			&f.ParentID,
			&f.IsPublic,
			&f.CreatedAt,
			&f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		crumbs = append(crumbs, &f)
	}
	return crumbs, nil
}

func (r *FolderRepository) ListByUserID(ctx context.Context, userID string, parentID *string) ([]*model.Folder, error) {
	var query string
	var args []any

	if parentID == nil || *parentID == "" {
		query = `
			SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
			FROM folders
			WHERE user_id = $1 AND parent_id IS NULL AND deleted_at IS NULL
			ORDER BY name ASC
		`
		args = []any{userID}
	} else {
		query = `
			SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
			FROM folders
			WHERE user_id = $1 AND parent_id = $2 AND deleted_at IS NULL
			ORDER BY name ASC
		`
		args = []any{userID, *parentID}
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		var f model.Folder
		err := rows.Scan(
			&f.ID,
			&f.UserID,
			&f.Name,
			&f.ParentID,
			&f.IsPublic,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, &f)
	}
	return folders, nil
}

func (r *FolderRepository) ListPublicSubfolders(ctx context.Context, parentID string) ([]*model.Folder, error) {
	query := `
		SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
		FROM folders
		WHERE parent_id = $1 AND deleted_at IS NULL
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query, parentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		var f model.Folder
		err := rows.Scan(
			&f.ID,
			&f.UserID,
			&f.Name,
			&f.ParentID,
			&f.IsPublic,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, &f)
	}
	return folders, nil
}

func (r *FolderRepository) ListAllFlat(ctx context.Context, userID string) ([]*model.Folder, error) {
	query := `
		SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
		FROM folders
		WHERE user_id = $1 AND deleted_at IS NULL
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folders []*model.Folder
	for rows.Next() {
		var f model.Folder
		err := rows.Scan(
			&f.ID,
			&f.UserID,
			&f.Name,
			&f.ParentID,
			&f.IsPublic,
			&f.CreatedAt,
			&f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		folders = append(folders, &f)
	}
	return folders, nil
}

func (r *FolderRepository) UpdateParent(ctx context.Context, id string, parentID *string) error {
	query := `UPDATE folders SET parent_id = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, parentID, id)
	return err
}

func (r *FolderRepository) Rename(ctx context.Context, id string, name string) error {
	query := `UPDATE folders SET name = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, name, id)
	return err
}

func (r *FolderRepository) UpdatePublicStatus(ctx context.Context, id string, isPublic bool) error {
	query := `UPDATE folders SET is_public = $1, updated_at = NOW() WHERE id = $2 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, isPublic, id)
	return err
}

func (r *FolderRepository) SoftDelete(ctx context.Context, id string) error {
	queryFolder := `UPDATE folders SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, queryFolder, id)
	if err != nil {
		return err
	}

	queryFiles := `UPDATE files SET deleted_at = NOW(), updated_at = NOW() WHERE folder_id = $1`
	_, err = r.db.Exec(ctx, queryFiles, id)
	return err
}

func (r *FolderRepository) FindByName(ctx context.Context, userID, name string, parentID *string) (*model.Folder, error) {
	var query string
	var args []any
	if parentID == nil || *parentID == "" {
		query = `
			SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
			FROM folders
			WHERE user_id = $1 AND name = $2 AND parent_id IS NULL AND deleted_at IS NULL
			LIMIT 1
		`
		args = []any{userID, name}
	} else {
		query = `
			SELECT id, user_id, name, parent_id, is_public, created_at, updated_at
			FROM folders
			WHERE user_id = $1 AND name = $2 AND parent_id = $3 AND deleted_at IS NULL
			LIMIT 1
		`
		args = []any{userID, name, *parentID}
	}

	var folder model.Folder
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&folder.ID,
		&folder.UserID,
		&folder.Name,
		&folder.ParentID,
		&folder.IsPublic,
		&folder.CreatedAt,
		&folder.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find folder by name: %w", err)
	}
	return &folder, nil
}

