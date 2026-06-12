package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ConnectionTester performs a live getMe call against the Bale/Telegram bot
// APIs using the real tokens from env (TASK-0036 BUG-8). Tokens never leave
// the server; errors are sanitized because Go's *url.Error embeds the request
// URL — which contains the token.
type ConnectionTester struct {
	baleToken string
	tgToken   string
}

func NewConnectionTester(baleToken, tgToken string) *ConnectionTester {
	return &ConnectionTester{baleToken: baleToken, tgToken: tgToken}
}

var ErrTokenNotConfigured = errors.New("bot token not configured")

type getMeResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		Username string `json:"username"`
	} `json:"result"`
	Description string `json:"description"`
}

// Test calls <api>/getMe for the given platform ("bale" | "telegram").
// Returns the bot username and round-trip latency on success.
func (t *ConnectionTester) Test(ctx context.Context, platform string) (username string, latency time.Duration, err error) {
	var base string
	switch platform {
	case "telegram":
		if t.tgToken == "" {
			return "", 0, ErrTokenNotConfigured
		}
		base = "https://api.telegram.org/bot" + t.tgToken
	default: // bale
		if t.baleToken == "" {
			return "", 0, ErrTokenNotConfigured
		}
		base = "https://tapi.bale.ai/bot" + t.baleToken
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/getMe", nil)
	if err != nil {
		return "", 0, errors.New("building request failed")
	}

	start := time.Now()
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// Do NOT wrap err: *url.Error stringifies the full URL incl. the token.
		if errors.Is(err, context.DeadlineExceeded) || ctx.Err() != nil {
			return "", 0, errors.New("timeout reaching bot API")
		}
		return "", 0, errors.New("network error reaching bot API")
	}
	defer resp.Body.Close()
	latency = time.Since(start)

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	var parsed getMeResponse
	if jsonErr := json.Unmarshal(body, &parsed); jsonErr != nil || !parsed.OK {
		desc := parsed.Description
		if desc == "" {
			desc = http.StatusText(resp.StatusCode)
		}
		return "", latency, fmt.Errorf("bot API error: %s (status %d)", desc, resp.StatusCode)
	}
	return parsed.Result.Username, latency, nil
}
