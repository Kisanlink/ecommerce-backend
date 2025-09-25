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
	fmt.Println("Starting coupon seeding script...")

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

	// Define the coupons from the image
	coupons := []struct {
		code        string
		description string
		discount    float64
		usageLimit  int
	}{
		{
			code:        "KAUSHIKMAX18",
			description: "KaushikMax18 - 18% discount coupon",
			discount:    18.0,
			usageLimit:  50,
		},
		{
			code:        "CHARANPRIME18",
			description: "CharanPrime18 - 18% discount coupon",
			discount:    18.0,
			usageLimit:  50,
		},
		{
			code:        "MANIELITE18",
			description: "ManiElite18 - 18% discount coupon",
			discount:    18.0,
			usageLimit:  50,
		},
		{
			code:        "MEHERFEST18",
			description: "MeherFest18 - 18% discount coupon",
			discount:    18.0,
			usageLimit:  50,
		},
		{
			code:        "DEAL5",
			description: "Deal5 - 5% discount coupon",
			discount:    5.0,
			usageLimit:  50,
		},
		{
			code:        "SAVE10",
			description: "Save10 - 10% discount coupon",
			discount:    10.0,
			usageLimit:  50,
		},
		{
			code:        "OFFER15",
			description: "Offer15 - 15% discount coupon",
			discount:    15.0,
			usageLimit:  50,
		},
		{
			code:        "MEGA5",
			description: "Mega5 - 5% discount coupon",
			discount:    5.0,
			usageLimit:  50,
		},
		{
			code:        "SUPER10",
			description: "Super10 - 10% discount coupon",
			discount:    10.0,
			usageLimit:  50,
		},
	}

	fmt.Printf("Seeding %d coupons...\n", len(coupons))

	// Create each coupon
	createdCount := 0
	for _, couponData := range coupons {
		// Check if coupon already exists
		var existingCoupon models.Coupon
		_, err := repoManager.CouponRepository.GetByCode(ctx, couponData.code, &existingCoupon)
		if err == nil {
			fmt.Printf("Coupon %s already exists, skipping...\n", couponData.code)
			continue
		}

		// Create new coupon
		coupon := models.NewCoupon(
			couponData.code,
			couponData.description,
			"system", // created by
		)

		// Create coupon rule
		rule := models.NewCouponRule(
			coupon.ID,
			models.CouponRuleTypePercentage,
			couponData.discount,
		)
		rule.SetMinAmount(100.0) // Minimum order amount

		// Set max discount as pointer
		maxDiscount := 1000.0
		rule.MaxDiscount = &maxDiscount

		// Set user limit as pointer
		rule.UserLimit = &couponData.usageLimit

		rule.SetFirstTimeOnly(false) // Can be used multiple times

		// Add rule to coupon
		coupon.AddRule(rule)

		// Set usage count
		coupon.Count = couponData.usageLimit
		coupon.UsedCount = 0
		coupon.IsActive = true

		// Save to database
		if err := repoManager.CouponRepository.Create(ctx, coupon); err != nil {
			logger.Error("Failed to create coupon",
				zap.String("code", couponData.code),
				zap.Error(err))
			fmt.Printf("Failed to create coupon %s: %v\n", couponData.code, err)
		} else {
			createdCount++
			fmt.Printf("Created coupon: %s (%.0f%% discount, usage limit: %d)\n",
				couponData.code, couponData.discount, couponData.usageLimit)
		}
	}

	fmt.Printf("\nSuccessfully created %d out of %d coupons.\n", createdCount, len(coupons))

	// Verify coupons were created
	fmt.Println("\nVerifying created coupons...")
	allCoupons, err := repoManager.CouponRepository.GetAll(ctx, 100, 0)
	if err != nil {
		log.Printf("Failed to verify coupons: %v", err)
		return
	}

	fmt.Printf("Total coupons in database: %d\n", len(allCoupons))
	for _, coupon := range allCoupons {
		fmt.Printf("- %s: %s (Active: %t, Usage: %d/%d)\n",
			coupon.Code, coupon.Description, coupon.IsActive, coupon.UsedCount, coupon.Count)
	}
}
