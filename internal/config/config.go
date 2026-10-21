package config

import (
	"log"

	"os"
	"strconv"

	"github.com/joho/godotenv"
)

// LoadEnv loads environment variables from .env file
func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found or error loading .env, using system env vars")
	}
}

// PostgresEnvConfig holds PostgreSQL connection configuration from env
type PostgresEnvConfig struct {
	Host           string
	Port           string
	User           string
	Password       string
	DBName         string
	SSLMode        string
	ChannelBinding string
	MaxConns       int
	IdleConns      int
}

// GetPostgresEnvConfig returns Postgres config from environment variables
func GetPostgresEnvConfig() PostgresEnvConfig {
	maxConns, _ := strconv.Atoi(os.Getenv("DB_POSTGRES_MAX_CONNS"))
	if maxConns == 0 {
		maxConns = 10
	}
	idleConns, _ := strconv.Atoi(os.Getenv("DB_POSTGRES_IDLE_CONNS"))
	if idleConns == 0 {
		idleConns = 5
	}
	return PostgresEnvConfig{
		Host:           getEnvOrDefault("DB_POSTGRES_HOST", ""),
		Port:           getEnvOrDefault("DB_POSTGRES_PORT", "5432"),
		User:           getEnvOrDefault("DB_POSTGRES_USER", ""),
		Password:       getEnvOrDefault("DB_POSTGRES_PASSWORD", ""),
		DBName:         getEnvOrDefault("DB_POSTGRES_DBNAME", ""),
		SSLMode:        getEnvOrDefault("DB_POSTGRES_SSLMODE", "require"),
		ChannelBinding: os.Getenv("DB_POSTGRES_CHANNEL_BINDING"),
		MaxConns:       maxConns,
		IdleConns:      idleConns,
	}
}

// RedisEnvConfig holds Redis connection configuration from env
type RedisEnvConfig struct {
	Host            string
	Port            string
	Password        string
	DB              int
	PoolSize        int
	DialTimeoutSec  int
	ReadTimeoutSec  int
	WriteTimeoutSec int
	PoolTimeoutSec  int
	IdleTimeoutMin  int
	MaxConnAgeMin   int
}

// GetRedisEnvConfig returns Redis config from environment variables
func GetRedisEnvConfig() RedisEnvConfig {
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	poolSize, _ := strconv.Atoi(os.Getenv("REDIS_POOL_SIZE"))
	if poolSize == 0 {
		poolSize = 10
	}
	dialTimeout, _ := strconv.Atoi(os.Getenv("REDIS_DIAL_TIMEOUT_SEC"))
	if dialTimeout == 0 {
		dialTimeout = 5
	}
	readTimeout, _ := strconv.Atoi(os.Getenv("REDIS_READ_TIMEOUT_SEC"))
	if readTimeout == 0 {
		readTimeout = 3
	}
	writeTimeout, _ := strconv.Atoi(os.Getenv("REDIS_WRITE_TIMEOUT_SEC"))
	if writeTimeout == 0 {
		writeTimeout = 3
	}
	poolTimeout, _ := strconv.Atoi(os.Getenv("REDIS_POOL_TIMEOUT_SEC"))
	if poolTimeout == 0 {
		poolTimeout = 4
	}
	idleTimeout, _ := strconv.Atoi(os.Getenv("REDIS_IDLE_TIMEOUT_MIN"))
	if idleTimeout == 0 {
		idleTimeout = 300
	}
	maxConnAge, _ := strconv.Atoi(os.Getenv("REDIS_MAX_CONN_AGE_MIN"))
	if maxConnAge == 0 {
		maxConnAge = 60
	}

	return RedisEnvConfig{
		Host:            getEnvOrDefault("REDIS_HOST", "localhost"),
		Port:            getEnvOrDefault("REDIS_PORT", "6379"),
		Password:        os.Getenv("REDIS_PASSWORD"),
		DB:              db,
		PoolSize:        poolSize,
		DialTimeoutSec:  dialTimeout,
		ReadTimeoutSec:  readTimeout,
		WriteTimeoutSec: writeTimeout,
		PoolTimeoutSec:  poolTimeout,
		IdleTimeoutMin:  idleTimeout,
		MaxConnAgeMin:   maxConnAge,
	}
}

// CartEnvConfig holds cart-specific configuration from env
type CartEnvConfig struct {
	ExpiryHours int
	MaxItems    int
}

// GetCartEnvConfig returns cart config from environment variables
func GetCartEnvConfig() CartEnvConfig {
	expiryHours, _ := strconv.Atoi(os.Getenv("CART_EXPIRY_HOURS"))
	if expiryHours == 0 {
		expiryHours = 168 // 7 days default
	}
	maxItems, _ := strconv.Atoi(os.Getenv("CART_MAX_ITEMS"))
	if maxItems == 0 {
		maxItems = 50 // default max items
	}

	return CartEnvConfig{
		ExpiryHours: expiryHours,
		MaxItems:    maxItems,
	}
}

// InventoryEnvConfig holds inventory-specific configuration from env
type InventoryEnvConfig struct {
	DefaultMinStockLevel int
	DefaultMaxStockLevel int
	DefaultReorderPoint  int
}

// GetInventoryEnvConfig returns inventory config from environment variables
func GetInventoryEnvConfig() InventoryEnvConfig {
	minStockLevel, _ := strconv.Atoi(os.Getenv("INVENTORY_DEFAULT_MIN_STOCK_LEVEL"))
	if minStockLevel == 0 {
		minStockLevel = 10 // default min stock level
	}
	maxStockLevel, _ := strconv.Atoi(os.Getenv("INVENTORY_DEFAULT_MAX_STOCK_LEVEL"))
	if maxStockLevel == 0 {
		maxStockLevel = 1000 // default max stock level
	}
	reorderPoint, _ := strconv.Atoi(os.Getenv("INVENTORY_DEFAULT_REORDER_POINT"))
	if reorderPoint == 0 {
		reorderPoint = 20 // default reorder point
	}

	return InventoryEnvConfig{
		DefaultMinStockLevel: minStockLevel,
		DefaultMaxStockLevel: maxStockLevel,
		DefaultReorderPoint:  reorderPoint,
	}
}

// SwaggerEnvConfig holds Swagger documentation configuration from env
type SwaggerEnvConfig struct {
	Enabled      bool
	Title        string
	Description  string
	Version      string
	Host         string
	BasePath     string
	ContactName  string
	ContactURL   string
	ContactEmail string
	LicenseName  string
	LicenseURL   string
}

// GetSwaggerEnvConfig returns Swagger config from environment variables
func GetSwaggerEnvConfig() SwaggerEnvConfig {
	enabled := getEnvOrDefault("ENABLE_DOCS", "true") == "true"

	return SwaggerEnvConfig{
		Enabled:      enabled,
		Title:        getEnvOrDefault("SWAGGER_TITLE", "Ecommerce Backend API"),
		Description:  getEnvOrDefault("SWAGGER_DESCRIPTION", "A comprehensive ecommerce backend API with authentication, product management, cart, orders, and payment integration."),
		Version:      getEnvOrDefault("SWAGGER_VERSION", "1.0.0"),
		Host:         getEnvOrDefault("SWAGGER_HOST", "localhost:8080"),
		BasePath:     getEnvOrDefault("SWAGGER_BASE_PATH", "/api/v1"),
		ContactName:  getEnvOrDefault("SWAGGER_CONTACT_NAME", "API Support"),
		ContactURL:   getEnvOrDefault("SWAGGER_CONTACT_URL", "http://www.kisanlink.com/support"),
		ContactEmail: getEnvOrDefault("SWAGGER_CONTACT_EMAIL", "support@kisanlink.com"),
		LicenseName:  getEnvOrDefault("SWAGGER_LICENSE_NAME", "MIT"),
		LicenseURL:   getEnvOrDefault("SWAGGER_LICENSE_URL", "https://opensource.org/licenses/MIT"),
	}
}

// WhatsAppEnvConfig holds WhatsApp Business API configuration from env
type WhatsAppEnvConfig struct {
	Enabled        bool
	AccessToken    string
	PhoneNumberID  string
	BusinessAcctID string
	WebhookToken   string
	APIBaseURL     string
	OTPTemplate    string
	OrderTemplate  string
}

// GetWhatsAppEnvConfig returns WhatsApp config from environment variables
func GetWhatsAppEnvConfig() WhatsAppEnvConfig {
	enabled := getEnvOrDefault("WHATSAPP_ENABLED", "false") == "true"

	return WhatsAppEnvConfig{
		Enabled:        enabled,
		AccessToken:    os.Getenv("WHATSAPP_ACCESS_TOKEN"),
		PhoneNumberID:  os.Getenv("WHATSAPP_PHONE_NUMBER_ID"),
		BusinessAcctID: os.Getenv("WHATSAPP_BUSINESS_ACCOUNT_ID"),
		WebhookToken:   os.Getenv("WHATSAPP_WEBHOOK_TOKEN"),
		APIBaseURL:     getEnvOrDefault("WHATSAPP_API_BASE_URL", "https://graph.facebook.com/v22.0"),
		OTPTemplate:    os.Getenv("WHATSAPP_OTP_TEMPLATE_NAME"),
		OrderTemplate:  getEnvOrDefault("WHATSAPP_ORDER_TEMPLATE_NAME", "mani_test1"),
	}
}

// R2Config holds Cloudflare R2 configuration from env
type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicDomain    string // Optional: for public URL generation (e.g., pub-xxx.r2.dev or custom domain)
	Endpoint        string // Auto-generated from AccountID
}

// GetR2Config returns Cloudflare R2 config from environment variables
func GetR2Config() R2Config {
	accountID := os.Getenv("R2_ACCOUNT_ID")
	endpoint := ""
	if accountID != "" {
		endpoint = "https://" + accountID + ".r2.cloudflarestorage.com"
	}

	return R2Config{
		AccountID:       accountID,
		AccessKeyID:     os.Getenv("R2_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("R2_SECRET_ACCESS_KEY"),
		BucketName:      getEnvOrDefault("R2_BUCKET_NAME", "amrti"),
		PublicDomain:    os.Getenv("R2_PUBLIC_DOMAIN"), // e.g., "pub-xxx.r2.dev" or custom domain
		Endpoint:        endpoint,
	}
}

// IsR2Configured returns true if R2 is properly configured
func (c R2Config) IsR2Configured() bool {
	return c.AccountID != "" && c.AccessKeyID != "" && c.SecretAccessKey != "" && c.BucketName != ""
}

// getEnvOrDefault returns environment variable value or default if empty
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
