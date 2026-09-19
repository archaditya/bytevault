-- 028_create_subscription_audit_logs.sql
-- Audit log for all subscription lifecycle events, webhooks, and billing actions

CREATE TABLE IF NOT EXISTS subscription_audit_logs (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID REFERENCES users(id) ON DELETE CASCADE,
    subscription_id   UUID REFERENCES subscriptions(id) ON DELETE CASCADE,
    transaction_id    UUID REFERENCES transactions(id) ON DELETE CASCADE,
    
    event_type        VARCHAR(80) NOT NULL,
    event_source      VARCHAR(30) NOT NULL,   -- 'api', 'webhook', 'scheduler', 'admin'
    
    payload           JSONB DEFAULT '{}',
    
    status            VARCHAR(20) NOT NULL DEFAULT 'success',  -- 'success', 'error', 'skipped'
    error_message     TEXT,
    
    ip_address        VARCHAR(45),
    user_agent        TEXT,
    
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sub_audit_logs_user_id ON subscription_audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_sub_audit_logs_event_type ON subscription_audit_logs(event_type);
CREATE INDEX IF NOT EXISTS idx_sub_audit_logs_created_at ON subscription_audit_logs(created_at);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_sub_audit_logs_created_at;
DROP INDEX IF EXISTS idx_sub_audit_logs_event_type;
DROP INDEX IF EXISTS idx_sub_audit_logs_user_id;
DROP TABLE IF EXISTS subscription_audit_logs;
