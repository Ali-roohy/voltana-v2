ALTER TABLE cars
    DROP COLUMN IF EXISTS catalog_car_id,
    DROP COLUMN IF EXISTS spec_overrides;
