package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

type CartItem struct {
	ID         string  `json:"id" gorm:"type:varchar(255);primary_key"`
	CartID     string  `json:"cart_id" gorm:"not null;index"`
	ProductID  string  `json:"product_id" gorm:"not null;index"`
	Quantity   int     `json:"quantity" gorm:"not null"`
	UnitPrice  float64 `json:"unit_price" gorm:"not null"`
	TotalPrice float64 `json:"total_price" gorm:"not null"`
}

func main() {
	fmt.Println("=== Checking Cart Items Database ===")

	// Load environment variables
	config.LoadEnv()

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
		// Add S3 configuration
		S3Region: os.Getenv("DB_S3_REGION"),
		S3Bucket: os.Getenv("DB_S3_BUCKET"),
	}

	ctx := context.Background()
	pgManager := db.NewPostgresManager(pgConfig, logger)
	if err := pgManager.Connect(ctx); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	database, err := pgManager.GetDB(ctx, false)
	if err != nil {
		log.Fatalf("Error getting database connection: %v", err)
	}

	// Delete cart items with empty IDs
	result := database.Where("id = ?", "").Delete(&CartItem{})
	if result.Error != nil {
		log.Fatalf("Error deleting empty ID cart items: %v", result.Error)
	}

	fmt.Printf("Deleted %d cart items with empty IDs\n", result.RowsAffected)

	// Also delete cart items with empty cart_id
	result = database.Where("cart_id = ?", "").Delete(&CartItem{})
	if result.Error != nil {
		log.Fatalf("Error deleting empty cart_id cart items: %v", result.Error)
	}

	fmt.Printf("Deleted %d cart items with empty cart_id\n", result.RowsAffected)

	// Show remaining cart items
	var allItems []CartItem
	database.Find(&allItems)

	fmt.Printf("\nRemaining cart items in database:\n")
	for i, item := range allItems {
		fmt.Printf("  %d: ID='%s' (len:%d), CartID='%s' (len:%d), ProductID='%s'\n",
			i+1, item.ID, len(item.ID), item.CartID, len(item.CartID), item.ProductID)
	}

	fmt.Println("\n=== Database Cleanup Complete ===")
}
