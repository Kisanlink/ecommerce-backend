package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/Kisanlink/ecommerce-backend/docs"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/server"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// @title Ecommerce Backend API
// @version 1.0.0
// @description A comprehensive ecommerce backend API with authentication, product management, cart, orders, and payment integration.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.kisanlink.com/support
// @contact.email support@kisanlink.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey FirebaseAuth
// @in header
// @name Authorization
// @description Firebase ID token for authentication

// executeSQLMigrations executes SQL migration files
func executeSQLMigrations(db *gorm.DB) error {
	migrationsDir := "migrations"

	// List of migration files to execute in order
	migrationFiles := []string{
		"create_coupons_table.sql",
		"create_images_table.sql",
		"add_multiple_images_columns.sql",
		"add_delhivery_columns.sql",
		"add_discount_columns_to_orders.sql",
	}

	for _, filename := range migrationFiles {
		filePath := filepath.Join(migrationsDir, filename)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("Migration file %s not found, skipping", filePath)
			continue
		}

		// Read SQL file
		sqlContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", filePath, err)
		}

		// Split SQL content by semicolon and execute each statement
		statements := strings.Split(string(sqlContent), ";")
		for _, statement := range statements {
			statement = strings.TrimSpace(statement)
			if statement == "" || strings.HasPrefix(statement, "--") {
				continue
			}

			if err := db.Exec(statement).Error; err != nil {
				// Log warning but don't fail - some statements might already exist
				log.Printf("Warning: Failed to execute SQL statement in %s: %v", filename, err)
			}
		}

		log.Printf("Executed migration file: %s", filename)
	}

	return nil
}

func main() {
	config.LoadEnv()
	fmt.Println("Starting amrti-backend server...")

	// Initialize Swagger documentation
	docs.SwaggerInfo.Title = "Ecommerce Backend API"
	docs.SwaggerInfo.Description = "A comprehensive ecommerce backend API with authentication, product management, cart, orders, and payment integration."
	docs.SwaggerInfo.Version = "1.0.0"
	docs.SwaggerInfo.Host = "localhost:8080"
	docs.SwaggerInfo.BasePath = "/api/v1"
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	// Initialize Firebase
	if err := config.InitFirebase(); err != nil {
		log.Printf("Warning: Firebase initialization failed: %v", err)
		log.Println("Firebase authentication will not be available")
	}

	// Initialize zap logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Listen for interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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

	ctx = context.Background()
	pgManager := db.NewPostgresManager(pgConfig, logger)
	if err := pgManager.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect to Postgres via plugin: %v", err)
	}
	defer func() {
		logger.Info("Closing database connections...")
		if err := pgManager.Close(); err != nil {
			logger.Error("Failed to close Postgres connection", zap.Error(err))
		} else {
			logger.Info("Database connections closed successfully")
		}
	}()

	// Initialize S3Manager
	var s3Manager *db.S3Manager
	if pgConfig.S3Bucket != "" {
		s3Manager = db.NewS3Manager(pgConfig, logger)
		if err := s3Manager.Connect(ctx); err != nil {
			log.Printf("Warning: Failed to connect to S3: %v", err)
			log.Println("S3 image upload will not be available")
		} else {
			log.Println("Successfully connected to S3")
			defer func() {
				logger.Info("Closing S3 connections...")
				if err := s3Manager.Close(); err != nil {
					logger.Error("Failed to close S3 connection", zap.Error(err))
				} else {
					logger.Info("S3 connections closed successfully")
				}
			}()
		}
	}

	// Ensure users.id is VARCHAR to support string IDs like "USER..."
	// This guards against older schemas where id might be BIGINT.
	if dbConn, err := pgManager.GetDB(ctx, false); err == nil {
		var dataType string
		// Check current data type of users.id
		type row struct{ DataType string }
		var r row
		if qerr := dbConn.Raw(
			`SELECT data_type FROM information_schema.columns 
			 WHERE table_schema = 'public' AND table_name = 'users' AND column_name = 'id'`,
		).Scan(&r).Error; qerr == nil {
			dataType = r.DataType
		}
		if dataType != "character varying" && dataType != "text" && dataType != "" {
			// Drop default (e.g., sequence) if any, then alter type to VARCHAR(255)
			_ = dbConn.Exec(`ALTER TABLE "users" ALTER COLUMN "id" DROP DEFAULT;`).Error
			_ = dbConn.Exec(`ALTER TABLE "users" ALTER COLUMN "id" TYPE VARCHAR(255) USING "id"::text;`).Error
		}
	}

	// Auto-migrate all models so columns/rows are created automatically
	// Note: Recipe model is excluded as it's stored in DynamoDB
	err := pgManager.AutoMigrateModels(ctx,
		&models.User{},
		// &models.Recipe{}, // Stored in DynamoDB, not PostgreSQL
		&models.Product{},
		&models.Image{},     // Add Image model for auto-migration
		&models.Inventory{}, // Add Inventory model for auto-migration
		&models.Order{},
		&models.OrderItem{},
		&models.Cart{},
		&models.CartItem{},
		&models.Address{},
		&models.Profile{},
		&models.Favorite{},
		&models.Coupon{},     // Add Coupon model for auto-migration
		&models.CouponRule{}, // Add CouponRule model for auto-migration
	)
	if err != nil {
		log.Fatalf("Failed to auto-migrate models: %v", err)
	}

	// Print all table names to verify connection
	dbConn, err := pgManager.GetDB(ctx, false)
	if err != nil {
		log.Fatalf("Failed to get DB: %v", err)
	}

	// Execute SQL migrations for initial data
	if err := executeSQLMigrations(dbConn); err != nil {
		log.Printf("Warning: Failed to execute SQL migrations: %v", err)
	}

	// Ensure carts table has coupon fields with sane defaults and no NULLs
	// 1) Add columns if missing
	_ = dbConn.Exec(`ALTER TABLE "carts" ADD COLUMN IF NOT EXISTS "coupon_code" VARCHAR(50);`).Error
	_ = dbConn.Exec(`ALTER TABLE "carts" ADD COLUMN IF NOT EXISTS "discount_amount" DOUBLE PRECISION;`).Error
	// 2) Backfill NULLs
	_ = dbConn.Exec(`UPDATE "carts" SET "coupon_code" = '' WHERE "coupon_code" IS NULL;`).Error
	_ = dbConn.Exec(`UPDATE "carts" SET "discount_amount" = 0 WHERE "discount_amount" IS NULL;`).Error
	// 3) Apply defaults and NOT NULL constraints
	_ = dbConn.Exec(`ALTER TABLE "carts" ALTER COLUMN "coupon_code" SET DEFAULT '';`).Error
	_ = dbConn.Exec(`ALTER TABLE "carts" ALTER COLUMN "coupon_code" SET NOT NULL;`).Error
	_ = dbConn.Exec(`ALTER TABLE "carts" ALTER COLUMN "discount_amount" SET DEFAULT 0;`).Error
	_ = dbConn.Exec(`ALTER TABLE "carts" ALTER COLUMN "discount_amount" SET NOT NULL;`).Error

	var tables []string
	dbConn.Raw("SELECT tablename FROM pg_tables WHERE schemaname = 'public'").Scan(&tables)
	fmt.Println("Postgres tables:", tables)

	// Create repository manager with database manager
	repoManager := repositories.NewRepositoryManager(pgManager, logger)

	// Create service manager with repository manager and S3Manager (following the development rules)
	serviceManager := services.NewServiceManager(repoManager, s3Manager)

	// Start the main server with both repository and service managers
	srv, err := server.NewServer(repoManager, serviceManager)
	if err != nil {
		log.Fatalf("Failed to initialize server: %v", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    ":" + port,
		Handler: srv,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting server", zap.String("port", port))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Wait for interrupt signal
	<-sigChan
	logger.Info("Shutdown signal received, initiating graceful shutdown...")

	// Create shutdown context with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP server gracefully
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server forced to shutdown", zap.Error(err))
	} else {
		logger.Info("Server exited gracefully")
	}

	logger.Info("Application shutdown complete")
}
