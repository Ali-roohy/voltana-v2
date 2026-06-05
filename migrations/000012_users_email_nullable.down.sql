-- Remove any NULL-email users before restoring NOT NULL (safe on rollback; none should exist).
DELETE FROM users WHERE email IS NULL;
DROP INDEX IF EXISTS uq_users_email;
ALTER TABLE users ADD CONSTRAINT users_email_key UNIQUE (email);
ALTER TABLE users ALTER COLUMN email SET NOT NULL;
