package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	"voltana-api/internal/repository"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
)

var (
	// ErrPushDisabled — no VAPID keys configured (handlers translate to 503).
	ErrPushDisabled = errors.New("web push is not configured")
	// ErrInvalidEndpoint — subscription endpoint failed validation (→ 400).
	ErrInvalidEndpoint = errors.New("invalid push endpoint")
)

// PushPayload is the JSON the service worker receives.
type PushPayload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url,omitempty"`
}

// PushService manages web-push subscriptions and delivery (TASK-0039).
// The VAPID private key never leaves this service.
type PushService struct {
	subs            repository.PushSubscriptionRepository
	vapidPublicKey  string
	vapidPrivateKey string
	subscriber      string // VAPID sub claim (mailto: or https: origin)
}

func NewPushService(subs repository.PushSubscriptionRepository, vapidPublic, vapidPrivate, subscriber string) *PushService {
	if subscriber == "" {
		subscriber = "https://voltana.app"
	}
	return &PushService{
		subs:            subs,
		vapidPublicKey:  vapidPublic,
		vapidPrivateKey: vapidPrivate,
		subscriber:      subscriber,
	}
}

// Enabled reports whether VAPID keys are configured.
func (s *PushService) Enabled() bool {
	return s.vapidPublicKey != "" && s.vapidPrivateKey != ""
}

// VAPIDPublicKey returns the applicationServerKey for PushManager.subscribe.
func (s *PushService) VAPIDPublicKey() (string, error) {
	if !s.Enabled() {
		return "", ErrPushDisabled
	}
	return s.vapidPublicKey, nil
}

// Subscribe stores a browser subscription after validating the endpoint
// (SSRF guard: https only, no localhost / private-IP hosts — the server later
// POSTs to this URL).
func (s *PushService) Subscribe(ctx context.Context, userID uuid.UUID, endpoint, p256dh, auth string) error {
	if !s.Enabled() {
		return ErrPushDisabled
	}
	if err := validatePushEndpoint(endpoint); err != nil {
		return err
	}
	if p256dh == "" || auth == "" {
		return fmt.Errorf("%w: missing keys", ErrInvalidEndpoint)
	}
	return s.subs.Create(ctx, userID, endpoint, p256dh, auth)
}

func (s *PushService) Unsubscribe(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return s.subs.DeleteByEndpoint(ctx, userID, endpoint)
}

// SendToUser fans a payload out to all of the user's subscriptions. Endpoints
// answering 404/410 (browser unsubscribed) are pruned. Returns how many pushes
// were accepted by the push services.
func (s *PushService) SendToUser(ctx context.Context, userID uuid.UUID, p PushPayload) (int, error) {
	if !s.Enabled() {
		return 0, ErrPushDisabled
	}
	subs, err := s.subs.ListByUser(ctx, userID)
	if err != nil {
		return 0, err
	}
	body, _ := json.Marshal(p)
	sent := 0
	for _, sub := range subs {
		resp, err := webpush.SendNotificationWithContext(ctx, body, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
		}, &webpush.Options{
			Subscriber:      s.subscriber,
			VAPIDPublicKey:  s.vapidPublicKey,
			VAPIDPrivateKey: s.vapidPrivateKey,
			TTL:             3600,
		})
		if err != nil {
			// Network errors may stringify the endpoint URL — that's the user's
			// own push-service URL, not a secret, but keep logs terse.
			log.Printf("push: send to sub %s failed: %v", sub.ID, errTail(err))
			continue
		}
		func() {
			defer resp.Body.Close()
			switch {
			case resp.StatusCode == 404 || resp.StatusCode == 410:
				// Browser revoked the subscription — prune.
				if delErr := s.subs.DeleteByID(ctx, sub.ID); delErr != nil {
					log.Printf("push: prune sub %s: %v", sub.ID, delErr)
				}
			case resp.StatusCode >= 200 && resp.StatusCode < 300:
				sent++
			default:
				log.Printf("push: sub %s: push service status %d", sub.ID, resp.StatusCode)
			}
		}()
	}
	return sent, nil
}

// NotifySOHDrop satisfies the analytics trigger: fire-and-forget battery alert.
func (s *PushService) NotifySOHDrop(userID uuid.UUID, carName string, sohPct float64) {
	if !s.Enabled() {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_, err := s.SendToUser(ctx, userID, PushPayload{
			Title: "⚠️ هشدار سلامت باتری",
			Body:  fmt.Sprintf("سلامت باتری «%s» به %.0f٪ رسید", carName, sohPct),
			URL:   "/",
		})
		if err != nil && !errors.Is(err, ErrPushDisabled) {
			log.Printf("push: soh alert for user %s: %v", userID, err)
		}
	}()
}

// validatePushEndpoint allows only https URLs on public hostnames. Browsers'
// push services are always public https; anything else is a misbehaving client
// (or an SSRF attempt — the server POSTs to this URL).
func validatePushEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("%w: must be an https URL", ErrInvalidEndpoint)
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".local") {
		return fmt.Errorf("%w: forbidden host", ErrInvalidEndpoint)
	}
	if ip := net.ParseIP(host); ip != nil {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified() {
			return fmt.Errorf("%w: forbidden host", ErrInvalidEndpoint)
		}
	}
	return nil
}

// errTail keeps only the final ": "-separated segment of an error message —
// enough to diagnose (timeout / connection refused) without the full URL noise.
func errTail(err error) string {
	msg := err.Error()
	if i := strings.LastIndex(msg, ": "); i >= 0 && i+2 < len(msg) {
		return msg[i+2:]
	}
	return msg
}
