-- Migration 038: Add views tracking to folders table
ALTER TABLE folders ADD COLUMN IF NOT EXISTS views BIGINT NOT NULL DEFAULT 0;
