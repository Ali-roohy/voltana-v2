// TASK-0042 — the ONE extensible consumption estimator (binding architecture rule).
//
// estimateConsumption(context) returns kWh/100km. Today it returns the car's
// average; FEAT-2 (season) and FEAT-4 (regen/elevation) plug into the SAME context
// object by adding terms here — no feature forks its own consumption math.
//
// SOC prediction helpers (FEAT-1) build on it. They are pure functions so they can
// be unit-tested and reused at entry time.

import type { ChargingSession } from "./api";
import type { Car } from "@/features/cars/api";
import type { CatalogCar } from "@/features/catalog/api";

// Fallback when a car has no efficiency history yet (a reasonable mixed-driving EV
// average, within the BUG-4 sanity band).
export const DEFAULT_CONSUMPTION_KWH_PER_100KM = 16;

export interface ConsumptionContext {
  /** The car's historical average kWh/100km (or the default fallback). */
  carAvgKwhPer100km: number;
  /** Reserved for FEAT-2: Jalali month (1–12) → seasonal multiplier. */
  month?: number;
  /** Reserved for FEAT-4: 0..1 regenerative-braking factor. */
  regenFactor?: number;
  /** Reserved for FEAT-4: previous session shared this location (≈ no elevation change). */
  sameLocationAsPrev?: boolean;
}

/**
 * The single consumption estimate (kWh/100km). Layers are applied to the car's
 * average in one place:
 *   - FEAT-4: regenerative braking lowers effective consumption × (1 − regenFactor).
 *   - FEAT-2 (pending): seasonal multiplier on ctx.month plugs in HERE.
 */
export function estimateConsumption(ctx: ConsumptionContext): number {
  const seasonMultiplier = 1; // FEAT-2 will replace with monthMultiplier(ctx.month)
  return ctx.carAvgKwhPer100km * seasonMultiplier * (1 - clamp01(ctx.regenFactor ?? 0));
}

/** raw kWh/100km adjusted for regen — used for the raw-vs-adjusted session detail (FEAT-4). */
export function applyRegen(rawKwhPer100km: number, regenFactor: number | undefined): number {
  return rawKwhPer100km * (1 - clamp01(regenFactor ?? 0));
}

function clamp01(v: number): number {
  return Math.max(0, Math.min(1, v));
}

/** Resolve a car's usable battery capacity (kWh): override → catalog → null. */
export function resolveUsableCapacity(
  car: Car | undefined,
  catalogById: Map<string, CatalogCar>,
): number | null {
  if (!car) return null;
  const ov = car.spec_overrides?.battery_capacity_kwh;
  const ovNum = typeof ov === "number" ? ov : typeof ov === "string" ? parseFloat(ov) : NaN;
  if (Number.isFinite(ovNum) && ovNum > 0) return ovNum;
  if (car.catalog_car_id) {
    const cat = catalogById.get(car.catalog_car_id);
    if (cat?.usable_kwh && cat.usable_kwh > 0) return cat.usable_kwh;
    if (cat?.battery_capacity_kwh && cat.battery_capacity_kwh > 0) return cat.battery_capacity_kwh;
  }
  return null;
}

/**
 * The car's average kWh/100km from its session history (efficiency values within
 * the sane band), or the default fallback when there's no usable history.
 */
export function carAverageConsumption(carId: string, sessions: ChargingSession[]): number {
  const effs = sessions
    .filter((s) => s.car_id === carId && s.efficiency_kwh_per_100km != null)
    .map((s) => s.efficiency_kwh_per_100km as number)
    .filter((e) => e >= 5 && e <= 40);
  if (effs.length === 0) return DEFAULT_CONSUMPTION_KWH_PER_100KM;
  return effs.reduce((a, b) => a + b, 0) / effs.length;
}

/**
 * Predicted START SOC: the previous session's end SOC minus the SOC consumed over
 * the distance driven since (FEAT-1). Returns null when it can't be estimated.
 * `tripKm` = current odometer − previous session's odometer (≥ 0).
 */
export function predictStartSoc(
  prevEndSoc: number | null | undefined,
  tripKm: number | null,
  ctx: ConsumptionContext,
  usableCapacityKwh: number | null,
): number | null {
  if (prevEndSoc == null) return null;
  if (!tripKm || tripKm <= 0 || !usableCapacityKwh || usableCapacityKwh <= 0) {
    return clampSoc(prevEndSoc); // no distance info → just carry the previous end SOC
  }
  const kwhUsed = (tripKm * estimateConsumption(ctx)) / 100;
  const socUsed = (kwhUsed / usableCapacityKwh) * 100;
  return clampSoc(prevEndSoc - socUsed);
}

/**
 * Predicted END SOC: start SOC plus the SOC added by the session's energy, capped
 * at 100 (FEAT-1). Returns null when inputs are missing.
 */
export function predictEndSoc(
  startSoc: number | null,
  energyKwh: number | null,
  usableCapacityKwh: number | null,
): number | null {
  if (startSoc == null || !energyKwh || energyKwh <= 0 || !usableCapacityKwh || usableCapacityKwh <= 0) {
    return null;
  }
  const socAdded = (energyKwh / usableCapacityKwh) * 100;
  return clampSoc(startSoc + socAdded);
}

function clampSoc(v: number): number {
  return Math.max(0, Math.min(100, Math.round(v)));
}
