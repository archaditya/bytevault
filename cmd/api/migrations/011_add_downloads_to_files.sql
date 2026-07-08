ALTER TABLE files ADD COLUMN IF NOT EXISTS downloads INT NOT NULL DEFAULT 0;
---- create above / drop below ----
ALTER TABLE files DROP COLUMN IF EXISTS downloads;
