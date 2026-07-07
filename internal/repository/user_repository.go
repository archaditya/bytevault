package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	status := "active"
	if user.Status != nil {
		status = *user.Status
	}

	query := `
		INSERT INTO users (email, password, first_name, last_name, avatar_url, is_verified, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, email, password, first_name, last_name, avatar_url, is_verified, status, created_at, updated_at, deleted_at
	`

	var created model.User
	err := r.db.QueryRow(ctx, query, user.Email, user.Password, user.FirstName, user.LastName, user.AvatarURL, user.IsVerified, status).Scan(
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
	created.HasPassword = created.Password != nil
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
	user.HasPassword = user.Password != nil
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
	user.HasPassword = user.Password != nil
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
	// Treat NULL status as active
	err = r.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE deleted_at IS NULL AND (status IS NULL OR status != 'inactive')").Scan(&activeUsers)
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

func (r *UserRepository) UpdatePassword(ctx context.Context, id string, hashedPassword string) error {
	query := `UPDATE users SET password = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id, hashedPassword)
	return err
}

func (r *UserRepository) UpdateAvatarURL(ctx context.Context, id string, avatarURL string) error {
	query := `UPDATE users SET avatar_url = $2, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, id, avatarURL)
	return err
}

func (r *UserRepository) GetUserStorageStats(ctx context.Context, userID string) (int64, int64, error) {
	var totalFiles int64
	var totalStorage int64
	err := r.db.QueryRow(ctx, "SELECT COUNT(*), COALESCE(SUM(file_size), 0) FROM files WHERE user_id = $1 AND deleted_at IS NULL AND status = 'READY'", userID).Scan(&totalFiles, &totalStorage)
	return totalFiles, totalStorage, err
}

func (r *UserRepository) MarkVerified(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx,
		`UPDATE users SET is_verified = true, updated_at = NOW() WHERE id = $1`,
		userID,
	)
	return err
}


// PurgeDeletedUserFiles removes file records for users soft-deleted before the cutoff.
// User rows are kept to prevent free-tier quota re-abuse.
func (r *UserRepository) PurgeDeletedUserFiles(ctx context.Context, cutoff time.Time) ([]string, error) {
	query := `
		DELETE FROM files
		WHERE user_id IN (SELECT id FROM users WHERE deleted_at IS NOT NULL AND deleted_at < $1)
		RETURNING storage_key
	`
	rows, err := r.db.Query(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to purge deleted user files: %w", err)
	}
	defer rows.Close()

	var keys []string
	for rows.Next() {
		var key string
		if err := rows.Scan(&key); err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	return keys, nil
}

func (r *UserRepository) ListAllActiveIDs(ctx context.Context) ([]string, error) {
	// Treat NULL status as active
	query := `SELECT id FROM users WHERE deleted_at IS NULL AND (status IS NULL OR status != 'inactive')`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func (r *UserRepository) FindUserIDsByRole(ctx context.Context, roleName string) ([]string, error) {
	query := `
		SELECT ur.user_id 
		FROM user_roles ur 
		JOIN roles r ON ur.role_id = r.id 
		JOIN users u ON ur.user_id = u.id
		WHERE r.name = $1 AND u.deleted_at IS NULL AND (u.status IS NULL OR u.status != 'inactive')
	`
	rows, err := r.db.Query(ctx, query, roleName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}
