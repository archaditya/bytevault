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
	UserID       string
	FilterFolder bool
	FolderID     *string
	Search       string
	SortBy       string // name, size, date
	SortDir      string // asc, desc
	Limit        int
	Cursor       string // RFC3339 timestamp
	IsPublic     *bool
}

type FileRepository struct {
	db *pgxpool.Pool
}

func NewFileRepository(db *pgxpool.Pool) *FileRepository {
	return &FileRepository{db: db}
}

// buildPrefixTsQuery converts words into prefix tsquery format: "mon" -> "mon:*" or "home th" -> "home:* & th:*"
func buildPrefixTsQuery(search string) string {
	words := strings.Fields(strings.TrimSpace(search))
	var parts []string
	for _, w := range words {
		cleaned := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				return r
			}
			return -1
		}, w)
		if cleaned != "" {
			parts = append(parts, cleaned+":*")
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " & ")
}

func (r *FileRepository) Create(ctx context.Context, file *model.File) error {
	query := `
		INSERT INTO files (user_id, filename, storage_provider, bucket, storage_key, file_size, content_type, is_public, status, folder_id, tags, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
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
		file.Tags,
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create file record: %w", err)
	}
	return nil
}

func (r *FileRepository) FindByID(ctx context.Context, id string) (*model.File, error) {
	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, thumbnail_key, file_size, content_type, is_public, status, folder_id, created_at, updated_at, downloads, tags, COALESCE(nsfw_score, 0)
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
		&file.ThumbnailKey,
		&file.FileSize,
		&file.ContentType,
		&file.IsPublic,
		&file.Status,
		&file.FolderID,
		&file.CreatedAt,
		&file.UpdatedAt,
		&file.Downloads,
		&file.Tags,
		&file.NSFWScore,
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
	conditions = append(conditions, "(status = 'READY' OR status = 'PENDING_SCAN')")

	// IsPublic, Search, or Folder filter
	if params.IsPublic != nil && *params.IsPublic {
		conditions = append(conditions, "is_public = true")
	} else if params.Search != "" {
		trimmed := strings.TrimSpace(params.Search)
		prefixQuery := buildPrefixTsQuery(trimmed)
		likePattern := "%" + trimmed + "%"

		if prefixQuery != "" {
			conditions = append(conditions, fmt.Sprintf(
				"(search_vector @@ to_tsquery('english', $%d) OR search_vector @@ plainto_tsquery('english', $%d) OR filename ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d OR content_text ILIKE $%d)",
				argIndex, argIndex+1, argIndex+2, argIndex+2, argIndex+2,
			))
			args = append(args, prefixQuery, trimmed, likePattern)
			argIndex += 3
		} else {
			conditions = append(conditions, fmt.Sprintf(
				"(filename ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d OR content_text ILIKE $%d)",
				argIndex, argIndex, argIndex,
			))
			args = append(args, likePattern)
			argIndex++
		}
	} else if params.FilterFolder {
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
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, thumbnail_key, file_size, content_type, is_public, status, folder_id, created_at, updated_at, downloads, tags, COALESCE(nsfw_score, 0)
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
			&f.ThumbnailKey,
			&f.FileSize,
			&f.ContentType,
			&f.IsPublic,
			&f.Status,
			&f.FolderID,
			&f.CreatedAt,
			&f.UpdatedAt,
			&f.Downloads,
			&f.Tags,
			&f.NSFWScore,
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

func (r *FileRepository) RenameFile(ctx context.Context, id string, filename string) error {
	query := `UPDATE files SET filename = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, filename, id)
	return err
}

func (r *FileRepository) GetUserStorageUsed(ctx context.Context, userID string) (int64, error) {
	query := `SELECT COALESCE(SUM(file_size), 0) FROM files WHERE user_id = $1 AND deleted_at IS NULL AND status IN ('READY', 'UPLOADING')`
	var total int64
	err := r.db.QueryRow(ctx, query, userID).Scan(&total)
	return total, err
}

func (r *FileRepository) SoftDelete(ctx context.Context, id string) error {
	query := `UPDATE files SET deleted_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}


// PurgeSoftDeletedBefore hard-deletes file records soft-deleted before the cutoff.
// Returns storage keys so the caller can remove objects from the storage provider.
func (r *FileRepository) PurgeSoftDeletedBefore(ctx context.Context, cutoff time.Time) ([]string, error) {
	query := `DELETE FROM files WHERE deleted_at IS NOT NULL AND deleted_at < $1 RETURNING storage_key`
	rows, err := r.db.Query(ctx, query, cutoff)
	if err != nil {
		return nil, fmt.Errorf("failed to purge soft-deleted files: %w", err)
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

// ListAllFiles returns all files across all users (admin use) with search and cursor pagination
func (r *FileRepository) ListAllFiles(ctx context.Context, search string, limit int, cursor string) ([]*model.File, string, error) {
	var conditions []string
	var args []any
	argIndex := 1

	conditions = append(conditions, "deleted_at IS NULL")
	conditions = append(conditions, "status = 'READY'")

	if search != "" {
		trimmed := strings.TrimSpace(search)
		prefixQuery := buildPrefixTsQuery(trimmed)
		likePattern := "%" + trimmed + "%"

		if prefixQuery != "" {
			conditions = append(conditions, fmt.Sprintf(
				"(search_vector @@ to_tsquery('english', $%d) OR search_vector @@ plainto_tsquery('english', $%d) OR filename ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d OR content_text ILIKE $%d)",
				argIndex, argIndex+1, argIndex+2, argIndex+2, argIndex+2,
			))
			args = append(args, prefixQuery, trimmed, likePattern)
			argIndex += 3
		} else {
			conditions = append(conditions, fmt.Sprintf(
				"(filename ILIKE $%d OR array_to_string(tags, ' ') ILIKE $%d OR content_text ILIKE $%d)",
				argIndex, argIndex, argIndex,
			))
			args = append(args, likePattern)
			argIndex++
		}
	}

	if cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, cursor)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("created_at < $%d", argIndex))
			args = append(args, cursorTime)
			argIndex++
		}
	}

	// Use relevance ranking when searching, otherwise default chronological order
	orderClause := "created_at DESC"
	if search != "" {
		orderClause = fmt.Sprintf(
			"ts_rank(search_vector, plainto_tsquery('english', $1)) DESC, created_at DESC",
		)
	}

	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, thumbnail_key, file_size, content_type, is_public, status, folder_id, created_at, updated_at, downloads, tags, COALESCE(nsfw_score, 0)
		FROM files
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY ` + orderClause + `
	`

	limitVal := 20
	if limit > 0 {
		limitVal = limit
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
		if err := rows.Scan(&f.ID, &f.UserID, &f.Filename, &f.StorageProvider, &f.Bucket, &f.StorageKey, &f.ThumbnailKey, &f.FileSize, &f.ContentType, &f.IsPublic, &f.Status, &f.FolderID, &f.CreatedAt, &f.UpdatedAt, &f.Downloads, &f.Tags, &f.NSFWScore); err != nil {
			return nil, "", err
		}
		files = append(files, &f)
	}

	nextCursor := ""
	if len(files) == limitVal && len(files) > 0 {
		nextCursor = files[len(files)-1].CreatedAt.Format(time.RFC3339)
	}

	return files, nextCursor, nil
}

// ListAllSharedFiles returns all publicly shared files across all users (admin use) with search and cursor pagination
func (r *FileRepository) ListAllSharedFiles(ctx context.Context, search string, limit int, cursor string) ([]*model.File, string, error) {
	var conditions []string
	var args []any
	argIndex := 1

	conditions = append(conditions, "f.deleted_at IS NULL")
	conditions = append(conditions, "f.status = 'READY'")
	conditions = append(conditions, "f.is_public = true")

	if search != "" {
		trimmed := strings.TrimSpace(search)
		prefixQuery := buildPrefixTsQuery(trimmed)
		likePattern := "%" + trimmed + "%"

		if prefixQuery != "" {
			conditions = append(conditions, fmt.Sprintf(
				"(f.search_vector @@ to_tsquery('english', $%d) OR f.search_vector @@ plainto_tsquery('english', $%d) OR f.filename ILIKE $%d OR array_to_string(f.tags, ' ') ILIKE $%d OR f.content_text ILIKE $%d)",
				argIndex, argIndex+1, argIndex+2, argIndex+2, argIndex+2,
			))
			args = append(args, prefixQuery, trimmed, likePattern)
			argIndex += 3
		} else {
			conditions = append(conditions, fmt.Sprintf(
				"(f.filename ILIKE $%d OR array_to_string(f.tags, ' ') ILIKE $%d OR f.content_text ILIKE $%d)",
				argIndex, argIndex, argIndex,
			))
			args = append(args, likePattern)
			argIndex++
		}
	}

	if cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, cursor)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("f.created_at < $%d", argIndex))
			args = append(args, cursorTime)
			argIndex++
		}
	}

	// Use relevance ranking when searching, otherwise default chronological order
	orderClause := "f.created_at DESC"
	if search != "" {
		orderClause = fmt.Sprintf(
			"ts_rank(f.search_vector, plainto_tsquery('english', $1)) DESC, f.created_at DESC",
		)
	}

	query := `
		SELECT f.id, f.user_id, f.filename, f.storage_provider, f.bucket, f.storage_key, f.thumbnail_key, f.file_size, f.content_type, f.is_public, f.status, f.folder_id, f.created_at, f.updated_at, f.downloads, f.tags, COALESCE(f.nsfw_score, 0),
		       COALESCE(u.first_name || ' ' || u.last_name, '') as owner_name, COALESCE(u.email, '') as owner_email
		FROM files f
		LEFT JOIN users u ON f.user_id = u.id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY ` + orderClause + `
	`

	limitVal := 20
	if limit > 0 {
		limitVal = limit
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
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.Filename, &f.StorageProvider, &f.Bucket, &f.StorageKey, &f.ThumbnailKey,
			&f.FileSize, &f.ContentType, &f.IsPublic, &f.Status, &f.FolderID,
			&f.CreatedAt, &f.UpdatedAt, &f.Downloads, &f.Tags, &f.NSFWScore, &f.OwnerName, &f.OwnerEmail,
		); err != nil {
			return nil, "", err
		}
		files = append(files, &f)
	}

	nextCursor := ""
	if len(files) == limitVal && len(files) > 0 {
		nextCursor = files[len(files)-1].CreatedAt.Format(time.RFC3339)
	}

	return files, nextCursor, nil
}

// GetAllStorageKeys returns a map of all active file storage keys AND thumbnail keys.
func (r *FileRepository) GetAllStorageKeys(ctx context.Context) (map[string]bool, error) {
	query := `SELECT storage_key, thumbnail_key FROM files`
	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make(map[string]bool)
	for rows.Next() {
		var key string
		var thumbKey *string
		if err := rows.Scan(&key, &thumbKey); err == nil {
			keys[key] = true
			if thumbKey != nil && *thumbKey != "" {
				keys[*thumbKey] = true
			}
		}
	}
	return keys, nil
}

func (r *FileRepository) IncrementDownloads(ctx context.Context, id string) error {
	query := `UPDATE files SET downloads = downloads + 1, updated_at = NOW() WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

func (r *FileRepository) UpdateThumbnailKey(ctx context.Context, id string, thumbnailKey string) error {
	query := `UPDATE files SET thumbnail_key = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, thumbnailKey, id)
	return err
}

// UpdateContentText stores extracted document text for full-text search indexing.
// The DB trigger automatically regenerates the search_vector column from filename + content_text + tags.
func (r *FileRepository) UpdateContentText(ctx context.Context, id string, text string) error {
	query := `UPDATE files SET content_text = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, text, id)
	return err
}

// AppendToContentText appends additional text (e.g., AI-generated labels) to existing content_text.
// This preserves any previously extracted document text while adding new searchable content.
func (r *FileRepository) AppendToContentText(ctx context.Context, id string, additionalText string) error {
	query := `UPDATE files SET content_text = COALESCE(content_text, '') || E'\n' || $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, additionalText, id)
	return err
}

// UpdateTags replaces the tags array for a file.
// The DB trigger automatically regenerates the search_vector to include the new tags.
func (r *FileRepository) UpdateTags(ctx context.Context, id string, tags []string) error {
	query := `UPDATE files SET tags = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, tags, id)
	return err
}

func (r *FileRepository) ListPublicFilesByFolderID(ctx context.Context, folderID string) ([]*model.File, error) {
	query := `
		SELECT id, user_id, filename, storage_provider, bucket, storage_key, thumbnail_key, file_size, content_type, is_public, status, folder_id, created_at, updated_at, downloads, tags, COALESCE(nsfw_score, 0)
		FROM files
		WHERE folder_id = $1 AND status = 'READY' AND deleted_at IS NULL
		ORDER BY filename ASC
	`
	rows, err := r.db.Query(ctx, query, folderID)
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
			&f.ThumbnailKey,
			&f.FileSize,
			&f.ContentType,
			&f.IsPublic,
			&f.Status,
			&f.FolderID,
			&f.CreatedAt,
			&f.UpdatedAt,
			&f.Downloads,
			&f.Tags,
			&f.NSFWScore,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &f)
	}
	return files, nil
}

// UpdateNSFWScore stores the NSFW detection score for a file.
func (r *FileRepository) UpdateNSFWScore(ctx context.Context, id string, score float64) error {
	query := `UPDATE files SET nsfw_score = $1, updated_at = NOW() WHERE id = $2`
	_, err := r.db.Exec(ctx, query, score, id)
	return err
}

// ListFlaggedFiles returns files with FLAGGED_REVIEW status for admin moderation.
func (r *FileRepository) ListFlaggedFiles(ctx context.Context, limit int, cursor string) ([]*model.File, string, error) {
	var conditions []string
	var args []any
	argIndex := 1

	conditions = append(conditions, "f.deleted_at IS NULL")
	conditions = append(conditions, "f.status = 'FLAGGED_REVIEW'")

	if cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, cursor)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("f.created_at < $%d", argIndex))
			args = append(args, cursorTime)
			argIndex++
		}
	}

	query := `
		SELECT f.id, f.user_id, f.filename, f.storage_provider, f.bucket, f.storage_key, f.thumbnail_key, f.file_size, f.content_type, f.is_public, f.status, f.folder_id, f.created_at, f.updated_at, f.downloads, f.tags, COALESCE(f.nsfw_score, 0),
		       COALESCE(u.first_name || ' ' || u.last_name, '') as owner_name, COALESCE(u.email, '') as owner_email
		FROM files f
		LEFT JOIN users u ON f.user_id = u.id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY f.nsfw_score DESC, f.created_at DESC
	`

	limitVal := 20
	if limit > 0 {
		limitVal = limit
	}
	query += fmt.Sprintf(" LIMIT %d", limitVal)
	_ = argIndex // suppress unused warning

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.Filename, &f.StorageProvider, &f.Bucket, &f.StorageKey, &f.ThumbnailKey,
			&f.FileSize, &f.ContentType, &f.IsPublic, &f.Status, &f.FolderID,
			&f.CreatedAt, &f.UpdatedAt, &f.Downloads, &f.Tags, &f.NSFWScore, &f.OwnerName, &f.OwnerEmail,
		); err != nil {
			return nil, "", err
		}
		files = append(files, &f)
	}

	nextCursor := ""
	if len(files) == limitVal && len(files) > 0 {
		nextCursor = files[len(files)-1].CreatedAt.Format(time.RFC3339)
	}

	return files, nextCursor, nil
}

// ListBlockedNSFWFiles returns files that were auto-blocked by NSFW detection (audit trail).
func (r *FileRepository) ListBlockedNSFWFiles(ctx context.Context, limit int, cursor string) ([]*model.File, string, error) {
	var conditions []string
	var args []any
	argIndex := 1

	conditions = append(conditions, "f.status = 'BLOCKED_NSFW'")

	if cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339, cursor)
		if err == nil {
			conditions = append(conditions, fmt.Sprintf("f.created_at < $%d", argIndex))
			args = append(args, cursorTime)
			argIndex++
		}
	}

	query := `
		SELECT f.id, f.user_id, f.filename, f.storage_provider, f.bucket, f.storage_key, f.thumbnail_key, f.file_size, f.content_type, f.is_public, f.status, f.folder_id, f.created_at, f.updated_at, f.downloads, f.tags, COALESCE(f.nsfw_score, 0),
		       COALESCE(u.first_name || ' ' || u.last_name, '') as owner_name, COALESCE(u.email, '') as owner_email
		FROM files f
		LEFT JOIN users u ON f.user_id = u.id
		WHERE ` + strings.Join(conditions, " AND ") + `
		ORDER BY f.created_at DESC
	`

	limitVal := 20
	if limit > 0 {
		limitVal = limit
	}
	query += fmt.Sprintf(" LIMIT %d", limitVal)
	_ = argIndex

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()

	var files []*model.File
	for rows.Next() {
		var f model.File
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.Filename, &f.StorageProvider, &f.Bucket, &f.StorageKey, &f.ThumbnailKey,
			&f.FileSize, &f.ContentType, &f.IsPublic, &f.Status, &f.FolderID,
			&f.CreatedAt, &f.UpdatedAt, &f.Downloads, &f.Tags, &f.NSFWScore, &f.OwnerName, &f.OwnerEmail,
		); err != nil {
			return nil, "", err
		}
		files = append(files, &f)
	}

	nextCursor := ""
	if len(files) == limitVal && len(files) > 0 {
		nextCursor = files[len(files)-1].CreatedAt.Format(time.RFC3339)
	}

	return files, nextCursor, nil
}

// GetModerationStats returns counts for admin moderation dashboard.
func (r *FileRepository) GetModerationStats(ctx context.Context) (totalBlocked int, totalFlagged int, err error) {
	query := `
		SELECT
			COALESCE(SUM(CASE WHEN status = 'BLOCKED_NSFW' THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN status = 'FLAGGED_REVIEW' THEN 1 ELSE 0 END), 0)
		FROM files
	`
	err = r.db.QueryRow(ctx, query).Scan(&totalBlocked, &totalFlagged)
	return
}
