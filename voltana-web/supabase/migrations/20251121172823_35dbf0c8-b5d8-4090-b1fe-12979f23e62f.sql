-- Add default_car_id to user_settings table
ALTER TABLE public.user_settings 
ADD COLUMN default_car_id UUID REFERENCES public.cars(id) ON DELETE SET NULL;

-- Add index for better performance
CREATE INDEX idx_user_settings_default_car ON public.user_settings(default_car_id);