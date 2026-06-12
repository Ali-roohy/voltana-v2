import { useNavigate } from "react-router-dom";
import { useLanguage } from "@/contexts/LanguageContext";
import { Settings, Car, ChevronDown, Shield, Users, BookOpen } from "lucide-react";
import { Button } from "@/components/ui/button";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/components/ui/dropdown-menu";
import { toast } from "sonner";
import { useQueryClient } from "@tanstack/react-query";
import { cn } from "@/lib/utils";
import { useCars } from "@/features/cars/hooks";
import { useSettings, useUpdateSettings } from "@/features/settings/hooks";
import { useMe } from "@/features/auth/hooks";

export const Header = () => {
  const { language } = useLanguage();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: cars = [] } = useCars();
  const { data: settings } = useSettings();
  const { data: me } = useMe();
  const updateSettings = useUpdateSettings();

  const defaultCarId = settings?.default_car_id ?? null;
  const defaultCar = cars.find((car) => car.id === defaultCarId);

  const handleCarChange = (carId: string) => {
    // Setting the default car is a settings PUT (full replace) — preserve rates.
    updateSettings.mutate(
      {
        default_car_id: carId,
        peak_rate: settings?.peak_rate ?? 0,
        mid_rate: settings?.mid_rate ?? 0,
        offpeak_rate: settings?.offpeak_rate ?? 0,
      },
      {
        onSuccess: () => {
          // bug #2: refresh dependent data via the query cache instead of window.location.reload()
          queryClient.invalidateQueries({ queryKey: ["charging-sessions"] });
          queryClient.invalidateQueries({ queryKey: ["dashboard"] });
          toast.success(language === "fa" ? "ماشین پیش‌فرض تغییر کرد" : "Default car changed");
        },
        onError: () =>
          toast.error(language === "fa" ? "خطا در تغییر ماشین پیش‌فرض" : "Error changing default car"),
      },
    );
  };

  return (
    <header className="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div className="container flex h-14 sm:h-16 items-center justify-between px-2 sm:px-4">
        <div className="gap-2 sm:gap-4 items-center justify-start flex flex-row">
          <h1 onClick={() => navigate("/")} className="text-base sm:text-xl font-bold cursor-pointer hover:text-primary transition-colors text-center">
            {language === "fa" ? "مدیریت شارژ" : "Charge Manager"}
          </h1>

          {cars.length > 0 && (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="outline" size="sm" disabled={updateSettings.isPending} className={cn("gap-1 sm:gap-2 bg-background h-8 sm:h-9 text-xs sm:text-sm px-2 sm:px-3", language === "fa" ? "flex-row-reverse" : "")}>
                  <Car className="h-3 w-3 sm:h-4 sm:w-4" />
                  {defaultCar ? (
                    <span className="max-w-[80px] sm:max-w-[150px] truncate">{defaultCar.name}</span>
                  ) : (
                    <span className="hidden sm:inline">{language === "fa" ? "انتخاب ماشین" : "Select Car"}</span>
                  )}
                  <ChevronDown className="h-3 w-3 sm:h-4 sm:w-4 opacity-50" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align={language === "fa" ? "end" : "start"} className="w-48 sm:w-56 bg-background">
                {cars.map((car) => (
                  <DropdownMenuItem key={car.id} onClick={() => handleCarChange(car.id)} className={cn("cursor-pointer", car.id === defaultCarId && "bg-primary/10 font-semibold")}>
                    <div className="flex items-center gap-2 w-full">
                      <Car className="h-3 w-3 sm:h-4 sm:w-4" />
                      <span className="truncate text-justify font-thin text-xs font-sans">{car.name}</span>
                      {car.id === defaultCarId && <span className="mr-auto text-primary">✓</span>}
                    </div>
                  </DropdownMenuItem>
                ))}
              </DropdownMenuContent>
            </DropdownMenu>
          )}
        </div>

        <div className="flex items-center gap-1 sm:gap-2">
          {me?.is_admin && (
            <>
              <Button variant="ghost" size="icon" onClick={() => navigate("/admin/users")} title={language === "fa" ? "مدیریت کاربران" : "Manage Users"} className="h-8 w-8 sm:h-10 sm:w-10">
                <Users className="h-4 w-4 sm:h-5 sm:w-5" />
              </Button>
              <Button variant="ghost" size="icon" onClick={() => navigate("/admin/stations")} title={language === "fa" ? "مدیریت ایستگاه‌ها" : "Manage Stations"} className="h-8 w-8 sm:h-10 sm:w-10">
                <Shield className="h-4 w-4 sm:h-5 sm:w-5" />
              </Button>
            </>
          )}
          <Button variant="ghost" size="icon" onClick={() => navigate("/cars")} title={language === "fa" ? "خودروهای من" : "My Cars"} className="h-8 w-8 sm:h-10 sm:w-10">
            <Car className="h-4 w-4 sm:h-5 sm:w-5" />
          </Button>
          <Button variant="ghost" size="icon" onClick={() => navigate("/catalog")} title={language === "fa" ? "کاتالوگ خودروها" : "EV Catalog"} className="h-8 w-8 sm:h-10 sm:w-10">
            <BookOpen className="h-4 w-4 sm:h-5 sm:w-5" />
          </Button>
          <Button variant="ghost" size="icon" onClick={() => navigate("/settings")} title={language === "fa" ? "تنظیمات" : "Settings"} className="h-8 w-8 sm:h-10 sm:w-10">
            <Settings className="h-4 w-4 sm:h-5 sm:w-5" />
          </Button>
        </div>
      </div>
    </header>
  );
};
