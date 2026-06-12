import type { ChargingSession } from "@/features/charging/api";
import type { Settings } from "@/features/settings/api";

export type Currency = 'toman' | 'rial' | 'usd';

// Static fallback rate — 1 USD ≈ 500,000 Toman (update when a live feed is wired).
const USD_RATE = 500_000;

/**
 * Format a Toman amount into the user's chosen currency string, including unit.
 * Conversion is display-only; stored amounts are always in Toman.
 */
export function formatCost(amount: number, currency: Currency = 'toman'): string {
  const n = (v: number) => new Intl.NumberFormat('fa-IR').format(Math.round(v));
  switch (currency) {
    case 'rial':
      return `${n(amount * 10)} ریال`;
    case 'usd':
      return `$${(amount / USD_RATE).toFixed(2)}`;
    default:
      return `${n(amount)} تومان`;
  }
}

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

// The rates a session must be priced with (FEAT-6): its creation-time snapshot.
// Changing rates later must never re-price old sessions, so the snapshot wins;
// `fallback` (current settings) applies only to legacy rows without one.
export function ratesForSession(
  session: Pick<ChargingSession, "rate_peak_at_time" | "rate_mid_at_time" | "rate_offpeak_at_time">,
  fallback: Rates,
): Rates {
  if (
    session.rate_peak_at_time != null &&
    session.rate_mid_at_time != null &&
    session.rate_offpeak_at_time != null
  ) {
    return {
      peak: session.rate_peak_at_time,
      mid: session.rate_mid_at_time,
      offpeak: session.rate_offpeak_at_time,
    };
  }
  return fallback;
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
