package database

import (
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectPostgres() error {
	dsn := os.Getenv("POSTGRES_DSN")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to Postgres: %w", err)
	}
	DB = db
	return nil
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
