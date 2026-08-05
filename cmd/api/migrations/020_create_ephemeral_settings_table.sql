-- Migration 020: Clean Ephemeral Settings Governance Table
CREATE TABLE IF NOT EXISTS ephemeral_settings (
    id INT PRIMARY KEY DEFAULT 1,
    max_file_size_gb NUMERIC NOT NULL DEFAULT 2,
    max_downloads_cap INT NOT NULL DEFAULT 1,
    rate_limit_24h INT NOT NULL DEFAULT 2,
    expiry_minutes INT NOT NULL DEFAULT 60,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT single_row_check CHECK (id = 1)
);

-- Seed initial default configuration row
INSERT INTO ephemeral_settings (id, max_file_size_gb, max_downloads_cap, rate_limit_24h, expiry_minutes)
VALUES (1, 2, 1, 2, 60)
ON CONFLICT (id) DO NOTHING;
