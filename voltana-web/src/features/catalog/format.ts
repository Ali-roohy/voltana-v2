import type { CatalogCar } from "./api";

// Persian-digit number formatting + null-safe unit helper shared by the
// catalog card, detail drawer and comparison table.
export function fmt(n: number | null | undefined, unit = ""): string {
  if (n === null || n === undefined) return "—";
  const s = n.toLocaleString("fa-IR", { maximumFractionDigits: 2 });
  return unit ? `${s} ${unit}` : s;
}

export function text(v: string | null | undefined): string {
  return v && v.trim() ? v : "—";
}

// Rough price-tier rank derived from the Persian market-segment text — the
// dataset has no prices, only segment wording (TASK-0033 Scope Out).
export function segmentTier(segment: string | null): number {
  if (!segment) return 2;
  const luxury = segment.includes("لوکس");
  const mid = segment.includes("میان");
  if (luxury && mid) return 3; // میان‌رده رو به لوکس
  if (luxury) return 4;
  if (mid) return 2;
  if (segment.includes("اقتصاد")) return 1;
  return 2;
}

export type SortKey = "range" | "acceleration" | "tier";

export function sortCars(cars: CatalogCar[], key: SortKey): CatalogCar[] {
  const sorted = [...cars];
  switch (key) {
    case "range": // longest range first
      sorted.sort((a, b) => (b.range_km ?? -1) - (a.range_km ?? -1));
      break;
    case "acceleration": // quickest 0→100 first; nulls last
      sorted.sort(
        (a, b) => (a.acceleration_0_100_s ?? Infinity) - (b.acceleration_0_100_s ?? Infinity),
      );
      break;
    case "tier": // most premium first
      sorted.sort((a, b) => segmentTier(b.segment) - segmentTier(a.segment));
      break;
  }
  return sorted;
}

export type BatteryBucket = "all" | "small" | "medium" | "large";

export function inBatteryBucket(car: CatalogCar, bucket: BatteryBucket): boolean {
  if (bucket === "all") return true;
  const kwh = car.battery_capacity_kwh;
  if (kwh === null) return false;
  if (bucket === "small") return kwh < 60;
  if (bucket === "medium") return kwh >= 60 && kwh <= 90;
  return kwh > 90;
}
