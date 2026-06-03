-- Remove seeded catalog data and the idempotency constraint.
-- (cars.ev_model_id is ON DELETE SET NULL, so existing cars are unaffected.)
DELETE FROM ev_models;
ALTER TABLE ev_models DROP CONSTRAINT IF EXISTS ev_models_name_en_key;
