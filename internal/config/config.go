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
