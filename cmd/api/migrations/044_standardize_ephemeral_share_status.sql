-- 044_standardize_ephemeral_share_status.sql
-- Normalizes any historical 'READY' statuses to 'ACTIVE' and adds check constraint for valid EphemeralState values.

UPDATE ephemeral_shares
SET status = 'ACTIVE'
WHERE status = 'READY';

ALTER TABLE ephemeral_shares
DROP CONSTRAINT IF EXISTS chk_ephemeral_shares_status;

ALTER TABLE ephemeral_shares
ADD CONSTRAINT chk_ephemeral_shares_status
CHECK (status IN ('INITIATED', 'UPLOADING', 'ACTIVE', 'BURNED', 'EXPIRED', 'FAILED', 'PURGED'));

---- create above / drop below ----

ALTER TABLE ephemeral_shares
DROP CONSTRAINT IF EXISTS chk_ephemeral_shares_status;
