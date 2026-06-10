package bot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// LinkCallback is implemented by service.AuthService to handle the bot ↔
// account linking handshake. Keeping a minimal interface here prevents a direct
// import of the service package.
type LinkCallback interface {
	ConsumeBotLinkToken(ctx context.Context, token string) (userID string, found bool, err error)
	StorePendingLink(ctx context.Context, platform, chatID, userID string) error
	ConsumePendingLink(ctx context.Context, platform, chatID string) (userID string, found bool, err error)
	CompleteBotLink(ctx context.Context, userIDStr, platform, chatID, phone string) error
	// StoreRegistrationContact stores a phone→chatID mapping when a user shares
	// their phone after a bare /start (B5, TASK-0026).
	StoreRegistrationContact(ctx context.Context, platform, chatID, phone string) error
}

// Poller runs a getUpdates long-poll loop for one bot platform (Bale or
// Telegram). Started as a goroutine from main.go; shuts down when ctx is done.
type Poller struct {
	baseURL  string // e.g. "https://tapi.bale.ai/bot<token>"
	platform string // "bale" | "telegram" (for keys + logging)
	cb       LinkCallback
}

func NewPoller(baseURL, platform string, cb LinkCallback) *Poller {
	return &Poller{baseURL: baseURL, platform: platform, cb: cb}
}

// ProbeBot does a single /getMe to check if the bot API is reachable.
// Returns nil on success, an error with HTTP status on failure.
func ProbeBot(baseURL string) error {
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(baseURL + "/getMe")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		snippet := string(b)
		if len(snippet) > 80 {
			snippet = snippet[:80]
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet)
	}
	return nil
}

// Run starts the polling loop. Call as `go poller.Run(ctx)`.
// Errors back off exponentially (5 s → 10 s → … → 60 s) to avoid log spam
// during temporary outages or network filtering.
func (p *Poller) Run(ctx context.Context) {
	var offset int64
	backoff := 5 * time.Second
	const maxBackoff = 60 * time.Second
	log.Printf("bot: %s poller started", p.platform)
	for {
		if ctx.Err() != nil {
			return
		}
		updates, err := p.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("bot: %s getUpdates error (retry in %s): %v", p.platform, backoff, err)
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return
			}
			if backoff < maxBackoff {
				backoff *= 2
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
			}
			continue
		}
		backoff = 5 * time.Second // reset on success
		for _, u := range updates {
			if u.UpdateID >= offset {
				offset = u.UpdateID + 1
			}
			p.handleUpdate(ctx, u)
		}
	}
}

func (p *Poller) handleUpdate(ctx context.Context, u update) {
	if u.Message == nil {
		return
	}
	msg := u.Message
	chatIDStr := fmt.Sprintf("%d", msg.Chat.ID)

	// "/start <link-token>" — user tapped a deep link from Settings.
	if strings.HasPrefix(msg.Text, "/start ") {
		token := strings.TrimSpace(strings.TrimPrefix(msg.Text, "/start "))
		if token == "" {
			return
		}
		userID, found, err := p.cb.ConsumeBotLinkToken(ctx, token)
		if err != nil || !found {
			if err != nil {
				log.Printf("bot: %s /start: consume token: %v", p.platform, err)
			}
			return
		}
		if err := p.cb.StorePendingLink(ctx, p.platform, chatIDStr, userID); err != nil {
			log.Printf("bot: %s /start: store pending: %v", p.platform, err)
			return
		}
		_ = p.sendContactRequest(ctx, chatIDStr)
		return
	}

	// "/start" with no token — cold-start registration flow (B5).
	if msg.Text == "/start" {
		_ = p.sendRegistrationContactRequest(ctx, chatIDStr)
		return
	}

	// Contact share — user tapped "share phone" button.
	if msg.Contact != nil && msg.Contact.PhoneNumber != "" {
		userID, found, err := p.cb.ConsumePendingLink(ctx, p.platform, chatIDStr)
		if err != nil || !found {
			// No pending account-link: store registration contact (B5).
			if storeErr := p.cb.StoreRegistrationContact(ctx, p.platform, chatIDStr, msg.Contact.PhoneNumber); storeErr != nil {
				log.Printf("bot: %s store reg contact: %v", p.platform, storeErr)
				return
			}
			_ = p.sendText(ctx, chatIDStr, "✅ شماره ثبت شد. می‌توانید از اپلیکیشن ولتانا ثبت‌نام کنید.")
			return
		}
		if err := p.cb.CompleteBotLink(ctx, userID, p.platform, chatIDStr, msg.Contact.PhoneNumber); err != nil {
			log.Printf("bot: %s complete link: %v", p.platform, err)
			return
		}
		_ = p.sendText(ctx, chatIDStr, "✅ اتصال حساب انجام شد. می‌توانید با شماره تلفن وارد شوید.")
	}
}

func (p *Poller) sendContactRequest(ctx context.Context, chatID string) error {
	return p.sendJSON(ctx, "sendMessage", map[string]any{
		"chat_id": chatID,
		"text":    "برای اتصال حساب ولتانا، لطفاً شماره تلفن خود را به اشتراک بگذارید:",
		"reply_markup": map[string]any{
			"keyboard": [][]map[string]any{
				{{"text": "📱 اشتراک‌گذاری شماره تلفن", "request_contact": true}},
			},
			"one_time_keyboard": true,
			"resize_keyboard":   true,
		},
	})
}

// sendRegistrationContactRequest prompts a new user to share their phone
// number as the first step of the cold-start registration flow (B5).
func (p *Poller) sendRegistrationContactRequest(ctx context.Context, chatID string) error {
	return p.sendJSON(ctx, "sendMessage", map[string]any{
		"chat_id": chatID,
		"text":    "برای ثبت‌نام در ولتانا، شماره تلفن خود را به اشتراک بگذارید:",
		"reply_markup": map[string]any{
			"keyboard": [][]map[string]any{
				{{"text": "📱 اشتراک‌گذاری شماره تلفن", "request_contact": true}},
			},
			"one_time_keyboard": true,
			"resize_keyboard":   true,
		},
	})
}

func (p *Poller) sendText(ctx context.Context, chatID, text string) error {
	return p.sendJSON(ctx, "sendMessage", map[string]any{
		"chat_id":      chatID,
		"text":         text,
		"reply_markup": map[string]any{"remove_keyboard": true},
	})
}

func (p *Poller) sendJSON(ctx context.Context, method string, body any) error {
	payload, _ := json.Marshal(body)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/"+method, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("bot %s %s: %d: %s", p.platform, method, resp.StatusCode, b)
	}
	return nil
}

// ── getUpdates long-poll ───────────────────────────────────────────────────────

type update struct {
	UpdateID int64    `json:"update_id"`
	Message  *message `json:"message"`
}

type message struct {
	Text    string   `json:"text"`
	Chat    chat     `json:"chat"`
	Contact *contact `json:"contact"`
}

type chat struct {
	ID int64 `json:"id"`
}

type contact struct {
	PhoneNumber string `json:"phone_number"`
}

type updatesResponse struct {
	OK     bool     `json:"ok"`
	Result []update `json:"result"`
}

func (p *Poller) getUpdates(ctx context.Context, offset int64) ([]update, error) {
	url := fmt.Sprintf("%s/getUpdates?offset=%d&timeout=30", p.baseURL, offset)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 35 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		snippet := string(b)
		if len(snippet) > 120 {
			snippet = snippet[:120]
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet)
	}
	var r updatesResponse
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("parse updates: %w", err)
	}
	if !r.OK {
		return nil, fmt.Errorf("getUpdates not OK: %s", b)
	}
	return r.Result, nil
}
