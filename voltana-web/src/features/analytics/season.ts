// TASK-0042 FEAT-2 — seasonal expected-consumption config (single source of truth).
//
// Shared by the monthly-chart "expected band" AND estimateConsumption's season slot
// (architecture rule). Keyed by Jalali month so summer (AC) and winter (heater)
// raise the expected consumption — making a high month read as normal.

import { format as formatJalali } from "date-fns-jalali";

// Multiplier on the baseline consumption, by Jalali month (tunable).
export const SEASON_MULTIPLIER: Record<number, number> = {
  1: 1.0, 2: 1.0, 3: 1.0, // فروردین–خرداد  (معتدل)
  4: 1.15, 5: 1.15, 6: 1.15, // تیر–شهریور    (گرم / کولر)
  7: 1.0, 8: 1.0, 9: 1.0, // مهر–آذر         (معتدل)
  10: 1.2, 11: 1.2, 12: 1.2, // دی–اسفند      (سرد / بخاری)
};

// Half-width of the expected band around the seasonal expectation (±15%).
export const SEASON_BAND_TOLERANCE = 0.15;

export type Season = "warm" | "cold" | "mild";

export function monthMultiplier(jalaliMonth: number | undefined): number {
  if (!jalaliMonth) return 1;
  return SEASON_MULTIPLIER[jalaliMonth] ?? 1;
}

export function seasonOf(jalaliMonth: number): Season {
  if (jalaliMonth >= 4 && jalaliMonth <= 6) return "warm";
  if (jalaliMonth >= 10 && jalaliMonth <= 12) return "cold";
  return "mild";
}

// Jalali month (1–12) from a Date or a "YYYY-MM" bucket key.
export function jalaliMonthOf(input: Date | string): number {
  let d: Date;
  if (typeof input === "string") {
    const m = /^(\d{4})-(\d{2})$/.exec(input);
    d = m ? new Date(Number(m[1]), Number(m[2]) - 1, 1) : new Date(input);
  } else {
    d = input;
  }
  return Number(formatJalali(d, "M"));
}
