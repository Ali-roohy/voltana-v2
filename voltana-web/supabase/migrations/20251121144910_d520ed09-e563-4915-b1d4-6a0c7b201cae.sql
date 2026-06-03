-- Make license_plate nullable in cars table
ALTER TABLE public.cars ALTER COLUMN license_plate DROP NOT NULL;