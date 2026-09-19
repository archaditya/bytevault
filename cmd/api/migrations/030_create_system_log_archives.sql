-- 030_create_system_log_archives.sql
-- Tracks daily app log archives across local server, Cloudflare R2, and 6-month retention purging.

CREATE TABLE IF NOT EXISTS system_log_archives (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    log_date               DATE NOT NULL UNIQUE,
    file_name              VARCHAR(100) NOT NULL,
    storage_location       VARCHAR(20) NOT NULL DEFAULT 'local', -- 'local', 'r2', 'purged'
    storage_key            TEXT,                                 -- e.g. logs/app/2026/09/18-09-2026.app.log.gz
    local_path             TEXT,                                 -- e.g. logs/18-09-2026.app.log
    file_size_bytes        BIGINT NOT NULL DEFAULT 0,
    compressed_size_bytes  BIGINT NOT NULL DEFAULT 0,
    archived_to_r2_at      TIMESTAMPTZ,
    purged_at              TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_system_logs_date ON system_log_archives(log_date);
CREATE INDEX IF NOT EXISTS idx_system_logs_location ON system_log_archives(storage_location);
