-- Migration 039: Create user consents and add ticket number to contact queries

CREATE TABLE IF NOT EXISTS user_consents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    consent_type VARCHAR(64) NOT NULL,
    version VARCHAR(32) NOT NULL DEFAULT '2026.1',
    status VARCHAR(16) NOT NULL DEFAULT 'granted',
    ip_address VARCHAR(64),
    user_agent TEXT,
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_consents_user_id ON user_consents(user_id);
CREATE INDEX IF NOT EXISTS idx_user_consents_type ON user_consents(consent_type);

-- Add ticket_number to contact_queries
ALTER TABLE contact_queries ADD COLUMN IF NOT EXISTS ticket_number VARCHAR(32);
CREATE UNIQUE INDEX IF NOT EXISTS idx_contact_queries_ticket_number ON contact_queries(ticket_number) WHERE ticket_number IS NOT NULL;
