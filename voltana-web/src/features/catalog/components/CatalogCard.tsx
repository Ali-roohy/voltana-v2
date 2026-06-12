import { motion } from "framer-motion";
import { Battery, CarFront, Gauge, Zap } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { cssColorFor } from "@/lib/dynamic-theme";
import type { CatalogCar } from "../api";
import { fmt, text } from "../format";

interface CatalogCardProps {
  car: CatalogCar;
  onDetails: (car: CatalogCar) => void;
}

export const CatalogCard = ({ car, onDetails }: CatalogCardProps) => (
  <motion.div layout initial={{ opacity: 0, y: 8 }} animate={{ opacity: 1, y: 0 }}>
    <Card className="h-full overflow-hidden hover:shadow-lg transition-shadow">
      {/* hero — gradient from the first exterior color (no real photos yet) */}
      <div
        className="flex h-24 items-end justify-between p-4 text-white"
        style={{
          background: `linear-gradient(135deg, ${cssColorFor(car.exterior_colors[0] ?? "")}cc, hsl(var(--primary)))`,
        }}
      >
        <div className="min-w-0">
          <h3 className="truncate text-lg font-bold drop-shadow">{car.name_fa}</h3>
          <p className="truncate text-xs opacity-90" dir="ltr">{car.name_en}</p>
        </div>
        <div className="flex shrink-0 items-center gap-1 rounded-full bg-black/25 px-2 py-1 text-xs">
          <CarFront className="h-3.5 w-3.5" />
          {text(car.body_type)}
        </div>
      </div>

      <CardContent className="space-y-3 p-4">
        {/* color swatches (first 3) */}
        <div className="flex items-center gap-1.5">
          {car.exterior_colors.slice(0, 3).map((c) => (
            <span
              key={c}
              title={c}
              className="h-4 w-4 rounded-full border border-black/10 shadow-sm"
              style={{ backgroundColor: cssColorFor(c) }}
            />
          ))}
          {car.exterior_colors.length > 3 && (
            <span className="text-xs text-muted-foreground">+{car.exterior_colors.length - 3}</span>
          )}
          <Badge variant="secondary" className="mr-auto">
            {text(car.adas_level)} ADAS
          </Badge>
        </div>

        <div className="space-y-1.5 text-sm">
          <div className="flex items-center gap-2">
            <Battery className="h-4 w-4 shrink-0 text-primary" />
            <span>
              {fmt(car.battery_capacity_kwh, "kWh")}
              {car.cell_type ? ` · ${car.cell_type}` : ""} · {fmt(car.range_km, "km")}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Gauge className="h-4 w-4 shrink-0 text-primary" />
            <span>
              {fmt(car.motor_power_kw, "kW")} · ۰→۱۰۰: {fmt(car.acceleration_0_100_s, "ث")}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <Zap className="h-4 w-4 shrink-0 text-primary" />
            <span>
              DC {fmt(car.dc_charge_kw, "kW")}
              {car.fast_charge_to_80_min !== null
                ? ` · ${fmt(car.fast_charge_to_80_min)} دقیقه شارژ سریع`
                : ""}
            </span>
          </div>
        </div>

        <Button className="w-full" size="sm" onClick={() => onDetails(car)}>
          جزئیات
        </Button>
      </CardContent>
    </Card>
  </motion.div>
);
