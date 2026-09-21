-- Migration 034: Increase Ephemeral Rate Limit to prevent false blocking on shared office/NAT IPs
UPDATE ephemeral_settings 
SET rate_limit_24h = 50, updated_at = NOW() 
WHERE id = 1 AND rate_limit_24h < 50;

ALTER TABLE ephemeral_settings 
ALTER COLUMN rate_limit_24h SET DEFAULT 50;
