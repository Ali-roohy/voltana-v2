-- TASK-0018: per-session odometer reading for efficiency (kWh/100km).
-- Nullable + no default so existing rows are unaffected (NULL = not recorded).
ALTER TABLE charging_sessions ADD COLUMN odometer_km INT;
ALTER TABLE charging_sessions
    ADD CONSTRAINT charging_sessions_odometer_km_check
    CHECK (odometer_km IS NULL OR odometer_km >= 0);
