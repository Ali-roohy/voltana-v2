-- Add new columns for energy consumption by time period
ALTER TABLE charging_sessions 
ADD COLUMN energy_peak_kwh numeric DEFAULT 0,
ADD COLUMN energy_mid_kwh numeric DEFAULT 0,
ADD COLUMN energy_offpeak_kwh numeric DEFAULT 0;

-- Migrate existing data (assuming all existing sessions are mid-peak)
UPDATE charging_sessions 
SET energy_mid_kwh = energy_kwh 
WHERE energy_mid_kwh = 0;

-- Create settings table for electricity rates
CREATE TABLE IF NOT EXISTS user_settings (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL REFERENCES auth.users(id),
  rate_peak numeric DEFAULT 5000,
  rate_mid numeric DEFAULT 3000,
  rate_offpeak numeric DEFAULT 1500,
  created_at timestamp with time zone DEFAULT now(),
  updated_at timestamp with time zone DEFAULT now(),
  UNIQUE(user_id)
);

-- Enable RLS
ALTER TABLE user_settings ENABLE ROW LEVEL SECURITY;

-- RLS policies for user_settings
CREATE POLICY "Users can view their own settings"
  ON user_settings FOR SELECT
  USING (auth.uid() = user_id);

CREATE POLICY "Users can insert their own settings"
  ON user_settings FOR INSERT
  WITH CHECK (auth.uid() = user_id);

CREATE POLICY "Users can update their own settings"
  ON user_settings FOR UPDATE
  USING (auth.uid() = user_id);

-- Trigger for updated_at
CREATE TRIGGER update_user_settings_updated_at
  BEFORE UPDATE ON user_settings
  FOR EACH ROW
  EXECUTE FUNCTION update_updated_at_column();