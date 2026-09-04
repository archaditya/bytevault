-- 024_add_nsfw_moderation.sql
-- Adds NSFW detection scoring, user restriction/strike system, and moderation appeals.

-- NSFW score on files (0.0 = safe, 1.0 = explicit)
ALTER TABLE files ADD COLUMN IF NOT EXISTS nsfw_score REAL DEFAULT 0;

-- User restriction & strike system
ALTER TABLE users ADD COLUMN IF NOT EXISTS nsfw_strikes INTEGER DEFAULT 0;
ALTER TABLE users ADD COLUMN IF NOT EXISTS restricted_until TIMESTAMPTZ;
ALTER TABLE users ADD COLUMN IF NOT EXISTS restriction_reason TEXT;

-- Moderation appeals table
CREATE TABLE IF NOT EXISTS moderation_appeals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending, approved, rejected
    admin_notes TEXT,
    reviewed_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_at TIMESTAMPTZ
);

-- Indexes for efficient admin queries
CREATE INDEX IF NOT EXISTS idx_files_nsfw_flagged ON files (nsfw_score) WHERE nsfw_score > 0.5;
CREATE INDEX IF NOT EXISTS idx_users_restricted ON users (restricted_until) WHERE restricted_until IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_appeals_pending ON moderation_appeals (status) WHERE status = 'pending';
CREATE INDEX IF NOT EXISTS idx_appeals_user ON moderation_appeals (user_id);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_appeals_user;
DROP INDEX IF EXISTS idx_appeals_pending;
DROP INDEX IF EXISTS idx_users_restricted;
DROP INDEX IF EXISTS idx_files_nsfw_flagged;
DROP TABLE IF EXISTS moderation_appeals;
ALTER TABLE users DROP COLUMN IF EXISTS restriction_reason;
ALTER TABLE users DROP COLUMN IF EXISTS restricted_until;
ALTER TABLE users DROP COLUMN IF EXISTS nsfw_strikes;
ALTER TABLE files DROP COLUMN IF EXISTS nsfw_score;
