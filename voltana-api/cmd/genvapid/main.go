// Command genvapid generates a fresh VAPID keypair for Web Push (TASK-0039/0041).
//
// Usage:
//
//	cd voltana-api && go run ./cmd/genvapid
//
// It prints an env block to stdout, ready to paste into a .env file:
//
//	VAPID_PUBLIC_KEY=...
//	VAPID_PRIVATE_KEY=...
//
// Generate a FRESH pair per environment — never reuse dev keys in production.
package main

import (
	"fmt"
	"os"

	webpush "github.com/SherClockHolmes/webpush-go"
)

func main() {
	// GenerateVAPIDKeys returns (privateKey, publicKey, err).
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		fmt.Fprintln(os.Stderr, "genvapid: failed to generate VAPID keys:", err)
		os.Exit(1)
	}

	fmt.Printf("VAPID_PUBLIC_KEY=%s\n", pub)
	fmt.Printf("VAPID_PRIVATE_KEY=%s\n", priv)
}
