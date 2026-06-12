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
  full_name: string | null;
  phone: string | null;
  bale_linked: boolean;
  telegram_linked: boolean;
  password_set: boolean;
}

export interface OTPRequestResult {
  status?: "deep_link" | "awaiting_contact_share" | "not_registered";
  bale_url?: string | null;
  telegram_url?: string | null;
}

export interface OTPContactStatusResult {
  // awaiting_bot: deep-link session, user hasn't opened the bot yet (neutral
  // wait state — the frontend keeps polling without a countdown).
  status: "awaiting_contact_share" | "awaiting_bot" | "otp_sent" | "expired";
}

export interface OTPConfig {
  delivery_method: "deeplink" | "contact_share";
  bale_username?: string;
  tg_username?: string;
}

export const getMe = () => api.get<Me>("/v1/me");

export async function register(
  email: string,
  password: string,
  fullName?: string,
  phone?: string,
): Promise<void> {
  await api.post("/auth/register", {
    email,
    password,
    full_name: fullName || undefined,
    phone: phone || undefined,
  });
}

export async function login(email: string, password: string, stayLoggedIn = false): Promise<void> {
  const res = await api.post<TokenResponse>("/auth/login", {
    email,
    password,
    stay_logged_in: stayLoggedIn,
  });
  authStore.setToken(res.access_token);
  authStore.setEmail(email);
}

export async function loginWithPhone(
  phone: string,
  password: string,
  stayLoggedIn = false,
): Promise<void> {
  const res = await api.post<TokenResponse>("/auth/login/phone", {
    phone,
    password,
    stay_logged_in: stayLoggedIn,
  });
  authStore.setToken(res.access_token);
}

export async function setPassword(password: string): Promise<void> {
  await api.post("/v1/account/set-password", { password });
}

export async function getOTPConfig(): Promise<OTPConfig> {
  return api.get<OTPConfig>("/auth/otp/config");
}

export async function getOTPContactStatus(
  phone: string,
  platform: string,
): Promise<OTPContactStatusResult> {
  const params = new URLSearchParams({ phone, platform });
  return api.get<OTPContactStatusResult>(`/auth/otp/contact-status?${params}`);
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

// Request a 6-digit OTP. In contact_share mode, always resolves (202, anti-enum).
// In deeplink mode, may return { status: "deep_link", bale_url, telegram_url } (200).
export async function requestOTP(
  phone: string,
  platform: "bale" | "telegram",
  mode: "login" | "register" = "register",
): Promise<OTPRequestResult> {
  const res = await api.post<OTPRequestResult>("/auth/otp/request", { phone, platform, mode });
  return res ?? {};
}

// Verify a 6-digit OTP and log in. Sets the in-memory access token on success.
// Throws ApiError with code "INVALID_OTP" (+ data.remaining_attempts) or "OTP_LOCKED".
export async function verifyOTP(
  phone: string,
  code: string,
  platform: "bale" | "telegram",
  stayLoggedIn = false,
): Promise<void> {
  const res = await api.post<TokenResponse>("/auth/otp/verify", {
    phone,
    code,
    platform,
    stay_logged_in: stayLoggedIn,
  });
  authStore.setToken(res.access_token);
}

// Verify a registration OTP and create a new account. Sets the in-memory access token on success.
// Throws ApiError with code "PHONE_TAKEN" (409), "INVALID_OTP" (401), or "OTP_LOCKED" (401).
export async function registerWithOTP(
  phone: string,
  code: string,
  platform: "bale" | "telegram",
  email?: string,
): Promise<void> {
  const res = await api.post<TokenResponse>("/auth/otp/register", {
    phone,
    code,
    platform,
    ...(email ? { email } : {}),
  });
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
