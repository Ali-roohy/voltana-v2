import { useState, useEffect } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card";
import { useSystemSettings, useUpdateSystemSettings } from "@/features/admin-system/hooks";
import type { SystemSettings } from "@/features/admin-system/api";

export function AdminSystemSettings() {
  const { data: settings, isLoading } = useSystemSettings();
  const updateSettings = useUpdateSystemSettings();
  const [selected, setSelected] = useState<SystemSettings["otp_delivery_method"]>("deeplink");

  useEffect(() => {
    if (settings) setSelected(settings.otp_delivery_method);
  }, [settings]);

  const handleSave = () => {
    updateSettings.mutate(
      { otp_delivery_method: selected },
      {
        onSuccess: () => toast.success("تنظیمات ذخیره شد"),
        onError: () => toast.error("خطا در ذخیره تنظیمات"),
      },
    );
  };

  if (isLoading) return null;

  const isDirty = settings?.otp_delivery_method !== selected;

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
