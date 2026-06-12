import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { SpecOverrides } from "@/features/cars/api";
import { useCreateCar } from "@/features/cars/hooks";
import type { CatalogCar } from "../api";
import { SPEC_FIELD_SECTIONS, catalogValueOf } from "../fields";
import { ColorPicker } from "./ColorPicker";

interface CustomizeCarModalProps {
  car: CatalogCar;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

// "Add to my cars" customization (TASK-0034). Every catalog spec is pre-filled
// and editable; ONLY the fields the user actually changes are sent as
// spec_overrides — untouched specs keep following the catalog.
export const CustomizeCarModal = ({ car, open, onOpenChange }: CustomizeCarModalProps) => {
  const navigate = useNavigate();
  const createCar = useCreateCar();

  const [name, setName] = useState(car.name_fa);
  const [licensePlate, setLicensePlate] = useState("");
  const [odometerKm, setOdometerKm] = useState(0);
  const [exteriorColor, setExteriorColor] = useState<string | null>(null);
  const [interiorColor, setInteriorColor] = useState<string | null>(null);
  const [specs, setSpecs] = useState<Record<string, string>>(() => {
    const init: Record<string, string> = {};
    SPEC_FIELD_SECTIONS.forEach((sec) =>
      sec.fields.forEach((f) => {
        init[f.key] = catalogValueOf(car, f.key);
      }),
    );
    return init;
  });

  const buildOverrides = (): SpecOverrides => {
    const overrides: SpecOverrides = {};
    SPEC_FIELD_SECTIONS.forEach((sec) =>
      sec.fields.forEach((f) => {
        const current = specs[f.key].trim();
        const original = catalogValueOf(car, f.key);
        if (current === original) return; // unchanged → follow the catalog
        if (f.kind === "number") {
          const n = parseFloat(current);
          if (!Number.isNaN(n)) overrides[f.key] = n;
        } else if (current !== "") {
          overrides[f.key] = current;
        }
      }),
    );
    if (exteriorColor) overrides.exterior_color = exteriorColor;
    if (interiorColor) overrides.interior_color = interiorColor;
    return overrides;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      await createCar.mutateAsync({
        name: name.trim() || car.name_fa,
        catalog_car_id: car.id,
        spec_overrides: buildOverrides(),
        license_plate: licensePlate.trim() || null,
        odometer_km: odometerKm,
      });
      toast.success("خودرو اضافه شد");
      onOpenChange(false);
      navigate("/cars");
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[90vh] max-w-2xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>افزودن {car.name_fa} به خودروهای من</DialogTitle>
        </DialogHeader>

        <form onSubmit={handleSubmit} className="space-y-5">
          {/* my-car fields */}
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div className="space-y-1.5">
              <Label htmlFor="cc-name">نام خودرو</Label>
              <Input id="cc-name" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="cc-plate">پلاک (اختیاری)</Label>
              <Input id="cc-plate" value={licensePlate} onChange={(e) => setLicensePlate(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="cc-odo">کیلومتر فعلی</Label>
              <Input
                id="cc-odo"
                type="number"
                min={0}
                value={odometerKm}
                onChange={(e) => setOdometerKm(parseInt(e.target.value) || 0)}
              />
            </div>
          </div>

          {/* colors */}
          <div className="space-y-2">
            <Label>رنگ بدنه</Label>
            <ColorPicker colors={car.exterior_colors} selected={exteriorColor} onSelect={setExteriorColor} />
          </div>
          <div className="space-y-2">
            <Label>رنگ داخل کابین</Label>
            <ColorPicker colors={car.interior_colors} selected={interiorColor} onSelect={setInteriorColor} />
          </div>

          {/* all spec fields, pre-filled from the catalog */}
          <p className="text-xs text-muted-foreground">
            مشخصات از کاتالوگ پر شده‌اند — فقط مواردی که تغییر دهید به‌صورت شخصی‌سازی ذخیره می‌شوند.
          </p>
          {SPEC_FIELD_SECTIONS.map((sec) => (
            <section key={sec.title}>
              <h4 className="mb-2 border-b pb-1 text-sm font-bold text-primary">{sec.title}</h4>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                {sec.fields.map((f) => (
                  <div key={f.key} className="space-y-1">
                    <Label htmlFor={`cc-${f.key}`} className="text-xs text-muted-foreground">
                      {f.label}
                    </Label>
                    <Input
                      id={`cc-${f.key}`}
                      type={f.kind === "number" ? "number" : "text"}
                      step={f.kind === "number" ? "any" : undefined}
                      dir={f.kind === "number" ? "ltr" : undefined}
                      value={specs[f.key]}
                      onChange={(e) => setSpecs((p) => ({ ...p, [f.key]: e.target.value }))}
                    />
                  </div>
                ))}
              </div>
            </section>
          ))}

          <div className="flex gap-2 pt-2">
            <Button type="submit" className="flex-1" disabled={createCar.isPending}>
              ذخیره در خودروهای من
            </Button>
            <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
              انصراف
            </Button>
          </div>
        </form>
      </DialogContent>
    </Dialog>
  );
};
