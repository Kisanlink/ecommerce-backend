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

	// Update existing images to set s3_key = image_url for backward compatibility
	fmt.Println("Updating existing images with s3_key values...")

	// Execute the update query
	gormDB, err := pgManager.GetDB(ctx, false)
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	result := gormDB.Exec(`
		UPDATE images 
		SET s3_key = image_url 
		WHERE s3_key IS NULL OR s3_key = ''
	`)

	if result.Error != nil {
		log.Fatalf("Failed to update images: %v", result.Error)
	}

	rowsAffected := result.RowsAffected

	fmt.Printf("Successfully updated %d image records\n", rowsAffected)
	fmt.Println("Migration completed successfully!")
}
