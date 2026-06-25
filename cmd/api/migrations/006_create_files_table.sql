CREATE TABLE IF NOT EXISTS files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    storage_provider VARCHAR(50) NOT NULL, -- 'local', 'cloudinary', or 'r2'
    bucket VARCHAR(255),                  -- Cloud storage bucket name (e.g. for R2)
    storage_key TEXT NOT NULL,            -- e.g. user/123/docs/resume.pdf
    file_size BIGINT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    is_public BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_files_user_id ON files(user_id);
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files(deleted_at);
CREATE INDEX IF NOT EXISTS idx_files_provider_key ON files(storage_provider, storage_key);

---- create above / drop below ----

DROP TABLE IF EXISTS files;