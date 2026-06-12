import { useMemo, useState } from "react";
import { useQueries } from "@tanstack/react-query";
import { toast } from "sonner";
import { Plus, Pencil, Trash2, Loader2, MapPin } from "lucide-react";

import { Header } from "@/components/Header";
import { StationMapPicker } from "@/components/StationMapPicker";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table, TableBody, TableCell, TableHead, TableHeader, TableRow,
} from "@/components/ui/table";
import {
  Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle,
} from "@/components/ui/dialog";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent,
  AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { ApiError } from "@/lib/api";
import { getStation, type Station, type StationInput } from "@/features/stations/api";
import {
  useStations, useCreateStation, useUpdateStation, useDeleteStation,
} from "@/features/stations/hooks";

// ── form ────────────────────────────────────────────────────────────────────

interface FormState {
  name: string;
  operator: string;
  address: string;
  latitude: string;
  longitude: string;
  connector_types: string;
  power_kw: string;
}

const EMPTY_FORM: FormState = {
  name: "", operator: "", address: "", latitude: "", longitude: "",
  connector_types: "", power_kw: "",
};

function fromStation(s: Station): FormState {
  return {
    name: s.name,
    operator: s.operator ?? "",
    address: s.address ?? "",
    latitude: String(s.latitude),
    longitude: String(s.longitude),
    connector_types: s.connector_types ?? "",
    power_kw: s.power_kw != null ? String(s.power_kw) : "",
  };
}

const num = (v: string): number | null => {
  const t = v.trim();
  if (t === "") return null;
  const n = Number(t);
  return Number.isFinite(n) ? n : null;
};

// Mirrors the server validators so users get errors before the round-trip.
function validate(f: FormState): { input?: StationInput; error?: string } {
  if (!f.name.trim()) return { error: "نام ایستگاه الزامی است" };
  const lat = num(f.latitude);
  const lng = num(f.longitude);
  if (lat === null || lat < -90 || lat > 90) return { error: "عرض جغرافیایی باید بین ۹۰- و ۹۰ باشد" };
  if (lng === null || lng < -180 || lng > 180) return { error: "طول جغرافیایی باید بین ۱۸۰- و ۱۸۰ باشد" };
  let power: number | null = null;
  if (f.power_kw.trim() !== "") {
    const p = Number(f.power_kw);
    if (!Number.isInteger(p) || p < 1) return { error: "توان باید عدد صحیح مثبت باشد" };
    power = p;
  }
  return {
    input: {
      name: f.name,
      latitude: lat,
      longitude: lng,
      address: f.address,
      operator: f.operator,
      connector_types: f.connector_types,
      power_kw: power,
    },
  };
}

interface StationFormDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  editing: Station | null;
}

function StationFormDialog({ open, onOpenChange, editing }: StationFormDialogProps) {
  const [form, setForm] = useState<FormState>(EMPTY_FORM);
  // Re-seed the form whenever the dialog opens (add → blank; edit → prefilled).
  const seedKey = `${open}:${editing?.id ?? "new"}`;
  const [lastSeed, setLastSeed] = useState("");
  if (open && seedKey !== lastSeed) {
    setForm(editing ? fromStation(editing) : EMPTY_FORM);
    setLastSeed(seedKey);
  }

  const create = useCreateStation();
  const update = useUpdateStation();
  const pending = create.isPending || update.isPending;

  const set = (k: keyof FormState) => (e: React.ChangeEvent<HTMLInputElement>) =>
    setForm((f) => ({ ...f, [k]: e.target.value }));

  const handleSubmit = () => {
    const { input, error } = validate(form);
    if (error || !input) {
      toast.error(error);
      return;
    }
    const onSuccess = () => {
      toast.success(editing ? "ایستگاه به‌روزرسانی شد" : "ایستگاه اضافه شد");
      onOpenChange(false);
    };
    const onError = (err: unknown) => {
      const msg =
        err instanceof ApiError
          ? err.status === 403
            ? "فقط مدیران می‌توانند ایستگاه‌ها را تغییر دهند"
            : err.message
          : "خطای ناشناخته";
      toast.error(msg);
    };
    if (editing) update.mutate({ id: editing.id, input }, { onSuccess, onError });
    else create.mutate(input, { onSuccess, onError });
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[90vh] overflow-y-auto" dir="rtl">
        <DialogHeader>
          <DialogTitle>{editing ? "ویرایش ایستگاه" : "افزودن ایستگاه"}</DialogTitle>
          <DialogDescription>
            روی نقشه کلیک کنید یا مارکر را بکشید تا مختصات تنظیم شود
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4">
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="name">نام *</Label>
              <Input id="name" value={form.name} onChange={set("name")} />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="operator">اپراتور</Label>
              <Input id="operator" value={form.operator} onChange={set("operator")} />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="address">آدرس</Label>
            <Input id="address" value={form.address} onChange={set("address")} />
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="connector_types">انواع کانکتور (با کاما جدا کنید)</Label>
              <Input id="connector_types" value={form.connector_types} onChange={set("connector_types")} placeholder="CCS2, Type2" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="power_kw">توان (kW)</Label>
              <Input id="power_kw" type="number" min={1} value={form.power_kw} onChange={set("power_kw")} />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label htmlFor="latitude">عرض جغرافیایی *</Label>
              <Input id="latitude" value={form.latitude} onChange={set("latitude")} inputMode="decimal" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="longitude">طول جغرافیایی *</Label>
              <Input id="longitude" value={form.longitude} onChange={set("longitude")} inputMode="decimal" />
            </div>
          </div>

          <StationMapPicker
            latitude={num(form.latitude)}
            longitude={num(form.longitude)}
            onChange={(lat, lng) =>
              setForm((f) => ({ ...f, latitude: String(lat), longitude: String(lng) }))
            }
          />
        </div>

        <DialogFooter className="gap-2 sm:gap-0">
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>
            انصراف
          </Button>
          <Button onClick={handleSubmit} disabled={pending}>
            {pending && <Loader2 className="w-4 h-4 animate-spin ml-2" />}
            {editing ? "ذخیره" : "افزودن"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

// ── page ──────────────────────────────────────────────────────────────────────

const AdminStations = () => {
  const { data: markers, isLoading, isError } = useStations();
  const del = useDeleteStation();

  const [formOpen, setFormOpen] = useState(false);
  const [editing, setEditing] = useState<Station | null>(null);
  const [toDelete, setToDelete] = useState<{ id: string; name: string } | null>(null);

  // The list endpoint returns markers only (no operator/address); hydrate full
  // detail per row so the table can show every column. Bounded reference data,
  // and each result is cached under ["stations", id] — reused when editing.
  const details = useQueries({
    queries: (markers ?? []).map((m) => ({
      queryKey: ["stations", m.id],
      queryFn: () => getStation(m.id),
      staleTime: 60_000,
    })),
  });
  const byId = useMemo(() => {
    const map = new Map<string, Station>();
    for (const q of details) if (q.data) map.set(q.data.id, q.data);
    return map;
  }, [details]);

  const openAdd = () => { setEditing(null); setFormOpen(true); };
  const openEdit = (id: string) => { setEditing(byId.get(id) ?? null); setFormOpen(true); };

  const confirmDelete = () => {
    if (!toDelete) return;
    del.mutate(toDelete.id, {
      onSuccess: () => { toast.success("ایستگاه حذف شد"); setToDelete(null); },
      onError: (err) => {
        const msg = err instanceof ApiError
          ? (err.status === 403 ? "فقط مدیران می‌توانند ایستگاه‌ها را حذف کنند" : err.message)
          : "خطا در حذف ایستگاه";
        toast.error(msg);
        setToDelete(null);
      },
    });
  };

  return (
    <div className="min-h-screen app-page-bg" dir="rtl">
      <Header />
      <main className="container mx-auto p-3 sm:p-4 space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-2xl sm:text-3xl font-bold flex items-center gap-2">
            <MapPin className="w-6 h-6" /> مدیریت ایستگاه‌های شارژ
          </h1>
          <Button onClick={openAdd}>
            <Plus className="w-4 h-4 ml-2" /> افزودن ایستگاه
          </Button>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>ایستگاه‌ها</CardTitle>
          </CardHeader>
          <CardContent>
            {isLoading && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Loader2 className="w-4 h-4 animate-spin" /> در حال بارگذاری…
              </div>
            )}
            {isError && <p className="text-sm text-destructive">خطا در بارگذاری ایستگاه‌ها</p>}
            {!isLoading && !isError && markers?.length === 0 && (
              <p className="text-sm text-muted-foreground">ایستگاهی ثبت نشده است</p>
            )}

            {!isLoading && !isError && (markers?.length ?? 0) > 0 && (
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead className="text-right">نام</TableHead>
                      <TableHead className="text-right">اپراتور</TableHead>
                      <TableHead className="text-right">آدرس</TableHead>
                      <TableHead className="text-right">مختصات</TableHead>
                      <TableHead className="text-right">کانکتور</TableHead>
                      <TableHead className="text-right">توان</TableHead>
                      <TableHead className="text-right">عملیات</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {markers!.map((m) => {
                      const d = byId.get(m.id);
                      return (
                        <TableRow key={m.id}>
                          <TableCell className="font-medium">{m.name}</TableCell>
                          <TableCell>{d?.operator ?? "—"}</TableCell>
                          <TableCell className="max-w-[200px] truncate">{d?.address ?? "—"}</TableCell>
                          <TableCell className="tabular-nums text-xs">
                            {m.latitude.toFixed(5)}, {m.longitude.toFixed(5)}
                          </TableCell>
                          <TableCell>
                            <div className="flex flex-wrap gap-1">
                              {m.connector_types
                                ? m.connector_types.split(",").map((ct) => (
                                    <Badge key={ct} variant="outline">{ct.trim()}</Badge>
                                  ))
                                : "—"}
                            </div>
                          </TableCell>
                          <TableCell>{m.power_kw != null ? `${m.power_kw} kW` : "—"}</TableCell>
                          <TableCell>
                            <div className="flex gap-1">
                              <Button
                                variant="ghost" size="icon"
                                onClick={() => openEdit(m.id)}
                                disabled={!d}
                                title="ویرایش"
                              >
                                <Pencil className="w-4 h-4" />
                              </Button>
                              <Button
                                variant="ghost" size="icon"
                                onClick={() => setToDelete({ id: m.id, name: m.name })}
                                title="حذف"
                              >
                                <Trash2 className="w-4 h-4 text-destructive" />
                              </Button>
                            </div>
                          </TableCell>
                        </TableRow>
                      );
                    })}
                  </TableBody>
                </Table>
              </div>
            )}
          </CardContent>
        </Card>
      </main>

      <StationFormDialog open={formOpen} onOpenChange={setFormOpen} editing={editing} />

      <AlertDialog open={!!toDelete} onOpenChange={(o) => !o && setToDelete(null)}>
        <AlertDialogContent dir="rtl">
          <AlertDialogHeader>
            <AlertDialogTitle>حذف ایستگاه</AlertDialogTitle>
            <AlertDialogDescription>
              آیا از حذف «{toDelete?.name}» مطمئن هستید؟ این عملیات قابل بازگشت نیست.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter className="gap-2 sm:gap-0">
            <AlertDialogCancel disabled={del.isPending}>انصراف</AlertDialogCancel>
            <AlertDialogAction
              onClick={(e) => { e.preventDefault(); confirmDelete(); }}
              disabled={del.isPending}
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
            >
              {del.isPending && <Loader2 className="w-4 h-4 animate-spin ml-2" />}
              حذف
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
};

export default AdminStations;
