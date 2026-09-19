-- 025_create_packages.sql
-- Defines subscription packages/tiers

CREATE TABLE IF NOT EXISTS packages (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  VARCHAR(50) NOT NULL UNIQUE,         -- 'free', 'pro', 'premium'
    display_name          VARCHAR(100) NOT NULL,               -- 'Free', 'Pro', 'Premium'
    description           TEXT,
    price_paise           INTEGER NOT NULL DEFAULT 0,          -- Base price in paise (14900 = ₹149)
    gst_rate              NUMERIC(5,2) NOT NULL DEFAULT 18.00, -- GST percentage
    price_with_gst_paise  INTEGER NOT NULL DEFAULT 0,          -- Tax-inclusive price for Razorpay
    currency              VARCHAR(3) NOT NULL DEFAULT 'INR',
    billing_period        VARCHAR(20) NOT NULL DEFAULT 'monthly',
    storage_limit_bytes   BIGINT NOT NULL DEFAULT 5368709120,   -- 5GB default
    max_file_size_bytes   BIGINT NOT NULL DEFAULT 2147483648,   -- 2GB default
    razorpay_plan_id      VARCHAR(100),                        -- Razorpay plan_id (null for free)
    is_active             BOOLEAN NOT NULL DEFAULT true,
    sort_order            INTEGER NOT NULL DEFAULT 0,
    features              JSONB DEFAULT '{}',                  -- {"max_shares": 5, ...}
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_packages_active ON packages (is_active, sort_order);
CREATE INDEX IF NOT EXISTS idx_packages_name ON packages (name);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_packages_name;
DROP INDEX IF EXISTS idx_packages_active;
DROP TABLE IF EXISTS packages;
