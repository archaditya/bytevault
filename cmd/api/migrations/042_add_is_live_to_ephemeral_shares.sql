-- Migration 042: Add is_live flag to ephemeral_shares to identify live pipelined shares
ALTER TABLE ephemeral_shares ADD COLUMN IF NOT EXISTS is_live BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_ephemeral_shares_is_live ON ephemeral_shares(is_live);
