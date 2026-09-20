-- Upload invite links for guest file collection (Client File Portal)
CREATE TABLE IF NOT EXISTS upload_invites (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token                VARCHAR(64) NOT NULL UNIQUE,
    owner_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_folder_id     UUID REFERENCES folders(id) ON DELETE SET NULL,
    label                VARCHAR(255) NOT NULL DEFAULT '',
    max_total_bytes      BIGINT NOT NULL DEFAULT 2147483648,    -- 2 GB
    max_files            INT NOT NULL DEFAULT 20,
    used_bytes           BIGINT NOT NULL DEFAULT 0,
    used_files           INT NOT NULL DEFAULT 0,
    passcode_hash        VARCHAR(255),
    status               VARCHAR(20) NOT NULL DEFAULT 'active', -- active, revoked, expired
    pending_notify_count INT NOT NULL DEFAULT 0,
    last_notified_at     TIMESTAMPTZ,
    expires_at           TIMESTAMPTZ NOT NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_upload_invites_token ON upload_invites(token);
CREATE INDEX idx_upload_invites_owner ON upload_invites(owner_id);
CREATE INDEX idx_upload_invites_pending_notify ON upload_invites(pending_notify_count) WHERE pending_notify_count > 0;

-- Tracks each file uploaded via an invite for audit trail and IP rate limiting
CREATE TABLE IF NOT EXISTS upload_invite_files (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invite_id   UUID NOT NULL REFERENCES upload_invites(id) ON DELETE CASCADE,
    file_id     UUID NOT NULL REFERENCES files(id) ON DELETE CASCADE,
    ip_address  VARCHAR(45),
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_upload_invite_files_invite ON upload_invite_files(invite_id);
CREATE INDEX idx_upload_invite_files_ip_created ON upload_invite_files(ip_address, created_at);

---- create above / drop below ----

DROP TABLE IF EXISTS upload_invite_files;
DROP TABLE IF EXISTS upload_invites;
