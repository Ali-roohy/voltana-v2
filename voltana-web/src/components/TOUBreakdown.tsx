import { useTranslation } from "react-i18next";
import { cn, formatNumber } from "@/lib/utils";

// One time-of-use bucket: its energy (kWh) and cost (Toman).
export interface TouSegment {
  kwh: number;
  cost: number;
}

interface TOUBreakdownProps {
  peak: TouSegment;
  mid: TouSegment;
  offpeak: TouSegment;
  // Defaults to the sum of the three segments when omitted.
  total?: TouSegment;
  // "inline" = compact, for a session card; "summary" = a little larger, for the dashboard.
  variant?: "inline" | "summary";
  className?: string;
}

type SegKey = "peak" | "mid" | "offpeak";

interface SegMeta {
  key: SegKey;
  seg: TouSegment;
  barClass: string;
  dotClass: string;
}

// Presentational stacked-bar breakdown of charging spend across peak/mid/off-peak.
// Pure (no data fetching) so it serves both a single session and the month aggregate.
export function TOUBreakdown({ peak, mid, offpeak, total, variant = "inline", className }: TOUBreakdownProps) {
  const { t } = useTranslation();
  const isSummary = variant === "summary";

  const segments: SegMeta[] = [
    { key: "peak", seg: peak, barClass: "bg-red-500", dotClass: "bg-red-500" },
    { key: "mid", seg: mid, barClass: "bg-amber-500", dotClass: "bg-amber-500" },
    { key: "offpeak", seg: offpeak, barClass: "bg-green-500", dotClass: "bg-green-500" },
  ];

  const totalSeg: TouSegment = total ?? {
    kwh: peak.kwh + mid.kwh + offpeak.kwh,
    cost: peak.cost + mid.cost + offpeak.cost,
  };

  // Only buckets with energy are shown; the bar is sized by each bucket's cost share.
  const present = segments.filter((s) => s.seg.kwh > 0);
  const costBasis = present.reduce((sum, s) => sum + s.seg.cost, 0);
  const hasBar = present.length > 0 && costBasis > 0;

  return (
    <div className={cn("w-full", isSummary ? "space-y-3" : "space-y-2", className)}>
      {hasBar && (
        <div className={cn("flex w-full overflow-hidden rounded-full bg-muted", isSummary ? "h-3" : "h-2")}>
          {present.map((s) => (
            <div
              key={s.key}
              className={s.barClass}
              style={{ width: `${(s.seg.cost / costBasis) * 100}%` }}
              title={`${t(`tou.${s.key}`)}: ${formatNumber(Math.round(s.seg.cost))} ${t("tou.toman")}`}
            />
          ))}
        </div>
      )}

      {present.length > 0 ? (
        <div className={cn("space-y-1", isSummary ? "text-sm" : "text-xs")}>
          {present.map((s) => (
            // RTL row: ● label on the start (right) side, value group on the end (left) side.
            <div key={s.key} className="flex items-center justify-between gap-2">
              <span className="flex items-center gap-1.5 text-muted-foreground">
                <span className={cn("inline-block rounded-full", isSummary ? "h-2.5 w-2.5" : "h-2 w-2", s.dotClass)} />
                {t(`tou.${s.key}`)}
              </span>
              {/* Isolate the Latin/number runs so bidi can't swap them with the Persian تومان unit. */}
              <span className="text-muted-foreground">
                <span dir="ltr">{formatNumber(s.seg.kwh.toFixed(2))} kWh</span>
                {" · "}
                <span dir="ltr">{formatNumber(Math.round(s.seg.cost))}</span> {t("tou.toman")}
              </span>
            </div>
          ))}
        </div>
      ) : (
        // Degraded state: a session that only has a grand total (kwh_charged), no split.
        <div className={cn("text-muted-foreground", isSummary ? "text-sm" : "text-xs")}>{t("tou.noBreakdown")}</div>
      )}

      <div
        className={cn(
          "flex items-center justify-between gap-2 border-t border-border/50 pt-1.5 font-medium",
          isSummary ? "text-sm" : "text-xs",
        )}
      >
        <span>{t("tou.total")}</span>
        {/* Isolate the Latin/number runs so bidi can't swap them with the Persian تومان unit. */}
        <span>
          <span dir="ltr">{formatNumber(totalSeg.kwh.toFixed(2))} kWh</span>
          {" · "}
          <span dir="ltr">{formatNumber(Math.round(totalSeg.cost))}</span> {t("tou.toman")}
        </span>
      </div>
    </div>
  );
}
