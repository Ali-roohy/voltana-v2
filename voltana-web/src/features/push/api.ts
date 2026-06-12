import { api } from "@/lib/api";

export interface VAPIDKeyResponse {
  vapid_public_key: string;
}

export const getVAPIDKey = () => api.get<VAPIDKeyResponse>("/v1/push/vapid-key");

// Browser PushSubscription.toJSON() shape.
export interface PushSubscriptionJSON {
  endpoint: string;
  keys: { p256dh: string; auth: string };
}

export const subscribePush = (sub: PushSubscriptionJSON) =>
  api.post<void>("/v1/account/push-subscription", sub);

export const unsubscribePush = (endpoint: string) =>
  api.del<void>("/v1/account/push-subscription", { endpoint });

export interface TestPushResult {
  success: boolean;
  message: string;
  sent?: number;
}

export const testPush = () => api.post<TestPushResult>("/v1/admin/test-push");
