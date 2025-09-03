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

type Cart struct {
	ID     string `json:"id" gorm:"type:varchar(255);primary_key"`
	UserID string `json:"user_id"`
}

type CartItem struct {
	ID        string `json:"id" gorm:"type:varchar(255);primary_key"`
	CartID    string `json:"cart_id"`
	ProductID string `json:"product_id"`
}

func main() {
	fmt.Println("=== Getting Existing IDs ===")

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

	// Get all cart IDs
	var carts []Cart
	database.Find(&carts)

	cartIDs := make([]string, len(carts))
	for i, cart := range carts {
		cartIDs[i] = cart.ID
	}

	fmt.Printf("Existing cart IDs: %v\n", cartIDs)

	// Get all cart item IDs
	var cartItems []CartItem
	database.Find(&cartItems)

	cartItemIDs := make([]string, len(cartItems))
	for i, item := range cartItems {
		cartItemIDs[i] = item.ID
	}

	fmt.Printf("Existing cart item IDs: %v\n", cartItemIDs)

	fmt.Println("\n=== ID Collection Complete ===")
}
