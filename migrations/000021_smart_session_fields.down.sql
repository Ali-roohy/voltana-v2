ALTER TABLE charging_sessions DROP COLUMN IF EXISTS trip_distance_km;
ALTER TABLE charging_sessions DROP COLUMN IF EXISTS charge_power_kw;
ALTER TABLE user_settings     DROP COLUMN IF EXISTS city;
ALTER TABLE user_settings     DROP COLUMN IF EXISTS regen_factor;
