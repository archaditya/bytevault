-- 031_update_default_storage_limits.sql
-- Elevate default user storage limits from legacy 1GB/100MB to 5GB/2GB per ByteVault policy.

ALTER TABLE users ALTER COLUMN storage_limit_bytes SET DEFAULT 5368709120;
ALTER TABLE users ALTER COLUMN max_file_size_bytes SET DEFAULT 2147483648;

-- Update all existing users who still have the legacy 1GB (1073741824) to 5GB (5368709120)
UPDATE users 
SET storage_limit_bytes = 5368709120 
WHERE storage_limit_bytes = 1073741824 OR storage_limit_bytes IS NULL;

-- Update all existing users who still have the legacy 100MB (104857600) to 2GB (2147483648)
UPDATE users 
SET max_file_size_bytes = 2147483648 
WHERE max_file_size_bytes = 104857600 OR max_file_size_bytes IS NULL;

---- create above / drop below ----

ALTER TABLE users ALTER COLUMN storage_limit_bytes SET DEFAULT 1073741824;
ALTER TABLE users ALTER COLUMN max_file_size_bytes SET DEFAULT 104857600;
