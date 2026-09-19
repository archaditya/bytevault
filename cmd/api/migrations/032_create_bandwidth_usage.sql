-- 032_create_bandwidth_usage.sql
-- Metering table for egress data transfer (downloads, public shares, instant shares)

CREATE TABLE IF NOT EXISTS bandwidth_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID REFERENCES users(id) ON DELETE SET NULL,
    file_id           UUID,
    bytes_transferred BIGINT NOT NULL,
    transfer_type     VARCHAR(30) NOT NULL, -- 'file_download', 'public_share', 'instant_share'
    ip_address        VARCHAR(45),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bandwidth_created_at ON bandwidth_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bandwidth_user_id ON bandwidth_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_bandwidth_transfer_type ON bandwidth_logs (transfer_type, created_at DESC);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_bandwidth_transfer_type;
DROP INDEX IF EXISTS idx_bandwidth_user_id;
DROP INDEX IF EXISTS idx_bandwidth_created_at;
DROP TABLE IF EXISTS bandwidth_logs;
