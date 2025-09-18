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
	fmt.Println("Fixing coupon rules for all coupons...")

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

	// Create repositories
	couponRepo := repositories.NewCouponRepository(pgManager, logger)
	couponRuleRepo := repositories.NewCouponRuleRepository(pgManager, logger)

	// Get all coupons
	coupons, err := couponRepo.GetAll(ctx, 100, 0)
	if err != nil {
		log.Fatalf("Failed to get coupons: %v", err)
	}

	fmt.Printf("Found %d coupons to process\n", len(coupons))

	// Process each coupon
	for _, coupon := range coupons {
		fmt.Printf("\n--- Processing coupon: %s ---\n", coupon.Code)

		// Check if coupon already has rules
		couponWithRules, err := couponRepo.GetByCodeWithRules(ctx, coupon.Code)
		if err != nil {
			fmt.Printf("Error getting coupon with rules: %v\n", err)
			continue
		}

		if len(couponWithRules.Rules) > 0 {
			fmt.Printf("Coupon %s already has %d rules, skipping\n", coupon.Code, len(couponWithRules.Rules))
			continue
		}

		// Create a default rule for this coupon
		fmt.Printf("Creating default rule for coupon %s\n", coupon.Code)

		// Create a percentage rule with 18% discount, minimum 100 amount
		rule := models.NewCouponRule(coupon.ID, models.CouponRuleTypePercentage, 18.0)
		rule.SetMinAmount(100.0)
		rule.SetMaxDiscount(1000.0)
		rule.SetFirstTimeOnly(false)

		// Save the rule
		if err := couponRuleRepo.Create(ctx, rule); err != nil {
			fmt.Printf("Failed to create rule for coupon %s: %v\n", coupon.Code, err)
			continue
		}

		fmt.Printf("✅ Created rule for coupon %s: %s discount, min amount: %.2f\n",
			coupon.Code, rule.RuleType, rule.MinAmount)
	}

	fmt.Println("\n✅ Coupon rules fix completed!")

	// Verify the results
	fmt.Println("\nVerifying results...")
	for _, coupon := range coupons {
		couponWithRules, err := couponRepo.GetByCodeWithRules(ctx, coupon.Code)
		if err != nil {
			fmt.Printf("Error verifying coupon %s: %v\n", coupon.Code, err)
			continue
		}

		fmt.Printf("Coupon %s: %d rules, Valid: %t\n",
			coupon.Code, len(couponWithRules.Rules), couponWithRules.IsValid())
	}
}
