import { useState, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { useNavigate } from "react-router-dom";
import { Plus, Pencil, Trash2, Car as CarIcon, Search, Star, X } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Header } from "@/components/Header";
import { useCars, useCreateCar, useUpdateCar, useDeleteCar } from "@/features/cars/hooks";
import type { Car, CarInput } from "@/features/cars/api";
import { useEVModels } from "@/features/ev-models/hooks";
import type { CatalogCar } from "@/features/catalog/api";
import { useSettings, useUpdateSettings } from "@/features/settings/hooks";
import { useCatalog } from "@/features/catalog/hooks";

const emptyForm: CarInput = {
  name: "",
  ev_model_id: "",
  license_plate: "",
  odometer_km: 0,
  catalog_car_id: null,
  spec_overrides: {},
};

export default function Cars() {
  const navigate = useNavigate();
  const { t, i18n } = useTranslation();
  const isRTL = i18n.language === "fa";
  const labels = {
    name: isRTL ? "نام خودرو" : "Car name",
    plate: isRTL ? "پلاک (اختیاری)" : "License plate (optional)",
    odometer: isRTL ? "کیلومتر" : "Odometer (km)",
  };

  const { data: cars = [] } = useCars();
  const { data: evModels = [] } = useEVModels();
  const { data: settings } = useSettings();
  const createCar = useCreateCar();
  const updateCar = useUpdateCar();
  const deleteCar = useDeleteCar();
  const updateSettings = useUpdateSettings();

  const [searchQuery, setSearchQuery] = useState("");
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [editingCar, setEditingCar] = useState<Car | null>(null);
  const [deleteCarId, setDeleteCarId] = useState<string | null>(null);
  const [formData, setFormData] = useState<CarInput>(emptyForm);

  // Lookup linked EV model details for display (Go cars store only ev_model_id).
  const modelById = useMemo(
    () => new Map(evModels.map((m) => [m.id, m] as const)),
    [evModels],
  );

  // Catalog lookup for catalog-linked cars (TASK-0034).
  const { data: catalogCars = [] } = useCatalog();
  const catalogById = useMemo(
    () => new Map(catalogCars.map((c) => [c.id, c] as const)),
    [catalogCars],
  );

  // TASK-0035: new cars link to the rich ev_catalog, not ev_models. The picker
  // searches the 23 cached catalog cars client-side (fa/en/brand).
  const filteredCatalog = useMemo<CatalogCar[]>(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return [];
    return catalogCars.filter(
      (c) =>
        c.name_fa.toLowerCase().includes(q) ||
        c.name_en.toLowerCase().includes(q) ||
        (c.brand ?? "").toLowerCase().includes(q),
    );
  }, [searchQuery, catalogCars]);

  const handleSelectCatalogCar = (catalogCar: CatalogCar) => {
    setFormData((prev) => ({
      ...prev,
      catalog_car_id: catalogCar.id,
      name: prev.name?.trim() ? prev.name : catalogCar.name_fa,
    }));
    setSearchQuery("");
  };

  const resetForm = () => {
    setFormData(emptyForm);
    setEditingCar(null);
    setIsDialogOpen(false);
    setSearchQuery("");
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const input: CarInput = {
      name: formData.name.trim(),
      ev_model_id: formData.ev_model_id || null,
      license_plate: formData.license_plate?.trim() || null,
      odometer_km: Number(formData.odometer_km) || 0,
      // PUT is full-replace — losing these would unlink the catalog car and
      // wipe the user's customizations (TASK-0034).
      catalog_car_id: formData.catalog_car_id || null,
      spec_overrides: formData.spec_overrides ?? {},
    };
    try {
      if (editingCar) {
        await updateCar.mutateAsync({ id: editingCar.id, input });
        toast.success(t("cars.carUpdated"));
      } else {
        await createCar.mutateAsync(input);
        toast.success(t("cars.carAdded"));
      }
      resetForm();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const handleEdit = (car: Car) => {
    setEditingCar(car);
    setFormData({
      name: car.name,
      ev_model_id: car.ev_model_id || "",
      license_plate: car.license_plate || "",
      odometer_km: car.odometer_km,
      catalog_car_id: car.catalog_car_id,
      spec_overrides: car.spec_overrides ?? {},
    });
    setIsDialogOpen(true);
  };

  const handleDelete = async () => {
    if (!deleteCarId) return;
    try {
      await deleteCar.mutateAsync(deleteCarId);
      toast.success(t("cars.carDeleted"));
    } catch (err) {
      toast.error((err as Error).message);
    }
    setDeleteCarId(null);
  };

  // Setting the default car is a settings PUT (full replace) — preserve the rates.
  const setDefaultCar = (carId: string) => {
    updateSettings.mutate(
      {
        default_car_id: carId,
        peak_rate: settings?.peak_rate ?? 0,
        mid_rate: settings?.mid_rate ?? 0,
        offpeak_rate: settings?.offpeak_rate ?? 0,
      },
      {
        onSuccess: () =>
          toast.success(isRTL ? "ماشین پیش‌فرض تنظیم شد" : "Default car set successfully"),
        onError: (err) => toast.error((err as Error).message),
      },
    );
  };

  return (
    <div className="min-h-screen bg-background">
      <Header />

      <div className="container mx-auto px-3 sm:px-4 py-4 sm:py-8">
        <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4 mb-6">
          <h2 className="text-2xl sm:text-3xl font-bold text-foreground">{t("cars.myCars")}</h2>
          <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
            <DialogTrigger asChild>
              <Button onClick={() => resetForm()} className="w-full sm:w-auto">
                <Plus className="w-4 h-4" />
                {t("cars.addCar")}
              </Button>
            </DialogTrigger>
            <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto mx-4">
              <DialogHeader>
                <DialogTitle>{editingCar ? t("cars.edit") : t("cars.addCar")}</DialogTitle>
              </DialogHeader>
              <form onSubmit={handleSubmit} className="space-y-4">
                {!editingCar && (
                  <div className="space-y-2">
                    <Label>{isRTL ? "جستجو در کاتالوگ" : "Search the catalog"}</Label>
                    <div className="relative">
                      <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder={isRTL ? "نام یا برند خودرو…" : "Car name or brand…"}
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="pl-10"
                      />
                    </div>
                    {filteredCatalog.length > 0 && (
                      <Card className="max-h-60 overflow-y-auto">
                        <CardContent className="p-0">
                          {filteredCatalog.map((catalogCar) => (
                            <button
                              key={catalogCar.id}
                              type="button"
                              onClick={() => handleSelectCatalogCar(catalogCar)}
                              className="w-full text-right p-3 hover:bg-accent transition-colors border-b border-border last:border-0"
                            >
                              <div className="font-semibold">
                                {catalogCar.name_fa}{" "}
                                <span className="text-xs font-normal text-muted-foreground" dir="ltr">
                                  {catalogCar.name_en}
                                </span>
                              </div>
                              <div className="text-sm text-muted-foreground">
                                {catalogCar.brand ?? "—"} • {catalogCar.battery_capacity_kwh ?? "?"} kWh
                                {catalogCar.cell_type ? ` ${catalogCar.cell_type}` : ""} •{" "}
                                {catalogCar.range_km ?? "?"} km
                              </div>
                            </button>
                          ))}
                        </CardContent>
                      </Card>
                    )}
                    <p className="text-xs text-muted-foreground">
                      {isRTL ? "برای شخصی‌سازی کامل مشخصات، از " : "For full customization, add from the "}
                      <button type="button" className="text-primary underline" onClick={() => navigate("/catalog")}>
                        {isRTL ? "کاتالوگ" : "catalog"}
                      </button>
                      {isRTL ? " اضافه کنید." : "."}
                    </p>
                  </div>
                )}

                <div className="space-y-2">
                  <Label htmlFor="name">{labels.name}</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    required
                  />
                </div>

                {formData.catalog_car_id && (
                  <p className="flex items-center gap-2 text-sm text-muted-foreground">
                    <span>
                      {isRTL ? "از کاتالوگ:" : "From catalog:"}{" "}
                      {catalogById.get(formData.catalog_car_id)?.name_fa ?? formData.catalog_car_id}
                    </span>
                    <button
                      type="button"
                      title={isRTL ? "حذف پیوند کاتالوگ" : "Unlink catalog car"}
                      onClick={() => setFormData((p) => ({ ...p, catalog_car_id: null }))}
                      className="text-muted-foreground hover:text-destructive"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </p>
                )}
                {/* legacy ev_model link — display only; new selections are catalog-only (TASK-0035) */}
                {formData.ev_model_id && (
                  <p className="text-sm text-muted-foreground">
                    {isRTL ? "مدل:" : "Model:"}{" "}
                    {modelById.get(formData.ev_model_id)?.name_en ?? formData.ev_model_id}
                  </p>
                )}

                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label htmlFor="license_plate">{labels.plate}</Label>
                    <Input
                      id="license_plate"
                      value={formData.license_plate ?? ""}
                      onChange={(e) => setFormData({ ...formData, license_plate: e.target.value })}
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="odometer_km">{labels.odometer}</Label>
                    <Input
                      id="odometer_km"
                      type="number"
                      min={0}
                      value={formData.odometer_km ?? 0}
                      onChange={(e) =>
                        setFormData({ ...formData, odometer_km: parseInt(e.target.value) || 0 })
                      }
                    />
                  </div>
                </div>

                <div className="flex gap-2 pt-4">
                  <Button type="submit" className="flex-1" disabled={createCar.isPending || updateCar.isPending}>
                    {t("common.save")}
                  </Button>
                  <Button type="button" variant="outline" onClick={resetForm}>
                    {t("common.cancel")}
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6">
          {cars.map((car) => {
            const model = car.ev_model_id ? modelById.get(car.ev_model_id) : undefined;
            const catalogCar = car.catalog_car_id ? catalogById.get(car.catalog_car_id) : undefined;
            // effective battery: user override → catalog → ev_model (mirrors the API's SOH chain)
            const overrideKwh = car.spec_overrides?.battery_capacity_kwh;
            const batteryKwh =
              typeof overrideKwh === "number"
                ? overrideKwh
                : catalogCar?.battery_capacity_kwh ?? model?.battery_capacity_kwh ?? null;
            return (
              <Card
                key={car.id}
                className={cn(
                  "hover:shadow-lg transition-all",
                  settings?.default_car_id === car.id &&
                    "border-2 border-yellow-500 bg-yellow-50/50 dark:bg-yellow-950/20",
                )}
              >
                <CardHeader className="p-4 sm:p-6">
                  <div className="flex items-start justify-between gap-2">
                    <div className="flex items-center gap-2 sm:gap-3 min-w-0">
                      <div className="p-2 sm:p-3 bg-primary/10 rounded-lg flex-shrink-0">
                        <CarIcon className="w-5 h-5 sm:w-6 sm:h-6 text-primary" />
                      </div>
                      <div className="min-w-0">
                        <CardTitle className="text-base sm:text-lg truncate">{car.name}</CardTitle>
                        <p className="text-xs sm:text-sm text-muted-foreground truncate">
                          {catalogCar ? (
                            <>
                              {isRTL ? catalogCar.name_fa : catalogCar.name_en}
                              <span className="mr-1 rounded bg-primary/10 px-1.5 py-0.5 text-[10px] text-primary">
                                کاتالوگ
                              </span>
                            </>
                          ) : model ? (
                            isRTL ? model.name_fa : model.name_en
                          ) : (
                            "—"
                          )}
                        </p>
                      </div>
                    </div>
                    <div className="flex gap-1 flex-shrink-0">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => setDefaultCar(car.id)}
                        className={cn("h-8 w-8", settings?.default_car_id === car.id && "text-yellow-500")}
                        title={isRTL ? "تنظیم به عنوان پیش‌فرض" : "Set as default"}
                      >
                        <Star className={`w-3.5 h-3.5 sm:w-4 sm:h-4 ${settings?.default_car_id === car.id ? "fill-current" : ""}`} />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => handleEdit(car)} className="h-8 w-8">
                        <Pencil className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
                      </Button>
                      <Button variant="ghost" size="icon" onClick={() => setDeleteCarId(car.id)} className="h-8 w-8">
                        <Trash2 className="w-3.5 h-3.5 sm:w-4 sm:h-4" />
                      </Button>
                    </div>
                  </div>
                </CardHeader>
                <CardContent className="p-4 sm:p-6 pt-0">
                  <div className="space-y-2">
                    <div className="flex justify-between text-xs sm:text-sm">
                      <span className="text-muted-foreground">{labels.odometer}:</span>
                      <span className="font-medium">{car.odometer_km.toLocaleString()} km</span>
                    </div>
                    {batteryKwh != null && (
                      <div className="flex justify-between text-xs sm:text-sm">
                        <span className="text-muted-foreground">{t("cars.batteryCapacity")}:</span>
                        <span className="font-medium">{batteryKwh} kWh</span>
                      </div>
                    )}
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>

        {cars.length === 0 && (
          <Card className="text-center py-12">
            <CardContent>
              <CarIcon className="w-16 h-16 mx-auto mb-4 text-muted-foreground" />
              <p className="text-muted-foreground">{t("cars.addCar")}</p>
            </CardContent>
          </Card>
        )}
      </div>

      <AlertDialog open={deleteCarId !== null} onOpenChange={() => setDeleteCarId(null)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("cars.delete")}</AlertDialogTitle>
            <AlertDialogDescription>{t("cars.confirmDelete")}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>{t("common.delete")}</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
