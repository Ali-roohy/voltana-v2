ALTER TABLE user_settings
    ALTER COLUMN peak_rate    SET DEFAULT 2000,
    ALTER COLUMN mid_rate     SET DEFAULT 1000,
    ALTER COLUMN offpeak_rate SET DEFAULT 500;

DELETE FROM system_settings
WHERE key IN ('default_peak_rate', 'default_mid_rate', 'default_offpeak_rate');

ALTER TABLE charging_sessions
    DROP COLUMN rate_peak_at_time,
    DROP COLUMN rate_mid_at_time,
    DROP COLUMN rate_offpeak_at_time;
