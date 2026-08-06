package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/archaditya/bytevault/internal/model"
)

type ShareRepository struct {
	db *pgxpool.Pool
}

func NewShareRepository(db *pgxpool.Pool) *ShareRepository {
	return &ShareRepository{db: db}
}

func (r *ShareRepository) FindByID(ctx context.Context, shareID string) (*model.Share, error) {
	query := `
		SELECT id, resource_type, resource_id, owner_id, grantee_email, grantee_id, permission, created_at, updated_at
		FROM shares
		WHERE id = $1
	`
	var s model.Share
	err := r.db.QueryRow(ctx, query, shareID).Scan(
		&s.ID, &s.ResourceType, &s.ResourceID, &s.OwnerID, &s.GranteeEmail, &s.GranteeID, &s.Permission, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("share not found")
		}
		return nil, err
	}
	return &s, nil
}

func (r *ShareRepository) Create(ctx context.Context, share *model.Share) error {
	query := `
		INSERT INTO shares (resource_type, resource_id, owner_id, grantee_email, grantee_id, permission, created_at, updated_at)
		VALUES ($1, $2, $3, LOWER($4), $5, $6, NOW(), NOW())
		ON CONFLICT (resource_type, resource_id, grantee_email) 
		DO UPDATE SET permission = EXCLUDED.permission, updated_at = NOW()
		RETURNING id, created_at, updated_at
	`
	return r.db.QueryRow(ctx, query,
		share.ResourceType,
		share.ResourceID,
		share.OwnerID,
		strings.ToLower(share.GranteeEmail),
		share.GranteeID,
		share.Permission,
	).Scan(&share.ID, &share.CreatedAt, &share.UpdatedAt)
}

func (r *ShareRepository) FindByResourceAndGrantee(ctx context.Context, resourceType, resourceID, granteeEmail string) (*model.Share, error) {
	query := `
		SELECT id, resource_type, resource_id, owner_id, grantee_email, grantee_id, permission, created_at, updated_at
		FROM shares
		WHERE resource_type = $1 AND resource_id = $2 AND LOWER(grantee_email) = LOWER($3)
	`
	var s model.Share
	err := r.db.QueryRow(ctx, query, resourceType, resourceID, granteeEmail).Scan(
		&s.ID, &s.ResourceType, &s.ResourceID, &s.OwnerID, &s.GranteeEmail, &s.GranteeID, &s.Permission, &s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &s, nil
}

func (r *ShareRepository) ListByResource(ctx context.Context, resourceType, resourceID, ownerID string) ([]*model.Share, error) {
	query := `
		SELECT id, resource_type, resource_id, owner_id, grantee_email, grantee_id, permission, created_at, updated_at
		FROM shares
		WHERE resource_type = $1 AND resource_id = $2 AND owner_id = $3
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, resourceType, resourceID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*model.Share
	for rows.Next() {
		var s model.Share
		if err := rows.Scan(&s.ID, &s.ResourceType, &s.ResourceID, &s.OwnerID, &s.GranteeEmail, &s.GranteeID, &s.Permission, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		shares = append(shares, &s)
	}
	return shares, nil
}

func (r *ShareRepository) ListSharedWithMe(ctx context.Context, granteeEmail string) ([]*model.Share, error) {
	query := `
		SELECT id, resource_type, resource_id, owner_id, grantee_email, grantee_id, permission, created_at, updated_at
		FROM shares
		WHERE LOWER(grantee_email) = LOWER($1)
		ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, granteeEmail)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []*model.Share
	for rows.Next() {
		var s model.Share
		if err := rows.Scan(&s.ID, &s.ResourceType, &s.ResourceID, &s.OwnerID, &s.GranteeEmail, &s.GranteeID, &s.Permission, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		shares = append(shares, &s)
	}
	return shares, nil
}

func (r *ShareRepository) Revoke(ctx context.Context, shareID, ownerID string) error {
	query := `DELETE FROM shares WHERE id = $1 AND owner_id = $2`
	res, err := r.db.Exec(ctx, query, shareID, ownerID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("share not found or unauthorized")
	}
	return nil
}
