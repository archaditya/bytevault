UPDATE users SET updated_at = created_at WHERE updated_at IS NULL;
ALTER TABLE users ALTER COLUMN updated_at SET DEFAULT NOW();
ALTER TABLE users ALTER COLUMN updated_at SET NOT NULL;

---- create above / drop below ----

ALTER TABLE users ALTER COLUMN updated_at DROP NOT NULL;
ALTER TABLE users ALTER COLUMN updated_at DROP DEFAULT;
