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
		INSERT INTO users (email, password, first_name, last_name, avatar_url, is_verified, status, storage_limit_bytes, max_file_size_bytes)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, email, password, first_name, last_name, avatar_url, is_verified, status, storage_limit_bytes, max_file_size_bytes, created_at, updated_at, deleted_at
	`

	var created model.User
	err := r.db.QueryRow(ctx, query, user.Email, user.Password, user.FirstName, user.LastName, user.AvatarURL, user.IsVerified, status, user.StorageLimitBytes, user.MaxFileSizeBytes).Scan(
		&created.ID,
		&created.Email,
		&created.Password,
		&created.FirstName,
		&created.LastName,
		&created.AvatarURL,
		&created.IsVerified,
		&created.Status,
		&created.StorageLimitBytes,
		&created.MaxFileSizeBytes,
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
		SELECT id, email, password, first_name, last_name, avatar_url, is_verified, status, storage_limit_bytes, max_file_size_bytes, created_at, updated_at, deleted_at
		FROM users
		WHERE LOWER(email) = LOWER($1) AND deleted_at IS NULL
	`

	var user model.User
	err := r.db.QueryRow(ctx, query, strings.TrimSpace(email)).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.FirstName,
		&user.LastName,
		&user.AvatarURL,
		&user.IsVerified,
		&user.Status,
		&user.StorageLimitBytes,
		&user.MaxFileSizeBytes,
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
		SELECT id, email, password, first_name, last_name, avatar_url, is_verified, status, storage_limit_bytes, max_file_size_bytes, created_at, updated_at, deleted_at
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
		&user.StorageLimitBytes,
		&user.MaxFileSizeBytes,
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

func (r *UserRepository) ListAll(ctx context.Context, search string, status string, role string, limit, offset int) ([]model.User, int, error) {
	var conditions []string
	var args []any
	argIndex := 1

	conditions = append(conditions, "u.deleted_at IS NULL")

	if search != "" {
		conditions = append(conditions, fmt.Sprintf("(u.email ILIKE $%d OR u.first_name ILIKE $%d OR u.last_name ILIKE $%d)", argIndex, argIndex, argIndex))
		args = append(args, "%"+search+"%")
		argIndex++
	}

	if status != "" {
		conditions = append(conditions, fmt.Sprintf("u.status = $%d", argIndex))
		args = append(args, status)
		argIndex++
	}

	if role != "" {
		conditions = append(conditions, fmt.Sprintf("r.name = $%d", argIndex))
		args = append(args, role)
		argIndex++
	}

	// Count query
	countQuery := `
		SELECT COUNT(DISTINCT u.id)
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE ` + strings.Join(conditions, " AND ")

	var total int
	err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Data query
	query := `
		SELECT DISTINCT u.id, u.email, u.first_name, u.last_name, u.avatar_url, u.is_verified, u.status, u.storage_limit_bytes, u.max_file_size_bytes, u.created_at, u.updated_at
		FROM users u
		LEFT JOIN user_roles ur ON u.id = ur.user_id
		LEFT JOIN roles r ON ur.role_id = r.id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY u.created_at DESC
	`

	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.AvatarURL,
			&u.IsVerified, &u.Status, &u.StorageLimitBytes, &u.MaxFileSizeBytes, &u.CreatedAt, &u.UpdatedAt,
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

func (r *UserRepository) UpdateDetails(ctx context.Context, id string, firstName *string, lastName *string, status *string, isVerified *bool, storageLimitBytes *int64, maxFileSizeBytes *int64) error {
	query := `
		UPDATE users
		SET first_name = COALESCE($2, first_name),
		    last_name = COALESCE($3, last_name),
		    status = COALESCE($4, status),
		    is_verified = COALESCE($5, is_verified),
		    storage_limit_bytes = COALESCE($6, storage_limit_bytes),
		    max_file_size_bytes = COALESCE($7, max_file_size_bytes),
		    updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
	`
	_, err := r.db.Exec(ctx, query, id, firstName, lastName, status, isVerified, storageLimitBytes, maxFileSizeBytes)
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

// GetAllAvatarURLs returns a map of all active avatar URLs.
func (r *UserRepository) GetAllAvatarURLs(ctx context.Context) (map[string]bool, error) {
	query := `SELECT avatar_url FROM users WHERE avatar_url IS NOT NULL AND deleted_at IS NULL`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	urls := make(map[string]bool)
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err == nil {
			urls[url] = true
		}
	}
	return urls, nil
}

// UpdateMFA sets the MFA secret and enabled flag for a user.
func (r *UserRepository) UpdateMFA(ctx context.Context, userID string, mfaEnabled bool, mfaSecret *string) error {
	query := `UPDATE users SET mfa_enabled = $2, mfa_secret = $3, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	_, err := r.db.Exec(ctx, query, userID, mfaEnabled, mfaSecret)
	return err
}

// GetMFAFields returns mfa_enabled and mfa_secret for a user by ID.
func (r *UserRepository) GetMFAFields(ctx context.Context, userID string) (bool, *string, error) {
	var enabled bool
	var secret *string
	err := r.db.QueryRow(ctx, `SELECT mfa_enabled, mfa_secret FROM users WHERE id = $1 AND deleted_at IS NULL`, userID).Scan(&enabled, &secret)
	if err != nil {
		return false, nil, err
	}
	return enabled, secret, nil
}

// --- Content Moderation: NSFW Strike & Restriction System ---

// IncrementNSFWStrikes atomically increments the user's NSFW strike count and returns the new count.
func (r *UserRepository) IncrementNSFWStrikes(ctx context.Context, userID string) (int, error) {
	var newCount int
	query := `UPDATE users SET nsfw_strikes = nsfw_strikes + 1, updated_at = NOW() WHERE id = $1 RETURNING nsfw_strikes`
	err := r.db.QueryRow(ctx, query, userID).Scan(&newCount)
	return newCount, err
}

// RestrictUser sets a temporary or permanent restriction on the user's account.
func (r *UserRepository) RestrictUser(ctx context.Context, userID string, until *time.Time, reason string) error {
	query := `UPDATE users SET restricted_until = $2, restriction_reason = $3, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID, until, reason)
	return err
}

// UnrestrictUser removes the restriction from the user's account and resets strikes.
func (r *UserRepository) UnrestrictUser(ctx context.Context, userID string) error {
	query := `UPDATE users SET restricted_until = NULL, restriction_reason = NULL, nsfw_strikes = 0, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// IsUserRestricted checks if a user is currently restricted (restricted_until > NOW).
func (r *UserRepository) IsUserRestricted(ctx context.Context, userID string) (bool, *time.Time, *string, error) {
	var restrictedUntil *time.Time
	var reason *string
	query := `SELECT restricted_until, restriction_reason FROM users WHERE id = $1 AND deleted_at IS NULL`
	err := r.db.QueryRow(ctx, query, userID).Scan(&restrictedUntil, &reason)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil, nil, nil
		}
		return false, nil, nil, err
	}
	if restrictedUntil == nil {
		return false, nil, nil, nil
	}
	if restrictedUntil.Before(time.Now()) {
		// Restriction expired — auto-clear
		_ = r.ClearExpiredRestriction(ctx, userID)
		return false, nil, nil, nil
	}
	return true, restrictedUntil, reason, nil
}

// ClearExpiredRestriction removes an expired restriction without resetting strikes.
func (r *UserRepository) ClearExpiredRestriction(ctx context.Context, userID string) error {
	query := `UPDATE users SET restricted_until = NULL, restriction_reason = NULL, updated_at = NOW() WHERE id = $1 AND restricted_until < NOW()`
	_, err := r.db.Exec(ctx, query, userID)
	return err
}

// ListRestrictedUsers returns all users currently under restriction (for admin dashboard).
func (r *UserRepository) ListRestrictedUsers(ctx context.Context) ([]model.User, error) {
	query := `
		SELECT id, email, first_name, last_name, nsfw_strikes, restricted_until, restriction_reason, created_at
		FROM users
		WHERE restricted_until IS NOT NULL AND restricted_until > NOW() AND deleted_at IS NULL
		ORDER BY restricted_until DESC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(&u.ID, &u.Email, &u.FirstName, &u.LastName, &u.NSFWStrikes, &u.RestrictedUntil, &u.RestrictionReason, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, nil
}

// GetRestrictedUsersCount returns the count of currently restricted users.
func (r *UserRepository) GetRestrictedUsersCount(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE restricted_until IS NOT NULL AND restricted_until > NOW() AND deleted_at IS NULL`
	err := r.db.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// --- Moderation Appeals ---

// CreateAppeal inserts a new moderation appeal from a restricted user.
func (r *UserRepository) CreateAppeal(ctx context.Context, userID string, reason string) (*model.ModerationAppeal, error) {
	var appeal model.ModerationAppeal
	query := `
		INSERT INTO moderation_appeals (user_id, reason) VALUES ($1, $2)
		RETURNING id, user_id, reason, status, created_at
	`
	err := r.db.QueryRow(ctx, query, userID, reason).Scan(
		&appeal.ID, &appeal.UserID, &appeal.Reason, &appeal.Status, &appeal.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create appeal: %w", err)
	}
	return &appeal, nil
}

// ListPendingAppeals returns all unresolved appeals for admin review.
func (r *UserRepository) ListPendingAppeals(ctx context.Context) ([]model.ModerationAppeal, error) {
	query := `
		SELECT a.id, a.user_id, a.reason, a.status, a.admin_notes, a.reviewed_by, a.created_at, a.reviewed_at,
		       COALESCE(u.email, '') as user_email, COALESCE(u.first_name || ' ' || u.last_name, '') as user_name
		FROM moderation_appeals a
		LEFT JOIN users u ON a.user_id = u.id
		WHERE a.status = 'pending'
		ORDER BY a.created_at ASC
	`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var appeals []model.ModerationAppeal
	for rows.Next() {
		var a model.ModerationAppeal
		if err := rows.Scan(
			&a.ID, &a.UserID, &a.Reason, &a.Status, &a.AdminNotes, &a.ReviewedBy,
			&a.CreatedAt, &a.ReviewedAt, &a.UserEmail, &a.UserName,
		); err != nil {
			return nil, err
		}
		appeals = append(appeals, a)
	}
	return appeals, nil
}

// ResolveAppeal marks an appeal as approved or rejected by the admin.
func (r *UserRepository) ResolveAppeal(ctx context.Context, appealID string, status string, adminNotes string, reviewedBy string) error {
	query := `
		UPDATE moderation_appeals
		SET status = $2, admin_notes = $3, reviewed_by = $4, reviewed_at = NOW()
		WHERE id = $1
	`
	_, err := r.db.Exec(ctx, query, appealID, status, adminNotes, reviewedBy)
	return err
}

// GetPendingAppealsCount returns the count of unresolved appeals.
func (r *UserRepository) GetPendingAppealsCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM moderation_appeals WHERE status = 'pending'`).Scan(&count)
	return count, err
}

// HasPendingAppeal checks if a user already has a pending appeal.
func (r *UserRepository) HasPendingAppeal(ctx context.Context, userID string) (bool, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM moderation_appeals WHERE user_id = $1 AND status = 'pending'`, userID).Scan(&count)
	return count > 0, err
}
