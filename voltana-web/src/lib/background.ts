// Theme background designer (TASK-0037 FEAT-3).
//
// The background is an independent layer UNDER the theme system: presets /
// dynamic car-color themes own the accent vars, this module owns only the page
// canvas. Stored in its own key so any theme × background combination works.
//
// Mechanism: pages render `.app-page-bg` / `.app-page-bg-gradient` (index.css).
// When a custom background is active we set `--app-custom-bg` on :root plus a
// `data-custom-bg` attribute on <html>; a single CSS rule then paints <body>
// and makes the page classes transparent.

const STORAGE_KEY = "voltana:bg";

export type GradientDirection = "to-b" | "to-br" | "to-r" | "to-tr";
export type PatternId = "dots" | "grid" | "stripes";

export type BackgroundStyle =
  | { type: "solid"; color: string }
  | { type: "gradient"; direction: GradientDirection; stops: string[] }
  | { type: "pattern"; id: PatternId };

const DIRECTION_CSS: Record<GradientDirection, string> = {
  "to-b": "to bottom",
  "to-br": "to bottom left", // RTL app: visually bottom-start
  "to-r": "to left",
  "to-tr": "to top left",
};

// Patterns derive from theme vars so they re-tint automatically when the user
// switches a preset or a dynamic car-color theme.
const PATTERN_CSS: Record<PatternId, string> = {
  dots: "radial-gradient(hsl(var(--primary) / 0.10) 1.5px, hsl(var(--background)) 1.5px) 0 0 / 22px 22px",
  grid: "linear-gradient(hsl(var(--primary) / 0.07) 1px, transparent 1px) 0 0 / 28px 28px, linear-gradient(90deg, hsl(var(--primary) / 0.07) 1px, hsl(var(--background)) 1px) 0 0 / 28px 28px",
  stripes:
    "repeating-linear-gradient(135deg, hsl(var(--primary) / 0.05) 0 12px, hsl(var(--background)) 12px 36px)",
};

export function backgroundToCSS(style: BackgroundStyle): string {
  switch (style.type) {
    case "solid":
      return style.color;
    case "gradient":
      return `linear-gradient(${DIRECTION_CSS[style.direction]}, ${style.stops.join(", ")})`;
    case "pattern":
      return PATTERN_CSS[style.id];
  }
}

export function getSavedBackground(): BackgroundStyle | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as BackgroundStyle;
    if (parsed.type === "solid" && typeof parsed.color === "string") return parsed;
    if (
      parsed.type === "gradient" &&
      Array.isArray(parsed.stops) &&
      parsed.stops.length >= 2 &&
      parsed.stops.length <= 3 &&
      parsed.direction in DIRECTION_CSS
    )
      return parsed;
    if (parsed.type === "pattern" && parsed.id in PATTERN_CSS) return parsed;
    return null;
  } catch {
    return null;
  }
}

export function applyBackground(style: BackgroundStyle | null): void {
  const root = document.documentElement;
  if (!style) {
    root.style.removeProperty("--app-custom-bg");
    root.removeAttribute("data-custom-bg");
    return;
  }
  root.style.setProperty("--app-custom-bg", backgroundToCSS(style));
  root.setAttribute("data-custom-bg", style.type);
}

export function saveBackground(style: BackgroundStyle | null): void {
  if (!style) {
    localStorage.removeItem(STORAGE_KEY);
  } else {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(style));
  }
  applyBackground(style);
}

export function applySavedBackground(): void {
  applyBackground(getSavedBackground());
}
