import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useLanguage } from "@/contexts/LanguageContext";
import { useAuth } from "@/hooks/useAuth";
import { ArrowLeft, Zap, Palette, Download } from "lucide-react";
import { usePWAInstall } from "@/lib/pwa-install";
import { BackgroundPicker } from "@/features/settings/BackgroundPicker";
import { BackupRestoreCard } from "@/features/settings/BackupRestoreCard";
import { NotificationsCard } from "@/features/settings/NotificationsCard";
import { DeleteAccountCard } from "@/features/settings/DeleteAccountCard";
import { AdminOTPPanel } from "@/features/settings/AdminOTPPanel";
import { AdminSystemSettings } from "@/features/settings/AdminSystemSettings";
import { useSettings, useUpdateSettings } from "@/features/settings/hooks";
import { useMe } from "@/features/auth/hooks";
import { useAppTheme } from "@/contexts/ThemeContext";
import { useAppFont } from "@/contexts/FontContext";
import { THEMES } from "@/lib/themes";
import { FONTS } from "@/lib/fonts";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

export default function Settings() {
  const { language } = useLanguage();
  const { user, loading: authLoading, signOut } = useAuth();
  const navigate = useNavigate();
  const isRTL = language === 'fa';
  const { themeId, setTheme } = useAppTheme();
  const { fontId, setFont } = useAppFont();

  const [ratePeak, setRatePeak] = useState<string>('2000');
  const [rateMid, setRateMid] = useState<string>('1000');
  const [rateOffpeak, setRateOffpeak] = useState<string>('500');
  const currency = 'toman' as const;

  useEffect(() => {
    if (!authLoading && !user) {
      navigate('/auth');
    }
  }, [user, authLoading, navigate]);

  // Fetch user settings (GET auto-creates a default row server-side)
  const { data: settings, isLoading } = useSettings();
  const { data: me } = useMe(!!user);
  const { canInstall, install } = usePWAInstall();

  useEffect(() => {
    if (settings) {
      setRatePeak(settings.peak_rate?.toString() || '2000');
      setRateMid(settings.mid_rate?.toString() || '1000');
      setRateOffpeak(settings.offpeak_rate?.toString() || '500');
    }
  }, [settings]);

  // Save settings. PUT is a full replace, so we preserve the existing
  // default_car_id (otherwise saving rates would clear it).
  const saveMutation = useUpdateSettings();
  const handleSave = () =>
    saveMutation.mutate(
      {
        default_car_id: settings?.default_car_id ?? null,
        peak_rate: parseFloat(ratePeak) || 2000,
        mid_rate: parseFloat(rateMid) || 1000,
        offpeak_rate: parseFloat(rateOffpeak) || 500,
        currency,
      },
      {
        onSuccess: () => toast.success(isRTL ? 'تنظیمات با موفقیت ذخیره شد' : 'Settings saved successfully'),
        onError: (error) => {
          console.error('Error saving settings:', error);
          toast.error(isRTL ? 'خطا در ذخیره تنظیمات' : 'Error saving settings');
        },
      },
    );

  const texts = {
    title: isRTL ? 'تنظیمات' : 'Settings',
    electricityRates: isRTL ? 'نرخ‌های برق' : 'Electricity Rates',
    electricityDesc: isRTL ? 'نرخ مصرف برق را به تومان وارد کنید' : 'Enter electricity rates in Toman',
    peakRate: isRTL ? 'نرخ اوج بار (تومان/kWh)' : 'Peak Rate (Toman/kWh)',
    midRate: isRTL ? 'نرخ میان‌باری (تومان/kWh)' : 'Mid Rate (Toman/kWh)',
    offpeakRate: isRTL ? 'نرخ کم‌باری (تومان/kWh)' : 'Off-Peak Rate (Toman/kWh)',
    language: isRTL ? 'زبان' : 'Language',
    languageDesc: isRTL ? 'زبان نمایش برنامه را انتخاب کنید' : 'Select application display language',
    save: isRTL ? 'ذخیره تغییرات' : 'Save Changes',
    back: isRTL ? 'بازگشت' : 'Back',
    logout: isRTL ? 'خروج' : 'Logout',
  };

  if (authLoading || isLoading) {
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
    <div className="min-h-screen app-page-bg-gradient">
      {/* Header */}
      <header className="border-b bg-card/50 backdrop-blur-sm">
        <div className="container mx-auto px-3 sm:px-4 py-3 sm:py-4 flex items-center justify-between">
          <div className="flex items-center gap-2 sm:gap-3">
            <Button
              variant="ghost"
              size="icon"
              onClick={() => navigate('/')}
              className="hover:bg-muted h-8 w-8 sm:h-10 sm:w-10"
            >
              <ArrowLeft className={`w-4 h-4 sm:w-5 sm:h-5 ${isRTL ? 'rotate-180' : ''}`} />
            </Button>
            <h1 className="text-xl sm:text-2xl font-bold">{texts.title}</h1>
          </div>
          
          <Button variant="outline" onClick={signOut} size="sm" className="text-xs sm:text-sm">
            {texts.logout}
          </Button>
        </div>
      </header>

      {/* Main Content */}
      <main className="container mx-auto px-3 sm:px-4 py-4 sm:py-8 max-w-2xl">
        <div className="space-y-6">
          {/* PWA install (FEAT-2) — only when the browser offered an install prompt */}
          {canInstall && (
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Download className="w-5 h-5" />
                  نصب اپلیکیشن
                </CardTitle>
                <CardDescription>
                  ولتانا را مانند یک اپلیکیشن واقعی روی دستگاه خود نصب کنید
                </CardDescription>
              </CardHeader>
              <CardContent>
                <Button
                  className="w-full"
                  onClick={async () => {
                    const ok = await install();
                    if (ok) toast.success('اپلیکیشن نصب شد');
                  }}
                >
                  نصب اپلیکیشن
                </Button>
              </CardContent>
            </Card>
          )}

          {/* Electricity Rates Card */}
          <Card>
            <CardHeader>
              <CardTitle>{texts.electricityRates}</CardTitle>
              <CardDescription>{texts.electricityDesc}</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="ratePeak">{texts.peakRate}</Label>
                <Input
                  id="ratePeak"
                  type="number"
                  value={ratePeak}
                  onChange={(e) => setRatePeak(e.target.value)}
                  placeholder="5000"
                  dir={isRTL ? 'rtl' : 'ltr'}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="rateMid">{texts.midRate}</Label>
                <Input
                  id="rateMid"
                  type="number"
                  value={rateMid}
                  onChange={(e) => setRateMid(e.target.value)}
                  placeholder="3000"
                  dir={isRTL ? 'rtl' : 'ltr'}
                />
              </div>

              <div className="space-y-2">
                <Label htmlFor="rateOffpeak">{texts.offpeakRate}</Label>
                <Input
                  id="rateOffpeak"
                  type="number"
                  value={rateOffpeak}
                  onChange={(e) => setRateOffpeak(e.target.value)}
                  placeholder="1500"
                  dir={isRTL ? 'rtl' : 'ltr'}
                />
              </div>

              <Button
                onClick={handleSave}
                disabled={saveMutation.isPending}
                className="w-full"
              >
                {saveMutation.isPending ? (isRTL ? 'در حال ذخیره...' : 'Saving...') : texts.save}
              </Button>
            </CardContent>
          </Card>

          {/* Currency — fixed to Toman */}
          <Card>
            <CardHeader>
              <CardTitle>واحد ارزی</CardTitle>
              <CardDescription>واحد نمایش مبالغ</CardDescription>
            </CardHeader>
            <CardContent>
              <Button variant="default" className="w-full" disabled>
                تومان
              </Button>
            </CardContent>
          </Card>

          {/* Theme selector card */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Palette className="w-5 h-5" />
                {isRTL ? 'تم رنگی' : 'Color Theme'}
              </CardTitle>
              <CardDescription>
                {isRTL ? 'رنگ‌بندی رابط کاربری را انتخاب کنید' : 'Choose the app color scheme'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid grid-cols-4 gap-3">
                {THEMES.map((t) => {
                  const isActive = themeId === t.id;
                  return (
                    <button
                      key={t.id}
                      onClick={() => setTheme(t.id)}
                      className="flex flex-col items-center gap-1.5 group"
                      title={isRTL ? t.nameFa : t.nameEn}
                    >
                      {/* Color swatch circle */}
                      <div
                        className={cn(
                          'w-10 h-10 rounded-full border-2 transition-all',
                          isActive
                            ? 'border-foreground scale-110 shadow-md'
                            : 'border-transparent group-hover:border-muted-foreground group-hover:scale-105',
                        )}
                        style={{
                          background: `linear-gradient(135deg, ${t.swatchPrimary}, ${t.swatchAccent})`,
                        }}
                      />
                      <span
                        className={cn(
                          'text-[10px] text-center leading-tight transition-colors',
                          isActive ? 'text-foreground font-medium' : 'text-muted-foreground',
                        )}
                      >
                        {isRTL ? t.nameFa : t.nameEn}
                      </span>
                    </button>
                  );
                })}
              </div>
            </CardContent>
          </Card>

          {/* Background designer card (FEAT-3) */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Palette className="w-5 h-5" />
                پس‌زمینه
              </CardTitle>
              <CardDescription>
                رنگ ثابت، گرادیان یا الگو — مستقل از تم رنگی
              </CardDescription>
            </CardHeader>
            <CardContent>
              <BackgroundPicker />
            </CardContent>
          </Card>

          {/* Font selector card */}
          <Card>
            <CardHeader>
              <CardTitle>{isRTL ? 'فونت' : 'Font'}</CardTitle>
              <CardDescription>
                {isRTL ? 'فونت نمایش متن را انتخاب کنید' : 'Choose the display typeface'}
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-2">
              {FONTS.map((f) => {
                const isActive = fontId === f.id;
                return (
                  <button
                    key={f.id}
                    onClick={() => setFont(f.id)}
                    className={cn(
                      'w-full flex items-center justify-between px-3 py-2.5 rounded-md border text-sm transition-all',
                      isActive
                        ? 'border-primary bg-primary/10 text-foreground'
                        : 'border-transparent bg-muted/50 text-muted-foreground hover:bg-muted hover:text-foreground',
                    )}
                  >
                    <span style={{ fontFamily: f.stack }} className="text-base">
                      {isRTL ? f.previewFa : f.previewEn}
                    </span>
                    <span className="text-xs">{isRTL ? f.nameFa : f.nameEn}</span>
                  </button>
                );
              })}
            </CardContent>
          </Card>

          {/* Language Settings Card */}
          <Card>
            <CardHeader>
              <CardTitle>{texts.language}</CardTitle>
              <CardDescription>{texts.languageDesc}</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2">
                <Button
                  variant="default"
                  className="flex-1"
                  disabled
                >
                  فارسی
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Admin OTP test panel — visible only to admins */}
          {me?.is_admin && <AdminOTPPanel me={me} />}

          {/* Admin system settings — OTP delivery method */}
          {me?.is_admin && <AdminSystemSettings />}

          {/* Notifications (TASK-0039) */}
          <NotificationsCard isAdmin={!!me?.is_admin} />

          {/* Backup & restore (FEAT-4) — end of Settings */}
          <BackupRestoreCard />

          {/* Self-delete (FEAT-5) — hidden for the permanent admin; the API
              enforces the last-admin guard regardless */}
          {me && !me.is_admin && <DeleteAccountCard />}
        </div>
      </main>
    </div>
  );
}
