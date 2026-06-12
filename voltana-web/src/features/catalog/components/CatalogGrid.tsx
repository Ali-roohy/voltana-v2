import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { cssColorFor } from "@/lib/dynamic-theme";
import type { CatalogCar } from "../api";
import { fmt, text } from "../format";
import { CatalogCard } from "./CatalogCard";

interface CatalogGridProps {
  cars: CatalogCar[];
  mode: "grid" | "list";
  onDetails: (car: CatalogCar) => void;
}

// Grid mode: responsive card grid (1 / 2 / 3 columns).
// List mode: dense spec rows for side-by-side scanning on desktop.
export const CatalogGrid = ({ cars, mode, onDetails }: CatalogGridProps) => {
  if (cars.length === 0) {
    return (
      <p className="py-12 text-center text-muted-foreground">
        خودرویی با این فیلترها پیدا نشد
      </p>
    );
  }

  if (mode === "grid") {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {cars.map((car) => (
          <CatalogCard key={car.id} car={car} onDetails={onDetails} />
        ))}
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-lg border">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="text-right">خودرو</TableHead>
            <TableHead className="text-right">بدنه</TableHead>
            <TableHead className="text-right">باتری</TableHead>
            <TableHead className="text-right">برد</TableHead>
            <TableHead className="text-right">توان</TableHead>
            <TableHead className="text-right">۰→۱۰۰</TableHead>
            <TableHead className="text-right">شارژ DC</TableHead>
            <TableHead className="text-right">ADAS</TableHead>
            <TableHead className="text-right">رنگ‌ها</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          {cars.map((car) => (
            <TableRow key={car.id} className="cursor-pointer" onClick={() => onDetails(car)}>
              <TableCell>
                <div className="font-medium">{car.name_fa}</div>
                <div className="text-xs text-muted-foreground" dir="ltr">
                  {car.name_en}
                </div>
              </TableCell>
              <TableCell>{text(car.body_type)}</TableCell>
              <TableCell>
                {fmt(car.battery_capacity_kwh, "kWh")}
                {car.cell_type ? ` ${car.cell_type}` : ""}
              </TableCell>
              <TableCell>{fmt(car.range_km, "km")}</TableCell>
              <TableCell>{fmt(car.motor_power_kw, "kW")}</TableCell>
              <TableCell>{fmt(car.acceleration_0_100_s, "ث")}</TableCell>
              <TableCell>{fmt(car.dc_charge_kw, "kW")}</TableCell>
              <TableCell>
                <Badge variant="secondary">{text(car.adas_level)}</Badge>
              </TableCell>
              <TableCell>
                <div className="flex gap-1">
                  {car.exterior_colors.slice(0, 4).map((c) => (
                    <span
                      key={c}
                      title={c}
                      className="h-3.5 w-3.5 rounded-full border border-black/10"
                      style={{ backgroundColor: cssColorFor(c) }}
                    />
                  ))}
                </div>
              </TableCell>
              <TableCell>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={(e) => {
                    e.stopPropagation();
                    onDetails(car);
                  }}
                >
                  جزئیات
                </Button>
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
};
