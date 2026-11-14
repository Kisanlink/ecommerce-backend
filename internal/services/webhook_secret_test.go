package services

import (
	"testing"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"go.uber.org/zap"
)

// PAY-02 regression: when RAZORPAY_WEBHOOK_SECRET is unset, ParseWebhook must
// fail closed. Otherwise the HMAC is computed with an empty key, which an
// attacker can reproduce to forge order.paid events and fulfill unpaid orders.
func TestParseWebhook_FailsClosedOnEmptySecret(t *testing.T) {
	s := &PaymentService{
		config: config.RazorpayConfig{WebhookSecret: ""},
		logger: zap.NewNop(),
	}
	body := []byte(`{"event":"order.paid"}`)

	// The signature an attacker would compute against the empty key.
	forged := s.generateSignature(string(body), "")

	if _, err := s.ParseWebhook(body, forged); err == nil {
		t.Fatal("expected ParseWebhook to reject the webhook when no secret is configured")
	}
}

// With a secret configured, a correctly-signed webhook passes and a wrong
// signature is rejected.
func TestParseWebhook_VerifiesWithConfiguredSecret(t *testing.T) {
	const secret = "whsec_test_value"
	s := &PaymentService{
		config: config.RazorpayConfig{WebhookSecret: secret},
		logger: zap.NewNop(),
	}
	body := []byte(`{"event":"order.paid","payload":{}}`)

	valid := s.generateSignature(string(body), secret)
	if _, err := s.ParseWebhook(body, valid); err != nil {
		t.Fatalf("expected a validly-signed webhook to pass, got: %v", err)
	}

	if _, err := s.ParseWebhook(body, "deadbeefdeadbeef"); err == nil {
		t.Fatal("expected an invalid signature to be rejected")
	}
}
