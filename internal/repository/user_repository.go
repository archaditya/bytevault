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
		&created.Password,
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

	result, err := r.db.Exec(ctx, query, id, deletedBy)
	if err != nil {
		return fmt.Errorf("failed to soft delete user: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrUserNotFound
	}

	return nil
}

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

	var totalFiles int64
	var totalStorage int64
	err = r.db.QueryRow(ctx, "SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM files WHERE deleted_at IS NULL AND status = 'READY'").Scan(&totalFiles, &totalStorage)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, "SELECT storage_provider, COALESCE(SUM(file_size), 0), COUNT(*) FROM files WHERE deleted_at IS NULL AND status = 'READY' GROUP BY storage_provider")
	var providerStats []map[string]any
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var provider string
			var size int64
			var count int
			if err := rows.Scan(&provider, &size, &count); err == nil {
				providerStats = append(providerStats, map[string]any{
					"provider":   provider,
					"used_bytes": size,
					"file_count": count,
				})
			}
		}
	}

	return map[string]any{
		"total_users":      totalUsers,
		"active_users":     activeUsers,
		"verified_users":   verifiedUsers,
		"active_sessions":  totalSessions,
		"total_files":      totalFiles,
		"total_storage":    totalStorage,
		"provider_storage": providerStats,
	}, nil
}

func (r *UserRepository) UpdateDetails(ctx context.Context, id string, firstName *string, lastName *string, status *string, isVerified *bool) error {
	query := `
		UPDATE users
		SET first_name = COALESCE($2, first_name),
		    last_name = COALESCE($3, last_name),
		    status = COALESCE($4, status),
		    is_verified = COALESCE($5, is_verified),
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, id, firstName, lastName, status, isVerified)
	return err
}

func (r *UserRepository) GetUserStorageStats(ctx context.Context, userID string) (int64, int64, error) {
	var totalFiles int64
	var totalStorage int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM files WHERE user_id = $1 AND deleted_at IS NULL AND status = 'READY'", userID).Scan(&totalFiles, &totalStorage)
	return totalFiles, totalStorage, err
}
