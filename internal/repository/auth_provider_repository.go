package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

var (
	ErrAuthProviderNotFound = errors.New("auth provider link not found")
)

type AuthProviderRepository struct {
	db *pgxpool.Pool
}

func NewAuthProviderRepository(db *pgxpool.Pool) *AuthProviderRepository {
	return &AuthProviderRepository{db: db}
}

func (r *AuthProviderRepository) Create(ctx context.Context, ap *model.AuthProvider) (*model.AuthProvider, error) {
	query := `
		INSERT INTO auth_providers (user_id, provider, provider_user_id, email)
		VALUES ($1, $2, $3, $4)
		RETURNING id, user_id, provider, provider_user_id, email, created_at, updated_at
	`

	var created model.AuthProvider
	err := r.db.QueryRow(ctx, query, ap.UserID, ap.Provider, ap.ProviderUserID, ap.Email).Scan(
		&created.ID,
		&created.UserID,
		&created.Provider,
		&created.ProviderUserID,
		&created.Email,
		&created.CreatedAt,
		&created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth provider link: %w", err)
	}

	return &created, nil
}

func (r *AuthProviderRepository) FindByProviderAndID(ctx context.Context, provider, providerUserID string) (*model.AuthProvider, error) {
	query := `
		SELECT id, user_id, provider, provider_user_id, email, created_at, updated_at
		FROM auth_providers
		WHERE provider = $1 AND provider_user_id = $2 AND deleted_at IS NULL
	`

	var ap model.AuthProvider
	err := r.db.QueryRow(ctx, query, provider, providerUserID).Scan(
		&ap.ID,
		&ap.UserID,
		&ap.Provider,
		&ap.ProviderUserID,
		&ap.Email,
		&ap.CreatedAt,
		&ap.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAuthProviderNotFound
		}
		return nil, fmt.Errorf("failed to find auth provider: %w", err)
	}

	return &ap, nil
}
