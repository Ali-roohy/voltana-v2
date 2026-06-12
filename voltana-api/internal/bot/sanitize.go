package bot

import (
	"errors"
	"regexp"
)

// Bot API URLs embed the token in the path ("…/bot<token>/getUpdates"), and
// Go's *url.Error stringifies the full URL — so any raw HTTP error from this
// package would leak the token into logs (TASK-0038). Every error that can
// carry a URL must pass through sanitizeURLErr before being returned.

// botTokenRe matches the "/bot<token>" path segment for both Bale and
// Telegram URLs (token runs until the next slash, whitespace, or quote).
var botTokenRe = regexp.MustCompile(`/bot[^/\s"]+`)

// maskBotToken replaces any bot-token path segment in s with "/bot***".
func maskBotToken(s string) string {
	return botTokenRe.ReplaceAllString(s, "/bot***")
}

// sanitizeURLErr returns an error safe to log: same text as err with any bot
// token masked. The original error is deliberately NOT wrapped, so the token
// cannot resurface via errors.Unwrap/%+v.
func sanitizeURLErr(err error) error {
	if err == nil {
		return nil
	}
	return errors.New(maskBotToken(err.Error()))
}
