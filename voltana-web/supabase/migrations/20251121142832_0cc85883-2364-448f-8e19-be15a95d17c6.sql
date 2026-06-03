-- Create ev_models table for predefined electric vehicle models
CREATE TABLE public.ev_models (
  id uuid NOT NULL DEFAULT gen_random_uuid() PRIMARY KEY,
  name_fa text NOT NULL,
  name_en text NOT NULL,
  vehicle_type text NOT NULL,
  battery_chemistry text NOT NULL,
  battery_capacity numeric NOT NULL,
  range_km integer NOT NULL,
  battery_voltage text,
  max_dc_charge_kw numeric,
  usable_capacity_kwh numeric NOT NULL,
  battery_warranty text,
  max_soc_percent integer DEFAULT 90,
  min_soc_percent integer DEFAULT 20,
  estimated_cycles integer,
  max_ac_charge_kw numeric,
  max_dc_accept_kw numeric,
  created_at timestamp with time zone DEFAULT now(),
  updated_at timestamp with time zone DEFAULT now()
);

-- Enable RLS
ALTER TABLE public.ev_models ENABLE ROW LEVEL SECURITY;

-- Public read access for ev_models (all users can view the list)
CREATE POLICY "Anyone can view ev_models"
ON public.ev_models
FOR SELECT
TO authenticated
USING (true);

-- Only admins can insert/update/delete ev_models
CREATE POLICY "Admins can insert ev_models"
ON public.ev_models
FOR INSERT
TO authenticated
WITH CHECK (public.has_role(auth.uid(), 'admin'));

CREATE POLICY "Admins can update ev_models"
ON public.ev_models
FOR UPDATE
TO authenticated
USING (public.has_role(auth.uid(), 'admin'));

CREATE POLICY "Admins can delete ev_models"
ON public.ev_models
FOR DELETE
TO authenticated
USING (public.has_role(auth.uid(), 'admin'));

-- Add trigger for updated_at
CREATE TRIGGER update_ev_models_updated_at
BEFORE UPDATE ON public.ev_models
FOR EACH ROW
EXECUTE FUNCTION public.update_updated_at_column();

-- Update cars table to add reference to ev_model (optional)
ALTER TABLE public.cars
ADD COLUMN ev_model_id uuid REFERENCES public.ev_models(id) ON DELETE SET NULL;