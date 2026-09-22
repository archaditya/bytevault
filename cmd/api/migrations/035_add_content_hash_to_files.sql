-- Migration 035: Add content hash and folder filename indexes for duplicate detection and deduplication
ALTER TABLE files ADD COLUMN IF NOT EXISTS content_hash VARCHAR(64);

CREATE INDEX IF NOT EXISTS idx_files_user_content_hash ON files(user_id, content_hash) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_files_user_folder_filename ON files(user_id, COALESCE(folder_id, '00000000-0000-0000-0000-000000000000'), filename) WHERE deleted_at IS NULL;
