-- TASK-0037 FEAT-6 — per-session rate snapshots + admin default rates.

-- 1. Snapshot columns: the rates in force when the session was created.
--    Cost math and analytics must use these, never the user's CURRENT rates.
ALTER TABLE charging_sessions
    ADD COLUMN rate_peak_at_time    NUMERIC(10,4),
    ADD COLUMN rate_mid_at_time     NUMERIC(10,4),
    ADD COLUMN rate_offpeak_at_time NUMERIC(10,4);

-- 2. Backfill existing sessions from the owner's current rates (best available
--    approximation of the historical rate). Owners without a settings row stay
--    NULL — consumers fall back to current rates for those legacy rows.
UPDATE charging_sessions cs
SET rate_peak_at_time    = us.peak_rate,
    rate_mid_at_time     = us.mid_rate,
    rate_offpeak_at_time = us.offpeak_rate
FROM user_settings us
WHERE us.user_id = cs.user_id;

-- 3. Admin-managed default rates move to system_settings; the hardcoded
--    user_settings column defaults are removed (the creation path now copies
--    the admin defaults explicitly).
INSERT INTO system_settings (key, value) VALUES
    ('default_peak_rate',    '2000'),
    ('default_mid_rate',     '1000'),
    ('default_offpeak_rate', '500')
ON CONFLICT (key) DO NOTHING;

ALTER TABLE user_settings
    ALTER COLUMN peak_rate    DROP DEFAULT,
    ALTER COLUMN mid_rate     DROP DEFAULT,
    ALTER COLUMN offpeak_rate DROP DEFAULT;
