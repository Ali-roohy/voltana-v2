-- E.164-normalized phone for OTP-request lookup; partial-unique: many NULLs
-- allowed, non-null phones must be globally unique.
ALTER TABLE users ADD COLUMN phone            TEXT;
ALTER TABLE users ADD COLUMN bale_chat_id     TEXT;
ALTER TABLE users ADD COLUMN telegram_chat_id TEXT;

CREATE UNIQUE INDEX uq_users_phone     ON users (phone)            WHERE phone IS NOT NULL;
CREATE UNIQUE INDEX uq_users_bale_chat ON users (bale_chat_id)     WHERE bale_chat_id IS NOT NULL;
CREATE UNIQUE INDEX uq_users_tg_chat   ON users (telegram_chat_id) WHERE telegram_chat_id IS NOT NULL;
