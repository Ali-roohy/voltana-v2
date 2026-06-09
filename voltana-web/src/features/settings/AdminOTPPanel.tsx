import { useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { type Me } from "@/features/auth/api";
import { testOTPDelivery } from "./api";

type Platform = "bale" | "telegram" | "email";
type CardState = { status: "idle" | "loading" | "success" | "error"; message: string };

const INITIAL: CardState = { status: "idle", message: "" };

const NOT_LINKED_MSG: Record<Platform, string> = {
  bale: "بله متصل نشده — قبل از تست باید در تنظیمات «اتصال بله» را انجام دهید",
  telegram: "تلگرام متصل نشده — قبل از تست باید در تنظیمات «اتصال تلگرام» را انجام دهید",
  email: "ایمیل تنظیم نشده — برای تست ایمیل باید آدرس ایمیل روی حساب شما ثبت باشد",
};

interface PlatformCardProps {
  icon: string;
  label: string;
  linked: boolean;
  linkInfo: string;
  platform: Platform;
}

function PlatformCard({ icon, label, linked, linkInfo, platform }: PlatformCardProps) {
  const [state, setState] = useState<CardState>(INITIAL);

  const handleSend = async () => {
    if (state.status === "loading") return;

    if (!linked) {
      toast.warning(NOT_LINKED_MSG[platform]);
      return;
    }

    setState({ status: "loading", message: "" });
    try {
      const res = await testOTPDelivery(platform);
      if (res.success) {
        setState({ status: "success", message: res.message });
      } else {
        setState({ status: "error", message: res.message });
      }
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "خطای ناشناخته";
      setState({ status: "error", message: msg });
    }
  };

  return (
    <Card>
      <CardHeader className="pb-2">
        <CardTitle className="text-base flex items-center gap-2">
          <span>{icon}</span>
          <span>{label}</span>
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <p className="text-sm text-muted-foreground">{linkInfo}</p>

        <Button
          size="sm"
          variant="outline"
          disabled={state.status === "loading"}
          onClick={state.status !== "idle" ? () => setState(INITIAL) : handleSend}
          className="w-full"
        >
          {state.status === "loading" ? (
            <>
              <Loader2 className="w-4 h-4 ml-2 animate-spin" />
              در حال ارسال...
            </>
          ) : state.status !== "idle" ? (
            "ارسال مجدد"
          ) : (
            "ارسال تست"
          )}
        </Button>

        {state.status === "success" && (
          <p className="text-sm text-green-600 dark:text-green-400 flex items-center gap-1">
            <span>✅</span>
            <span>کد ارسال شد</span>
          </p>
        )}
        {state.status === "error" && (
          <p className="text-sm text-red-600 dark:text-red-400 flex items-center gap-1">
            <span>❌</span>
            <span>خطا: {state.message}</span>
          </p>
        )}
      </CardContent>
    </Card>
  );
}

interface AdminOTPPanelProps {
  me: Me;
}

export function AdminOTPPanel({ me }: AdminOTPPanelProps) {
  return (
    <div className="space-y-4">
      <div>
        <h2 className="text-lg font-semibold">🔧 تست OTP</h2>
        <p className="text-sm text-muted-foreground mt-1">
          ارسال کد تست به کانال‌های متصل — فقط برای ادمین
        </p>
      </div>

      <PlatformCard
        icon="🟣"
        label="بله"
        platform="bale"
        linked={me.bale_linked}
        linkInfo={me.bale_linked ? `متصل${me.phone ? ` · ${me.phone}` : ""}` : "بله متصل نیست"}
      />

      <PlatformCard
        icon="✈️"
        label="تلگرام"
        platform="telegram"
        linked={me.telegram_linked}
        linkInfo={me.telegram_linked ? "متصل" : "تلگرام متصل نیست"}
      />

      <PlatformCard
        icon="✉️"
        label="ایمیل"
        platform="email"
        linked={!!me.email}
        linkInfo={me.email ? me.email : "ایمیل تنظیم نشده"}
      />
    </div>
  );
}
