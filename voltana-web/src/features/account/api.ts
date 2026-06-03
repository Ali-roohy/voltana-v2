import { api } from "@/lib/api";

export interface BotLinkResponse {
  bale_url?: string;
  telegram_url?: string;
}

// POST /v1/account/bot-link (JWT-protected)
// Returns deep links for the configured bot platforms.
// The user taps the link → opens the bot → shares their verified phone.
export const requestBotLink = () => api.post<BotLinkResponse>("/v1/account/bot-link");
