import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useSystemSettings, useUpdateSystemSettings } from "@/features/admin-system/hooks";
import type { SystemSettings } from "@/features/admin-system/api";

export function AdminSystemSettings() {
  const { data: settings, isLoading } = useSystemSettings();
  const updateSettings = useUpdateSystemSettings();
  const [selected, setSelected] = useState<SystemSettings["otp_delivery_method"]>("deeplink");
  const [defPeak, setDefPeak] = useState("2000");
  const [defMid, setDefMid] = useState("1000");
  const [defOffpeak, setDefOffpeak] = useState("500");

  useEffect(() => {
    if (settings) {
      setSelected(settings.otp_delivery_method);
      setDefPeak(String(settings.default_peak_rate ?? 2000));
      setDefMid(String(settings.default_mid_rate ?? 1000));
      setDefOffpeak(String(settings.default_offpeak_rate ?? 500));
    }
  }, [settings]);

  const handleSave = () => {
    updateSettings.mutate(
      {
        otp_delivery_method: selected,
        default_peak_rate: parseFloat(defPeak) || 0,
        default_mid_rate: parseFloat(defMid) || 0,
        default_offpeak_rate: parseFloat(defOffpeak) || 0,
      },
      {
        onSuccess: () => toast.success("تنظیمات ذخیره شد"),
        onError: () => toast.error("خطا در ذخیره تنظیمات"),
      },
    );
  };

  if (isLoading) return null;

  const isDirty =
    settings?.otp_delivery_method !== selected ||
    String(settings?.default_peak_rate ?? "") !== defPeak ||
    String(settings?.default_mid_rate ?? "") !== defMid ||
    String(settings?.default_offpeak_rate ?? "") !== defOffpeak;

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <span>⚙️</span> روش ارسال OTP
        </CardTitle>
        <CardDescription>
          نحوه تحویل کد OTP به کاربران جدید را تنظیم کنید
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="space-y-3">
          <label
            className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
              selected === "deeplink" ? "border-primary bg-primary/5" : "border-border"
            }`}
          >
            <input
              type="radio"
              name="otp_delivery"
              value="deeplink"
              checked={selected === "deeplink"}
              onChange={() => setSelected("deeplink")}
              className="mt-1"
            />
            <div>
              <p className="font-medium text-sm">لینک مستقیم (پیشنهادی)</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                کاربر بدون اشتراک‌گذاری مخاطب، از طریق لینک مستقیم کد OTP دریافت می‌کند
              </p>
            </div>
          </label>

          <label
            className={`flex items-start gap-3 p-3 rounded-lg border cursor-pointer transition-colors ${
              selected === "contact_share" ? "border-primary bg-primary/5" : "border-border"
            }`}
          >
            <input
              type="radio"
              name="otp_delivery"
              value="contact_share"
              checked={selected === "contact_share"}
              onChange={() => setSelected("contact_share")}
              className="mt-1"
            />
            <div>
              <p className="font-medium text-sm">اشتراک‌گذاری مخاطب (قدیمی)</p>
              <p className="text-xs text-muted-foreground mt-0.5">
                کاربر باید ابتدا /start را در ربات اجرا کرده و شماره تلفن را به اشتراک بگذارد
              </p>
            </div>
          </label>
        </div>

        <div className="space-y-3 pt-2 border-t">
          <div>
            <p className="font-medium text-sm">نرخ‌های پیش‌فرض برق (تومان/کیلووات‌ساعت)</p>
            <p className="text-xs text-muted-foreground mt-0.5">
              برای کاربران جدید کپی می‌شود — نرخ کاربران فعلی و هزینه جلسات قبلی تغییر نمی‌کند
            </p>
          </div>
          <div className="grid grid-cols-3 gap-2">
            <div className="space-y-1">
              <Label htmlFor="def-peak" className="text-xs">اوج بار</Label>
              <Input id="def-peak" type="number" dir="ltr" value={defPeak} onChange={(e) => setDefPeak(e.target.value)} />
            </div>
            <div className="space-y-1">
              <Label htmlFor="def-mid" className="text-xs">میان‌باری</Label>
              <Input id="def-mid" type="number" dir="ltr" value={defMid} onChange={(e) => setDefMid(e.target.value)} />
            </div>
            <div className="space-y-1">
              <Label htmlFor="def-offpeak" className="text-xs">کم‌باری</Label>
              <Input id="def-offpeak" type="number" dir="ltr" value={defOffpeak} onChange={(e) => setDefOffpeak(e.target.value)} />
            </div>
          </div>
        </div>

        <Button
          className="w-full"
          onClick={handleSave}
          disabled={updateSettings.isPending || !isDirty}
        >
          {updateSettings.isPending ? "در حال ذخیره..." : "ذخیره"}
        </Button>
      </CardContent>
    </Card>
  );
}
