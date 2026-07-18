ALTER TABLE users ADD COLUMN IF NOT EXISTS storage_limit_bytes BIGINT DEFAULT 1073741824;

---- create above / drop below ----
ALTER TABLE users DROP COLUMN IF EXISTS storage_limit_bytes;