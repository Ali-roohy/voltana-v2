-- Revert to the bounded SOH check. NOTE: if any snapshot with soh_pct > 100 exists
-- this will fail; that is intentional (the down migration assumes pre-BUG-5 data).
ALTER TABLE battery_health_snapshots
    DROP CONSTRAINT IF EXISTS battery_health_snapshots_soh_pct_check;

ALTER TABLE battery_health_snapshots
    ADD CONSTRAINT battery_health_snapshots_soh_pct_check CHECK (soh_pct > 0 AND soh_pct <= 100);
