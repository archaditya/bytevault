-- 027_create_transactions.sql
-- Tracks every financial transaction, invoice number, and Razorpay payment details

CREATE SEQUENCE IF NOT EXISTS invoice_number_seq START WITH 1001;

CREATE TABLE IF NOT EXISTS transactions (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    subscription_id       UUID REFERENCES subscriptions(id) ON DELETE CASCADE,
    package_id            UUID REFERENCES packages(id) ON DELETE CASCADE,
    
    -- Razorpay payment identifiers
    razorpay_payment_id   VARCHAR(100),
    razorpay_order_id     VARCHAR(100),
    razorpay_signature    VARCHAR(256),
    razorpay_invoice_id   VARCHAR(100),
    
    -- Amount details
    amount_paise          INTEGER NOT NULL,
    tax_paise             INTEGER NOT NULL DEFAULT 0,
    total_paise           INTEGER NOT NULL,
    currency              VARCHAR(3) NOT NULL DEFAULT 'INR',
    
    -- Transaction metadata: type ('subscription_charge', 'upgrade', 'refund')
    type                  VARCHAR(30) NOT NULL,
    -- Status: pending, captured, failed, refunded
    status                VARCHAR(30) NOT NULL DEFAULT 'pending',
    
    description           TEXT,
    failure_reason        TEXT,
    
    -- Invoice
    invoice_number        VARCHAR(50) UNIQUE,
    invoice_generated     BOOLEAN NOT NULL DEFAULT false,
    
    metadata              JSONB DEFAULT '{}',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_transactions_user_id ON transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_transactions_subscription_id ON transactions(subscription_id);
CREATE INDEX IF NOT EXISTS idx_transactions_invoice_number ON transactions(invoice_number);
CREATE INDEX IF NOT EXISTS idx_transactions_razorpay_payment_id ON transactions(razorpay_payment_id);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_transactions_razorpay_payment_id;
DROP INDEX IF EXISTS idx_transactions_invoice_number;
DROP INDEX IF EXISTS idx_transactions_subscription_id;
DROP INDEX IF EXISTS idx_transactions_user_id;
DROP TABLE IF EXISTS transactions;
DROP SEQUENCE IF EXISTS invoice_number_seq;
