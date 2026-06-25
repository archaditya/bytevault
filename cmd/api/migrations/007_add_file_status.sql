ALTER TABLE files ADD COLUMN status VARCHAR(50) NOT NULL DEFAULT 'READY';
---- create above / drop below ----
ALTER TABLE files DROP COLUMN status;
