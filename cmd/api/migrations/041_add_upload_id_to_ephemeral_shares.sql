-- Migration 041: Add upload_id to ephemeral_shares to track and abort incomplete multipart uploads
ALTER TABLE ephemeral_shares ADD COLUMN IF NOT EXISTS upload_id TEXT;
