-- TASK-0042 Section B migration: smart-session fields.
ALTER TABLE charging_sessions ADD COLUMN IF NOT EXISTS trip_distance_km DECIMAL(10,2);
ALTER TABLE charging_sessions ADD COLUMN IF NOT EXISTS charge_power_kw  DECIMAL(10,2);
ALTER TABLE user_settings     ADD COLUMN IF NOT EXISTS city             TEXT;
ALTER TABLE user_settings     ADD COLUMN IF NOT EXISTS regen_factor     NUMERIC NOT NULL DEFAULT 0.10;

-- Backfill trip_distance_km from consecutive odometers (per car, by time);
-- only positive deltas (the odometer is cumulative — BUG-4).
WITH d AS (
    SELECT id,
           odometer_km - LAG(odometer_km) OVER (PARTITION BY car_id ORDER BY started_at) AS km
    FROM charging_sessions
)
UPDATE charging_sessions cs
SET trip_distance_km = d.km
FROM d
WHERE cs.id = d.id AND d.km > 0;
