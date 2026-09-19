-- 026_create_subscriptions.sql
-- Subscriptions table tracks user subscription lifecycles with Razorpay

CREATE TABLE IF NOT EXISTS subscriptions (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    package_id                UUID NOT NULL REFERENCES packages(id) ON DELETE CASCADE,
    
    -- Razorpay identifiers
    razorpay_subscription_id  VARCHAR(100) UNIQUE,
    razorpay_customer_id      VARCHAR(100),
    razorpay_payment_id       VARCHAR(100),
    
    -- Lifecycle: created, authenticated, active, pending, halted, cancelled, completed, paused, expired
    status                    VARCHAR(30) NOT NULL DEFAULT 'active',
    
    current_period_start      TIMESTAMPTZ,
    current_period_end        TIMESTAMPTZ,
    cancelled_at              TIMESTAMPTZ,
    cancel_at_cycle_end       BOOLEAN NOT NULL DEFAULT false,
    paused_at                 TIMESTAMPTZ,
    
    -- Upgrade/downgrade tracking
    pending_package_id        UUID REFERENCES packages(id) ON DELETE SET NULL,
    upgraded_from_id          UUID REFERENCES subscriptions(id) ON DELETE SET NULL,
    
    -- Grace period for administrative deletion
    grace_period_end          TIMESTAMPTZ,
    
    metadata                  JSONB DEFAULT '{}',
    created_at                TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
CREATE INDEX IF NOT EXISTS idx_subscriptions_razorpay_id ON subscriptions(razorpay_subscription_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_active_user ON subscriptions(user_id) 
    WHERE status IN ('active', 'authenticated', 'pending', 'created');

---- create above / drop below ----

DROP INDEX IF EXISTS idx_subscriptions_active_user;
DROP INDEX IF EXISTS idx_subscriptions_razorpay_id;
DROP INDEX IF EXISTS idx_subscriptions_status;
DROP INDEX IF EXISTS idx_subscriptions_user_id;
DROP TABLE IF EXISTS subscriptions;
