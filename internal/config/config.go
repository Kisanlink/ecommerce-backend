// config.go - Loads environment variables from .env
// Author: Mani-KisanLink
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
	Host      string
	Port      string
	User      string
	Password  string
	DBName    string
	SSLMode   string
	MaxConns  int
	IdleConns int
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
		Host:      os.Getenv("DB_POSTGRES_HOST"),
		Port:      os.Getenv("DB_POSTGRES_PORT"),
		User:      os.Getenv("DB_POSTGRES_USER"),
		Password:  os.Getenv("DB_POSTGRES_PASSWORD"),
		DBName:    os.Getenv("DB_POSTGRES_DBNAME"),
		SSLMode:   os.Getenv("DB_POSTGRES_SSLMODE"),
		MaxConns:  maxConns,
		IdleConns: idleConns,
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

// getEnvOrDefault returns environment variable value or default if empty
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
