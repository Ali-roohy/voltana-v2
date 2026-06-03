-- users table is in 000001_init_schema.up.sql
-- This migration adds the email_verification_tokens table used by POST /auth/register.
-- Tokens are hashed (SHA-256) so the raw token is never stored.

CREATE TABLE email_verification_tokens (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) UNIQUE NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX evt_user_id_idx   ON email_verification_tokens(user_id);
CREATE INDEX evt_expires_at_idx ON email_verification_tokens(expires_at);
