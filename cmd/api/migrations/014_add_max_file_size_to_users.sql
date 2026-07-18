ALTER TABLE users ADD COLUMN IF NOT EXISTS max_file_size_bytes BIGINT DEFAULT 104857600;

---- create above / drop below ----
ALTER TABLE users DROP COLUMN IF EXISTS max_file_size_bytes;
