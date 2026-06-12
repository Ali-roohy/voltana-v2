import { useCallback, useEffect, useState } from "react";
import { getVAPIDKey, subscribePush, unsubscribePush } from "./api";

// applicationServerKey must be a Uint8Array of the raw VAPID public key.
function urlBase64ToUint8Array(base64: string): Uint8Array {
  const padding = "=".repeat((4 - (base64.length % 4)) % 4);
  const b64 = (base64 + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(b64);
  return Uint8Array.from([...raw].map((c) => c.charCodeAt(0)));
}

export type PushState =
  | "unsupported"   // browser has no PushManager / Notification
  | "disabled"      // server has no VAPID keys (503)
  | "denied"        // user blocked notifications
  | "off"           // supported, not subscribed
  | "on"            // subscribed
  | "loading";

// usePushNotifications drives the Settings «اعلان‌ها» toggle: resolves the
// current state on mount and exposes enable()/disable().
export function usePushNotifications() {
  const [state, setState] = useState<PushState>("loading");

  useEffect(() => {
    let cancelled = false;
    (async () => {
      if (!("serviceWorker" in navigator) || !("PushManager" in window) || !("Notification" in window)) {
        if (!cancelled) setState("unsupported");
        return;
      }
      try {
        await getVAPIDKey();
      } catch {
        if (!cancelled) setState("disabled");
        return;
      }
      if (Notification.permission === "denied") {
        if (!cancelled) setState("denied");
        return;
      }
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      if (!cancelled) setState(sub ? "on" : "off");
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const enable = useCallback(async () => {
    setState("loading");
    try {
      const permission = await Notification.requestPermission();
      if (permission !== "granted") {
        setState(permission === "denied" ? "denied" : "off");
        return false;
      }
      const { vapid_public_key } = await getVAPIDKey();
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.subscribe({
        userVisibleOnly: true,
        applicationServerKey: urlBase64ToUint8Array(vapid_public_key),
      });
      const json = sub.toJSON();
      await subscribePush({
        endpoint: sub.endpoint,
        keys: { p256dh: json.keys?.p256dh ?? "", auth: json.keys?.auth ?? "" },
      });
      setState("on");
      return true;
    } catch {
      setState("off");
      return false;
    }
  }, []);

  const disable = useCallback(async () => {
    setState("loading");
    try {
      const reg = await navigator.serviceWorker.ready;
      const sub = await reg.pushManager.getSubscription();
      if (sub) {
        await unsubscribePush(sub.endpoint);
        await sub.unsubscribe();
      }
      setState("off");
    } catch {
      setState("off");
    }
  }, []);

  return { state, enable, disable };
}
