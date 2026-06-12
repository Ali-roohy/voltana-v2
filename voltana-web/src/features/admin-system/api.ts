import { api } from "@/lib/api";

export interface SystemSettings {
  otp_delivery_method: "deeplink" | "contact_share";
  // Admin default rates copied into NEW users' settings at creation (FEAT-6).
  default_peak_rate: number;
  default_mid_rate: number;
  default_offpeak_rate: number;
}

export async function getSystemSettings(): Promise<SystemSettings> {
  return api.get<SystemSettings>("/v1/admin/system-settings");
}

export async function updateSystemSettings(patch: Partial<SystemSettings>): Promise<SystemSettings> {
  return api.put<SystemSettings>("/v1/admin/system-settings", patch);
}
