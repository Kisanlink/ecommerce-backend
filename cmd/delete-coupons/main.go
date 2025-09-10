package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

func main() {
	config.LoadEnv()
	fmt.Println("Starting coupon deletion script...")

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

	// Get all coupons first
	coupons, err := repoManager.CouponRepository.GetAll(ctx, 100, 0)
	if err != nil {
		log.Fatalf("Failed to get coupons: %v", err)
	}

	fmt.Printf("Found %d coupons:\n", len(coupons))
	for i, coupon := range coupons {
		fmt.Printf("%d. Code: %s, Description: %s, Active: %t\n",
			i+1, coupon.Code, coupon.Description, coupon.IsActive)
	}

	if len(coupons) == 0 {
		fmt.Println("No coupons found to delete.")
		return
	}

	// Ask for confirmation
	fmt.Print("\nDo you want to delete ALL coupons? (yes/no): ")
	var response string
	fmt.Scanln(&response)

	if response != "yes" {
		fmt.Println("Operation cancelled.")
		return
	}

	// Delete all coupons
	deletedCount := 0
	for _, coupon := range coupons {
		if err := repoManager.CouponRepository.Delete(ctx, coupon.ID); err != nil {
			logger.Error("Failed to delete coupon",
				zap.String("code", coupon.Code),
				zap.Error(err))
			fmt.Printf("Failed to delete coupon %s: %v\n", coupon.Code, err)
		} else {
			deletedCount++
			fmt.Printf("Deleted coupon: %s\n", coupon.Code)
		}
	}

	fmt.Printf("\nSuccessfully deleted %d out of %d coupons.\n", deletedCount, len(coupons))
}
