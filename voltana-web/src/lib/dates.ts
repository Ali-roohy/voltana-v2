// Centralized, locale-aware date formatting (TASK-0042 BUG-1).
//
// This is the ONLY place in the app that decides Jalali vs. Gregorian. Every
// date/month rendered anywhere — chart axes, tooltips, labels — must go through
// here. Adding English later then flips every date to Gregorian automatically,
// with zero per-component changes. Do not call toLocaleDateString / date-fns /
// date-fns-jalali for display formatting outside this module.
//
// `language` is the value from LanguageContext ('fa' | 'en'); anything other
// than 'fa' is treated as English (Gregorian).

import { format as formatGregorian } from "date-fns";
import { format as formatJalali } from "date-fns-jalali";
import { faIR } from "date-fns-jalali/locale/fa-IR";

export type AppLanguage = "fa" | "en";

function isFa(language: string): boolean {
  return language === "fa";
}

// date-fns-jalali emits Latin digits; Persian users expect Persian numerals
// (the previous Intl `toLocaleDateString('fa-IR')` path produced them), so map
// 0-9 → ۰-۹ for the fa branch to avoid a visual regression.
const PERSIAN_DIGITS = ["۰", "۱", "۲", "۳", "۴", "۵", "۶", "۷", "۸", "۹"];
function toPersianDigits(s: string): string {
  return s.replace(/[0-9]/g, (d) => PERSIAN_DIGITS[Number(d)]);
}

// Accept the shapes our charts/data actually carry: a Date, an epoch ms number,
// an ISO string, or a "YYYY-MM" month bucket (the monthly-trend dataKey).
function toDate(input: Date | string | number): Date {
  if (input instanceof Date) return input;
  if (typeof input === "string") {
    const ym = /^(\d{4})-(\d{2})$/.exec(input);
    if (ym) return new Date(Number(ym[1]), Number(ym[2]) - 1, 1);
  }
  return new Date(input);
}

/**
 * Month label with year. fa → Jalali ("آذر ۱۴۰۴"); en → Gregorian ("Dec 2025").
 * Accepts a Date, ISO string, epoch ms, or a "YYYY-MM" bucket key.
 */
export function formatMonth(input: Date | string | number, language: string): string {
  const d = toDate(input);
  return isFa(language)
    ? toPersianDigits(formatJalali(d, "MMMM yyyy", { locale: faIR }))
    : formatGregorian(d, "MMM yyyy");
}

/**
 * Short day label. fa → Jalali ("۱۲ آذر"); en → Gregorian ("Dec 12").
 * Accepts a Date, ISO string, or epoch ms.
 */
export function formatDate(input: Date | string | number, language: string): string {
  const d = toDate(input);
  return isFa(language)
    ? toPersianDigits(formatJalali(d, "d MMMM", { locale: faIR }))
    : formatGregorian(d, "MMM d");
}
