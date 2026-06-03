// In-memory access-token store (ADR-003). The access token is NEVER persisted to
// localStorage/sessionStorage — only the refresh token lives in an httpOnly cookie.
// On a full page reload the token is gone and is restored via POST /auth/refresh.

type Listener = () => void;

export interface AuthUser {
  id: string;
  email?: string;
  // present only for API-compat with the old Supabase user shape; always undefined now
  user_metadata?: { full_name?: string };
}

let accessToken: string | null = null;
let email: string | null = null;
const listeners = new Set<Listener>();

function notify() {
  listeners.forEach((l) => l());
}

export const authStore = {
  getToken: (): string | null => accessToken,
  setToken(token: string | null) {
    accessToken = token;
    notify();
  },
  setEmail(value: string | null) {
    email = value;
  },
  clear() {
    accessToken = null;
    email = null;
    notify();
  },
  subscribe(listener: Listener): () => void {
    listeners.add(listener);
    return () => {
      listeners.delete(listener);
    };
  },
};

// currentUser decodes the access-token payload (no verification — the server
// verifies on every request). The user id is the JWT `sub` claim.
export function currentUser(): AuthUser | null {
  if (!accessToken) return null;
  try {
    const payload = accessToken.split(".")[1];
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"));
    const claims = JSON.parse(json) as { sub?: string };
    if (!claims.sub) return null;
    return { id: claims.sub, email: email ?? undefined };
  } catch {
    return null;
  }
}
