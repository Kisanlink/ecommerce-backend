package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/server"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

func main() {
	config.LoadEnv()
	fmt.Println("Starting amrti-backend server...")

	// Initialize Firebase
	if err := config.InitFirebase(); err != nil {
		log.Printf("Warning: Firebase initialization failed: %v", err)
		log.Println("Firebase authentication will not be available")
	}

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
		log.Fatalf("Failed to connect to Postgres via plugin: %v", err)
	}

	// Auto-migrate all models so columns/rows are created automatically
	err := pgManager.AutoMigrateModels(ctx,
		&models.User{},
		&models.Recipe{},
		&models.Product{},
		&models.Order{},
		&models.OrderItem{},
		&models.Cart{},
		&models.CartItem{},
		&models.Address{},
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate models: %v", err)
	}

	// Print all table names to verify connection
	dbConn, err := pgManager.GetDB(ctx, false)
	if err != nil {
		log.Fatalf("Failed to get DB: %v", err)
	}

	var tables []string
	dbConn.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables)
	fmt.Println("Postgres tables:", tables)

	// Create repository manager with database manager
	repoManager := repositories.NewRepositoryManager(pgManager)

	// Start the main server
	srv, err := server.NewServer(repoManager)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	if err := srv.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
