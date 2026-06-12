// TASK-0033 — dynamic theme generated from a car's exterior color name.
//
// The catalog stores Persian color names ("سفید مروارید", "آبی متالیک", …),
// not hex values. A keyword table maps the name to a base hue, the hue is
// classified warm / cool / neutral, and a palette is generated over the same
// CSS variables the static presets in lib/themes.ts override. The result is
// persisted to the same localStorage key ('voltana:theme') with a "dynamic:"
// prefix — picking a static preset in Settings simply overwrites the key, so
// the two systems coexist.

import { OVERRIDEABLE_VARS } from "./themes";

const STORAGE_KEY = "voltana:theme";
const DYNAMIC_PREFIX = "dynamic:";

interface ColorDef {
  /** keyword found inside the Persian color name */
  keywords: string[];
  /** representative hue (0-360); null = achromatic */
  hue: number | null;
  /** swatch the UI shows for this color name */
  css: string;
}

// Order matters: more specific names (آبی نفتی, سرمه‌ای) before generic ones.
const COLOR_DEFS: ColorDef[] = [
  { keywords: ["آبی نفتی", "سرمه‌ای", "سرمه ای"], hue: 215, css: "#1e3a5f" },
  { keywords: ["آبی"], hue: 215, css: "#2563eb" },
  { keywords: ["فیروزه"], hue: 180, css: "#14b8a6" },
  { keywords: ["سبز"], hue: 150, css: "#16a34a" },
  { keywords: ["بنفش", "یاسی"], hue: 270, css: "#8b5cf6" },
  { keywords: ["قرمز", "آلبالویی", "زرشکی"], hue: 0, css: "#dc2626" },
  { keywords: ["نارنجی", "مسی", "کارامل"], hue: 25, css: "#ea580c" },
  { keywords: ["زرد", "طلایی"], hue: 45, css: "#eab308" },
  { keywords: ["قهوه‌ای", "قهوه ای", "شکلاتی", "برنز"], hue: 25, css: "#92400e" },
  { keywords: ["صورتی", "گلبهی"], hue: 340, css: "#ec4899" },
  { keywords: ["بژ", "کرم", "شنی"], hue: 40, css: "#d6c49a" },
  { keywords: ["سفید", "مروارید"], hue: null, css: "#f4f4f5" },
  { keywords: ["مشکی", "سیاه"], hue: null, css: "#27272a" },
  { keywords: ["خاکستری", "تیتانیوم", "گرافیت", "دودی"], hue: null, css: "#71717a" },
  { keywords: ["نقره‌ای", "نقره ای"], hue: null, css: "#c0c4cc" },
];

function findColorDef(name: string): ColorDef | null {
  for (const def of COLOR_DEFS) {
    if (def.keywords.some((k) => name.includes(k))) return def;
  }
  return null;
}

/** CSS color for a Persian color-name swatch (gray fallback for unknowns). */
export function cssColorFor(name: string): string {
  return findColorDef(name)?.css ?? "#9ca3af";
}

export type PaletteKind = "warm" | "cool" | "neutral";

export function paletteKindFor(name: string): PaletteKind {
  const def = findColorDef(name);
  if (!def || def.hue === null) return "neutral";
  // 80..300 covers green→blue→purple; everything else (reds/oranges/yellows
  // and the pink wrap-around) reads warm.
  return def.hue >= 80 && def.hue <= 300 ? "cool" : "warm";
}

interface Palette {
  primary: { h: number; s: number; l: number };
  glow: { h: number; s: number; l: number };
}

function buildPalette(name: string): Palette {
  const def = findColorDef(name);
  const kind = paletteKindFor(name);
  if (!def || def.hue === null) {
    // Neutral: a calm slate-blue primary with a soft cyan glow, nudged darker
    // for black/gray cars and lighter for white/silver.
    const light = name.includes("سفید") || name.includes("نقره") || name.includes("مروارید");
    return {
      primary: { h: 220, s: 15, l: light ? 55 : 40 },
      glow: { h: 200, s: 30, l: light ? 65 : 50 },
    };
  }
  const h = def.hue;
  // Accent leans further warm for warm hues, further cool for cool hues.
  const glowH = (kind === "warm" ? h + 20 : h - 25 + 360) % 360;
  return {
    primary: { h, s: 80, l: 52 },
    glow: { h: glowH, s: 90, l: 58 },
  };
}

const hsl = (c: { h: number; s: number; l: number }) => `${c.h} ${c.s}% ${c.l}%`;

function varsFor(name: string): Record<string, string> {
  const { primary, glow } = buildPalette(name);
  return {
    "--primary": hsl(primary),
    "--primary-foreground": "0 0% 100%",
    "--primary-glow": hsl(glow),
    "--accent": hsl(glow),
    "--accent-foreground": "0 0% 100%",
    "--ring": hsl(primary),
    "--gradient-primary": `linear-gradient(135deg, hsl(${hsl(primary)}), hsl(${hsl(glow)}))`,
    "--shadow-soft": `0 4px 20px -4px hsl(${hsl(primary)} / 0.25)`,
    "--shadow-glow": `0 0 40px hsl(${hsl(glow)} / 0.4)`,
  };
}

/**
 * Shift the whole app to a palette derived from the car color and persist it.
 * The brief `theme-transition` class makes components ease into the new colors
 * (rule in index.css) instead of snapping.
 */
export function applyDynamicTheme(colorName: string): void {
  const root = document.documentElement;
  root.classList.add("theme-transition");
  OVERRIDEABLE_VARS.forEach((v) => root.style.removeProperty(v));
  Object.entries(varsFor(colorName)).forEach(([k, v]) => root.style.setProperty(k, v));
  localStorage.setItem(STORAGE_KEY, DYNAMIC_PREFIX + colorName);
  window.setTimeout(() => root.classList.remove("theme-transition"), 600);
}

/** The persisted dynamic color name, or null when a static preset is active. */
export function getSavedDynamicColor(): string | null {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved && saved.startsWith(DYNAMIC_PREFIX)) {
    return saved.slice(DYNAMIC_PREFIX.length);
  }
  return null;
}
