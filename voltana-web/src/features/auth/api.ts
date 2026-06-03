import { api } from "@/lib/api";
import { authStore } from "@/lib/auth-store";

interface TokenResponse {
  access_token: string;
}

// Identity for the authenticated user from GET /v1/me. `is_admin` is sourced here
// (not from the access token, which deliberately omits it) so the UI can gate the
// admin area. The API stays the real boundary — AdminOnly re-checks on every write.
export interface Me {
  id: string;
  email: string;
  is_admin: boolean;
  phone: string | null;
  bale_linked: boolean;
  telegram_linked: boolean;
}

export const getMe = () => api.get<Me>("/v1/me");

export async function register(email: string, password: string): Promise<void> {
  await api.post("/auth/register", { email, password });
}

export async function login(email: string, password: string): Promise<void> {
  const res = await api.post<TokenResponse>("/auth/login", { email, password });
  authStore.setToken(res.access_token);
  authStore.setEmail(email);
}

// Consume a verification token from the emailed link. Returns the API message
// ("email verified" / "email already verified").
export async function verifyEmail(token: string): Promise<string> {
  const res = await api.post<{ message: string }>("/auth/verify-email", { token });
  return res.message;
}

// Request a fresh verification email. Always resolves for a well-formed request
// (the server replies 202 regardless of whether the account exists/is verified).
export async function resendVerification(email: string): Promise<void> {
  await api.post("/auth/resend-verification", { email });
}

// Request a 6-digit OTP sent to the user's linked Bale/Telegram bot chat.
// Always resolves (server replies 202 regardless of whether the phone is linked).
export async function requestOTP(phone: string): Promise<void> {
  await api.post("/auth/otp/request", { phone });
}

// Verify a 6-digit OTP and log in. Sets the in-memory access token on success.
export async function verifyOTP(phone: string, code: string): Promise<void> {
  const res = await api.post<TokenResponse>("/auth/otp/verify", { phone, code });
  authStore.setToken(res.access_token);
}

export async function logout(): Promise<void> {
  try {
    await api.post("/auth/logout");
  } finally {
    authStore.clear();
  }
}

// Restore a session from the httpOnly refresh cookie. Deduped to a single
// in-flight call so multiple useAuth() consumers mounting at once don't each
// hit /auth/refresh (refresh-token rotation would invalidate all but the first).
let restorePromise: Promise<boolean> | null = null;

async function doRestore(): Promise<boolean> {
  try {
    const res = await api.post<TokenResponse>("/auth/refresh");
    if (res?.access_token) {
      authStore.setToken(res.access_token);
      return true;
    }
  } catch {
    /* no valid refresh cookie — not logged in */
  }
  return false;
}

export function restoreSessionOnce(): Promise<boolean> {
  if (!restorePromise) {
    restorePromise = doRestore().finally(() => {
      restorePromise = null;
    });
  }
  return restorePromise;
}
