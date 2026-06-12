import { api } from "@/lib/api";

export interface Settings {
  default_car_id: string | null;
  peak_rate: number;
  mid_rate: number;
  offpeak_rate: number;
  currency: 'toman' | 'rial' | 'usd';
  created_at: string;
  updated_at: string;
}

// PUT /v1/settings is a full replace — always send every field.
export interface SettingsUpdate {
  default_car_id: string | null;
  peak_rate: number;
  mid_rate: number;
  offpeak_rate: number;
  currency: 'toman' | 'rial' | 'usd';
}

export const getSettings = () => api.get<Settings>("/v1/settings");
export const updateSettings = (body: SettingsUpdate) => api.put<Settings>("/v1/settings", body);

export interface TestOTPResult {
  success: boolean;
  message: string;
}

export const testOTPDelivery = (platform: "bale" | "telegram" | "email") =>
  api.post<TestOTPResult>("/v1/admin/test-otp", { platform });

export interface TestBotConnectionResult {
  ok: boolean;
  bot_username?: string;
  latency_ms?: number;
  error?: string;
}

// Server-side getMe with the env bot token (admin-only; the token never reaches the client).
export const testBotConnection = (platform: "bale" | "telegram") =>
  api.post<TestBotConnectionResult>("/v1/admin/test-bot-connection", { platform });

// ── Backup & restore (TASK-0037 FEAT-4) ──────────────────────────────────────

export interface ImportStats {
  cars: number;
  sessions: number;
  snapshots: number;
}

export interface ImportResult {
  message: string;
  imported: ImportStats;
}

// The payload is an opaque versioned document — the client never interprets it.
export const exportAccountData = () => api.get<Record<string, unknown>>("/v1/account/export");
export const importAccountData = (backup: unknown) =>
  api.post<ImportResult>("/v1/account/import", backup);

// ── Self-delete account (TASK-0037 FEAT-5) ───────────────────────────────────

export const deleteAccount = () => api.del<void>("/v1/account");
