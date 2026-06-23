-- TASK-0042 BUG-5: allow SOH > 100%.
-- A healthy/new battery whose measured usable energy exceeds the catalog nameplate
-- (regen, downhill, no AC/heater) legitimately estimates above 100%. Drop the
-- upper bound; keep the lower bound (> 0) the SOH floor relies on.
ALTER TABLE battery_health_snapshots
    DROP CONSTRAINT IF EXISTS battery_health_snapshots_soh_pct_check;

ALTER TABLE battery_health_snapshots
    ADD CONSTRAINT battery_health_snapshots_soh_pct_check CHECK (soh_pct > 0);
