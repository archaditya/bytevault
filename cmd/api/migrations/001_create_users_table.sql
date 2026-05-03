CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    email VARCHAR(255) NOT NULL,
    password TEXT,

    first_name VARCHAR(255),
    last_name VARCHAR(255),
    avatar_url TEXT,

    is_verified BOOLEAN NOT NULL DEFAULT false,
    status VARCHAR(255),

    created_by UUID,
    updated_by UUID,
    deleted_by UUID,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_email on users(email);


---- create above / drop below ----
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS "pgcrypto";