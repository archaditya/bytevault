package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/archaditya/bytevault/internal/model"
)

type UploadInviteRepository struct {
	db *pgxpool.Pool
}

func NewUploadInviteRepository(db *pgxpool.Pool) *UploadInviteRepository {
	return &UploadInviteRepository{db: db}
}

// Create inserts a new upload invite.
func (r *UploadInviteRepository) Create(ctx context.Context, invite *model.UploadInvite) error {
	query := `
		INSERT INTO upload_invites (token, owner_id, target_folder_id, label, max_total_bytes, max_files, passcode_hash, status, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at`
	return r.db.QueryRow(ctx, query,
		invite.Token, invite.OwnerID, invite.TargetFolderID, invite.Label,
		invite.MaxTotalBytes, invite.MaxFiles, invite.PasscodeHash,
		invite.Status, invite.ExpiresAt,
	).Scan(&invite.ID, &invite.CreatedAt, &invite.UpdatedAt)
}

// FindByToken retrieves an invite by its public token, joining owner info for guest display.
func (r *UploadInviteRepository) FindByToken(ctx context.Context, token string) (*model.UploadInvite, error) {
	query := `
		SELECT i.id, i.token, i.owner_id, i.target_folder_id, i.label,
		       i.max_total_bytes, i.max_files, i.used_bytes, i.used_files,
		       i.passcode_hash, i.status, i.pending_notify_count, i.last_notified_at,
		       i.expires_at, i.created_at, i.updated_at,
		       COALESCE(NULLIF(TRIM(COALESCE(u.first_name, '') || ' ' || COALESCE(u.last_name, '')), ''), u.email) AS owner_name,
		       u.email AS owner_email
		FROM upload_invites i
		JOIN users u ON u.id = i.owner_id
		WHERE i.token = $1`

	inv := &model.UploadInvite{}
	err := r.db.QueryRow(ctx, query, token).Scan(
		&inv.ID, &inv.Token, &inv.OwnerID, &inv.TargetFolderID, &inv.Label,
		&inv.MaxTotalBytes, &inv.MaxFiles, &inv.UsedBytes, &inv.UsedFiles,
		&inv.PasscodeHash, &inv.Status, &inv.PendingNotifyCount, &inv.LastNotifiedAt,
		&inv.ExpiresAt, &inv.CreatedAt, &inv.UpdatedAt,
		&inv.OwnerName, &inv.OwnerEmail,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find invite by token: %w", err)
	}
	inv.HasPasscode = inv.PasscodeHash != nil && *inv.PasscodeHash != ""
	return inv, nil
}

// FindByID retrieves an invite by UUID (for owner management).
func (r *UploadInviteRepository) FindByID(ctx context.Context, id string) (*model.UploadInvite, error) {
	query := `
		SELECT id, token, owner_id, target_folder_id, label,
		       max_total_bytes, max_files, used_bytes, used_files,
		       passcode_hash, status, pending_notify_count, last_notified_at,
		       expires_at, created_at, updated_at
		FROM upload_invites WHERE id = $1`

	inv := &model.UploadInvite{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.Token, &inv.OwnerID, &inv.TargetFolderID, &inv.Label,
		&inv.MaxTotalBytes, &inv.MaxFiles, &inv.UsedBytes, &inv.UsedFiles,
		&inv.PasscodeHash, &inv.Status, &inv.PendingNotifyCount, &inv.LastNotifiedAt,
		&inv.ExpiresAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find invite by ID: %w", err)
	}
	inv.HasPasscode = inv.PasscodeHash != nil && *inv.PasscodeHash != ""
	return inv, nil
}

// ListByOwner paginates invites for the owner's dashboard.
func (r *UploadInviteRepository) ListByOwner(ctx context.Context, ownerID string, limit, offset int) ([]model.UploadInvite, int, error) {
	countQuery := `SELECT COUNT(*) FROM upload_invites WHERE owner_id = $1`
	var total int
	if err := r.db.QueryRow(ctx, countQuery, ownerID).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, token, owner_id, target_folder_id, label,
		       max_total_bytes, max_files, used_bytes, used_files,
		       passcode_hash IS NOT NULL AND passcode_hash != '' AS has_passcode,
		       status, expires_at, created_at, updated_at
		FROM upload_invites
		WHERE owner_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, ownerID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var invites []model.UploadInvite
	for rows.Next() {
		var inv model.UploadInvite
		if err := rows.Scan(
			&inv.ID, &inv.Token, &inv.OwnerID, &inv.TargetFolderID, &inv.Label,
			&inv.MaxTotalBytes, &inv.MaxFiles, &inv.UsedBytes, &inv.UsedFiles,
			&inv.HasPasscode, &inv.Status,
			&inv.ExpiresAt, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		invites = append(invites, inv)
	}
	return invites, total, nil
}

// UpdateStatus sets the invite status (e.g. revoked, expired).
func (r *UploadInviteRepository) UpdateStatus(ctx context.Context, id, status string) error {
	query := `UPDATE upload_invites SET status = $1, updated_at = NOW() WHERE id = $2`
	tag, err := r.db.Exec(ctx, query, status, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("invite not found")
	}
	return nil
}

// ReserveUsage atomically reserves bytes and file count for an upload.
// Returns true if reservation succeeded, false if limits exceeded or invite invalid.
// This prevents race conditions when multiple guests upload simultaneously.
func (r *UploadInviteRepository) ReserveUsage(ctx context.Context, id string, fileSize int64) (bool, error) {
	query := `
		UPDATE upload_invites
		SET used_bytes = used_bytes + $1, used_files = used_files + 1, updated_at = NOW()
		WHERE id = $2
		  AND status = 'active'
		  AND expires_at > NOW()
		  AND used_bytes + $1 <= max_total_bytes
		  AND used_files + 1 <= max_files
		RETURNING id`
	var resultID string
	err := r.db.QueryRow(ctx, query, fileSize, id).Scan(&resultID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to reserve usage: %w", err)
	}
	return true, nil
}

// ReleaseUsage decrements reserved bytes and file count (for abandoned/failed uploads).
func (r *UploadInviteRepository) ReleaseUsage(ctx context.Context, id string, fileSize int64) error {
	query := `
		UPDATE upload_invites
		SET used_bytes = GREATEST(used_bytes - $1, 0),
		    used_files = GREATEST(used_files - 1, 0),
		    updated_at = NOW()
		WHERE id = $2`
	_, err := r.db.Exec(ctx, query, fileSize, id)
	return err
}

// IncrementPendingNotify atomically increments the pending notification counter.
func (r *UploadInviteRepository) IncrementPendingNotify(ctx context.Context, id string) error {
	query := `UPDATE upload_invites SET pending_notify_count = pending_notify_count + 1 WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// FlushPendingNotifications atomically reads and resets pending notification counters
// for invites that have accumulated files since the last notification (10-minute batch window).
func (r *UploadInviteRepository) FlushPendingNotifications(ctx context.Context) ([]model.UploadInvite, []int, error) {
	query := `
		WITH pending AS (
			SELECT id, owner_id, label, token, pending_notify_count
			FROM upload_invites
			WHERE pending_notify_count > 0
			  AND (last_notified_at IS NULL OR last_notified_at < NOW() - INTERVAL '10 minutes')
			FOR UPDATE
		),
		updated AS (
			UPDATE upload_invites i
			SET pending_notify_count = 0, last_notified_at = NOW()
			FROM pending p
			WHERE i.id = p.id
		)
		SELECT id, owner_id, label, token, pending_notify_count FROM pending`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	var invites []model.UploadInvite
	var counts []int
	for rows.Next() {
		var inv model.UploadInvite
		var count int
		if err := rows.Scan(&inv.ID, &inv.OwnerID, &inv.Label, &inv.Token, &count); err != nil {
			return nil, nil, err
		}
		invites = append(invites, inv)
		counts = append(counts, count)
	}
	return invites, counts, nil
}

// RecordFile inserts an audit row for a file uploaded via an invite.
func (r *UploadInviteRepository) RecordFile(ctx context.Context, record *model.UploadInviteFile) error {
	query := `
		INSERT INTO upload_invite_files (invite_id, file_id, ip_address, user_agent)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at`
	return r.db.QueryRow(ctx, query,
		record.InviteID, record.FileID, record.IPAddress, record.UserAgent,
	).Scan(&record.ID, &record.CreatedAt)
}

// CountUploadsByIPInWindow counts total file uploads from an IP in the last 24 hours across all invites.
func (r *UploadInviteRepository) CountUploadsByIPInWindow(ctx context.Context, ip string) (int, error) {
	query := `SELECT COUNT(*) FROM upload_invite_files WHERE ip_address = $1 AND created_at > NOW() - INTERVAL '24 hours'`
	var count int
	err := r.db.QueryRow(ctx, query, ip).Scan(&count)
	return count, err
}

// CountActiveByOwner counts non-revoked, non-expired invites for an owner.
func (r *UploadInviteRepository) CountActiveByOwner(ctx context.Context, ownerID string) (int, error) {
	query := `SELECT COUNT(*) FROM upload_invites WHERE owner_id = $1 AND status = 'active' AND expires_at > NOW()`
	var count int
	err := r.db.QueryRow(ctx, query, ownerID).Scan(&count)
	return count, err
}

// ExpireStaleInvites marks expired invites (called by scheduler).
func (r *UploadInviteRepository) ExpireStaleInvites(ctx context.Context) (int64, error) {
	query := `UPDATE upload_invites SET status = 'expired', updated_at = NOW() WHERE status = 'active' AND expires_at <= NOW()`
	tag, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// FindByFileID finds the invite associated with a file (for CompleteUpload re-validation).
func (r *UploadInviteRepository) FindByFileID(ctx context.Context, fileID string) (*model.UploadInvite, error) {
	query := `
		SELECT i.id, i.token, i.owner_id, i.target_folder_id, i.label,
		       i.max_total_bytes, i.max_files, i.used_bytes, i.used_files,
		       i.passcode_hash, i.status, i.pending_notify_count, i.last_notified_at,
		       i.expires_at, i.created_at, i.updated_at
		FROM upload_invites i
		JOIN upload_invite_files uif ON uif.invite_id = i.id
		WHERE uif.file_id = $1`

	inv := &model.UploadInvite{}
	err := r.db.QueryRow(ctx, query, fileID).Scan(
		&inv.ID, &inv.Token, &inv.OwnerID, &inv.TargetFolderID, &inv.Label,
		&inv.MaxTotalBytes, &inv.MaxFiles, &inv.UsedBytes, &inv.UsedFiles,
		&inv.PasscodeHash, &inv.Status, &inv.PendingNotifyCount, &inv.LastNotifiedAt,
		&inv.ExpiresAt, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find invite by file ID: %w", err)
	}
	inv.HasPasscode = inv.PasscodeHash != nil && *inv.PasscodeHash != ""
	return inv, nil
}

// StaleInviteUpload holds the data needed to clean up an abandoned upload.
type StaleInviteUpload struct {
	InviteID   string
	FileID     string
	FileSize   int64
	StorageKey string
}

// ListStaleUploads finds UPLOADING files created via invites that are older than 2 hours.
// These are abandoned sessions where the guest never completed the PUT.
func (r *UploadInviteRepository) ListStaleUploads(ctx context.Context) ([]StaleInviteUpload, error) {
	query := `
		SELECT uif.invite_id, f.id, f.file_size, f.storage_key
		FROM upload_invite_files uif
		JOIN files f ON f.id = uif.file_id
		WHERE f.status = 'UPLOADING' AND uif.created_at < NOW() - INTERVAL '2 hours'`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []StaleInviteUpload
	for rows.Next() {
		var row StaleInviteUpload
		if err := rows.Scan(&row.InviteID, &row.FileID, &row.FileSize, &row.StorageKey); err != nil {
			return nil, err
		}
		results = append(results, row)
	}
	return results, nil
}
