ALTER TABLE user_settings
  ADD COLUMN currency TEXT NOT NULL DEFAULT 'toman'
    CHECK (currency IN ('toman', 'rial', 'usd'));
