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
	isLive := os.Getenv("RAZORPAY_LIVE") == "true"

	return RazorpayConfig{
		KeyID:         getEnvOrDefault("RAZORPAY_KEY_ID", "rzp_live_RCbm4zwG4WzwGV"),
		KeySecret:     getEnvOrDefault("RAZORPAY_KEY_SECRET", "ANrUne3rX1A6ezzmlfBEqBNx"),
		AccountID:     getEnvOrDefault("RAZORPAY_ACCOUNT_ID", "R3vQGP5Eqlwg3t"),
		BaseURL:       getEnvOrDefault("RAZORPAY_BASE_URL", "https://api.razorpay.com/v1"),
		WebhookSecret: getEnvOrDefault("RAZORPAY_WEBHOOK_SECRET", "Kisanlink@2025"),
		IsLive:        isLive,
	}
}
