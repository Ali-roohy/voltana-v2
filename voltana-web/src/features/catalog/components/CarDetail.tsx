import { useState } from "react";
import { CarFront } from "lucide-react";
import { Badge } from "@/components/ui/badge";
import { Checkbox } from "@/components/ui/checkbox";
import { Sheet, SheetContent, SheetHeader, SheetTitle } from "@/components/ui/sheet";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { applyDynamicTheme, cssColorFor, getSavedDynamicColor } from "@/lib/dynamic-theme";
import type { CatalogCar } from "../api";
import { fmt, text } from "../format";
import { CarComparison } from "./CarComparison";
import { ColorPicker } from "./ColorPicker";

interface CarDetailProps {
  car: CatalogCar | null;
  allCars: CatalogCar[];
  compareIds: string[];
  onToggleCompare: (id: string) => void;
  onClose: () => void;
}

interface SpecSection {
  title: string;
  rows: [string, string][];
}

// All 43 source fields, organized in the 6 sections from the task spec.
function sections(car: CatalogCar): SpecSection[] {
  return [
    {
      title: "عمومی",
      rows: [
        ["نام فارسی", car.name_fa],
        ["نام انگلیسی", car.name_en],
        ["برند", text(car.brand)],
        ["کلاس", text(car.class)],
        ["نوع بدنه", text(car.body_type)],
        ["بدنه", text(car.body_style_fa)],
        ["سطح بازار", text(car.segment)],
        ["کشور", text(car.country)],
        ["واردکننده", text(car.importer)],
        ["پلتفرم", text(car.platform)],
      ],
    },
    {
      title: "باتری",
      rows: [
        ["ظرفیت", fmt(car.battery_capacity_kwh, "kWh")],
        ["قابل استفاده", fmt(car.usable_kwh, "kWh")],
        ["ولتاژ", text(car.battery_voltage)],
        ["نوع سلول", text(car.cell_type)],
        ["برند سلول", text(car.cell_brand)],
        ["خنک‌کاری", text(car.cooling)],
        ["برد", fmt(car.range_km, "km")],
        ["استاندارد برد", text(car.range_standard)],
        ["مصرف", fmt(car.consumption_kwh_per_100km, "kWh/100km")],
      ],
    },
    {
      title: "موتور و عملکرد",
      rows: [
        ["توان", fmt(car.motor_power_kw, "kW")],
        ["گشتاور", fmt(car.torque_nm, "Nm")],
        ["تعداد موتور", fmt(car.motor_count)],
        ["نوع موتور", text(car.motor_type)],
        ["شتاب ۰→۱۰۰", fmt(car.acceleration_0_100_s, "ثانیه")],
        ["حداکثر سرعت", fmt(car.max_speed_kmh, "km/h")],
        ["دیفرانسیل", text(car.drivetrain)],
      ],
    },
    {
      title: "شارژ",
      rows: [
        ["توان AC", fmt(car.ac_charge_kw, "kW")],
        ["کانکتور AC", text(car.ac_connector)],
        ["توان DC", fmt(car.dc_charge_kw, "kW")],
        ["کانکتور DC", text(car.dc_connector)],
        ["بازه شارژ سریع", text(car.fast_charge_window)],
        ["زمان شارژ سریع", fmt(car.fast_charge_to_80_min, "دقیقه")],
        ["V2L", text(car.v2l)],
        ["V2G", text(car.v2g)],
        ["OTA", text(car.ota)],
      ],
    },
    {
      title: "ADAS",
      rows: [
        ["سطح", text(car.adas_level)],
        ["تعداد رادار", fmt(car.radar_count)],
        ["تعداد دوربین", fmt(car.camera_count)],
      ],
    },
    {
      title: "راحتی و ابعاد",
      rows: [
        ["وزن", fmt(car.weight_kg, "kg")],
        ["صندوق", fmt(car.trunk_liters, "لیتر")],
        ["رنگ‌های داخل کابین", car.interior_colors.length ? car.interior_colors.join("، ") : "—"],
      ],
    },
  ];
}

export const CarDetail = ({ car, allCars, compareIds, onToggleCompare, onClose }: CarDetailProps) => {
  const [selectedColor, setSelectedColor] = useState<string | null>(getSavedDynamicColor());

  if (!car) return null;

  const compareCars = allCars.filter((c) => compareIds.includes(c.id));
  const heroColor = cssColorFor(selectedColor ?? car.exterior_colors[0] ?? "");

  const pickColor = (color: string) => {
    setSelectedColor(color);
    applyDynamicTheme(color); // live theme shift + localStorage persist
  };

  return (
    <Sheet open={!!car} onOpenChange={(open) => !open && onClose()}>
      <SheetContent side="left" className="w-full overflow-y-auto p-0 sm:max-w-xl">
        {/* banner — tinted by the selected exterior color */}
        <div
          className="flex h-28 flex-col justify-end p-4 text-white"
          style={{ background: `linear-gradient(135deg, ${heroColor}d0, hsl(var(--primary)))` }}
        >
          <SheetHeader className="space-y-0 text-right">
            <SheetTitle className="text-white drop-shadow">{car.name_fa}</SheetTitle>
          </SheetHeader>
          <div className="mt-1 flex items-center gap-2 text-xs">
            <span dir="ltr">{car.name_en}</span>
            <span className="flex items-center gap-1 rounded-full bg-black/25 px-2 py-0.5">
              <CarFront className="h-3 w-3" />
              {text(car.body_type)}
            </span>
            <Badge variant="secondary">{text(car.adas_level)} ADAS</Badge>
          </div>
        </div>

        <Tabs defaultValue="specs" className="p-4">
          <TabsList className="grid w-full grid-cols-4">
            <TabsTrigger value="specs">مشخصات</TabsTrigger>
            <TabsTrigger value="colors">رنگ‌ها</TabsTrigger>
            <TabsTrigger value="compare">مقایسه</TabsTrigger>
            <TabsTrigger value="notes">توضیحات</TabsTrigger>
          </TabsList>

          <TabsContent value="specs" className="mt-4 space-y-5">
            {sections(car).map((sec) => (
              <section key={sec.title}>
                <h4 className="mb-2 border-b pb-1 text-sm font-bold text-primary">{sec.title}</h4>
                <dl className="grid grid-cols-1 gap-x-6 gap-y-1.5 sm:grid-cols-2">
                  {sec.rows.map(([label, value]) => (
                    <div key={label} className="flex items-baseline justify-between gap-2 text-sm">
                      <dt className="shrink-0 text-muted-foreground">{label}</dt>
                      <dd className="text-left font-medium">{value}</dd>
                    </div>
                  ))}
                </dl>
              </section>
            ))}
          </TabsContent>

          <TabsContent value="colors" className="mt-4 space-y-3">
            <p className="text-sm text-muted-foreground">
              با انتخاب رنگ بدنه، تم برنامه به‌صورت زنده با همان رنگ هماهنگ می‌شود.
            </p>
            <ColorPicker
              colors={car.exterior_colors}
              selected={selectedColor}
              onSelect={pickColor}
            />
          </TabsContent>

          <TabsContent value="compare" className="mt-4 space-y-4">
            <p className="text-sm text-muted-foreground">
              تا ۳ خودرو برای مقایسه انتخاب کنید ({compareIds.length.toLocaleString("fa-IR")}/۳)
            </p>
            <div className="max-h-44 space-y-1 overflow-y-auto rounded-md border p-2">
              {allCars.map((c) => {
                const checked = compareIds.includes(c.id);
                const full = compareIds.length >= 3 && !checked;
                return (
                  <label
                    key={c.id}
                    className={`flex cursor-pointer items-center gap-2 rounded p-1.5 text-sm hover:bg-muted ${full ? "opacity-40" : ""}`}
                  >
                    <Checkbox
                      checked={checked}
                      disabled={full}
                      onCheckedChange={() => onToggleCompare(c.id)}
                    />
                    <span>{c.name_fa}</span>
                    <span className="text-xs text-muted-foreground" dir="ltr">
                      {c.name_en}
                    </span>
                  </label>
                );
              })}
            </div>
            <CarComparison cars={compareCars} />
          </TabsContent>

          <TabsContent value="notes" className="mt-4">
            <p className="text-sm leading-7">{text(car.notes)}</p>
          </TabsContent>
        </Tabs>
      </SheetContent>
    </Sheet>
  );
};
