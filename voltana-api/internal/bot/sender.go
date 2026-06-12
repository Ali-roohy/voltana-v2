// Package bot implements the OTPSender interface (Bale, Telegram, log-dev) and
// the long-poll worker that drives the bot-linking handshake. All concrete types
// here satisfy service.OTPSender structurally — no import of the service package
// is needed to implement the interface.
package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"voltana-api/internal/service"
)

// BaleSender sends OTP codes via the Bale Bot API (preferred, Iranian).
type BaleSender struct {
	token string
}

func NewBaleSender(token string) *BaleSender { return &BaleSender{token: token} }

func (s *BaleSender) Platform() service.Platform { return service.PlatformBale }

func (s *BaleSender) Send(ctx context.Context, chatID, code string) error {
	return sendMessage(ctx, "https://tapi.bale.ai/bot"+s.token, chatID,
		fmt.Sprintf("کد ورود ولتانا: *%s*\n\nاین کد ۵ دقیقه اعتبار دارد.", code))
}

// TelegramSender sends OTP codes via the Telegram Bot API (fallback; may be
// filtered by Iranian ISPs — Bale is the production primary).
type TelegramSender struct {
	token string
}

func NewTelegramSender(token string) *TelegramSender { return &TelegramSender{token: token} }

func (s *TelegramSender) Platform() service.Platform { return service.PlatformTelegram }

func (s *TelegramSender) Send(ctx context.Context, chatID, code string) error {
	return sendMessage(ctx, "https://api.telegram.org/bot"+s.token, chatID,
		fmt.Sprintf("Voltana login code: *%s*\n\nExpires in 5 minutes.", code))
}

// LogOTPSender is used when no bot token is configured (dev). It logs that an
// OTP send was requested — never the code itself — so QA can confirm the path
// is exercised without a real bot.
type LogOTPSender struct{}

func (LogOTPSender) Platform() service.Platform { return service.PlatformBale }

func (LogOTPSender) Send(_ context.Context, chatID, _ string) error {
	log.Printf("bot: LogOTPSender — no bot token; OTP send skipped for chatID=%s", chatID)
	return nil
}

// ── internal HTTP helper ───────────────────────────────────────────────────────

func sendMessage(ctx context.Context, baseURL, chatID, text string) error {
	payload, _ := json.Marshal(map[string]any{
		"chat_id":         chatID,
		"text":            text,
		"parse_mode":      "Markdown",
		"protect_content": true, // prevents forwarding / saving
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/sendMessage", bytes.NewReader(payload))
	if err != nil {
		return sanitizeURLErr(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// %v of the sanitized error — never wrap the raw *url.Error (token in URL).
		return fmt.Errorf("bot sendMessage: %v", sanitizeURLErr(err))
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bot sendMessage: status %d: %s", resp.StatusCode, body)
	}
	return nil
}
