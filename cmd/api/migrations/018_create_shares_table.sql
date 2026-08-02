-- Migration 018: Granular User-to-User File & Folder Sharing Table
CREATE TABLE IF NOT EXISTS shares (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type TEXT NOT NULL, -- 'file' or 'folder'
    resource_id UUID NOT NULL,
    owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    grantee_email TEXT NOT NULL,
    grantee_id UUID REFERENCES users(id) ON DELETE CASCADE,
    permission TEXT NOT NULL DEFAULT 'VIEWER', -- 'VIEWER', 'EDITOR'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(resource_type, resource_id, grantee_email)
);

CREATE INDEX IF NOT EXISTS idx_shares_grantee_email ON shares(LOWER(grantee_email));
CREATE INDEX IF NOT EXISTS idx_shares_grantee_id ON shares(grantee_id);
CREATE INDEX IF NOT EXISTS idx_shares_resource ON shares(resource_type, resource_id);
