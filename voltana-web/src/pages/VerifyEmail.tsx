import { useEffect, useRef, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { verifyEmail } from "@/features/auth/api";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useTranslation } from "react-i18next";
import { Zap, CheckCircle2, XCircle, Loader2 } from "lucide-react";

type Status = "loading" | "success" | "error";

export default function VerifyEmail() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [params] = useSearchParams();
  const [status, setStatus] = useState<Status>("loading");
  const ran = useRef(false);

  // Verification is single-use: fire exactly once on mount (action bootstrap,
  // not data fetching — same pattern as the session-restore in useAuth).
  useEffect(() => {
    if (ran.current) return;
    ran.current = true;
    const token = params.get("token");
    if (!token) {
      setStatus("error");
      return;
    }
    verifyEmail(token)
      .then(() => setStatus("success"))
      .catch(() => setStatus("error"));
  }, [params]);

  return (
    <div className="min-h-screen flex items-center justify-center app-page-bg-gradient p-4">
      <Card className="w-full max-w-md shadow-soft">
        <CardHeader className="text-center">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-primary mb-4 shadow-glow mx-auto">
            <Zap className="w-8 h-8 text-white" />
          </div>
          <CardTitle className="text-2xl">{t("auth.verifyEmail")}</CardTitle>
        </CardHeader>
        <CardContent className="text-center space-y-4">
          {status === "loading" && (
            <div className="flex flex-col items-center gap-3 text-muted-foreground py-4">
              <Loader2 className="w-8 h-8 animate-spin" />
              <p>{t("auth.verifying")}</p>
            </div>
          )}
          {status === "success" && (
            <div className="flex flex-col items-center gap-3 py-2">
              <CheckCircle2 className="w-12 h-12 text-green-500" />
              <p>{t("auth.verifiedSuccess")}</p>
              <Button className="w-full" onClick={() => navigate("/auth")}>
                {t("auth.login")}
              </Button>
            </div>
          )}
          {status === "error" && (
            <div className="flex flex-col items-center gap-3 py-2">
              <XCircle className="w-12 h-12 text-destructive" />
              <p>{t("auth.verifyFailed")}</p>
              <Button variant="outline" className="w-full" onClick={() => navigate("/auth")}>
                {t("auth.backToLogin")}
              </Button>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
