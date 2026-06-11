-- Migration 000014: Fix user_settings default rates (Toman, not raw Rial).
-- Old defaults: 0 (from 000001); new defaults: 2000 / 1000 / 500 (Toman/kWh).
-- Also updates existing rows that still have the 0 placeholder defaults.

ALTER TABLE user_settings
    ALTER COLUMN peak_rate     SET DEFAULT 2000,
    ALTER COLUMN mid_rate      SET DEFAULT 1000,
    ALTER COLUMN offpeak_rate  SET DEFAULT 500;

UPDATE user_settings
SET    peak_rate    = 2000,
       mid_rate     = 1000,
       offpeak_rate = 500
WHERE  peak_rate = 0
  AND  mid_rate  = 0
  AND  offpeak_rate = 0;
