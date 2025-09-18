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
	config.LoadEnv()
	fmt.Println("Fixing coupon_rules table structure...")

	// Initialize zap logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Get Postgres config from env
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

	ctx := context.Background()
	pgManager := db.NewPostgresManager(pgConfig, logger)
	if err := pgManager.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to Postgres: %v", err)
	}
	defer func() {
		logger.Info("Closing database connections...")
		if err := pgManager.Close(); err != nil {
			logger.Error("Failed to close Postgres connection", zap.Error(err))
		} else {
			logger.Info("Database connections closed successfully")
		}
	}()

	// Get database connection
	dbConn, err := pgManager.GetDB(ctx, false)
	if err != nil {
		log.Fatalf("Failed to get DB: %v", err)
	}

	fmt.Println("Adding missing columns to coupon_rules table...")

	// Add missing columns
	alterStatements := []string{
		"ALTER TABLE coupon_rules ADD COLUMN IF NOT EXISTS created_by VARCHAR(255) DEFAULT ''",
		"ALTER TABLE coupon_rules ADD COLUMN IF NOT EXISTS updated_by VARCHAR(255) DEFAULT ''",
		"ALTER TABLE coupon_rules ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMP",
		"ALTER TABLE coupon_rules ADD COLUMN IF NOT EXISTS deleted_by VARCHAR(255)",
	}

	for _, stmt := range alterStatements {
		fmt.Printf("Executing: %s\n", stmt)
		if err := dbConn.Exec(stmt).Error; err != nil {
			log.Printf("Warning: Failed to execute statement: %v", err)
		} else {
			fmt.Println("✅ Success")
		}
	}

	fmt.Println("\n✅ Coupon_rules table structure fixed!")

	// Verify the table structure
	fmt.Println("\nVerifying table structure...")
	var columns []string
	err = dbConn.Raw("SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'coupon_rules' ORDER BY ordinal_position").Scan(&columns).Error
	if err != nil {
		log.Printf("Error checking table structure: %v", err)
	} else {
		fmt.Printf("Coupon_rules table columns: %v\n", columns)
	}
}
