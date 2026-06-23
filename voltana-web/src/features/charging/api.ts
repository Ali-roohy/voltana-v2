import { api } from "@/lib/api";

export interface ChargingSession {
  id: string;
  car_id: string;
  started_at: string;
  ended_at: string | null;
  location: string | null;
  kwh_charged: number | null;
  energy_peak_kwh: number | null;
  energy_mid_kwh: number | null;
  energy_offpeak_kwh: number | null;
  start_soc: number | null;
  end_soc: number | null;
  cost: number | null;
  notes: string | null;
  odometer_km: number | null;
  // FEAT-3: optional charger power (kW). TASK-0042 migration: trip_distance_km is
  // server-maintained (odometer delta), null when not derivable.
  charge_power_kw: number | null;
  trip_distance_km: number | null;
  // Rate snapshot (FEAT-6): the owner's rates when the session was created.
  // Cost math must use these — never the current settings. Null only on
  // legacy rows whose owner had no settings at backfill time.
  rate_peak_at_time: number | null;
  rate_mid_at_time: number | null;
  rate_offpeak_at_time: number | null;
  // Derived server-side (kWh/100km) when this and the previous session both have
  // an odometer reading with a positive distance; otherwise null.
  efficiency_kwh_per_100km: number | null;
  created_at: string;
  updated_at: string;
}

// Full-replace payload for POST/PUT. Cost is omitted on normal saves (the Go API
// computes it from the per-period energy × the user's rates); it is only sent
// when the user manually overrides it.
export interface ChargingInput {
  car_id: string;
  started_at: string;
  ended_at?: string | null;
  location?: string | null;
  kwh_charged?: number | null;
  energy_peak_kwh?: number | null;
  energy_mid_kwh?: number | null;
  energy_offpeak_kwh?: number | null;
  start_soc?: number | null;
  end_soc?: number | null;
  cost?: number | null;
  odometer_km?: number | null;
  charge_power_kw?: number | null;
}

interface ListResponse<T> {
  items: T[];
  limit: number;
  offset: number;
  total: number;
}

// Optional server-side filter for the history list. Dates are serialized to RFC3339;
// `to` is treated as inclusive of that whole day (see end-of-day handling below).
export interface ChargingListFilter {
  car_id?: string;
  from?: Date;
  to?: Date;
}

export async function listChargingSessions(filter?: ChargingListFilter): Promise<ChargingSession[]> {
  const params = new URLSearchParams({ limit: "100" });
  if (filter?.car_id) params.set("car_id", filter.car_id);
  if (filter?.from) {
    const from = new Date(filter.from);
    from.setHours(0, 0, 0, 0); // start of the selected day
    params.set("from", from.toISOString());
  }
  if (filter?.to) {
    const to = new Date(filter.to);
    to.setHours(23, 59, 59, 999); // inclusive end of the selected day
    params.set("to", to.toISOString());
  }
  const res = await api.get<ListResponse<ChargingSession>>(`/v1/charging-sessions?${params.toString()}`);
  return res.items;
}

export const createChargingSession = (input: ChargingInput) =>
  api.post<ChargingSession>("/v1/charging-sessions", input);
export const updateChargingSession = (id: string, input: ChargingInput) =>
  api.put<ChargingSession>(`/v1/charging-sessions/${id}`, input);
export const deleteChargingSession = (id: string) =>
  api.del<void>(`/v1/charging-sessions/${id}`);
