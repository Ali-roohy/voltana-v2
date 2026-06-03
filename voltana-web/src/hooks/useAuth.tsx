import { useState, useEffect, useCallback } from "react";
import { useNavigate } from "react-router-dom";
import { authStore, currentUser, type AuthUser } from "@/lib/auth-store";
import { logout as apiLogout, restoreSessionOnce } from "@/features/auth/api";

// useAuth backs the app's auth gate with the Go JWT flow: access token in memory
// (auth-store), refresh token in an httpOnly cookie. Same return shape as before
// ({ user, session, loading, signOut }) so existing consumers keep working.
export function useAuth() {
  const [user, setUser] = useState<AuthUser | null>(() => currentUser());
  const [loading, setLoading] = useState(true);
  const navigate = useNavigate();

  useEffect(() => {
    const unsubscribe = authStore.subscribe(() => setUser(currentUser()));
    let active = true;
    (async () => {
      // On first load there is no in-memory token — try to restore from the
      // refresh cookie before deciding the user is logged out.
      if (!authStore.getToken()) {
        await restoreSessionOnce();
      }
      if (active) {
        setUser(currentUser());
        setLoading(false);
      }
    })();
    return () => {
      active = false;
      unsubscribe();
    };
  }, []);

  const signOut = useCallback(async () => {
    await apiLogout();
    navigate("/auth");
  }, [navigate]);

  // `session` is kept for API compatibility with the old Supabase shape.
  const session = user ? { user } : null;
  return { user, session, loading, signOut };
}
