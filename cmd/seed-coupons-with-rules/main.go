package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/database"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewDevelopment()
	defer logger.Sync()

	// Connect to database
	if err := database.ConnectPostgres(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations to ensure tables exist
	if err := database.RunMigrationsWithDB(); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create database manager and connect
	dbManager := db.NewDatabaseManager()
	ctx := context.Background()
	if err := dbManager.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer dbManager.Close()

	// Get the PostgreSQL manager
	postgresManager := dbManager.GetPostgresManager()
	if postgresManager == nil {
		log.Fatalf("PostgreSQL manager not available")
	}

	couponRepo := repositories.NewCouponRepository(postgresManager, logger)
	couponRuleRepo := repositories.NewCouponRuleRepository(postgresManager, logger)

	// Create sample coupons with rules
	if err := createSampleCoupons(ctx, couponRepo, couponRuleRepo, logger); err != nil {
		log.Fatalf("Failed to create sample coupons: %v", err)
	}

	logger.Info("Sample coupons with rules created successfully!")
}

func createSampleCoupons(ctx context.Context, couponRepo repositories.CouponRepositoryInterface, couponRuleRepo repositories.CouponRuleRepositoryInterface, logger *zap.Logger) error {
	// Sample coupons data
	coupons := []struct {
		code        string
		description string
		count       int
		createdBy   string
		rules       []struct {
			ruleType      models.CouponRuleType
			discountValue float64
			minAmount     float64
			maxDiscount   float64
			validFrom     *time.Time
			validUntil    *time.Time
			firstTimeOnly bool
			categories    []string
			products      []string
		}
	}{
		{
			code:        "WELCOME20",
			description: "20% off for new users",
			count:       100,
			createdBy:   "admin",
			rules: []struct {
				ruleType      models.CouponRuleType
				discountValue float64
				minAmount     float64
				maxDiscount   float64
				validFrom     *time.Time
				validUntil    *time.Time
				firstTimeOnly bool
				categories    []string
				products      []string
			}{
				{
					ruleType:      models.CouponRuleTypePercentage,
					discountValue: 20,
					minAmount:     100,
					maxDiscount:   50,
					firstTimeOnly: true,
					validFrom:     timePtr(time.Now()),
					validUntil:    timePtr(time.Now().AddDate(0, 1, 0)), // 1 month from now
				},
			},
		},
		{
			code:        "SAVE50",
			description: "50 rupees off on orders above 200",
			count:       50,
			createdBy:   "admin",
			rules: []struct {
				ruleType      models.CouponRuleType
				discountValue float64
				minAmount     float64
				maxDiscount   float64
				validFrom     *time.Time
				validUntil    *time.Time
				firstTimeOnly bool
				categories    []string
				products      []string
			}{
				{
					ruleType:      models.CouponRuleTypeFixed,
					discountValue: 50,
					minAmount:     200,
					validFrom:     timePtr(time.Now()),
					validUntil:    timePtr(time.Now().AddDate(0, 0, 15)), // 15 days from now
				},
			},
		},
		{
			code:        "FREESHIP",
			description: "Free shipping on all orders",
			count:       200,
			createdBy:   "admin",
			rules: []struct {
				ruleType      models.CouponRuleType
				discountValue float64
				minAmount     float64
				maxDiscount   float64
				validFrom     *time.Time
				validUntil    *time.Time
				firstTimeOnly bool
				categories    []string
				products      []string
			}{
				{
					ruleType:   models.CouponRuleTypeFreeShipping,
					minAmount:  0,
					validFrom:  timePtr(time.Now()),
					validUntil: timePtr(time.Now().AddDate(0, 0, 30)), // 30 days from now
				},
			},
		},
		{
			code:        "VEGGIE15",
			description: "15% off on vegetables",
			count:       75,
			createdBy:   "admin",
			rules: []struct {
				ruleType      models.CouponRuleType
				discountValue float64
				minAmount     float64
				maxDiscount   float64
				validFrom     *time.Time
				validUntil    *time.Time
				firstTimeOnly bool
				categories    []string
				products      []string
			}{
				{
					ruleType:      models.CouponRuleTypePercentage,
					discountValue: 15,
					minAmount:     50,
					maxDiscount:   25,
					categories:    []string{"vegetables", "organic"},
					validFrom:     timePtr(time.Now()),
					validUntil:    timePtr(time.Now().AddDate(0, 0, 20)), // 20 days from now
				},
			},
		},
		{
			code:        "MEGA30",
			description: "30% off on orders above 500",
			count:       25,
			createdBy:   "admin",
			rules: []struct {
				ruleType      models.CouponRuleType
				discountValue float64
				minAmount     float64
				maxDiscount   float64
				validFrom     *time.Time
				validUntil    *time.Time
				firstTimeOnly bool
				categories    []string
				products      []string
			}{
				{
					ruleType:      models.CouponRuleTypePercentage,
					discountValue: 30,
					minAmount:     500,
					maxDiscount:   100,
					validFrom:     timePtr(time.Now()),
					validUntil:    timePtr(time.Now().AddDate(0, 0, 7)), // 7 days from now
				},
			},
		},
	}

	// Create coupons and their rules
	for _, couponData := range coupons {
		// Check if coupon already exists
		var existingCoupon models.Coupon
		_, err := couponRepo.GetByCode(ctx, couponData.code, &existingCoupon)
		if err == nil {
			logger.Info("Coupon already exists, skipping",
				zap.String("code", couponData.code))
			continue
		}

		// Create coupon
		coupon := models.NewCoupon(couponData.code, couponData.description, couponData.createdBy)
		coupon.Count = couponData.count

		if err := couponRepo.Create(ctx, coupon); err != nil {
			return fmt.Errorf("failed to create coupon %s: %w", couponData.code, err)
		}

		logger.Info("Created coupon",
			zap.String("code", couponData.code),
			zap.String("id", coupon.ID))

		// Create rules for the coupon
		for _, ruleData := range couponData.rules {
			rule := models.NewCouponRule(coupon.ID, ruleData.ruleType, ruleData.discountValue)
			rule.SetMinAmount(ruleData.minAmount)
			if ruleData.maxDiscount > 0 {
				rule.SetMaxDiscount(ruleData.maxDiscount)
			}
			rule.SetValidityPeriod(ruleData.validFrom, ruleData.validUntil)
			rule.SetFirstTimeOnly(ruleData.firstTimeOnly)

			// Add categories
			for _, category := range ruleData.categories {
				rule.AddCategory(category)
			}

			// Add products
			for _, product := range ruleData.products {
				rule.AddProduct(product)
			}

			if err := couponRuleRepo.Create(ctx, rule); err != nil {
				return fmt.Errorf("failed to create rule for coupon %s: %w", couponData.code, err)
			}

			logger.Info("Created coupon rule",
				zap.String("coupon_code", couponData.code),
				zap.String("rule_id", rule.ID),
				zap.String("rule_type", string(ruleData.ruleType)))
		}
	}

	return nil
}

// Helper function to create time pointers
func timePtr(t time.Time) *time.Time {
	return &t
}
