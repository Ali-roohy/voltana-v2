import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Label } from "@/components/ui/label";
import {
  type BackgroundStyle,
  type GradientDirection,
  type PatternId,
  getSavedBackground,
  saveBackground,
  backgroundToCSS,
} from "@/lib/background";

type Mode = "default" | "solid" | "gradient" | "pattern";

const MODES: { id: Mode; label: string }[] = [
  { id: "default", label: "پیش‌فرض" },
  { id: "solid", label: "رنگ ثابت" },
  { id: "gradient", label: "گرادیان" },
  { id: "pattern", label: "الگو" },
];

const DIRECTIONS: { id: GradientDirection; label: string }[] = [
  { id: "to-b", label: "↓ عمودی" },
  { id: "to-br", label: "↘ مورب" },
  { id: "to-r", label: "→ افقی" },
  { id: "to-tr", label: "↗ مورب معکوس" },
];

const PATTERNS: { id: PatternId; label: string }[] = [
  { id: "dots", label: "نقطه‌ای" },
  { id: "grid", label: "شبکه‌ای" },
  { id: "stripes", label: "راه‌راه" },
];

// Every change applies live AND persists (same behavior as the theme swatches).
export function BackgroundPicker() {
  const saved = getSavedBackground();
  const [mode, setMode] = useState<Mode>(saved?.type ?? "default");
  const [solidColor, setSolidColor] = useState(saved?.type === "solid" ? saved.color : "#0d1b2a");
  const [direction, setDirection] = useState<GradientDirection>(
    saved?.type === "gradient" ? saved.direction : "to-br",
  );
  const [stops, setStops] = useState<string[]>(
    saved?.type === "gradient" ? saved.stops : ["#0d1b2a", "#1b4965"],
  );
  const [patternId, setPatternId] = useState<PatternId>(
    saved?.type === "pattern" ? saved.id : "dots",
  );

  const apply = (style: BackgroundStyle | null) => saveBackground(style);

  const applyMode = (m: Mode) => {
    setMode(m);
    if (m === "default") apply(null);
    else if (m === "solid") apply({ type: "solid", color: solidColor });
    else if (m === "gradient") apply({ type: "gradient", direction, stops });
    else apply({ type: "pattern", id: patternId });
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-4 gap-2">
        {MODES.map((m) => (
          <Button
            key={m.id}
            size="sm"
            variant={mode === m.id ? "default" : "outline"}
            onClick={() => applyMode(m.id)}
          >
            {m.label}
          </Button>
        ))}
      </div>

      {mode === "solid" && (
        <div className="flex items-center gap-3">
          <Label htmlFor="bg-solid-color" className="shrink-0">رنگ پس‌زمینه</Label>
          <input
            id="bg-solid-color"
            type="color"
            value={solidColor}
            onChange={(e) => {
              setSolidColor(e.target.value);
              apply({ type: "solid", color: e.target.value });
            }}
            className="h-9 w-16 cursor-pointer rounded border bg-transparent"
          />
        </div>
      )}

      {mode === "gradient" && (
        <div className="space-y-3">
          <div className="grid grid-cols-4 gap-2">
            {DIRECTIONS.map((d) => (
              <Button
                key={d.id}
                size="sm"
                variant={direction === d.id ? "default" : "outline"}
                className="text-xs"
                onClick={() => {
                  setDirection(d.id);
                  apply({ type: "gradient", direction: d.id, stops });
                }}
              >
                {d.label}
              </Button>
            ))}
          </div>
          <div className="flex items-center gap-2">
            {stops.map((stop, i) => (
              <input
                key={i}
                type="color"
                aria-label={`رنگ ${i + 1}`}
                value={stop}
                onChange={(e) => {
                  const next = [...stops];
                  next[i] = e.target.value;
                  setStops(next);
                  apply({ type: "gradient", direction, stops: next });
                }}
                className="h-9 w-14 cursor-pointer rounded border bg-transparent"
              />
            ))}
            {stops.length < 3 ? (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  const next = [...stops, "#324a5f"];
                  setStops(next);
                  apply({ type: "gradient", direction, stops: next });
                }}
              >
                + رنگ
              </Button>
            ) : (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => {
                  const next = stops.slice(0, 2);
                  setStops(next);
                  apply({ type: "gradient", direction, stops: next });
                }}
              >
                − حذف
              </Button>
            )}
          </div>
        </div>
      )}

      {mode === "pattern" && (
        <div className="grid grid-cols-3 gap-2">
          {PATTERNS.map((p) => (
            <button
              key={p.id}
              type="button"
              onClick={() => {
                setPatternId(p.id);
                apply({ type: "pattern", id: p.id });
              }}
              className={`h-16 rounded-lg border-2 text-xs font-medium ${
                patternId === p.id ? "border-primary" : "border-border"
              }`}
              style={{ background: backgroundToCSS({ type: "pattern", id: p.id }) }}
            >
              {p.label}
            </button>
          ))}
        </div>
      )}

      <p className="text-xs text-muted-foreground">
        پس‌زمینه مستقل از تم رنگی است — با تغییر تم، الگوها خودکار هم‌رنگ می‌شوند.
      </p>
    </div>
  );
}
