package database

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectPostgres() error {
	dsn, err := buildPostgresDSN()
	if err != nil {
		return err
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to initialize Postgres connection pool: %w", err)
	}

	cfg := config.GetPostgresEnvConfig()
	if cfg.MaxConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxConns)
	}
	if cfg.IdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.IdleConns)
	}
	sqlDB.SetConnMaxLifetime(60 * time.Minute)
	sqlDB.SetConnMaxIdleTime(15 * time.Minute)

	DB = db
	return nil
}

func buildPostgresDSN() (string, error) {
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		return dsn, nil
	}

	cfg := config.GetPostgresEnvConfig()
	if cfg.Host == "" || cfg.User == "" || cfg.Password == "" || cfg.DBName == "" {
		return "", fmt.Errorf("POSTGRES_DSN or DB_POSTGRES_* variables must be set for Neon connection")
	}

	if cfg.Port == "" {
		cfg.Port = "5432"
	}
	if cfg.SSLMode == "" {
		cfg.SSLMode = "require"
	}

	parts := []string{
		fmt.Sprintf("host=%s", cfg.Host),
		fmt.Sprintf("user=%s", cfg.User),
		fmt.Sprintf("password=%s", cfg.Password),
		fmt.Sprintf("dbname=%s", cfg.DBName),
		fmt.Sprintf("port=%s", cfg.Port),
		fmt.Sprintf("sslmode=%s", cfg.SSLMode),
		"TimeZone=Asia/Kolkata",
	}

	if cfg.ChannelBinding != "" {
		parts = append(parts, fmt.Sprintf("channel_binding=%s", cfg.ChannelBinding))
	}

	return strings.Join(parts, " "), nil
}

// ConnectAndMigrate connects to PostgreSQL and runs migrations
func ConnectAndMigrate() error {
	if err := ConnectPostgres(); err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := RunMigrationsWithDB(); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}
