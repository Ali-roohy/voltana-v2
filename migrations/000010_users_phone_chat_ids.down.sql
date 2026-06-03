DROP INDEX IF EXISTS uq_users_tg_chat;
DROP INDEX IF EXISTS uq_users_bale_chat;
DROP INDEX IF EXISTS uq_users_phone;

ALTER TABLE users DROP COLUMN IF EXISTS telegram_chat_id;
ALTER TABLE users DROP COLUMN IF EXISTS bale_chat_id;
ALTER TABLE users DROP COLUMN IF EXISTS phone;
