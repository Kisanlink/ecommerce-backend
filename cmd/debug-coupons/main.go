package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

func main() {
	config.LoadEnv()
	fmt.Println("Debugging coupon system...")

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

	// Create repository manager
	repoManager := repositories.NewRepositoryManager(pgManager, logger)

	// Check if coupons table exists
	fmt.Println("\n1. Checking if coupons table exists...")
	dbConn, err := pgManager.GetDB(ctx, false)
	if err != nil {
		log.Fatalf("Failed to get DB: %v", err)
	}

	var tableExists bool
	err = dbConn.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'coupons')").Scan(&tableExists).Error
	if err != nil {
		log.Printf("Error checking table existence: %v", err)
	} else {
		fmt.Printf("Coupons table exists: %t\n", tableExists)
	}

	// Get all coupons
	fmt.Println("\n2. Getting all coupons from database...")
	coupons, err := repoManager.CouponRepository.GetAll(ctx, 100, 0)
	if err != nil {
		log.Printf("Failed to get coupons: %v", err)
	} else {
		fmt.Printf("Found %d coupons:\n", len(coupons))
		for i, coupon := range coupons {
			fmt.Printf("%d. Code: %s, Description: %s, Active: %t, Usage: %d/%d\n",
				i+1, coupon.Code, coupon.Description, coupon.IsActive, coupon.UsedCount, coupon.Count)
		}
	}

	// Test specific coupon lookup
	fmt.Println("\n3. Testing specific coupon lookup...")
	testCodes := []string{"KaushikMax18", "KAUSHIKMAX18", "kaushikmax18"}
	for _, code := range testCodes {
		var coupon models.Coupon
		_, err := repoManager.CouponRepository.GetByCode(ctx, code, &coupon)
		if err != nil {
			fmt.Printf("Coupon %s: NOT FOUND (%v)\n", code, err)
		} else {
			fmt.Printf("Coupon %s: FOUND - %s (Active: %t)\n", code, coupon.Description, coupon.IsActive)
		}
	}

	// Check cart table structure
	fmt.Println("\n4. Checking cart table structure...")
	var columns []string
	err = dbConn.Raw("SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'carts' ORDER BY ordinal_position").Scan(&columns).Error
	if err != nil {
		log.Printf("Error checking cart table structure: %v", err)
	} else {
		fmt.Printf("Cart table columns: %v\n", columns)
	}

	// Check if any carts exist
	fmt.Println("\n5. Checking existing carts...")
	var cartCount int64
	err = dbConn.Raw("SELECT COUNT(*) FROM carts").Scan(&cartCount).Error
	if err != nil {
		log.Printf("Error counting carts: %v", err)
	} else {
		fmt.Printf("Total carts in database: %d\n", cartCount)
	}
}
