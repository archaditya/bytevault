-- Drop the redundant push_tokens table.
-- FCM tokens are now stored exclusively in the user_devices table (migration 005).
-- Existing tokens will be re-registered automatically on next user login.
DROP TABLE IF EXISTS push_tokens;

---- create above / drop below ----

-- push_tokens was originally created in migration 010.
-- If rolling back, that migration will recreate it.
