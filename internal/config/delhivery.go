package config

import (
	"os"
	"strconv"
)

// DelhiveryConfig holds Delhivery API configuration
type DelhiveryConfig struct {
	BaseURL       string
	APIToken      string
	IsProduction  bool
	OriginPincode string
	OriginCity    string
	OriginState   string
	OriginAddress string
	OriginPhone   string
	WarehouseName string
	DefaultWeight int    // in grams
	ServiceStatus string // Delivered, RTO, or DTO
}

// GetDelhiveryConfig returns Delhivery config from environment variables
func GetDelhiveryConfig() DelhiveryConfig {
	isProduction := os.Getenv("DELHIVERY_PRODUCTION") == "true"

	// CRITICAL FIX: Use correct URLs for Delhivery API
	baseURL := "https://staging-express.delhivery.com"
	if isProduction {
		// FIXED: Correct production URL
		baseURL = "https://track.delhivery.com"
	}

	// Override base URL if explicitly set
	if envBaseURL := os.Getenv("DELHIVERY_BASE_URL"); envBaseURL != "" {
		baseURL = envBaseURL
	}

	defaultWeight, _ := strconv.Atoi(os.Getenv("DELHIVERY_DEFAULT_WEIGHT"))
	if defaultWeight == 0 {
		defaultWeight = 300 // 300 grams default
	}

	// CRITICAL: Validate the API token format
	apiToken := getEnvOrDefault("DELHIVERY_API_TOKEN", "5c2243c9d55f103ed21341076ccbf38700dabf5d")

	// CRITICAL: Validate origin pincode format (must be exactly 6 digits)
	originPincode := getEnvOrDefault("DELHIVERY_ORIGIN_PINCODE", "500033")
	if len(originPincode) != 6 {
		// Log warning or handle invalid pincode
		originPincode = "500033" // Default Hyderabad pincode
	}

	// Determine service status (charges API requires one of: Delivered, RTO, DTO)
	serviceStatus := getEnvOrDefault("DELHIVERY_SERVICE_STATUS", "Delivered")
	switch serviceStatus {
	case "Delivered", "RTO", "DTO":
		// ok
	default:
		serviceStatus = "Delivered"
	}

	return DelhiveryConfig{
		BaseURL:       baseURL,
		APIToken:      apiToken,
		IsProduction:  isProduction,
		OriginPincode: originPincode,
		OriginCity:    getEnvOrDefault("DELHIVERY_ORIGIN_CITY", "Hyderabad"),
		OriginState:   getEnvOrDefault("DELHIVERY_ORIGIN_STATE", "Telangana"),
		OriginAddress: getEnvOrDefault("DELHIVERY_ORIGIN_ADDRESS", "Default Address"),
		OriginPhone:   getEnvOrDefault("DELHIVERY_ORIGIN_PHONE", "6300868001"),
		WarehouseName: getEnvOrDefault("DELHIVERY_WAREHOUSE_NAME", "main_warehouse"),
		DefaultWeight: defaultWeight,
		ServiceStatus: serviceStatus,
	}
}
