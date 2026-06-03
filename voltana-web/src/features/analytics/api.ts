import { api } from "@/lib/api";

// Lifetime fleet aggregates from GET /v1/analytics/dashboard (Redis-cached server-side).
export interface DashboardStats {
  total_kwh: number;
  total_cost: number;
  total_km: number;
  avg_kwh_per_100km: number | null; // null when total_km == 0
  session_count: number;
}

// One battery-health snapshot (delta-SOC estimate).
export interface BatterySnapshot {
  soh_pct: number;
  estimated_capacity_kwh: number;
  nominal_capacity_kwh: number;
  confidence: "low" | "medium" | "high";
  computed_at: string;
}

// The latest-snapshot endpoint returns either a snapshot or an insufficient-data marker.
export interface InsufficientData {
  status: "insufficient_data";
  qualifying_sessions: number;
}
export type BatteryHealth = BatterySnapshot | InsufficientData;

export function isInsufficient(b: BatteryHealth | undefined): b is InsufficientData {
  return !!b && (b as InsufficientData).status === "insufficient_data";
}

export const getDashboard = () => api.get<DashboardStats>("/v1/analytics/dashboard");

export const getBattery = (carId: string) => api.get<BatteryHealth>(`/v1/analytics/battery/${carId}`);

export const getBatteryHistory = (carId: string, limit = 30) =>
  api.get<{ items: BatterySnapshot[] }>(`/v1/analytics/battery/${carId}/history?limit=${limit}`);
