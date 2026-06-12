import { QueryClient } from "@tanstack/react-query";
import { authStore, currentUser } from "./auth-store";

export const queryClient = new QueryClient();

// Cached queries — notably ["me"], which gates AdminRoute/Header — must never
// outlive the identity they were fetched for. Clear the cache whenever the JWT
// subject changes: logout, login as a different user, or re-login after an
// out-of-band is_admin change. The silent refresh keeps the same sub, so it
// never triggers a clear.
let lastSub: string | null = currentUser()?.id ?? null;
authStore.subscribe(() => {
  const sub = currentUser()?.id ?? null;
  if (sub !== lastSub) {
    lastSub = sub;
    queryClient.clear();
  }
});
