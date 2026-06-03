import { api } from "@/lib/api";

export interface Settings {
  default_car_id: string | null;
  peak_rate: number;
  mid_rate: number;
  offpeak_rate: number;
  created_at: string;
  updated_at: string;
}

// PUT /v1/settings is a full replace — always send every field.
export interface SettingsUpdate {
  default_car_id: string | null;
  peak_rate: number;
  mid_rate: number;
  offpeak_rate: number;
}

export const getSettings = () => api.get<Settings>("/v1/settings");
export const updateSettings = (body: SettingsUpdate) => api.put<Settings>("/v1/settings", body);
