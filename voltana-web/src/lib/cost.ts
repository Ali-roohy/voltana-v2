import type { ChargingSession } from "@/features/charging/api";
import type { Settings } from "@/features/settings/api";

// Per-period electricity rates (Toman per kWh). Single source of truth for the
// time-of-use cost math used by the charging page and the dashboard.
export interface Rates {
  peak: number;
  mid: number;
  offpeak: number;
}

export interface TouCost {
  peak: number;
  mid: number;
  offpeak: number;
  total: number;
}

// Pull the per-period rates out of the user's settings, defaulting to 0 while
// settings are still loading (mirrors the old inline `rates` object).
export function ratesFromSettings(settings?: Settings | null): Rates {
  return {
    peak: settings?.peak_rate ?? 0,
    mid: settings?.mid_rate ?? 0,
    offpeak: settings?.offpeak_rate ?? 0,
  };
}

// Time-of-use cost breakdown for a session: each segment = energy × its rate, and
// `total` is always the sum of the three segments (so segments sum to total).
//
// A manual cost override (`session.cost`) is intentionally NOT applied here — it is
// a single number with no period split, so callers keep the override at the call
// site: `session.cost ?? calcCost(session, rates).total`.
export function calcCost(
  session: Pick<ChargingSession, "energy_peak_kwh" | "energy_mid_kwh" | "energy_offpeak_kwh">,
  rates: Rates,
): TouCost {
  const peak = (session.energy_peak_kwh ?? 0) * rates.peak;
  const mid = (session.energy_mid_kwh ?? 0) * rates.mid;
  const offpeak = (session.energy_offpeak_kwh ?? 0) * rates.offpeak;
  return { peak, mid, offpeak, total: peak + mid + offpeak };
}
