package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

func main() {
	// Load configuration
	config.LoadEnv()

	// Initialize logger
	logger, _ := zap.NewProduction()

	// Get database configuration
	pgEnv := config.GetPostgresEnvConfig()
	pgConfig := &db.Config{
		PostgresHost:      pgEnv.Host,
		PostgresPort:      pgEnv.Port,
		PostgresUser:      pgEnv.User,
		PostgresPassword:  pgEnv.Password,
		PostgresDBName:    pgEnv.DBName,
		PostgresSSLMode:   pgEnv.SSLMode,
		PostgresMaxConns:  pgEnv.MaxConns,
		PostgresIdleConns: pgEnv.IdleConns,
	}

	// Initialize database connection
	ctx := context.Background()
	pgManager := db.NewPostgresManager(pgConfig, logger)
	if err := pgManager.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer func() {
		if err := pgManager.Close(); err != nil {
			logger.Error("Failed to close Postgres connection", zap.Error(err))
		}
	}()

	// Get database connection
	gormDB, err := pgManager.GetDB(ctx, false)
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	fmt.Println("Running database migration to fix s3_key column...")

	// Step 1: Add the s3_key column as nullable first
	fmt.Println("Step 1: Adding s3_key column...")
	result1 := gormDB.Exec("ALTER TABLE images ADD COLUMN IF NOT EXISTS s3_key TEXT")
	if result1.Error != nil {
		log.Fatalf("Failed to add s3_key column: %v", result1.Error)
	}
	fmt.Println("✓ s3_key column added successfully")

	// Step 2: Update existing records
	fmt.Println("Step 2: Updating existing records...")
	result2 := gormDB.Exec(`
		UPDATE images 
		SET s3_key = image_url 
		WHERE s3_key IS NULL OR s3_key = ''
	`)
	if result2.Error != nil {
		log.Fatalf("Failed to update images: %v", result2.Error)
	}
	fmt.Printf("✓ Successfully updated %d image records\n", result2.RowsAffected)

	fmt.Println("Migration completed successfully!")
	fmt.Println("You can now start the server with: go run ./cmd/server")
}

