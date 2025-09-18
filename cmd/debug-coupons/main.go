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

	// Check if coupon_rules table exists
	fmt.Println("\n2. Checking if coupon_rules table exists...")
	err = dbConn.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'coupon_rules')").Scan(&tableExists).Error
	if err != nil {
		log.Printf("Error checking coupon_rules table existence: %v", err)
	} else {
		fmt.Printf("Coupon_rules table exists: %t\n", tableExists)
	}

	// Get all coupons
	fmt.Println("\n3. Getting all coupons from database...")
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

	// Test specific coupon lookup with rules
	fmt.Println("\n4. Testing specific coupon lookup with rules...")
	testCodes := []string{"KAUSHIKMAX18", "MANIELITE18", "CHARANPRIME18", "MEHERFEST18"}
	for _, code := range testCodes {
		fmt.Printf("\n--- Testing coupon: %s ---\n", code)

		// Test basic lookup
		var coupon models.Coupon
		_, err := repoManager.CouponRepository.GetByCode(ctx, code, &coupon)
		if err != nil {
			fmt.Printf("Basic lookup: NOT FOUND (%v)\n", err)
			continue
		}
		fmt.Printf("Basic lookup: FOUND - %s (Active: %t)\n", coupon.Description, coupon.IsActive)

		// Test lookup with rules
		couponWithRules, err := repoManager.CouponRepository.GetByCodeWithRules(ctx, code)
		if err != nil {
			fmt.Printf("With rules lookup: FAILED (%v)\n", err)
			continue
		}
		fmt.Printf("With rules lookup: SUCCESS\n")
		fmt.Printf("  - Coupon ID: %s\n", couponWithRules.ID)
		fmt.Printf("  - Rules count: %d\n", len(couponWithRules.Rules))

		// Check if coupon is valid
		fmt.Printf("  - IsValid(): %t\n", couponWithRules.IsValid())

		// Check each rule
		for j, rule := range couponWithRules.Rules {
			fmt.Printf("  - Rule %d: Type=%s, Value=%.2f, MinAmount=%.2f, Valid=%t\n",
				j+1, rule.RuleType, rule.DiscountValue, rule.MinAmount, rule.IsValid())
		}

		// Test cart validation
		testCartTotal := 150.0
		fmt.Printf("  - CanUse(cartTotal=%.2f): %t\n", testCartTotal, couponWithRules.CanUse(testCartTotal, "test-user", true))
	}

	// Check cart table structure
	fmt.Println("\n5. Checking cart table structure...")
	var columns []string
	err = dbConn.Raw("SELECT column_name FROM information_schema.columns WHERE table_schema = 'public' AND table_name = 'carts' ORDER BY ordinal_position").Scan(&columns).Error
	if err != nil {
		log.Printf("Error checking cart table structure: %v", err)
	} else {
		fmt.Printf("Cart table columns: %v\n", columns)
	}

	// Check if any carts exist
	fmt.Println("\n6. Checking existing carts...")
	var cartCount int64
	err = dbConn.Raw("SELECT COUNT(*) FROM carts").Scan(&cartCount).Error
	if err != nil {
		log.Printf("Error counting carts: %v", err)
	} else {
		fmt.Printf("Total carts in database: %d\n", cartCount)
	}
}
