import { useMemo, useState } from "react";
import { LayoutGrid, List, Scale } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Skeleton } from "@/components/ui/skeleton";
import { Header } from "@/components/Header";
import type { CatalogCar } from "./api";
import { useCatalog } from "./hooks";
import { type BatteryBucket, type SortKey, inBatteryBucket, sortCars } from "./format";
import { CarDetail } from "./components/CarDetail";
import { CarComparison } from "./components/CarComparison";
import { CatalogGrid } from "./components/CatalogGrid";

const VIEW_KEY = "voltana:catalog-view";

const unique = (vals: (string | null)[]) =>
  [...new Set(vals.filter((v): v is string => !!v))].sort();

export default function CatalogPage() {
  const { data: cars = [], isLoading } = useCatalog();

  const [mode, setMode] = useState<"grid" | "list">(
    () => (localStorage.getItem(VIEW_KEY) === "list" ? "list" : "grid"),
  );
  const [brand, setBrand] = useState("all");
  const [bodyType, setBodyType] = useState("all");
  const [segment, setSegment] = useState("all");
  const [battery, setBattery] = useState<BatteryBucket>("all");
  const [sort, setSort] = useState<SortKey>("range");
  const [detailCar, setDetailCar] = useState<CatalogCar | null>(null);
  const [compareIds, setCompareIds] = useState<string[]>([]);
  const [compareOpen, setCompareOpen] = useState(false);

  const switchMode = (m: "grid" | "list") => {
    setMode(m);
    localStorage.setItem(VIEW_KEY, m);
  };

  const toggleCompare = (id: string) =>
    setCompareIds((prev) =>
      prev.includes(id) ? prev.filter((x) => x !== id) : prev.length >= 3 ? prev : [...prev, id],
    );

  const brands = useMemo(() => unique(cars.map((c) => c.brand)), [cars]);
  const bodyTypes = useMemo(() => unique(cars.map((c) => c.body_type)), [cars]);
  const segments = useMemo(() => unique(cars.map((c) => c.segment)), [cars]);

  const visible = useMemo(() => {
    const filtered = cars.filter(
      (c) =>
        (brand === "all" || c.brand === brand) &&
        (bodyType === "all" || c.body_type === bodyType) &&
        (segment === "all" || c.segment === segment) &&
        inBatteryBucket(c, battery),
    );
    return sortCars(filtered, sort);
  }, [cars, brand, bodyType, segment, battery, sort]);

  const compareCars = cars.filter((c) => compareIds.includes(c.id));

  return (
    <div className="min-h-screen app-page-bg">
      <Header />
      <main className="container space-y-4 px-2 py-4 sm:px-4 sm:py-6">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h2 className="text-xl font-bold sm:text-2xl">کاتالوگ خودروهای برقی</h2>
            <p className="text-sm text-muted-foreground">
              {cars.length.toLocaleString("fa-IR")} خودرو با مشخصات کامل بازار ایران
            </p>
          </div>
          <div className="flex items-center gap-1 rounded-lg border p-1">
            <Button
              variant={mode === "grid" ? "secondary" : "ghost"}
              size="sm"
              className="h-8 px-2"
              onClick={() => switchMode("grid")}
              title="نمایش کارتی"
            >
              <LayoutGrid className="h-4 w-4" />
            </Button>
            <Button
              variant={mode === "list" ? "secondary" : "ghost"}
              size="sm"
              className="h-8 px-2"
              onClick={() => switchMode("list")}
              title="نمایش جدولی"
            >
              <List className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* filters + sort */}
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
          <Select value={brand} onValueChange={setBrand}>
            <SelectTrigger><SelectValue placeholder="برند" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">همه برندها</SelectItem>
              {brands.map((b) => <SelectItem key={b} value={b}>{b}</SelectItem>)}
            </SelectContent>
          </Select>
          <Select value={bodyType} onValueChange={setBodyType}>
            <SelectTrigger><SelectValue placeholder="نوع بدنه" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">همه بدنه‌ها</SelectItem>
              {bodyTypes.map((b) => <SelectItem key={b} value={b}>{b}</SelectItem>)}
            </SelectContent>
          </Select>
          <Select value={segment} onValueChange={setSegment}>
            <SelectTrigger><SelectValue placeholder="سطح بازار" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">همه سطح‌ها</SelectItem>
              {segments.map((s) => <SelectItem key={s} value={s}>{s}</SelectItem>)}
            </SelectContent>
          </Select>
          <Select value={battery} onValueChange={(v) => setBattery(v as BatteryBucket)}>
            <SelectTrigger><SelectValue placeholder="باتری" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="all">هر ظرفیت باتری</SelectItem>
              <SelectItem value="small">کمتر از ۶۰ kWh</SelectItem>
              <SelectItem value="medium">۶۰ تا ۹۰ kWh</SelectItem>
              <SelectItem value="large">بیشتر از ۹۰ kWh</SelectItem>
            </SelectContent>
          </Select>
          <Select value={sort} onValueChange={(v) => setSort(v as SortKey)}>
            <SelectTrigger className="col-span-2 sm:col-span-1"><SelectValue placeholder="مرتب‌سازی" /></SelectTrigger>
            <SelectContent>
              <SelectItem value="range">بیشترین برد</SelectItem>
              <SelectItem value="acceleration">سریع‌ترین شتاب</SelectItem>
              <SelectItem value="tier">بالاترین رده بازار</SelectItem>
            </SelectContent>
          </Select>
        </div>

        {isLoading ? (
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <Skeleton key={i} className="h-64 rounded-lg" />
            ))}
          </div>
        ) : (
          <CatalogGrid cars={visible} mode={mode} onDetails={setDetailCar} />
        )}
      </main>

      {/* floating compare bar — visible once a comparison set exists */}
      {compareIds.length >= 2 && (
        <Button
          className="fixed bottom-20 left-4 z-40 shadow-lg md:bottom-6"
          onClick={() => setCompareOpen(true)}
        >
          <Scale className="ml-2 h-4 w-4" />
          مقایسه ({compareIds.length.toLocaleString("fa-IR")})
        </Button>
      )}

      <Dialog open={compareOpen} onOpenChange={setCompareOpen}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>مقایسه خودروها</DialogTitle>
          </DialogHeader>
          <CarComparison cars={compareCars} />
        </DialogContent>
      </Dialog>

      <CarDetail
        key={detailCar?.id ?? "closed"}
        car={detailCar}
        allCars={cars}
        compareIds={compareIds}
        onToggleCompare={toggleCompare}
        onClose={() => setDetailCar(null)}
      />
    </div>
  );
}
