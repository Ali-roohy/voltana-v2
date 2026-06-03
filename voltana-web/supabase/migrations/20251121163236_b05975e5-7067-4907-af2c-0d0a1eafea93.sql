-- Add odometer_km field to charging_sessions table
ALTER TABLE public.charging_sessions
ADD COLUMN odometer_km integer;

COMMENT ON COLUMN public.charging_sessions.odometer_km IS 'Odometer reading in kilometers at the time of charging';
