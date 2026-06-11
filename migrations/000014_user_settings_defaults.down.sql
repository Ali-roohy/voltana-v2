ALTER TABLE user_settings
    ALTER COLUMN peak_rate     SET DEFAULT 0,
    ALTER COLUMN mid_rate      SET DEFAULT 0,
    ALTER COLUMN offpeak_rate  SET DEFAULT 0;
