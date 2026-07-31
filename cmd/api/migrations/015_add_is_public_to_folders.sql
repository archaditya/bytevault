-- Migration 015: Add is_public column to folders table
ALTER TABLE folders ADD COLUMN IF NOT EXISTS is_public BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX IF NOT EXISTS idx_folders_is_public ON folders(is_public);
