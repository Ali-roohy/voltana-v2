import { cn } from "@/lib/utils";
import type { CatalogCar } from "../api";
import { fmt, segmentTier, text } from "../format";

interface CarComparisonProps {
  cars: CatalogCar[]; // 2–3 cars
}

interface MetricRow {
  label: string;
  /** numeric accessor for best/worst ranking — omit for text-only rows */
  value?: (c: CatalogCar) => number | null;
  display: (c: CatalogCar) => string;
  /** true when a larger value wins the row */
  higherIsBetter: boolean;
}

const ROWS: MetricRow[] = [
  { label: "باتری (kWh)", value: (c) => c.battery_capacity_kwh, display: (c) => fmt(c.battery_capacity_kwh), higherIsBetter: true },
  { label: "قابل استفاده (kWh)", value: (c) => c.usable_kwh, display: (c) => fmt(c.usable_kwh), higherIsBetter: true },
  { label: "برد (km)", value: (c) => c.range_km, display: (c) => fmt(c.range_km), higherIsBetter: true },
  { label: "مصرف (kWh/100km)", value: (c) => c.consumption_kwh_per_100km, display: (c) => fmt(c.consumption_kwh_per_100km), higherIsBetter: false },
  { label: "توان (kW)", value: (c) => c.motor_power_kw, display: (c) => fmt(c.motor_power_kw), higherIsBetter: true },
  { label: "گشتاور (Nm)", value: (c) => c.torque_nm, display: (c) => fmt(c.torque_nm), higherIsBetter: true },
  { label: "شتاب ۰→۱۰۰ (ثانیه)", value: (c) => c.acceleration_0_100_s, display: (c) => fmt(c.acceleration_0_100_s), higherIsBetter: false },
  { label: "حداکثر سرعت (km/h)", value: (c) => c.max_speed_kmh, display: (c) => fmt(c.max_speed_kmh), higherIsBetter: true },
  { label: "شارژ AC (kW)", value: (c) => c.ac_charge_kw, display: (c) => fmt(c.ac_charge_kw), higherIsBetter: true },
  { label: "شارژ DC (kW)", value: (c) => c.dc_charge_kw, display: (c) => fmt(c.dc_charge_kw), higherIsBetter: true },
  { label: "شارژ سریع (دقیقه)", value: (c) => c.fast_charge_to_80_min, display: (c) => fmt(c.fast_charge_to_80_min), higherIsBetter: false },
  { label: "سطح ADAS", display: (c) => text(c.adas_level), higherIsBetter: true },
  { label: "وزن (kg)", value: (c) => c.weight_kg, display: (c) => fmt(c.weight_kg), higherIsBetter: false },
  { label: "صندوق (L)", value: (c) => c.trunk_liters, display: (c) => fmt(c.trunk_liters), higherIsBetter: true },
  { label: "رده بازار", value: (c) => segmentTier(c.segment), display: (c) => text(c.segment), higherIsBetter: true },
];

// best/worst per numeric row: green = best value, red = worst (only when the
// cars actually differ; text-only rows like ADAS are never colored).
function rowHighlights(row: MetricRow, cars: CatalogCar[]): (null | "best" | "worst")[] {
  if (!row.value) return cars.map(() => null);
  const vals = cars.map((c) => row.value(c));
  const present = vals.filter((v): v is number => v !== null);
  if (present.length < 2) return cars.map(() => null);
  const max = Math.max(...present);
  const min = Math.min(...present);
  if (max === min) return cars.map(() => null);
  const bestVal = row.higherIsBetter ? max : min;
  const worstVal = row.higherIsBetter ? min : max;
  return vals.map((v) => (v === null ? null : v === bestVal ? "best" : v === worstVal ? "worst" : null));
}

export const CarComparison = ({ cars }: CarComparisonProps) => {
  if (cars.length < 2) {
    return (
      <p className="py-8 text-center text-sm text-muted-foreground">
        برای مقایسه حداقل ۲ خودرو انتخاب کنید
      </p>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse text-sm">
        <thead>
          <tr>
            <th className="sticky right-0 bg-background p-2 text-right font-medium text-muted-foreground">
              مشخصه
            </th>
            {cars.map((car) => (
              <th key={car.id} className="min-w-[110px] p-2 text-center">
                <div className="font-bold">{car.name_fa}</div>
                <div className="text-xs font-normal text-muted-foreground" dir="ltr">
                  {car.name_en}
                </div>
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {ROWS.map((row) => {
            const marks = rowHighlights(row, cars);
            return (
              <tr key={row.label} className="border-t">
                <td className="sticky right-0 bg-background p-2 text-right text-muted-foreground">
                  {row.label}
                </td>
                {cars.map((car, i) => (
                  <td
                    key={car.id}
                    className={cn(
                      "p-2 text-center",
                      marks[i] === "best" && "bg-green-500/10 font-semibold text-green-600 dark:text-green-400",
                      marks[i] === "worst" && "bg-red-500/10 text-red-600 dark:text-red-400",
                    )}
                  >
                    {row.display(car)}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
};
