-- Migration 036: System Settings Governance Table
CREATE TABLE IF NOT EXISTS system_settings (
    id INT PRIMARY KEY DEFAULT 1,
    max_upload_mb INT NOT NULL DEFAULT 100,
    rate_limit_per_hour INT NOT NULL DEFAULT 1000,
    allow_public_shares BOOLEAN NOT NULL DEFAULT true,
    maintenance_mode BOOLEAN NOT NULL DEFAULT false,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT single_system_settings_row CHECK (id = 1)
);

-- Seed initial default configuration row
INSERT INTO system_settings (id, max_upload_mb, rate_limit_per_hour, allow_public_shares, maintenance_mode)
VALUES (1, 100, 1000, true, false)
ON CONFLICT (id) DO NOTHING;
