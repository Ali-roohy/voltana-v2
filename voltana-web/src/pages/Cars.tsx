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
import { Plus, Pencil, Trash2, Car as CarIcon, Search, Star } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Header } from "@/components/Header";
import { useCars, useCreateCar, useUpdateCar, useDeleteCar } from "@/features/cars/hooks";
import type { Car, CarInput } from "@/features/cars/api";
import { useEVModels } from "@/features/ev-models/hooks";
import type { EVModel } from "@/features/ev-models/api";
import { useSettings, useUpdateSettings } from "@/features/settings/hooks";

const emptyForm: CarInput = { name: "", ev_model_id: "", license_plate: "", odometer_km: 0 };

export default function Cars() {
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

  const filteredModels = useMemo<EVModel[]>(() => {
    const q = searchQuery.trim().toLowerCase();
    if (!q) return [];
    return evModels.filter(
      (m) => m.name_fa.toLowerCase().includes(q) || m.name_en.toLowerCase().includes(q),
    );
  }, [searchQuery, evModels]);

  const handleSelectModel = (model: EVModel) => {
    setFormData((prev) => ({
      ...prev,
      ev_model_id: model.id,
      name: prev.name?.trim() ? prev.name : model.name_en,
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
                    <Label>{t("cars.searchModel")}</Label>
                    <div className="relative">
                      <Search className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
                      <Input
                        placeholder={t("cars.searchModel")}
                        value={searchQuery}
                        onChange={(e) => setSearchQuery(e.target.value)}
                        className="pl-10"
                      />
                    </div>
                    {filteredModels.length > 0 && (
                      <Card className="max-h-60 overflow-y-auto">
                        <CardContent className="p-0">
                          {filteredModels.map((model) => (
                            <button
                              key={model.id}
                              type="button"
                              onClick={() => handleSelectModel(model)}
                              className="w-full text-right p-3 hover:bg-accent transition-colors border-b border-border last:border-0"
                            >
                              <div className="font-semibold">
                                {isRTL ? model.name_fa : model.name_en}
                              </div>
                              <div className="text-sm text-muted-foreground">
                                {model.brand ?? "—"} • {model.battery_capacity_kwh ?? "?"} kWh •{" "}
                                {model.range_km ?? "?"} km
                              </div>
                            </button>
                          ))}
                        </CardContent>
                      </Card>
                    )}
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
                          {model ? (isRTL ? model.name_fa : model.name_en) : "—"}
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
                    {model?.battery_capacity_kwh != null && (
                      <div className="flex justify-between text-xs sm:text-sm">
                        <span className="text-muted-foreground">{t("cars.batteryCapacity")}:</span>
                        <span className="font-medium">{model.battery_capacity_kwh} kWh</span>
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
