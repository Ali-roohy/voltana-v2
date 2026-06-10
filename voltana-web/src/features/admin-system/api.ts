import { api } from "@/lib/api";

export interface SystemSettings {
  otp_delivery_method: "deeplink" | "contact_share";
}

export async function getSystemSettings(): Promise<SystemSettings> {
  return api.get<SystemSettings>("/v1/admin/system-settings");
}

export async function updateSystemSettings(patch: Partial<SystemSettings>): Promise<SystemSettings> {
  return api.put<SystemSettings>("/v1/admin/system-settings", patch);
}
