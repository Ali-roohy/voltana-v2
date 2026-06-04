import { useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from "@/components/ui/button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { useTranslation } from "react-i18next";
import { useLanguage } from "@/contexts/LanguageContext";
import { useAuth } from "@/hooks/useAuth";
import { Zap, Car, Bolt, Map, BatteryCharging } from "lucide-react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { LineChart, Line, BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts';
import { Header } from "@/components/Header";
import { TOUBreakdown } from "@/components/TOUBreakdown";
import { formatNumber } from "@/lib/utils";
import { calcCost, ratesFromSettings, formatCost } from "@/lib/cost";
import { useChargingSessions } from "@/features/charging/hooks";
import type { ChargingSession } from "@/features/charging/api";
import { useCars } from "@/features/cars/hooks";
import { useSettings } from "@/features/settings/hooks";
import { useDashboard, useBattery, useBatteryHistory } from "@/features/analytics/hooks";
import { isInsufficient } from "@/features/analytics/api";

const totalKwh = (s: ChargingSession) =>
  s.kwh_charged ?? (s.energy_peak_kwh ?? 0) + (s.energy_mid_kwh ?? 0) + (s.energy_offpeak_kwh ?? 0);

export default function Index() {
  const { t } = useTranslation();
  const { language } = useLanguage();
  const { user, loading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (!loading && !user) {
      navigate('/auth');
    }
  }, [user, loading, navigate]);

  const { data: settings } = useSettings();
  const currency = settings?.currency ?? 'toman';
  const { data: cars = [] } = useCars();
  const { data: sessions = [] } = useChargingSessions();

  const defaultCar = settings?.default_car_id ? cars.find((c) => c.id === settings.default_car_id) : undefined;

  // Server-side lifetime fleet stats (total km + avg efficiency the client can't derive).
  const { data: dashboard } = useDashboard();

  // Battery health is per-car: default to the user's default car, else the first car.
  // `pickedCarId` overrides via the selector (shown only for multi-car users).
  const [pickedCarId, setPickedCarId] = useState<string | undefined>(undefined);
  const batteryCarId = pickedCarId ?? settings?.default_car_id ?? cars[0]?.id;
  const batteryCar = cars.find((c) => c.id === batteryCarId);
  const { data: battery } = useBattery(batteryCarId);
  const { data: batteryHistory } = useBatteryHistory(batteryCarId);

  const sohTrend = (batteryHistory?.items ?? []).map((s) => ({
    date: new Date(s.computed_at).toLocaleDateString(language === 'fa' ? 'fa-IR' : 'en-US', { month: 'short', day: 'numeric' }),
    soh: s.soh_pct,
  }));
  const latestSoh = battery && !isInsufficient(battery) ? battery : null;

  const stats = useMemo(() => {
    const rates = ratesFromSettings(settings);
    const filtered = settings?.default_car_id
      ? sessions.filter((s) => s.car_id === settings.default_car_id)
      : sessions;
    const sorted = [...filtered].sort(
      (a, b) => new Date(a.started_at).getTime() - new Date(b.started_at).getTime(),
    );

    const totalEnergy = sorted.reduce((sum, s) => sum + totalKwh(s), 0);
    // Honor a manual cost override, else fall back to the rate-based sum (a bare
    // `s.cost ?? 0` undercounted sessions whose cost is server/rate-computed).
    const totalCost = sorted.reduce((sum, s) => sum + (s.cost ?? calcCost(s, rates).total), 0);

    // Current-month time-of-use breakdown (Gregorian YYYY-MM, same bucketing as the
    // energy trend below). Costs are rate-based via the shared helper.
    const ym = new Date().toISOString().slice(0, 7);
    const touMonth = sorted
      .filter((s) => s.started_at.slice(0, 7) === ym)
      .reduce(
        (acc, s) => {
          const c = calcCost(s, rates);
          acc.peak.kwh += s.energy_peak_kwh ?? 0;
          acc.peak.cost += c.peak;
          acc.mid.kwh += s.energy_mid_kwh ?? 0;
          acc.mid.cost += c.mid;
          acc.offpeak.kwh += s.energy_offpeak_kwh ?? 0;
          acc.offpeak.cost += c.offpeak;
          return acc;
        },
        { peak: { kwh: 0, cost: 0 }, mid: { kwh: 0, cost: 0 }, offpeak: { kwh: 0, cost: 0 } },
      );

    // Monthly energy + cost trend, sharing the same YYYY-MM buckets so the two
    // charts line up. Cost reuses the shared helper (no re-inlined rate math).
    const monthly: Record<string, { energy: number; cost: number }> = {};
    sorted.forEach((s) => {
      const month = s.started_at.slice(0, 7); // YYYY-MM
      const bucket = monthly[month] ?? (monthly[month] = { energy: 0, cost: 0 });
      bucket.energy += totalKwh(s);
      bucket.cost += s.cost ?? calcCost(s, rates).total;
    });
    const trend = Object.entries(monthly)
      .map(([month, v]) => ({ month, energy: Number(v.energy.toFixed(2)), cost: Math.round(v.cost) }))
      .sort((a, b) => a.month.localeCompare(b.month));

    // Headline figures (scoped to the same sessions as totalCost).
    const sessionCount = sorted.length;
    const avgCost = sessionCount > 0 ? totalCost / sessionCount : null;

    const socData = sorted
      .map((s) => ({
        date: new Date(s.started_at).toLocaleDateString('fa-IR', { month: 'short', day: 'numeric' }),
        socChange: (s.end_soc ?? 0) - (s.start_soc ?? 0),
        energy: totalKwh(s),
      }))
      .slice(-10);

    return {
      totalEnergy: totalEnergy.toFixed(1),
      totalCost: totalCost.toFixed(0),
      trend,
      socData,
      touMonth,
      sessionCount,
      avgCost,
    };
  }, [sessions, settings?.default_car_id, settings?.peak_rate, settings?.mid_rate, settings?.offpeak_rate]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="animate-pulse">
          <Zap className="w-12 h-12 text-primary" />
        </div>
      </div>
    );
  }
  if (!user) {
    return null;
  }
  return (
    <div className="min-h-screen bg-gradient-to-br from-background via-secondary to-background">
      <Header />

      <main className="container mx-auto px-3 sm:px-4 py-4 sm:py-8">
        <div className="mb-6 sm:mb-8">
          <h2 className="font-bold mb-2 text-xl sm:text-2xl">
            {t('dashboard.welcome')}, {user.user_metadata?.full_name || user.email}!
          </h2>
          <div className="flex flex-col sm:flex-row items-start sm:items-center gap-2 sm:gap-3">
            <p className="text-sm sm:text-base text-muted-foreground">{t('app.tagline')}</p>
            {defaultCar && (
              <div className="flex items-center gap-2 bg-primary/10 px-2 sm:px-3 py-1 rounded-full">
                <Car className="w-3 h-3 sm:w-4 sm:h-4 text-primary" />
                <span className="text-center font-medium text-xs">{defaultCar.name}</span>
              </div>
            )}
          </div>
        </div>

        {/* Quick Actions Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 sm:gap-4 mb-6 sm:mb-8">
          <Button variant="outline" className="h-auto py-6 sm:py-8 flex flex-col items-center gap-2 sm:gap-3 hover:border-primary hover:shadow-soft transition-all" onClick={() => navigate('/charging')}>
            <Bolt className="w-6 h-6 sm:w-8 sm:h-8 text-accent" />
            <div className="text-center">
              <div className="font-semibold text-sm sm:text-base">{t('nav.charging')}</div>
              <div className="text-xs text-muted-foreground">{t('charging.sessions')}</div>
            </div>
          </Button>

          <Button variant="outline" className="h-auto py-6 sm:py-8 flex flex-col items-center gap-2 sm:gap-3 hover:border-primary hover:shadow-soft transition-all" onClick={() => navigate('/map')}>
            <Map className="w-6 h-6 sm:w-8 sm:h-8 text-primary-glow" />
            <div className="text-center">
              <div className="font-semibold text-sm sm:text-base">{t('nav.map')}</div>
              <div className="text-xs text-muted-foreground">نقشه ایستگاه‌ها</div>
            </div>
          </Button>
        </div>

        {/* Stats Cards */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3 sm:gap-4 mb-6 sm:mb-8">
          <div className="bg-card rounded-xl sm:rounded-2xl p-4 sm:p-6 shadow-soft border border-border/50 hover:shadow-lg transition-shadow">
            <div className="text-xs sm:text-sm text-muted-foreground mb-1 sm:mb-2">{t('dashboard.totalCharge')}</div>
            <div className="text-2xl sm:text-3xl font-bold bg-gradient-primary bg-clip-text text-transparent">
              {formatNumber(stats.totalEnergy)} kWh
            </div>
          </div>
          <div className="bg-card rounded-xl sm:rounded-2xl p-4 sm:p-6 shadow-soft border border-border/50 hover:shadow-lg transition-shadow">
            <div className="text-xs sm:text-sm text-muted-foreground mb-1 sm:mb-2">{t('dashboard.totalCost')}</div>
            <div className="text-2xl sm:text-3xl font-bold">
              {formatCost(Number(stats.totalCost), currency)}
            </div>
          </div>
          <div className="bg-card rounded-xl sm:rounded-2xl p-4 sm:p-6 shadow-soft border border-border/50 hover:shadow-lg transition-shadow">
            <div className="text-xs sm:text-sm text-muted-foreground mb-1 sm:mb-2">{t('dashboard.avgCostPerSession')}</div>
            <div className="text-2xl sm:text-3xl font-bold text-accent">
              {stats.avgCost == null
                ? '—'
                : formatCost(Math.round(stats.avgCost), currency)}
            </div>
          </div>
          <div className="bg-card rounded-xl sm:rounded-2xl p-4 sm:p-6 shadow-soft border border-border/50 hover:shadow-lg transition-shadow">
            <div className="text-xs sm:text-sm text-muted-foreground mb-1 sm:mb-2">
              {language === 'fa' ? 'تعداد جلسات' : 'Sessions'}
            </div>
            <div className="text-2xl sm:text-3xl font-bold text-primary">{formatNumber(String(stats.socData.length ? sessions.length : 0))}</div>
          </div>
        </div>

        {/* This month — time-of-use cost breakdown */}
        <Card className="rounded-xl sm:rounded-2xl shadow-lg border-border/50 overflow-hidden mb-6 sm:mb-8">
          <CardHeader className="bg-gradient-to-r from-primary/10 to-accent/10 p-4 sm:p-6">
            <CardTitle className="flex items-center gap-2 text-sm sm:text-base">
              <Bolt className="w-4 h-4 sm:w-5 sm:h-5 text-primary" />
              {t('tou.thisMonth')}
            </CardTitle>
          </CardHeader>
          <CardContent className="pt-4 sm:pt-6 p-4 sm:p-6">
            <TOUBreakdown
              variant="summary"
              peak={stats.touMonth.peak}
              mid={stats.touMonth.mid}
              offpeak={stats.touMonth.offpeak}
            />
          </CardContent>
        </Card>

        {/* Charts */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4 sm:gap-6 mb-6 sm:mb-8">
          {/* Monthly Energy Trend Chart */}
          <Card className="rounded-xl sm:rounded-2xl shadow-lg border-border/50 overflow-hidden">
            <CardHeader className="bg-gradient-to-r from-primary/10 to-accent/10 p-4 sm:p-6">
              <CardTitle className="flex items-center gap-2 text-sm sm:text-base">
                <Zap className="w-4 h-4 sm:w-5 sm:h-5 text-primary" />
                {language === 'fa' ? 'مصرف ماهانه (kWh)' : 'Monthly Energy (kWh)'}
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-4 sm:pt-6 p-3 sm:p-6">
              <ResponsiveContainer width="100%" height={250}>
                <LineChart data={stats.trend} margin={{ top: 5, right: 5, left: -20, bottom: 5 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.3} />
                  <XAxis dataKey="month" tick={{ fontSize: 12 }} stroke="hsl(var(--muted-foreground))" />
                  <YAxis tick={{ fontSize: 12 }} stroke="hsl(var(--muted-foreground))" />
                  <Tooltip
                    contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', padding: '12px' }}
                    formatter={(value: number | string) => formatNumber(parseFloat(String(value)).toFixed(1))}
                  />
                  <Legend wrapperStyle={{ fontSize: '12px' }} iconType="circle" />
                  <Line
                    type="monotone"
                    dataKey="energy"
                    stroke="hsl(var(--primary))"
                    strokeWidth={3}
                    dot={{ fill: 'hsl(var(--primary))', r: 5 }}
                    activeDot={{ r: 7 }}
                    name={language === 'fa' ? 'انرژی (kWh)' : 'Energy (kWh)'}
                    animationDuration={1000}
                  />
                </LineChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>

          {/* Monthly Cost Trend Chart */}
          <Card className="rounded-xl sm:rounded-2xl shadow-lg border-border/50 overflow-hidden">
            <CardHeader className="bg-gradient-to-r from-accent/10 to-primary/10 p-4 sm:p-6">
              <CardTitle className="flex items-center gap-2 text-sm sm:text-base">
                <Bolt className="w-4 h-4 sm:w-5 sm:h-5 text-accent" />
                {t('dashboard.monthlyCost')}
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-4 sm:pt-6 p-3 sm:p-6">
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={stats.trend} margin={{ top: 5, right: 5, left: 5, bottom: 5 }}>
                  <defs>
                    <linearGradient id="colorCost" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="hsl(var(--accent))" stopOpacity={1} />
                      <stop offset="100%" stopColor="hsl(var(--primary))" stopOpacity={0.7} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.3} />
                  <XAxis dataKey="month" tick={{ fontSize: 12 }} stroke="hsl(var(--muted-foreground))" />
                  <YAxis tick={{ fontSize: 12 }} stroke="hsl(var(--muted-foreground))" tickFormatter={(v: number) => formatNumber(v)} />
                  <Tooltip
                    contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', padding: '12px' }}
                    formatter={(value: number | string) => formatCost(Number(value), currency)}
                  />
                  <Legend wrapperStyle={{ fontSize: '12px' }} iconType="circle" />
                  <Bar
                    dataKey="cost"
                    fill="url(#colorCost)"
                    radius={[8, 8, 0, 0]}
                    name={language === 'fa' ? `هزینه (${currency === 'usd' ? 'USD' : currency === 'rial' ? 'ریال' : 'تومان'})` : `Cost (${currency.toUpperCase()})`}
                    animationDuration={1000}
                  />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </div>

        {/* SOC trend — own row */}
        <div className="grid grid-cols-1 gap-4 sm:gap-6 mb-6 sm:mb-8">
          {/* SOC Change Chart */}
          <Card className="rounded-xl sm:rounded-2xl shadow-lg border-border/50 overflow-hidden">
            <CardHeader className="bg-gradient-to-r from-accent/10 to-primary/10 p-4 sm:p-6">
              <CardTitle className="flex items-center gap-2 text-sm sm:text-base">
                <Bolt className="w-4 h-4 sm:w-5 sm:h-5 text-accent" />
                {language === 'fa' ? 'تغییرات درصد شارژ (۱۰ جلسه اخیر)' : 'SOC Changes (Last 10 Sessions)'}
              </CardTitle>
            </CardHeader>
            <CardContent className="pt-4 sm:pt-6 p-3 sm:p-6">
              <ResponsiveContainer width="100%" height={250}>
                <BarChart data={stats.socData} margin={{ top: 5, right: 5, left: -20, bottom: 5 }}>
                  <defs>
                    <linearGradient id="colorBar" x1="0" y1="0" x2="0" y2="1">
                      <stop offset="0%" stopColor="hsl(var(--primary))" stopOpacity={1} />
                      <stop offset="100%" stopColor="hsl(var(--accent))" stopOpacity={0.8} />
                    </linearGradient>
                  </defs>
                  <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.3} />
                  <XAxis dataKey="date" tick={{ fontSize: 11 }} angle={-15} textAnchor="end" height={60} stroke="hsl(var(--muted-foreground))" />
                  <YAxis tick={{ fontSize: 12 }} stroke="hsl(var(--muted-foreground))" />
                  <Tooltip
                    contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', padding: '12px' }}
                    formatter={(value: number | string) => `${formatNumber(String(value))}%`}
                  />
                  <Legend wrapperStyle={{ fontSize: '12px' }} iconType="circle" />
                  <Bar dataKey="socChange" fill="url(#colorBar)" radius={[8, 8, 0, 0]} name={language === 'fa' ? 'تغییر SOC (%)' : 'SOC Change (%)'} animationDuration={1000} />
                </BarChart>
              </ResponsiveContainer>
            </CardContent>
          </Card>
        </div>

        {/* Fleet metrics (server-side lifetime aggregates) + battery health */}
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 sm:gap-4 mb-6 sm:mb-8">
          <div className="bg-card rounded-xl sm:rounded-2xl p-4 sm:p-6 shadow-soft border border-border/50">
            <div className="text-xs sm:text-sm text-muted-foreground mb-1 sm:mb-2">
              {language === 'fa' ? 'مسافت کل (km)' : 'Total distance (km)'}
            </div>
            <div className="text-2xl sm:text-3xl font-bold text-primary">
              {dashboard ? formatNumber(String(dashboard.total_km)) : '—'}
            </div>
          </div>
          <div className="bg-card rounded-xl sm:rounded-2xl p-4 sm:p-6 shadow-soft border border-border/50">
            <div className="text-xs sm:text-sm text-muted-foreground mb-1 sm:mb-2">
              {language === 'fa' ? 'میانگین مصرف (kWh/۱۰۰km)' : 'Avg efficiency (kWh/100km)'}
            </div>
            <div className="text-2xl sm:text-3xl font-bold text-accent">
              {dashboard?.avg_kwh_per_100km == null ? '—' : formatNumber(dashboard.avg_kwh_per_100km.toFixed(1))}
            </div>
          </div>
          <div className="bg-card rounded-xl sm:rounded-2xl p-4 sm:p-6 shadow-soft border border-border/50">
            <div className="text-xs sm:text-sm text-muted-foreground mb-1 sm:mb-2">
              {language === 'fa' ? 'سلامت باتری (SOH)' : 'Battery health (SOH)'}
            </div>
            {latestSoh ? (
              <div className="flex items-baseline gap-2">
                <span className="text-2xl sm:text-3xl font-bold bg-gradient-primary bg-clip-text text-transparent">
                  {formatNumber(latestSoh.soh_pct.toFixed(1))}%
                </span>
                <span className="text-xs text-muted-foreground">
                  {language === 'fa'
                    ? { low: 'اطمینان کم', medium: 'اطمینان متوسط', high: 'اطمینان بالا' }[latestSoh.confidence]
                    : `${latestSoh.confidence} confidence`}
                </span>
              </div>
            ) : (
              <div className="text-sm text-muted-foreground pt-1">
                {language === 'fa' ? 'داده کافی نیست' : 'Not enough data yet'}
              </div>
            )}
          </div>
        </div>

        {/* Battery health trend */}
        <div className="grid grid-cols-1 gap-4 sm:gap-6 mb-6 sm:mb-8">
          <Card className="rounded-xl sm:rounded-2xl shadow-lg border-border/50 overflow-hidden">
            <CardHeader className="bg-gradient-to-r from-primary/10 to-accent/10 p-4 sm:p-6">
              <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-2">
                <CardTitle className="flex items-center gap-2 text-sm sm:text-base">
                  <BatteryCharging className="w-4 h-4 sm:w-5 sm:h-5 text-primary" />
                  {language === 'fa' ? 'روند سلامت باتری' : 'Battery Health Trend'}
                </CardTitle>
                {cars.length > 1 && (
                  <Select value={batteryCarId} onValueChange={setPickedCarId}>
                    <SelectTrigger className="w-full sm:w-48 h-8 text-xs">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      {cars.map((c) => (
                        <SelectItem key={c.id} value={c.id}>{c.name}</SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>
            </CardHeader>
            <CardContent className="pt-4 sm:pt-6 p-3 sm:p-6">
              {sohTrend.length === 0 ? (
                <div className="text-center text-sm text-muted-foreground py-12">
                  {!batteryCar
                    ? (language === 'fa' ? 'خودرویی ثبت نشده است' : 'No car registered yet')
                    : (language === 'fa'
                        ? 'پس از ثبت چند جلسه شارژ با اختلاف شارژ کافی، روند سلامت باتری اینجا نمایش داده می‌شود'
                        : 'Log a few charging sessions with a large SOC swing to see the battery-health trend here')}
                </div>
              ) : (
                <ResponsiveContainer width="100%" height={250}>
                  <LineChart data={sohTrend} margin={{ top: 5, right: 5, left: -20, bottom: 5 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="hsl(var(--border))" opacity={0.3} />
                    <XAxis dataKey="date" tick={{ fontSize: 12 }} stroke="hsl(var(--muted-foreground))" />
                    <YAxis domain={[0, 100]} tick={{ fontSize: 12 }} stroke="hsl(var(--muted-foreground))" tickFormatter={(v: number) => `${v}%`} />
                    <Tooltip
                      contentStyle={{ backgroundColor: 'hsl(var(--card))', border: '1px solid hsl(var(--border))', borderRadius: '12px', padding: '12px' }}
                      formatter={(value: number | string) => `${formatNumber(parseFloat(String(value)).toFixed(1))}%`}
                    />
                    <Legend wrapperStyle={{ fontSize: '12px' }} iconType="circle" />
                    <Line
                      type="monotone"
                      dataKey="soh"
                      stroke="hsl(var(--primary))"
                      strokeWidth={3}
                      dot={{ fill: 'hsl(var(--primary))', r: 5 }}
                      activeDot={{ r: 7 }}
                      name={language === 'fa' ? 'سلامت باتری (%)' : 'SOH (%)'}
                      animationDuration={1000}
                    />
                  </LineChart>
                </ResponsiveContainer>
              )}
            </CardContent>
          </Card>
        </div>
      </main>
    </div>
  );
}
