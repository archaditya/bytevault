-- 029_add_webhook_idempotency.sql
-- Stores processed Razorpay webhook event IDs to prevent duplicate processing

CREATE TABLE IF NOT EXISTS processed_webhook_events (
    event_id                  VARCHAR(200) PRIMARY KEY,
    event_type                VARCHAR(80) NOT NULL,
    razorpay_subscription_id  VARCHAR(100),
    processed_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_processed_webhooks_processed_at ON processed_webhook_events(processed_at);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_processed_webhooks_processed_at;
DROP TABLE IF EXISTS processed_webhook_events;
