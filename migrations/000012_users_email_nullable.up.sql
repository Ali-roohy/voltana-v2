-- Allow users registered via phone+bot OTP to have no email address.
-- Replace the NOT NULL unique constraint with a partial unique index so that
-- multiple NULL-email rows are permitted while non-NULL emails remain unique.
ALTER TABLE users ALTER COLUMN email DROP NOT NULL;
ALTER TABLE users DROP CONSTRAINT users_email_key;
CREATE UNIQUE INDEX uq_users_email ON users (email) WHERE email IS NOT NULL;
