package config

import (
	"os"
)

// RazorpayConfig holds Razorpay payment configuration
type RazorpayConfig struct {
	KeyID         string
	KeySecret     string
	AccountID     string
	BaseURL       string
	WebhookSecret string
	IsLive        bool
}

// GetRazorpayConfig returns Razorpay config from environment variables
func GetRazorpayConfig() RazorpayConfig {
	// Check both RAZORPAY_LIVE and RAZORPAY_PRODUCTION for backward compatibility
	isLive := os.Getenv("RAZORPAY_LIVE") == "true" || os.Getenv("RAZORPAY_PRODUCTION") == "true"

	// Default to test credentials if not in live mode
	defaultKeyID := "rzp_test_your_test_key_here"
	defaultSecret := "your_test_secret_here"
	if isLive {
		defaultKeyID = "rzp_live_RCbm4zwG4WzwGV"
		defaultSecret = "ANrUne3rX1A6ezzmlfBEqBNx"
	}

	return RazorpayConfig{
		KeyID:         getEnvOrDefault("RAZORPAY_KEY_ID", defaultKeyID),
		KeySecret:     getEnvOrDefault("RAZORPAY_KEY_SECRET", defaultSecret),
		AccountID:     getEnvOrDefault("RAZORPAY_ACCOUNT_ID", "R3vQGP5Eqlwg3t"),
		BaseURL:       getEnvOrDefault("RAZORPAY_BASE_URL", "https://api.razorpay.com/v1"),
		WebhookSecret: getEnvOrDefault("RAZORPAY_WEBHOOK_SECRET", "Kisanlink@2025"),
		IsLive:        isLive,
	}
}
