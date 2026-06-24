// TASK-0042 FEAT-5 — shared suspicious-data severity + entry-time rule mirror.
//
// The backend computes warnings[] for the LIST (all non-blocking 🟡). At ENTRY
// time we mirror the SAME rule codes client-side for instant feedback, plus the
// 🔴 blocking errors (odometer monotonic, SOC range) that can't be persisted.
// Messages are kept verbatim-equal to the backend so list + form read identically.

export type Severity = "error" | "warning";
export type WarningField = "odometer" | "start_soc" | "end_soc" | "energy" | "duration";

export interface EntryFlag {
  code: string;
  message: string;
  severity: Severity;
  field: WarningField;
}

// Tailwind class fragments per severity (amber = warning, destructive = error).
// Matches the existing emerald-badge convention (direct utilities for semantic states).
export const severityStyle: Record<Severity, { text: string; fieldBorder: string }> = {
  error: { text: "text-destructive", fieldBorder: "border-destructive focus-visible:ring-destructive" },
  warning: { text: "text-amber-600 dark:text-amber-400", fieldBorder: "" },
};

// Sanity band (mirror of the backend BUG-4 / FEAT-5 thresholds).
const EFF_MIN = 5;
const EFF_MAX = 40;
const DURATION_MISMATCH = 2;

export interface EntryInput {
  odometer: number | null;
  prevOdometer: number | null;
  startSoc: number | null;
  endSoc: number | null;
  energyKwh: number | null; // total
  chargePowerKw: number | null;
  durationMin: number | null;
}

/**
 * Compute the entry-time flags for the current form values. Errors block save;
 * warnings are advisory. Field tells the form where to render each message.
 */
export function computeEntryFlags(i: EntryInput): EntryFlag[] {
  const flags: EntryFlag[] = [];

  // 🔴 odometer must increase (cumulative). Mirrors the 422 the API returns.
  if (i.odometer != null && i.prevOdometer != null && i.odometer <= i.prevOdometer) {
    flags.push({
      code: "odometer_not_increasing",
      severity: "error",
      field: "odometer",
      message: "کیلومترشمار باید از جلسه قبلی بیشتر باشد (کیلومترشمار تجمعی است)",
    });
  }
  // 🔴 SOC range.
  for (const [val, field] of [[i.startSoc, "start_soc"], [i.endSoc, "end_soc"]] as const) {
    if (val != null && (val < 0 || val > 100)) {
      flags.push({ code: "soc_out_of_range", severity: "error", field, message: "درصد شارژ باید بین ۰ تا ۱۰۰ باشد" });
    }
  }
  // 🟡 efficiency out of band (needs a positive trip).
  if (i.odometer != null && i.prevOdometer != null && i.energyKwh != null && i.energyKwh > 0) {
    const trip = i.odometer - i.prevOdometer;
    if (trip > 0) {
      const eff = i.energyKwh / (trip / 100);
      if (eff > EFF_MAX || eff < EFF_MIN) {
        flags.push({
          code: "efficiency_out_of_band",
          severity: "warning",
          field: "odometer",
          message: "مصرف غیرعادی (خارج از محدوده ۵ تا ۴۰ کیلووات‌ساعت در ۱۰۰ کیلومتر)",
        });
      }
    }
  }
  // 🟡 SOC decreasing on a charge.
  if (i.startSoc != null && i.endSoc != null && i.startSoc > i.endSoc) {
    flags.push({ code: "soc_decreasing", severity: "warning", field: "end_soc", message: "درصد شارژ پایان کمتر از شروع است" });
  }
  // 🟡 zero energy but SOC changed.
  if (i.startSoc != null && i.endSoc != null && i.startSoc !== i.endSoc && (i.energyKwh == null || i.energyKwh === 0)) {
    flags.push({ code: "zero_energy_soc_changed", severity: "warning", field: "energy", message: "انرژی صفر ثبت شده اما درصد شارژ تغییر کرده است" });
  }
  // 🟡 duration implausible vs energy/power.
  if (i.durationMin != null && i.durationMin > 0 && i.chargePowerKw != null && i.chargePowerKw > 0 && i.energyKwh != null && i.energyKwh > 0) {
    const predicted = (i.energyKwh / i.chargePowerKw) * 60;
    if (predicted > i.durationMin * DURATION_MISMATCH || predicted < i.durationMin / DURATION_MISMATCH) {
      flags.push({ code: "duration_implausible", severity: "warning", field: "duration", message: "مدت زمان شارژ با انرژی و توان شارژ همخوانی ندارد" });
    }
  }

  return flags;
}

/** True if any flag is a blocking error (Save must be disabled). */
export function hasBlockingError(flags: EntryFlag[]): boolean {
  return flags.some((f) => f.severity === "error");
}
