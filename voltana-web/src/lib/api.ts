// Central HTTP client for the Go API (ADR-003). Responsibilities:
//  - prepend VITE_API_URL (defaults to same-origin, served by nginx)
//  - attach the in-memory access token as a Bearer header
//  - send/receive the httpOnly refresh cookie (credentials: "include")
//  - on 401, silently POST /auth/refresh ONCE and retry the original request
//  - surface the API's { error, code } envelope as a typed ApiError
//
// No component calls fetch() directly — all HTTP goes through here via the
// per-feature api.ts modules.

import { authStore } from "./auth-store";

const BASE = (import.meta.env.VITE_API_URL ?? "").replace(/\/$/, "");

export class ApiError extends Error {
  status: number;
  code: string;
  data?: Record<string, unknown>;
  constructor(status: number, code: string, message: string, data?: Record<string, unknown>) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.data = data;
  }
}

interface RequestOptions extends Omit<RequestInit, "body"> {
  body?: unknown;
}

// Single in-flight refresh shared across concurrent 401s, so refresh-token
// rotation can't invalidate parallel callers (only one /auth/refresh fires).
let refreshInFlight: Promise<string | null> | null = null;

async function doRefresh(): Promise<string | null> {
  try {
    const res = await fetch(`${BASE}/auth/refresh`, { method: "POST", credentials: "include" });
    if (!res.ok) return null;
    const data = (await res.json()) as { access_token?: string };
    return data.access_token ?? null;
  } catch {
    return null;
  }
}

function refreshOnce(): Promise<string | null> {
  if (!refreshInFlight) {
    refreshInFlight = doRefresh().finally(() => {
      refreshInFlight = null;
    });
  }
  return refreshInFlight;
}

export async function apiFetch<T = unknown>(
  path: string,
  options: RequestOptions = {},
  retry = true,
): Promise<T> {
  const { body, headers, ...rest } = options;
  const token = authStore.getToken();

  const res = await fetch(`${BASE}${path}`, {
    ...rest,
    credentials: "include",
    headers: {
      ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...headers,
    },
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (res.status === 401 && retry && path !== "/auth/refresh") {
    const newToken = await refreshOnce();
    if (newToken) {
      authStore.setToken(newToken);
      return apiFetch<T>(path, options, false);
    }
    authStore.clear();
    throw new ApiError(401, "UNAUTHORIZED", "session expired");
  }

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  const data = text ? JSON.parse(text) : undefined;

  if (!res.ok) {
    const code = (data && (data.code as string)) || "ERROR";
    const message = (data && (data.error as string)) || res.statusText;
    throw new ApiError(res.status, code, message, data as Record<string, unknown> | undefined);
  }
  return data as T;
}

export const api = {
  get: <T>(path: string) => apiFetch<T>(path, { method: "GET" }),
  post: <T>(path: string, body?: unknown) => apiFetch<T>(path, { method: "POST", body }),
  put: <T>(path: string, body?: unknown) => apiFetch<T>(path, { method: "PUT", body }),
  del: <T>(path: string) => apiFetch<T>(path, { method: "DELETE" }),
};
