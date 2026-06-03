-- Per-period energy split for time-of-use cost calculation (architecture doc §3).
-- charging_sessions already exists (000001) with a single kwh_charged (the total);
-- this adds the peak/mid/offpeak breakdown so the API can compute
--   cost = energy_peak_kwh*peak_rate + energy_mid_kwh*mid_rate + energy_offpeak_kwh*offpeak_rate
-- using the three rates in user_settings. kwh_charged is retained as the grand total.
ALTER TABLE charging_sessions
    ADD COLUMN energy_peak_kwh    DECIMAL(8,3) CHECK (energy_peak_kwh    >= 0),
    ADD COLUMN energy_mid_kwh     DECIMAL(8,3) CHECK (energy_mid_kwh     >= 0),
    ADD COLUMN energy_offpeak_kwh DECIMAL(8,3) CHECK (energy_offpeak_kwh >= 0);
