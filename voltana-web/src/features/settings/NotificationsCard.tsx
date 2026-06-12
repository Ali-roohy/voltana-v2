import { useState } from "react";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Bell, BellOff, Loader2 } from "lucide-react";
import { toast } from "sonner";
import { usePushNotifications } from "@/features/push/hooks";
import { testPush } from "@/features/push/api";

// «اعلان‌ها» (TASK-0039): web-push enable/disable + admin test-send. Hidden
// entirely when the browser can't do push or the server has no VAPID keys.
export function NotificationsCard({ isAdmin }: { isAdmin: boolean }) {
  const { state, enable, disable } = usePushNotifications();
  const [testing, setTesting] = useState(false);

  if (state === "unsupported" || state === "disabled") return null;

  const handleTest = async () => {
    setTesting(true);
    try {
      const res = await testPush();
      if (res.success) toast.success("اعلان تست ارسال شد");
      else toast.error(res.message);
    } catch {
      toast.error("خطا در ارسال اعلان تست");
    } finally {
      setTesting(false);
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Bell className="w-5 h-5" />
          اعلان‌ها
        </CardTitle>
        <CardDescription>
          هشدار افت سلامت باتری را روی همین دستگاه دریافت کنید
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-3">
        {state === "denied" ? (
          <p className="text-sm text-muted-foreground">
            اعلان‌ها در تنظیمات مرورگر مسدود شده‌اند — برای فعال‌سازی، دسترسی اعلان این سایت را در
            مرورگر آزاد کنید.
          </p>
        ) : (
          <Button
            variant={state === "on" ? "outline" : "default"}
            className="w-full gap-2"
            disabled={state === "loading"}
            onClick={async () => {
              if (state === "on") {
                await disable();
                toast.success("اعلان‌ها غیرفعال شد");
              } else {
                const ok = await enable();
                if (ok) toast.success("اعلان‌ها فعال شد");
              }
            }}
          >
            {state === "loading" ? (
              <Loader2 className="w-4 h-4 animate-spin" />
            ) : state === "on" ? (
              <>
                <BellOff className="w-4 h-4" />
                غیرفعال‌سازی اعلان‌ها
              </>
            ) : (
              <>
                <Bell className="w-4 h-4" />
                فعال‌سازی اعلان‌ها
              </>
            )}
          </Button>
        )}

        {isAdmin && state === "on" && (
          <Button size="sm" variant="outline" className="w-full" disabled={testing} onClick={handleTest}>
            {testing ? "در حال ارسال..." : "ارسال اعلان تست (ادمین)"}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}
