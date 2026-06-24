import { useState, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useLanguage } from "@/contexts/LanguageContext";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
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
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip";
import { Plus, Edit, Trash2, CalendarIcon, Zap, Clock, Pencil, Check, X, ChevronDown, FilterX, AlertTriangle, AlertCircle } from "lucide-react";
import { format } from "date-fns";
import { format as formatJalali } from "date-fns-jalali";
import { JalaliDatePicker } from "@/components/JalaliDatePicker";
import { TOUBreakdown } from "@/components/TOUBreakdown";
import { cn, formatNumber } from "@/lib/utils";
import { calcCost, ratesFromSettings, ratesForSession, formatCost } from "@/lib/cost";
import { SOCAnalysis } from "@/components/SOCAnalysis";
import { Header } from "@/components/Header";
import { useChargingSessions, useCreateSession, useUpdateSession, useDeleteSession } from "@/features/charging/hooks";
import type { ChargingSession, ChargingInput, ChargingListFilter } from "@/features/charging/api";
import { useCars } from "@/features/cars/hooks";
import { useCatalog } from "@/features/catalog/hooks";
import { useSettings } from "@/features/settings/hooks";
import {
  resolveUsableCapacity,
  carAverageConsumption,
  predictStartSoc,
  predictEndSoc,
  applyRegen,
} from "@/features/charging/consumption";
import { computeEntryFlags, hasBlockingError, severityStyle, type EntryFlag } from "@/features/charging/warnings";

interface FormData {
  car_id: string;
  date: Date;
  duration_minutes: string;
  energy_peak_kwh: string;
  energy_mid_kwh: string;
  energy_offpeak_kwh: string;
  location: string;
  charge_power_kw: string;
  start_soc: string;
  end_soc: string;
  odometer_km: string;
}

const emptyForm = (carId = ""): FormData => ({
  car_id: carId,
  date: new Date(),
  duration_minutes: "",
  energy_peak_kwh: "",
  energy_mid_kwh: "",
  energy_offpeak_kwh: "",
  location: "",
  charge_power_kw: "",
  start_soc: "",
  end_soc: "",
  odometer_km: "",
});

const Charging = () => {
  const { t } = useTranslation();
  const { language } = useLanguage();
  const navigate = useNavigate();

  const { data: cars = [] } = useCars();
  const { data: catalog = [] } = useCatalog();
  const { data: settings } = useSettings();
  const createSession = useCreateSession();
  const updateSession = useUpdateSession();
  const deleteSession = useDeleteSession();

  // ── History filters (applied server-side via the list query) ──
  const [selectedCarFilter, setSelectedCarFilter] = useState<string>("all");
  const [fromDate, setFromDate] = useState<Date | undefined>(undefined);
  const [toDate, setToDate] = useState<Date | undefined>(undefined);
  // Sessions are expanded by default (BUG-2); we track the set the user has
  // explicitly collapsed instead of a single expanded id. Empty set = all open.
  const [collapsedIds, setCollapsedIds] = useState<Set<string>>(new Set());
  const toggleCollapse = (id: string) =>
    setCollapsedIds((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });

  // from > to is a user error: show a message and don't query the bad range.
  const invalidRange = !!fromDate && !!toDate && fromDate > toDate;
  const filterActive = selectedCarFilter !== "all" || !!fromDate || !!toDate;
  const filter: ChargingListFilter | undefined =
    invalidRange || !filterActive
      ? undefined
      : {
          car_id: selectedCarFilter !== "all" ? selectedCarFilter : undefined,
          from: fromDate,
          to: toDate,
        };

  const { data: sessions = [], isLoading, isError } = useChargingSessions(filter);

  const [editingCost, setEditingCost] = useState<string | null>(null);
  const [editedCostValue, setEditedCostValue] = useState<string>("");
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const [isDeleteDialogOpen, setIsDeleteDialogOpen] = useState(false);
  const [editingSession, setEditingSession] = useState<ChargingSession | null>(null);
  const [deletingSessionId, setDeletingSessionId] = useState<string | null>(null);
  const [formData, setFormData] = useState<FormData>(emptyForm());
  // Required-field validation state — set on a failed submit, cleared as the user fills fields.
  const [errors, setErrors] = useState<{ car_id?: boolean; date?: boolean; energy?: boolean; duration?: boolean }>({});

  const carById = useMemo(() => new Map(cars.map((c) => [c.id, c] as const)), [cars]);

  // FEAT-3 location memory: most-recent charge_power_kw seen at each location,
  // derived from session history (the persisted source of truth — no extra store).
  // Keyed by trimmed, case-insensitive location.
  const powerByLocation = useMemo(() => {
    const m = new Map<string, number>();
    [...sessions]
      .sort((a, b) => new Date(a.started_at).getTime() - new Date(b.started_at).getTime())
      .forEach((s) => {
        const loc = s.location?.trim().toLowerCase();
        if (loc && s.charge_power_kw != null) m.set(loc, s.charge_power_kw);
      });
    return m;
  }, [sessions]);

  // The single newest session (by time) — backs last-location / last-power prefill.
  const lastSession = useMemo(
    () => [...sessions].sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())[0],
    [sessions],
  );

  const powerForLocation = (loc: string): string => {
    const p = powerByLocation.get(loc.trim().toLowerCase());
    return p != null ? String(p) : "";
  };

  // FEAT-1 smart SOC: catalog map (for usable capacity) + per-car previous session.
  const catalogById = useMemo(() => new Map(catalog.map((c) => [c.id, c] as const)), [catalog]);
  const prevSessionForCar = (carId: string): ChargingSession | undefined =>
    [...sessions]
      .filter((s) => s.car_id === carId)
      .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())[0];

  // FEAT-4 elevation heuristic: the immediately-earlier session for the same car.
  const chronoPrev = (session: ChargingSession): ChargingSession | undefined =>
    [...sessions]
      .filter((s) => s.car_id === session.car_id && new Date(s.started_at).getTime() < new Date(session.started_at).getTime())
      .sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime())[0];
  // True when the previous session was at a different location (possible elevation change).
  const elevationMayDiffer = (session: ChargingSession): boolean => {
    const prev = chronoPrev(session);
    const a = session.location?.trim().toLowerCase();
    const b = prev?.location?.trim().toLowerCase();
    return !!a && !!b && a !== b;
  };

  // Tracks whether start/end SOC are still auto-predicted (safe to overwrite). Once
  // the user edits a field it becomes manual and predictions stop touching it.
  const [socAuto, setSocAuto] = useState({ start: true, end: true });
  // FEAT-6: duration is optional + auto-predicted until the user edits it.
  const [durationAuto, setDurationAuto] = useState(true);

  // FEAT-6 duration prediction: energy / power × 60 (minutes). Power = the entered
  // charge power, else the power remembered for the location (FEAT-3 memory).
  const predictDurationMin = (fd: FormData): string => {
    const energy = energyTotal(fd);
    const power = parseFloat(fd.charge_power_kw) || (fd.location ? parseFloat(powerForLocation(fd.location)) : NaN);
    if (!energy || !Number.isFinite(power) || power <= 0) return "";
    return String(Math.round((energy / power) * 60));
  };

  // Inputs for the SOC predictions, derived from the currently-selected car.
  const regenFactor = settings?.regen_factor ?? 0;
  const predInputs = (carId: string) => {
    const car = carById.get(carId);
    return {
      prev: prevSessionForCar(carId),
      capacity: resolveUsableCapacity(car, catalogById),
      avg: carAverageConsumption(carId, sessions),
    };
  };

  const toInt = (v: string): number | null => (v.trim() === "" ? null : parseInt(v, 10));
  const energyTotal = (fd: FormData): number | null => {
    const t = (parseFloat(fd.energy_peak_kwh) || 0) + (parseFloat(fd.energy_mid_kwh) || 0) + (parseFloat(fd.energy_offpeak_kwh) || 0);
    return t > 0 ? t : null;
  };

  // Live predictions for the current form (also drive the gentle "differs a lot" hint).
  const predictedStart = useMemo(() => {
    const { prev, capacity, avg } = predInputs(formData.car_id);
    const odo = toInt(formData.odometer_km);
    const tripKm = odo != null && prev?.odometer_km != null ? odo - prev.odometer_km : null;
    return predictStartSoc(prev?.end_soc, tripKm, { carAvgKwhPer100km: avg, regenFactor }, capacity);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [formData.car_id, formData.odometer_km, sessions, catalogById]);

  const predictedEnd = useMemo(() => {
    const { capacity } = predInputs(formData.car_id);
    return predictEndSoc(toInt(formData.start_soc), energyTotal(formData), capacity);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [formData.car_id, formData.start_soc, formData.energy_peak_kwh, formData.energy_mid_kwh, formData.energy_offpeak_kwh, sessions, catalogById]);

  // FEAT-1 onChange handlers: recompute the auto fields, mark manual edits.
  const onOdometerChange = (value: string) => {
    setFormData((p) => {
      const next = { ...p, odometer_km: value };
      if (socAuto.start) {
        const { prev, capacity, avg } = predInputs(next.car_id);
        const odo = toInt(value);
        const tripKm = odo != null && prev?.odometer_km != null ? odo - prev.odometer_km : null;
        const ps = predictStartSoc(prev?.end_soc, tripKm, { carAvgKwhPer100km: avg, regenFactor }, capacity);
        if (ps != null) next.start_soc = String(ps);
      }
      return next;
    });
  };
  const onEnergyChange = (field: "energy_peak_kwh" | "energy_mid_kwh" | "energy_offpeak_kwh", value: string) => {
    setFormData((p) => {
      const next = { ...p, [field]: value };
      if (socAuto.end) {
        const { capacity } = predInputs(next.car_id);
        const pe = predictEndSoc(toInt(next.start_soc), energyTotal(next), capacity);
        if (pe != null) next.end_soc = String(pe);
      }
      if (durationAuto) next.duration_minutes = predictDurationMin(next) || next.duration_minutes;
      return next;
    });
    setErrors((p) => ({ ...p, energy: false }));
  };
  const onChargePowerChange = (value: string) => {
    setFormData((p) => {
      const next = { ...p, charge_power_kw: value };
      if (durationAuto) next.duration_minutes = predictDurationMin(next) || next.duration_minutes;
      return next;
    });
  };
  const onDurationChange = (value: string) => {
    setDurationAuto(false);
    setFormData((p) => ({ ...p, duration_minutes: value }));
  };
  const onStartSocChange = (value: string) => {
    setSocAuto((s) => ({ ...s, start: false }));
    setFormData((p) => {
      const next = { ...p, start_soc: value };
      if (socAuto.end) {
        const { capacity } = predInputs(next.car_id);
        const pe = predictEndSoc(toInt(value), energyTotal(next), capacity);
        if (pe != null) next.end_soc = String(pe);
      }
      return next;
    });
  };
  const onEndSocChange = (value: string) => {
    setSocAuto((s) => ({ ...s, end: false }));
    setFormData((p) => ({ ...p, end_soc: value }));
  };

  // Gentle hint when the user's value diverges a lot from the prediction (FEAT-1).
  const SOC_HINT_MARGIN = 15;
  const startHint = !socAuto.start && predictedStart != null && toInt(formData.start_soc) != null &&
    Math.abs((toInt(formData.start_soc) as number) - predictedStart) > SOC_HINT_MARGIN ? predictedStart : null;
  const endHint = !socAuto.end && predictedEnd != null && toInt(formData.end_soc) != null &&
    Math.abs((toInt(formData.end_soc) as number) - predictedEnd) > SOC_HINT_MARGIN ? predictedEnd : null;

  // FEAT-5 entry-time flags (client mirror of the backend rules). Errors block save.
  const entryFlags = useMemo<EntryFlag[]>(() => {
    const { prev } = predInputs(formData.car_id);
    return computeEntryFlags({
      odometer: toInt(formData.odometer_km),
      prevOdometer: prev?.odometer_km ?? null,
      startSoc: toInt(formData.start_soc),
      endSoc: toInt(formData.end_soc),
      energyKwh: energyTotal(formData),
      chargePowerKw: formData.charge_power_kw.trim() === "" ? null : parseFloat(formData.charge_power_kw),
      durationMin: toInt(formData.duration_minutes),
    });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [formData, sessions]);
  const flagsFor = (field: EntryFlag["field"]) => entryFlags.filter((f) => f.field === field);
  const fieldHasError = (field: EntryFlag["field"]) => flagsFor(field).some((f) => f.severity === "error");
  const saveBlocked = hasBlockingError(entryFlags);

  // Renders the inline flag messages under a field (🔴 error / 🟡 warning).
  const FieldFlags = ({ field }: { field: EntryFlag["field"] }) => (
    <>
      {flagsFor(field).map((f) => (
        <p key={f.code} className={cn("text-xs flex items-start gap-1", severityStyle[f.severity].text)}>
          {f.severity === "error" ? <AlertCircle className="h-3.5 w-3.5 mt-0.5 shrink-0" /> : <AlertTriangle className="h-3.5 w-3.5 mt-0.5 shrink-0" />}
          <span>{f.message}</span>
        </p>
      ))}
    </>
  );

  // The car a fresh form should pre-select: the user's default car if it still exists,
  // otherwise the first car in the list.
  const defaultCarId = useMemo(() => {
    if (settings?.default_car_id && carById.has(settings.default_car_id)) return settings.default_car_id;
    return cars.length > 0 ? cars[0].id : "";
  }, [settings?.default_car_id, carById, cars]);

  // Default the car filter to the user's default car once settings load.
  useEffect(() => {
    if (settings?.default_car_id && selectedCarFilter === "all") {
      setSelectedCarFilter(settings.default_car_id);
    }
  }, [settings?.default_car_id]); // eslint-disable-line react-hooks/exhaustive-deps

  // Newest-first for the history list (the API order is not guaranteed).
  const displaySessions = useMemo(
    () => [...sessions].sort((a, b) => new Date(b.started_at).getTime() - new Date(a.started_at).getTime()),
    [sessions],
  );

  const clearFilters = () => {
    setSelectedCarFilter("all");
    setFromDate(undefined);
    setToDate(undefined);
  };

  const rates = ratesFromSettings(settings);

  // A manual override wins; otherwise the rate-based sum (single source of truth in lib/cost).
  const getSessionCost = (session: ChargingSession) => session.cost ?? calcCost(session, ratesForSession(session, rates)).total;

  const totalKwh = (s: ChargingSession) =>
    s.kwh_charged ??
    (s.energy_peak_kwh ?? 0) + (s.energy_mid_kwh ?? 0) + (s.energy_offpeak_kwh ?? 0);

  const num = (v: string): number | null => (v.trim() === "" ? null : Number(v));

  const buildInput = (): ChargingInput => {
    const peak = parseFloat(formData.energy_peak_kwh) || 0;
    const mid = parseFloat(formData.energy_mid_kwh) || 0;
    const offpeak = parseFloat(formData.energy_offpeak_kwh) || 0;
    const total = peak + mid + offpeak;
    const started = formData.date;
    const durationMin = parseInt(formData.duration_minutes) || 0;
    const ended = durationMin > 0 ? new Date(started.getTime() + durationMin * 60000) : null;
    return {
      car_id: formData.car_id,
      started_at: started.toISOString(),
      ended_at: ended ? ended.toISOString() : null,
      location: formData.location.trim() || null,
      kwh_charged: total || null,
      energy_peak_kwh: peak || null,
      energy_mid_kwh: mid || null,
      energy_offpeak_kwh: offpeak || null,
      start_soc: num(formData.start_soc),
      end_soc: num(formData.end_soc),
      odometer_km: formData.odometer_km.trim() === "" ? null : parseInt(formData.odometer_km, 10),
      charge_power_kw: num(formData.charge_power_kw),
      // cost omitted — the Go API computes it from the per-period energy × rates
    };
  };

  // Required: car, date, total energy (kwh_charged) > 0, and duration > 0.
  // Optional: start/end SOC, location, notes, and the peak/mid/offpeak breakdown
  // (a session may have total energy without a per-period split).
  const validate = (): boolean => {
    const peak = parseFloat(formData.energy_peak_kwh) || 0;
    const mid = parseFloat(formData.energy_mid_kwh) || 0;
    const offpeak = parseFloat(formData.energy_offpeak_kwh) || 0;
    // FEAT-6: duration is now optional — only car, date, and total energy are required.
    const next = {
      car_id: !formData.car_id,
      date: !formData.date,
      energy: peak + mid + offpeak <= 0,
      duration: false,
    };
    setErrors(next);
    return !next.car_id && !next.date && !next.energy;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!validate()) {
      toast.error("لطفاً فیلدهای اجباری را پر کنید");
      return;
    }
    if (saveBlocked) {
      toast.error("لطفاً خطاهای فرم را برطرف کنید");
      return;
    }
    try {
      if (editingSession) {
        await updateSession.mutateAsync({ id: editingSession.id, input: buildInput() });
        toast.success("جلسه شارژ با موفقیت ویرایش شد");
      } else {
        await createSession.mutateAsync(buildInput());
        toast.success("جلسه شارژ با موفقیت ثبت شد");
      }
      resetForm();
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  // Open a fresh "add session" form, pre-selecting the default car (bug #1).
  const handleAddNew = () => {
    const next = emptyForm(defaultCarId);
    next.location = lastSession?.location ?? "";
    next.start_soc = lastSession?.end_soc?.toString() ?? "";
    // FEAT-3: pre-fill charge power remembered for the pre-filled location.
    next.charge_power_kw = next.location ? powerForLocation(next.location) : "";
    // start SOC starts auto-predicted (carries prev end SOC); end SOC auto until energy entered.
    next.start_soc = lastSession?.end_soc != null ? String(lastSession.end_soc) : "";
    setSocAuto({ start: true, end: true });
    setDurationAuto(true);
    setFormData(next);
    setEditingSession(null);
    setErrors({});
    setIsDialogOpen(true);
  };

  const handleEdit = (session: ChargingSession) => {
    setEditingSession(session);
    setErrors({});
    const start = new Date(session.started_at);
    const durationMin = session.ended_at
      ? Math.max(0, Math.round((new Date(session.ended_at).getTime() - start.getTime()) / 60000))
      : 0;
    setFormData({
      car_id: session.car_id,
      date: start,
      duration_minutes: durationMin ? durationMin.toString() : "",
      energy_peak_kwh: session.energy_peak_kwh?.toString() ?? "",
      energy_mid_kwh: session.energy_mid_kwh?.toString() ?? "",
      energy_offpeak_kwh: session.energy_offpeak_kwh?.toString() ?? "",
      location: session.location ?? "",
      charge_power_kw: session.charge_power_kw?.toString() ?? "",
      start_soc: session.start_soc?.toString() ?? "",
      end_soc: session.end_soc?.toString() ?? "",
      odometer_km: session.odometer_km?.toString() ?? "",
    });
    // Editing an existing session: its SOC + duration are user data — never auto-overwrite.
    setSocAuto({ start: false, end: false });
    setDurationAuto(false);
    setIsDialogOpen(true);
  };

  const handleDelete = async () => {
    if (!deletingSessionId) return;
    try {
      await deleteSession.mutateAsync(deletingSessionId);
      toast.success("جلسه شارژ با موفقیت حذف شد");
    } catch (err) {
      toast.error((err as Error).message);
    }
    setIsDeleteDialogOpen(false);
    setDeletingSessionId(null);
  };

  // Inline cost override. PUT is a full replace, so resend the whole session.
  const handleCostSave = async (session: ChargingSession) => {
    const newCost = parseFloat(editedCostValue);
    if (isNaN(newCost) || newCost < 0) {
      toast.error("لطفاً مقدار معتبر وارد کنید");
      return;
    }
    const input: ChargingInput = {
      car_id: session.car_id,
      started_at: session.started_at,
      ended_at: session.ended_at,
      location: session.location,
      kwh_charged: session.kwh_charged,
      energy_peak_kwh: session.energy_peak_kwh,
      energy_mid_kwh: session.energy_mid_kwh,
      energy_offpeak_kwh: session.energy_offpeak_kwh,
      start_soc: session.start_soc,
      end_soc: session.end_soc,
      odometer_km: session.odometer_km,
      charge_power_kw: session.charge_power_kw,
      cost: newCost,
    };
    try {
      await updateSession.mutateAsync({ id: session.id, input });
      toast.success("هزینه با موفقیت به‌روزرسانی شد");
      setEditingCost(null);
    } catch (err) {
      toast.error((err as Error).message);
    }
  };

  const resetForm = () => {
    const next = emptyForm(defaultCarId);
    next.location = lastSession?.location ?? "";
    next.start_soc = lastSession?.end_soc?.toString() ?? "";
    next.charge_power_kw = next.location ? powerForLocation(next.location) : "";
    next.start_soc = lastSession?.end_soc != null ? String(lastSession.end_soc) : "";
    setSocAuto({ start: true, end: true });
    setDurationAuto(true);
    setFormData(next);
    setEditingSession(null);
    setErrors({});
    setIsDialogOpen(false);
  };

  const formatDate = (iso: string) => {
    const d = new Date(iso);
    return language === "fa" ? formatJalali(d, "yyyy/MM/dd") : format(d, "yyyy/MM/dd");
  };

  const formatDuration = (session: ChargingSession) => {
    if (!session.ended_at) return "—";
    const minutes = Math.max(
      0,
      Math.round((new Date(session.ended_at).getTime() - new Date(session.started_at).getTime()) / 60000),
    );
    const hours = Math.floor(minutes / 60);
    const mins = minutes % 60;
    return `${hours}:${mins.toString().padStart(2, "0")}`;
  };

  const currency = settings?.currency ?? 'toman';
  const carName = (carId: string) => carById.get(carId)?.name ?? "—";

  return (
    <div className="min-h-screen app-page-bg">
      <Header />

      <main className="container mx-auto px-3 sm:px-4 py-4 sm:py-8">
        <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4 mb-4">
          <h2 className="text-2xl sm:text-3xl font-bold text-foreground">{t("charging.sessions")}</h2>
          <Button onClick={handleAddNew} disabled={cars.length === 0} className="w-full sm:w-auto">
            <Plus className="mr-2 h-4 w-4" />
            {t("charging.addSession")}
          </Button>
        </div>

        {/* History filters */}
        {cars.length > 0 && (
          <div className="flex flex-col sm:flex-row flex-wrap gap-2 sm:gap-3 items-stretch sm:items-end mb-2">
            {cars.length > 1 && (
              <div className="space-y-1">
                <Label className="text-xs text-muted-foreground">{language === "fa" ? "خودرو" : "Car"}</Label>
                <Select value={selectedCarFilter} onValueChange={setSelectedCarFilter}>
                  <SelectTrigger className="w-full sm:w-[180px]">
                    <SelectValue placeholder={language === "fa" ? "همه خودروها" : "All Cars"} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="all">{language === "fa" ? "همه خودروها" : "All Cars"}</SelectItem>
                    {cars.map((car) => (
                      <SelectItem key={car.id} value={car.id}>
                        {car.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">{t("charging.from")}</Label>
              <JalaliDatePicker date={fromDate} onDateChange={setFromDate} placeholder={t("charging.from")} className="w-full sm:w-[160px]" />
            </div>
            <div className="space-y-1">
              <Label className="text-xs text-muted-foreground">{t("charging.to")}</Label>
              <JalaliDatePicker date={toDate} onDateChange={setToDate} placeholder={t("charging.to")} className="w-full sm:w-[160px]" />
            </div>
            {filterActive && (
              <Button variant="ghost" onClick={clearFilters} className="sm:self-end">
                <FilterX className="h-4 w-4 mr-1" />
                {t("charging.clearFilters")}
              </Button>
            )}
          </div>
        )}
        {invalidRange && <p className="text-sm text-destructive mb-4">{t("charging.invalidRange")}</p>}
        <div className="mb-4" />

        {cars.length === 0 ? (
          <Card>
            <CardContent className="py-8 text-center">
              <p className="text-muted-foreground">ابتدا باید حداقل یک خودرو اضافه کنید</p>
              <Button className="mt-4" onClick={() => navigate("/cars")}>
                افزودن خودرو
              </Button>
            </CardContent>
          </Card>
        ) : invalidRange ? null : isLoading ? (
          <div className="py-12 text-center text-muted-foreground">
            {language === "fa" ? "در حال بارگذاری..." : "Loading..."}
          </div>
        ) : isError ? (
          <Card>
            <CardContent className="py-8 text-center">
              <p className="text-muted-foreground">
                {language === "fa" ? "خطا در بارگذاری جلسات" : "Failed to load sessions"}
              </p>
            </CardContent>
          </Card>
        ) : displaySessions.length === 0 ? (
          <Card>
            <CardContent className="py-8 text-center space-y-3">
              <p className="text-muted-foreground">
                {filterActive
                  ? t("charging.noSessionsInRange")
                  : language === "fa"
                    ? "هنوز جلسه‌ای ثبت نشده است"
                    : "No sessions yet"}
              </p>
              {filterActive && (
                <Button variant="outline" onClick={clearFilters}>
                  {t("charging.clearFilters")}
                </Button>
              )}
            </CardContent>
          </Card>
        ) : (
          <div className="grid gap-3 sm:gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {displaySessions.map((session) => {
              const isExpanded = !collapsedIds.has(session.id);
              const warns = session.warnings ?? [];
              const flagged = warns.length > 0;
              const hasEffWarn = warns.some((w) => w.code === "efficiency_out_of_band");
              return (
              <Card key={session.id} className={cn("overflow-hidden", flagged && "border-s-4 border-s-amber-500 bg-amber-500/5")}>
                {/* Summary row — tap to expand */}
                <div
                  role="button"
                  tabIndex={0}
                  aria-expanded={isExpanded}
                  onClick={() => toggleCollapse(session.id)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" || e.key === " ") {
                      e.preventDefault();
                      toggleCollapse(session.id);
                    }
                  }}
                  className="cursor-pointer"
                >
                  <CardHeader className="p-4 sm:p-6">
                    <CardTitle className="text-base sm:text-lg flex justify-between items-center gap-2">
                      <span className="flex items-center gap-2 min-w-0">
                        <ChevronDown
                          className={cn("h-4 w-4 flex-shrink-0 text-muted-foreground transition-transform", isExpanded && "rotate-180")}
                        />
                        <span className="truncate">{carName(session.car_id)}</span>
                        {flagged && (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild onClick={(e) => e.stopPropagation()}>
                                <Badge variant="outline" className="border-amber-500/50 text-amber-600 dark:text-amber-400 bg-amber-500/10 gap-1 px-1.5 shrink-0">
                                  <AlertTriangle className="h-3 w-3" />
                                  {formatNumber(String(warns.length))}
                                </Badge>
                              </TooltipTrigger>
                              <TooltipContent className="max-w-[260px] text-right" dir="rtl">
                                <ul className="space-y-1">
                                  {warns.map((w) => (
                                    <li key={w.code} className="text-xs">{w.message}</li>
                                  ))}
                                </ul>
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </span>
                      <div className="flex gap-1 flex-shrink-0">
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleEdit(session);
                          }}
                          className="h-8 w-8"
                        >
                          <Edit className="h-3.5 w-3.5" />
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={(e) => {
                            e.stopPropagation();
                            setDeletingSessionId(session.id);
                            setIsDeleteDialogOpen(true);
                          }}
                          className="h-8 w-8"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </Button>
                      </div>
                    </CardTitle>
                    {/* Compact summary: date · total kWh · total cost */}
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm text-muted-foreground pt-1">
                      <span className="flex items-center gap-1">
                        <CalendarIcon className="h-3.5 w-3.5" />
                        {formatDate(session.started_at)}
                      </span>
                      <span className="flex items-center gap-1">
                        <Zap className="h-3.5 w-3.5" />
                        {totalKwh(session).toFixed(2)} kWh
                      </span>
                      <span>{formatCost(getSessionCost(session), currency)}</span>
                      {session.efficiency_kwh_per_100km != null && (
                        <span className={cn("font-medium", hasEffWarn ? "text-amber-600 dark:text-amber-400" : "text-foreground")}>
                          {session.efficiency_kwh_per_100km.toFixed(1)} kWh/100km
                        </span>
                      )}
                    </div>
                  </CardHeader>
                </div>

                {/* Expanded detail */}
                {isExpanded && (
                  <CardContent className="p-4 sm:p-6 pt-0">
                    <div className="space-y-3 border-t border-border/50 pt-3">
                      {/* FEAT-5: warnings block first, so tap users see them without the tooltip */}
                      {flagged && (
                        <div className="space-y-1 rounded-md bg-amber-500/10 p-2">
                          {warns.map((w) => (
                            <p key={w.code} className="flex items-start gap-1.5 text-xs text-amber-600 dark:text-amber-400">
                              <AlertTriangle className="h-3.5 w-3.5 mt-0.5 shrink-0" />
                              <span>{w.message}</span>
                            </p>
                          ))}
                        </div>
                      )}
                      {/* Start time + duration */}
                      <div className="flex items-center gap-2 text-sm">
                        <Clock className="h-4 w-4 text-muted-foreground" />
                        <span>
                          {new Date(session.started_at).toLocaleTimeString(language === "fa" ? "fa-IR" : "en-US", {
                            hour: "2-digit",
                            minute: "2-digit",
                          })}
                          {" · "}
                          {language === "fa" ? "مدت" : "Duration"}: {formatDuration(session)}
                        </span>
                      </div>

                      {/* Cost (with inline override edit) */}
                      <div className="flex items-center gap-2 text-sm">
                        {editingCost === session.id ? (
                          <div className="flex items-center gap-2 flex-1">
                            <Input
                              type="number"
                              value={editedCostValue}
                              onChange={(e) => setEditedCostValue(e.target.value)}
                              className="h-7 w-32 text-sm"
                              autoFocus
                            />
                            <button
                              onClick={() => handleCostSave(session)}
                              className="p-1 hover:bg-green-100 dark:hover:bg-green-900 rounded transition-colors"
                            >
                              <Check className="w-4 h-4 text-green-600" />
                            </button>
                            <button
                              onClick={() => setEditingCost(null)}
                              className="p-1 hover:bg-red-100 dark:hover:bg-red-900 rounded transition-colors"
                            >
                              <X className="w-4 h-4 text-red-600" />
                            </button>
                          </div>
                        ) : (
                          <div className="flex items-center gap-2 flex-1">
                            <span>{formatCost(getSessionCost(session), currency)}</span>
                            <button
                              onClick={() => {
                                setEditingCost(session.id);
                                setEditedCostValue(getSessionCost(session).toString());
                              }}
                              className="p-1 hover:bg-muted rounded transition-colors"
                              title="ویرایش هزینه"
                            >
                              <Pencil className="w-3 h-3 text-muted-foreground" />
                            </button>
                          </div>
                        )}
                      </div>

                      {/* Time-of-use cost breakdown (degrades to total-only when no per-period split). */}
                      {(() => {
                        const c = calcCost(session, ratesForSession(session, rates));
                        return (
                          <TOUBreakdown
                            variant="inline"
                            peak={{ kwh: session.energy_peak_kwh ?? 0, cost: c.peak }}
                            mid={{ kwh: session.energy_mid_kwh ?? 0, cost: c.mid }}
                            offpeak={{ kwh: session.energy_offpeak_kwh ?? 0, cost: c.offpeak }}
                            total={{ kwh: totalKwh(session), cost: getSessionCost(session) }}
                          />
                        );
                      })()}

                      {session.location && (
                        <div className="text-sm text-muted-foreground">📍 {session.location}</div>
                      )}

                      {/* FEAT-4: raw vs regen-adjusted consumption */}
                      {session.efficiency_kwh_per_100km != null && (
                        <div className="text-sm">
                          <span className="text-muted-foreground">{language === "fa" ? "مصرف" : "Consumption"}: </span>
                          {language === "fa" ? "خام " : "raw "}{session.efficiency_kwh_per_100km.toFixed(1)}
                          {regenFactor > 0 && (
                            <>
                              {" · "}
                              {language === "fa" ? "با بازیابی " : "adjusted "}
                              {applyRegen(session.efficiency_kwh_per_100km, regenFactor).toFixed(1)}
                            </>
                          )}
                          {" kWh/100km"}
                        </div>
                      )}

                      {/* FEAT-4: soft elevation note when the previous session was elsewhere */}
                      {elevationMayDiffer(session) && (
                        <div className="text-xs text-muted-foreground">
                          {language === "fa"
                            ? "ممکن است اختلاف ارتفاع بر مصرف اثر گذاشته باشد"
                            : "Elevation difference may have affected consumption"}
                        </div>
                      )}

                      {session.notes && (
                        <div className="text-sm">
                          <span className="text-muted-foreground">{t("charging.notes")}: </span>
                          {session.notes}
                        </div>
                      )}

                      {session.start_soc !== null && session.end_soc !== null && (
                        <div className="pt-2">
                          <SOCAnalysis startSoc={session.start_soc} endSoc={session.end_soc} />
                        </div>
                      )}

                      {/* Close affordance — for users who have scrolled into the detail */}
                      <div className="pt-1 flex justify-center border-t border-border/30 mt-1">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-xs text-muted-foreground h-7 px-3"
                          onClick={() => toggleCollapse(session.id)}
                        >
                          <X className="w-3 h-3 mr-1" />
                          بستن
                        </Button>
                      </div>
                    </div>
                  </CardContent>
                )}
              </Card>
              );
            })}
          </div>
        )}

        <Dialog open={isDialogOpen} onOpenChange={setIsDialogOpen}>
          <DialogContent className="max-w-md">
            <DialogHeader>
              <DialogTitle>{editingSession ? "ویرایش جلسه شارژ" : t("charging.addSession")}</DialogTitle>
            </DialogHeader>
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="car_id" className={cn(errors.car_id && "text-destructive")}>خودرو</Label>
                <Select
                  value={formData.car_id}
                  onValueChange={(value) => {
                    setFormData({ ...formData, car_id: value });
                    setErrors((p) => ({ ...p, car_id: false }));
                  }}
                  required
                >
                  <SelectTrigger className={cn(errors.car_id && "border-destructive focus:ring-destructive")}>
                    <SelectValue placeholder="انتخاب خودرو" />
                  </SelectTrigger>
                  <SelectContent>
                    {cars.map((car) => (
                      <SelectItem key={car.id} value={car.id}>
                        {car.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label className={cn(errors.date && "text-destructive")}>{t("charging.date")}</Label>
                <JalaliDatePicker
                  date={formData.date}
                  onDateChange={(date) => {
                    if (date) {
                      setFormData({ ...formData, date });
                      setErrors((p) => ({ ...p, date: false }));
                    }
                  }}
                  placeholder="انتخاب تاریخ"
                  className={cn(errors.date && "border-destructive")}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="energy_peak_kwh" className={cn(errors.energy && "text-destructive")}>انرژی اوج بار (kWh)</Label>
                <Input id="energy_peak_kwh" type="number" step="0.01" min="0" value={formData.energy_peak_kwh}
                  className={cn(errors.energy && "border-destructive focus-visible:ring-destructive")}
                  onChange={(e) => onEnergyChange("energy_peak_kwh", e.target.value)} placeholder="0" />
              </div>

              <div className="space-y-2">
                <Label htmlFor="energy_mid_kwh" className={cn(errors.energy && "text-destructive")}>انرژی میان باری (kWh)</Label>
                <Input id="energy_mid_kwh" type="number" step="0.01" min="0" value={formData.energy_mid_kwh}
                  className={cn(errors.energy && "border-destructive focus-visible:ring-destructive")}
                  onChange={(e) => onEnergyChange("energy_mid_kwh", e.target.value)} placeholder="0" />
              </div>

              <div className="space-y-2">
                <Label htmlFor="energy_offpeak_kwh" className={cn(errors.energy && "text-destructive")}>انرژی کم باری (kWh)</Label>
                <Input id="energy_offpeak_kwh" type="number" step="0.01" min="0" value={formData.energy_offpeak_kwh}
                  className={cn(errors.energy && "border-destructive focus-visible:ring-destructive")}
                  onChange={(e) => onEnergyChange("energy_offpeak_kwh", e.target.value)} placeholder="0" />
              </div>
              {errors.energy && (
                <p className="text-sm text-destructive">حداقل مقدار انرژی شارژشده الزامی است</p>
              )}
              <FieldFlags field="energy" />

              <div className="space-y-2">
                <Label htmlFor="odometer_km">{language === "fa" ? "کیلومتر شمار (اختیاری)" : "Odometer (km, optional)"}</Label>
                <Input id="odometer_km" type="number" min="0" step="1" value={formData.odometer_km}
                  className={cn(fieldHasError("odometer") && severityStyle.error.fieldBorder)}
                  onChange={(e) => onOdometerChange(e.target.value)}
                  placeholder={language === "fa" ? "مثلاً ۱۲۳۴۵" : "e.g. 12345"} />
                <FieldFlags field="odometer" />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="location">مکان شارژ (اختیاری)</Label>
                  <Input id="location" value={formData.location}
                    onChange={(e) => {
                      const location = e.target.value;
                      // FEAT-3: when the location matches a known one, auto-fill its
                      // remembered charge power (unless the user already typed one).
                      const known = powerForLocation(location);
                      setFormData((p) => {
                        const next = {
                          ...p,
                          location,
                          charge_power_kw: p.charge_power_kw === "" && known !== "" ? known : p.charge_power_kw,
                        };
                        if (durationAuto) next.duration_minutes = predictDurationMin(next) || next.duration_minutes;
                        return next;
                      });
                    }}
                    placeholder="نام ایستگاه شارژ یا آدرس" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="charge_power_kw">{language === "fa" ? "توان شارژ (kW، اختیاری)" : "Charge power (kW, optional)"}</Label>
                  <Input id="charge_power_kw" type="number" step="0.1" min="0" value={formData.charge_power_kw}
                    onChange={(e) => setFormData({ ...formData, charge_power_kw: e.target.value })}
                    placeholder={language === "fa" ? "مثلاً ۷ یا ۵۰" : "e.g. 7 or 50"} />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="start_soc">SOC شروع (%)</Label>
                  <Input id="start_soc" type="number" min="0" max="100" value={formData.start_soc}
                    className={cn(fieldHasError("start_soc") && severityStyle.error.fieldBorder)}
                    onChange={(e) => onStartSocChange(e.target.value)} placeholder="30" />
                  {startHint != null && (
                    <p className="text-[11px] text-muted-foreground">
                      {language === "fa" ? `پیش‌بینی: حدود ${startHint}%` : `Predicted: ~${startHint}%`}
                    </p>
                  )}
                  <FieldFlags field="start_soc" />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="end_soc">SOC پایان (%)</Label>
                  <Input id="end_soc" type="number" min="0" max="100" value={formData.end_soc}
                    className={cn(fieldHasError("end_soc") && severityStyle.error.fieldBorder)}
                    onChange={(e) => onEndSocChange(e.target.value)} placeholder="80" />
                  {endHint != null && (
                    <p className="text-[11px] text-muted-foreground">
                      {language === "fa" ? `پیش‌بینی: حدود ${endHint}%` : `Predicted: ~${endHint}%`}
                    </p>
                  )}
                  <FieldFlags field="end_soc" />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="duration_minutes">{language === "fa" ? "مدت زمان (دقیقه، اختیاری)" : "Duration (minutes, optional)"}</Label>
                <Input
                  id="duration_minutes"
                  type="number"
                  min="1"
                  value={formData.duration_minutes}
                  onChange={(e) => onDurationChange(e.target.value)}
                  placeholder={language === "fa" ? "پیش‌بینی از انرژی و توان شارژ" : "predicted from energy & power"}
                />
                <FieldFlags field="duration" />
              </div>

              <DialogFooter>
                <Button type="button" variant="outline" onClick={resetForm}>
                  {t("common.cancel")}
                </Button>
                <Button type="submit" disabled={createSession.isPending || updateSession.isPending || saveBlocked}>
                  {createSession.isPending || updateSession.isPending ? "در حال ثبت..." : t("common.save")}
                </Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>

        <AlertDialog open={isDeleteDialogOpen} onOpenChange={setIsDeleteDialogOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>آیا مطمئن هستید؟</AlertDialogTitle>
              <AlertDialogDescription>
                این عملیات قابل بازگشت نیست. جلسه شارژ به طور دائم حذف خواهد شد.
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>لغو</AlertDialogCancel>
              <AlertDialogAction onClick={handleDelete}>حذف</AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </main>
    </div>
  );
};

export default Charging;
