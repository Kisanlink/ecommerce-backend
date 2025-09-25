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
	fmt.Println("Running coupon normalization migration...")

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

	fmt.Println("Executing coupon normalization migration...")

	// Step 1: Create the coupon_rules table
	fmt.Println("Step 1: Creating coupon_rules table...")
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS coupon_rules (
		id VARCHAR(255) PRIMARY KEY,
		coupon_id VARCHAR(255) NOT NULL,
		rule_type VARCHAR(20) NOT NULL CHECK (rule_type IN ('percentage', 'fixed', 'free_shipping')),
		discount_value DECIMAL(10,2) NOT NULL CHECK (discount_value >= 0),
		min_amount DECIMAL(10,2) DEFAULT 0 CHECK (min_amount >= 0),
		max_discount DECIMAL(10,2),
		valid_from TIMESTAMP,
		valid_until TIMESTAMP,
		user_limit INTEGER,
		first_time_only BOOLEAN DEFAULT false,
		categories JSONB,
		products JSONB,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		CONSTRAINT fk_coupon_rules_coupon_id FOREIGN KEY (coupon_id) REFERENCES coupons(id) ON DELETE CASCADE
	)`

	if err := dbConn.Exec(createTableSQL).Error; err != nil {
		log.Fatalf("Failed to create coupon_rules table: %v", err)
	}

	// Step 2: Create indexes
	fmt.Println("Step 2: Creating indexes...")
	indexes := []string{
		"CREATE INDEX IF NOT EXISTS idx_coupon_rules_coupon_id ON coupon_rules(coupon_id)",
		"CREATE INDEX IF NOT EXISTS idx_coupon_rules_type ON coupon_rules(rule_type)",
		"CREATE INDEX IF NOT EXISTS idx_coupon_rules_valid_dates ON coupon_rules(valid_from, valid_until)",
	}

	for _, indexSQL := range indexes {
		if err := dbConn.Exec(indexSQL).Error; err != nil {
			log.Printf("Warning: Failed to create index: %v", err)
		}
	}

	// Step 3: Migrate existing data
	fmt.Println("Step 3: Migrating existing coupon data...")
	migrateDataSQL := `
	INSERT INTO coupon_rules (
		id, coupon_id, rule_type, discount_value, min_amount, max_discount,
		valid_from, valid_until, user_limit, first_time_only, categories, products,
		created_at, updated_at
	)
	SELECT 
		'RULE_' || SUBSTRING(c.id, 1, 8) || '_' || EXTRACT(EPOCH FROM NOW())::TEXT,
		c.id as coupon_id,
		COALESCE(c.rules->>'type', 'percentage') as rule_type,
		COALESCE((c.rules->>'value')::DECIMAL(10,2), 0) as discount_value,
		COALESCE((c.rules->>'min_amount')::DECIMAL(10,2), 0) as min_amount,
		CASE 
			WHEN c.rules->>'max_discount' IS NOT NULL AND c.rules->>'max_discount' != '' 
			THEN (c.rules->>'max_discount')::DECIMAL(10,2)
			ELSE NULL
		END as max_discount,
		NULL as valid_from,
		NULL as valid_until,
		CASE 
			WHEN c.rules->>'user_limit' IS NOT NULL AND c.rules->>'user_limit' != '' 
			THEN (c.rules->>'user_limit')::INTEGER
			ELSE NULL
		END as user_limit,
		COALESCE((c.rules->>'first_time_only')::BOOLEAN, false) as first_time_only,
		NULL as categories,
		NULL as products,
		c.created_at,
		c.updated_at
	FROM coupons c
	WHERE c.rules IS NOT NULL
	ON CONFLICT (id) DO NOTHING`

	if err := dbConn.Exec(migrateDataSQL).Error; err != nil {
		log.Fatalf("Failed to migrate coupon data: %v", err)
	}

	// Step 4: Add constraints to coupons table
	fmt.Println("Step 4: Adding constraints to coupons table...")
	constraints := []string{
		"ALTER TABLE coupons ADD CONSTRAINT chk_coupons_count CHECK (count >= 0)",
		"ALTER TABLE coupons ADD CONSTRAINT chk_coupons_used_count CHECK (used_count >= 0 AND used_count <= count)",
	}

	for _, constraintSQL := range constraints {
		if err := dbConn.Exec(constraintSQL).Error; err != nil {
			log.Printf("Warning: Failed to add constraint: %v", err)
		}
	}

	fmt.Println("✅ Coupon normalization migration completed successfully!")

	// Verify the migration
	fmt.Println("\nVerifying migration...")

	// Check if coupon_rules table exists
	var tableExists bool
	err = dbConn.Raw("SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'coupon_rules')").Scan(&tableExists).Error
	if err != nil {
		log.Printf("Error checking table existence: %v", err)
	} else {
		fmt.Printf("Coupon_rules table exists: %t\n", tableExists)
	}

	// Check how many rules were created
	var ruleCount int64
	err = dbConn.Raw("SELECT COUNT(*) FROM coupon_rules").Scan(&ruleCount).Error
	if err != nil {
		log.Printf("Error counting rules: %v", err)
	} else {
		fmt.Printf("Total coupon rules created: %d\n", ruleCount)
	}

	// Show sample rules
	fmt.Println("\nSample coupon rules:")
	var rules []struct {
		ID            string  `json:"id"`
		CouponID      string  `json:"coupon_id"`
		RuleType      string  `json:"rule_type"`
		DiscountValue float64 `json:"discount_value"`
		MinAmount     float64 `json:"min_amount"`
	}

	err = dbConn.Raw("SELECT id, coupon_id, rule_type, discount_value, min_amount FROM coupon_rules LIMIT 5").Scan(&rules).Error
	if err != nil {
		log.Printf("Error fetching sample rules: %v", err)
	} else {
		for i, rule := range rules {
			fmt.Printf("%d. ID: %s, CouponID: %s, Type: %s, Value: %.2f, MinAmount: %.2f\n",
				i+1, rule.ID, rule.CouponID, rule.RuleType, rule.DiscountValue, rule.MinAmount)
		}
	}
}
