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
	ErrUserNotFound = errors.New("user not found")
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) (*model.User, error) {
	query := `
		INSERT INTO users (email, password, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password, first_name, last_name, avatar_url, is_verified, status, created_at, updated_at, deleted_at
	`

	var created model.User
	err := r.db.QueryRow(ctx, query, user.Email, user.Password, user.FirstName, user.LastName).Scan(
		&created.ID,
		&created.Email,
		&created.Password, // BUG FIX: was missing, must match RETURNING column order
		&created.FirstName,
		&created.LastName,
		&created.AvatarURL,
		&created.IsVerified,
		&created.Status,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("Failed to create user: %w", err)
	}

	return &created, nil
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, email, password, first_name, last_name, avatar_url, is_verified, status, created_at, updated_at, deleted_at
		FROM users
		WHERE email = $1 AND deleted_at IS NULL
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.IsVerified,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		// BUG FIX: was missing ErrNoRows check — without this,
		// "user not found" and "DB error" look the same to the caller
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by email: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (*model.User, error) {
	query := `
		SELECT id, email, password, first_name, last_name, avatar_url, is_verified, status, created_at, updated_at, deleted_at
		FROM users
		WHERE id = $1 AND deleted_at IS NULL
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.IsVerified,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user by id: %w", err)
	}

	return &user, nil
}

func (r *UserRepository) SoftDelete(ctx context.Context, id string, deletedBy string) error {
	query := `
		UPDATE users
		SET deleted_at = NOW(), deleted_by = $2, updated_at = NOW(), status = 'inactive'
		WHERE id = $1 AND deleted_at IS NULL
	`
	// BUG FIX: SQL strings use single quotes, not double quotes.
	// Double quotes in PostgreSQL mean column/table names, not string values.

	result, err := r.db.Exec(ctx, query, id, deletedBy)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

// ListAll returns paginated users (for admin)
func (r *UserRepository) ListAll(ctx context.Context, limit, offset int) ([]model.User, int, error) {
	var total int
	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	query := `
		SELECT id, email, first_name, last_name, avatar_url, is_verified, status, created_at, updated_at
		FROM users WHERE deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $1 OFFSET $2
	`
	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL,
			&u.IsVerified, &u.Status, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, u)
	}

	return users, total, nil
}

// GetStats returns system-wide stats for admin dashboard
func (r *UserRepository) GetStats(ctx context.Context) (map[string]any, error) {
	var totalUsers, activeUsers, verifiedUsers int

	err := r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL").Scan(&totalUsers)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND status != 'inactive'").Scan(&activeUsers)
	if err != nil {
		return nil, err
	}
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND is_verified = true").Scan(&verifiedUsers)
	if err != nil {
		return nil, err
	}

	var totalSessions int
	r.db.QueryRow(ctx, "SELECT COUNT(*) FROM sessions WHERE expires_at > NOW()").Scan(&totalSessions)

	return map[string]any{
		"total_users":    totalUsers,
		"active_users":   activeUsers,
		"verified_users": verifiedUsers,
		"active_sessions": totalSessions,
	}, nil
}