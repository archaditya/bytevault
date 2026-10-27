-- Migration 040: Create audit_logs table and add transfer metrics to files

CREATE TABLE IF NOT EXISTS audit_logs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    actor_type    TEXT NOT NULL DEFAULT 'user',
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   UUID,
    status        TEXT NOT NULL DEFAULT 'success',
    request_id    TEXT,
    ip_address    INET,
    user_agent    TEXT,
    metadata      JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT audit_logs_status_check
        CHECK (status IN ('success', 'failure'))
);

CREATE INDEX IF NOT EXISTS idx_audit_logs_tenant_time
    ON audit_logs (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_actor_time
    ON audit_logs (actor_user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_resource
    ON audit_logs (resource_type, resource_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_action_time
    ON audit_logs (action, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at
    ON audit_logs (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_audit_logs_request_id
    ON audit_logs (request_id);

-- Add upload metrics to files table
ALTER TABLE files
ADD COLUMN IF NOT EXISTS chunk_count INT DEFAULT 1,
ADD COLUMN IF NOT EXISTS upload_duration_ms BIGINT DEFAULT 0,
ADD COLUMN IF NOT EXISTS transfer_speed_bps BIGINT DEFAULT 0;

---- create above / drop below ----

DROP TABLE IF EXISTS audit_logs;

ALTER TABLE files
DROP COLUMN IF EXISTS chunk_count,
DROP COLUMN IF EXISTS upload_duration_ms,
DROP COLUMN IF EXISTS transfer_speed_bps;
