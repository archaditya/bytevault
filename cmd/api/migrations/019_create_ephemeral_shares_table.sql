-- Migration 019: Anonymous "Burn After Reading" Ephemeral Shares Table
CREATE TABLE IF NOT EXISTS ephemeral_shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token TEXT UNIQUE NOT NULL,
    storage_key TEXT NOT NULL,
    filename TEXT NOT NULL,
    file_size BIGINT NOT NULL,
    content_type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'READY', -- 'UPLOADING', 'READY', 'BURNED', 'EXPIRED'
    max_downloads INT NOT NULL DEFAULT 1,
    download_count INT NOT NULL DEFAULT 0,
    ip_address TEXT,
    user_agent TEXT,
    password_hash TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    burned_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_ephemeral_shares_token ON ephemeral_shares(token);
CREATE INDEX IF NOT EXISTS idx_ephemeral_shares_expires ON ephemeral_shares(expires_at) WHERE status = 'READY';
CREATE INDEX IF NOT EXISTS idx_ephemeral_shares_status ON ephemeral_shares(status);
