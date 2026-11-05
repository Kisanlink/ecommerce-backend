package config

import (
	"log"
	"os"
)

// RazorpayConfig holds Razorpay payment configuration
type RazorpayConfig struct {
	KeyID         string
	KeySecret     string
	AccountID     string
	BaseURL       string
	WebhookSecret string
	// MagicCheckoutSecret is the dedicated 1CC HMAC secret from the Magic Checkout
	// page in the Razorpay Dashboard (distinct from WebhookSecret). It is OPTIONAL:
	// when empty, Magic Checkout callback signature verification stays dormant and
	// the callbacks rely on their other protections (#7). When set, callbacks must
	// carry a valid X-Razorpay-Signature. Never required to boot, even in live mode.
	MagicCheckoutSecret string
	IsLive              bool
}

// GetRazorpayConfig returns Razorpay config from environment variables.
//
// Security: there are NO hardcoded live credentials. In live mode every secret
// (key id, key secret, webhook secret) MUST come from the environment; if any is
// missing the process refuses to start rather than fall back to a shared/committed
// value. In non-live (local/dev) mode only non-secret test placeholders are used,
// and the webhook secret is left empty so signature verification fails closed.
func GetRazorpayConfig() RazorpayConfig {
	// Check both RAZORPAY_LIVE and RAZORPAY_PRODUCTION for backward compatibility
	isLive := os.Getenv("RAZORPAY_LIVE") == "true" || os.Getenv("RAZORPAY_PRODUCTION") == "true"

	keyID := os.Getenv("RAZORPAY_KEY_ID")
	keySecret := os.Getenv("RAZORPAY_KEY_SECRET")
	webhookSecret := os.Getenv("RAZORPAY_WEBHOOK_SECRET")

	if isLive {
		missing := make([]string, 0, 3)
		if keyID == "" {
			missing = append(missing, "RAZORPAY_KEY_ID")
		}
		if keySecret == "" {
			missing = append(missing, "RAZORPAY_KEY_SECRET")
		}
		if webhookSecret == "" {
			missing = append(missing, "RAZORPAY_WEBHOOK_SECRET")
		}
		if len(missing) > 0 {
			// Fail fast: a live deployment must never run on default credentials.
			log.Fatalf("razorpay: live mode enabled but required env vars are unset: %v", missing)
		}
	} else {
		// Non-live convenience placeholders. These are NOT secrets.
		if keyID == "" {
			keyID = "rzp_test_your_test_key_here"
		}
		if keySecret == "" {
			keySecret = "your_test_secret_here"
		}
		// webhookSecret intentionally left as-is (possibly empty): webhook signature
		// verification must fail closed rather than accept a shared default secret.
	}

	return RazorpayConfig{
		KeyID:         keyID,
		KeySecret:     keySecret,
		AccountID:     os.Getenv("RAZORPAY_ACCOUNT_ID"),
		BaseURL:       getEnvOrDefault("RAZORPAY_BASE_URL", "https://api.razorpay.com/v1"),
		WebhookSecret: webhookSecret,
		// Optional 1CC secret; dormant when unset (see field doc). Not fail-fast
		// in live mode because callbacks have their own server-side protections.
		MagicCheckoutSecret: os.Getenv("RAZORPAY_MAGIC_CHECKOUT_SECRET"),
		IsLive:              isLive,
	}
}
