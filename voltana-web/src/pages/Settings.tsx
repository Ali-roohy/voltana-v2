import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { useLanguage } from "@/contexts/LanguageContext";
import { useAuth } from "@/hooks/useAuth";
import { ArrowLeft, Zap, Link2, CheckCircle2, Palette } from "lucide-react";
import { AdminOTPPanel } from "@/features/settings/AdminOTPPanel";
import { AdminSystemSettings } from "@/features/settings/AdminSystemSettings";
import { useSettings, useUpdateSettings } from "@/features/settings/hooks";
import { useMe } from "@/features/auth/hooks";
import { useBotLink } from "@/features/account/hooks";
import { useAppTheme } from "@/contexts/ThemeContext";
import { useAppFont } from "@/contexts/FontContext";
import { THEMES } from "@/lib/themes";
import { FONTS } from "@/lib/fonts";
import { cn } from "@/lib/utils";
import { toast } from "sonner";

export default function Settings() {
  const { language, setLanguage } = useLanguage();
  const { user, loading: authLoading, signOut } = useAuth();
  const navigate = useNavigate();
  const isRTL = language === 'fa';
  const { themeId, setTheme } = useAppTheme();
  const { fontId, setFont } = useAppFont();

  const [ratePeak, setRatePeak] = useState<string>('5000');
  const [rateMid, setRateMid] = useState<string>('3000');
  const [rateOffpeak, setRateOffpeak] = useState<string>('1500');
  const [currency, setCurrency] = useState<'toman' | 'rial' | 'usd'>('toman');

  useEffect(() => {
    if (!authLoading && !user) {
      navigate('/auth');
    }
  }, [user, authLoading, navigate]);

  // Fetch user settings (GET auto-creates a default row server-side)
  const { data: settings, isLoading } = useSettings();
  const { data: me } = useMe(!!user);
  const botLinkMutation = useBotLink();
  const [botLinks, setBotLinks] = useState<{ bale_url?: string; telegram_url?: string } | null>(null);

  useEffect(() => {
    if (settings) {
      setRatePeak(settings.peak_rate?.toString() || '5000');
      setRateMid(settings.mid_rate?.toString() || '3000');
      setRateOffpeak(settings.offpeak_rate?.toString() || '1500');
      setCurrency(settings.currency ?? 'toman');
    }
  }, [settings]);

  // Save settings. PUT is a full replace, so we preserve the existing
  // default_car_id (otherwise saving rates would clear it).
  const saveMutation = useUpdateSettings();
  const handleSave = () =>
    saveMutation.mutate(
      {
        default_car_id: settings?.default_car_id ?? null,
        peak_rate: parseFloat(ratePeak) || 5000,
        mid_rate: parseFloat(rateMid) || 3000,
        offpeak_rate: parseFloat(rateOffpeak) || 1500,
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
    <div className="min-h-screen bg-gradient-to-br from-background via-secondary to-background">
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

          {/* Currency selector card */}
          <Card>
            <CardHeader>
              <CardTitle>{isRTL ? 'واحد ارزی' : 'Currency'}</CardTitle>
              <CardDescription>
                {isRTL
                  ? 'واحد نمایش مبالغ را انتخاب کنید (ذخیره در حساب)'
                  : 'Currency unit for cost display (saved to account)'}
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex gap-2">
                {(['toman', 'rial', 'usd'] as const).map((c) => (
                  <Button
                    key={c}
                    variant={currency === c ? 'default' : 'outline'}
                    className="flex-1"
                    onClick={() => {
                      setCurrency(c);
                      saveMutation.mutate({
                        default_car_id: settings?.default_car_id ?? null,
                        peak_rate: parseFloat(ratePeak) || 5000,
                        mid_rate: parseFloat(rateMid) || 3000,
                        offpeak_rate: parseFloat(rateOffpeak) || 1500,
                        currency: c,
                      });
                    }}
                  >
                    {c === 'toman' ? 'تومان' : c === 'rial' ? 'ریال' : 'USD'}
                  </Button>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Bot link card */}
          <Card>
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Link2 className="w-5 h-5" />
                اتصال بله / تلگرام
              </CardTitle>
              <CardDescription>
                حساب خود را به بله یا تلگرام متصل کنید تا بتوانید با کد پیامکی وارد شوید.
              </CardDescription>
            </CardHeader>
            <CardContent className="space-y-3">
              {me?.bale_linked && (
                <div className="flex items-center gap-2 text-sm text-green-600 dark:text-green-400">
                  <CheckCircle2 className="w-4 h-4" />
                  بله متصل است
                  {me.phone && <span className="font-mono text-xs text-muted-foreground" dir="ltr">({me.phone})</span>}
                </div>
              )}
              {me?.telegram_linked && (
                <div className="flex items-center gap-2 text-sm text-blue-600 dark:text-blue-400">
                  <CheckCircle2 className="w-4 h-4" />
                  تلگرام متصل است
                </div>
              )}
              {botLinks ? (
                <div className="space-y-2">
                  <p className="text-sm text-muted-foreground">
                    روی لینک زیر کلیک کنید تا ربات باز شود. شماره تلفن را به اشتراک بگذارید تا اتصال کامل شود.
                  </p>
                  {botLinks.bale_url && (
                    <Button asChild variant="outline" className="w-full" size="sm">
                      <a href={botLinks.bale_url} target="_blank" rel="noopener noreferrer">
                        🟣 اتصال از طریق بله
                      </a>
                    </Button>
                  )}
                  {botLinks.telegram_url && (
                    <Button asChild variant="outline" className="w-full" size="sm">
                      <a href={botLinks.telegram_url} target="_blank" rel="noopener noreferrer">
                        🔵 اتصال از طریق تلگرام
                      </a>
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    size="sm"
                    className="w-full text-xs"
                    onClick={() => setBotLinks(null)}
                  >
                    بستن
                  </Button>
                </div>
              ) : (
                <Button
                  className="w-full"
                  variant="outline"
                  disabled={botLinkMutation.isPending}
                  onClick={() =>
                    botLinkMutation.mutate(undefined, {
                      onSuccess: (data) => setBotLinks(data),
                      onError: () => toast.error('خطا در دریافت لینک اتصال'),
                    })
                  }
                >
                  {botLinkMutation.isPending ? 'در حال دریافت لینک...' : 'دریافت لینک اتصال'}
                </Button>
              )}
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
                  onClick={() => setLanguage("fa")}
                  variant={language === "fa" ? "default" : "outline"}
                  className="flex-1"
                >
                  فارسی
                </Button>
                <Button
                  onClick={() => setLanguage("en")}
                  variant={language === "en" ? "default" : "outline"}
                  className="flex-1"
                >
                  English
                </Button>
              </div>
            </CardContent>
          </Card>

          {/* Admin OTP test panel — visible only to admins */}
          {me?.is_admin && <AdminOTPPanel me={me} />}

          {/* Admin system settings — OTP delivery method */}
          {me?.is_admin && <AdminSystemSettings />}
        </div>
      </main>
    </div>
  );
}
