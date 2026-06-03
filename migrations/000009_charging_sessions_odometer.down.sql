ALTER TABLE charging_sessions DROP CONSTRAINT IF EXISTS charging_sessions_odometer_km_check;
ALTER TABLE charging_sessions DROP COLUMN IF EXISTS odometer_km;
