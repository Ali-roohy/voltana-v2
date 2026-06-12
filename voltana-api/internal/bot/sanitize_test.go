package bot

import (
	"context"
	"strings"
	"testing"
)

const fakeToken = "1316617363:FAKESECRETxyz"

// All tests force REAL http errors against an unroutable address whose URL
// embeds fakeToken, then assert the token never appears in the error text
// (TASK-0038 — the live leak was *url.Error stringifying the full URL).

func assertNoToken(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error (unroutable address), got nil")
	}
	if strings.Contains(err.Error(), fakeToken) {
		t.Fatalf("error text leaks the bot token: %q", err.Error())
	}
	if !strings.Contains(err.Error(), "/bot***") {
		t.Fatalf("error text should contain the masked /bot*** marker, got: %q", err.Error())
	}
}

func TestGetUpdates_ErrorContainsNoToken(t *testing.T) {
	p := NewPoller("http://127.0.0.1:1/bot"+fakeToken, "bale", nil)
	_, err := p.getUpdates(context.Background(), 0)
	assertNoToken(t, err)
}

func TestSendJSON_ErrorContainsNoToken(t *testing.T) {
	p := NewPoller("http://127.0.0.1:1/bot"+fakeToken, "bale", nil)
	err := p.sendJSON(context.Background(), "sendMessage", map[string]any{"chat_id": "1"})
	assertNoToken(t, err)
}

func TestProbeBot_ErrorContainsNoToken(t *testing.T) {
	err := ProbeBot("http://127.0.0.1:1/bot" + fakeToken)
	assertNoToken(t, err)
}

func TestSendMessage_ErrorContainsNoToken(t *testing.T) {
	err := sendMessage(context.Background(), "http://127.0.0.1:1/bot"+fakeToken, "1", "hi")
	assertNoToken(t, err)
}

func TestMaskBotToken(t *testing.T) {
	in := `Get "https://tapi.bale.ai/bot` + fakeToken + `/getUpdates?offset=0": connection reset`
	got := maskBotToken(in)
	if strings.Contains(got, fakeToken) {
		t.Fatalf("token survived masking: %q", got)
	}
	want := `Get "https://tapi.bale.ai/bot***/getUpdates?offset=0": connection reset`
	if got != want {
		t.Errorf("masked = %q, want %q", got, want)
	}
}
