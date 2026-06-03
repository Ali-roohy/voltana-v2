-- Add SOC (State of Charge) fields to charging_sessions table
ALTER TABLE public.charging_sessions
ADD COLUMN start_soc_percent INTEGER,
ADD COLUMN end_soc_percent INTEGER;

-- Add check constraints for SOC values (0-100%)
ALTER TABLE public.charging_sessions
ADD CONSTRAINT start_soc_percent_range CHECK (start_soc_percent >= 0 AND start_soc_percent <= 100),
ADD CONSTRAINT end_soc_percent_range CHECK (end_soc_percent >= 0 AND end_soc_percent <= 100);

-- Add comment for documentation
COMMENT ON COLUMN public.charging_sessions.start_soc_percent IS 'Battery State of Charge at start of charging session (0-100%)';
COMMENT ON COLUMN public.charging_sessions.end_soc_percent IS 'Battery State of Charge at end of charging session (0-100%)';