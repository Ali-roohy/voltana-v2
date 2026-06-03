import { useEffect, type ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { Loader2 } from "lucide-react";
import { useAuth } from "@/hooks/useAuth";
import { useMe } from "@/features/auth/hooks";

// AdminRoute gates the admin area. Not-logged-in → /auth; logged-in non-admin → /.
// This is a convenience guard only: GET /v1/me sources is_admin, and the API's
// AdminOnly middleware re-checks on every write, so it remains the real boundary.
export function AdminRoute({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth();
  const navigate = useNavigate();
  // Only query identity once the auth session has settled and a user exists.
  const { data: me, isLoading: meLoading, isError } = useMe(!loading && !!user);

  const denied = !loading && !user ? "/auth" : isError || (me && !me.is_admin) ? "/" : null;

  useEffect(() => {
    if (denied) navigate(denied, { replace: true });
  }, [denied, navigate]);

  if (loading || (user && meLoading)) {
    return (
      <div className="min-h-screen flex items-center justify-center" dir="rtl">
        <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
      </div>
    );
  }

  if (denied || !me?.is_admin) return null; // redirecting

  return <>{children}</>;
}
