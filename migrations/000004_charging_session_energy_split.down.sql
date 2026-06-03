-- Reverse the per-period energy split (kwh_charged total is untouched).
ALTER TABLE charging_sessions
    DROP COLUMN IF EXISTS energy_peak_kwh,
    DROP COLUMN IF EXISTS energy_mid_kwh,
    DROP COLUMN IF EXISTS energy_offpeak_kwh;
