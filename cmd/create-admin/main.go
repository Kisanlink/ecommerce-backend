package main

import (
	"context"
	"log"
	"os"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

func main() {
	// Load environment variables
	config.LoadEnv()

	// Initialize zap logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Get Postgres config from env (same as main server)
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

	// Connect to database using the same method as main server
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
	database, err := pgManager.GetDB(ctx, false)
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	// Get email from command line argument or prompt
	email := "admin@amrti.com" // Default admin email
	if len(os.Args) > 1 {
		email = os.Args[1]
	}

	log.Printf("Setting up admin user for email: %s", email)

	// Check if user exists
	var existingUser models.User
	result := database.Where("email = ?", email).First(&existingUser)

	if result.Error == nil {
		// User exists, update role to admin
		log.Printf("User found: %s (ID: %s, Current Role: %s)", existingUser.Name, existingUser.ID, existingUser.Role)

		if strings.EqualFold(strings.TrimSpace(existingUser.Role), "admin") {
			log.Printf("✅ User %s is already an admin!", email)
			return
		}

		// Update role to admin
		existingUser.Role = "admin"
		if err := database.Save(&existingUser).Error; err != nil {
			log.Fatalf("❌ Failed to update user role: %v", err)
		}

		log.Printf("✅ Successfully updated user %s to admin role!", email)
	} else {
		// User doesn't exist, create new admin user
		log.Printf("User not found, creating new admin user...")

		// Create new admin user
		adminUser := &models.User{
			Email:       &email,
			Name:        "Admin User",
			Password:    "admin123", // You should change this
			FirebaseUID: nil,        // Will be set when user logs in with Firebase
			Role:        "admin",
			IsVerified:  true,
			IsActive:    true,
		}

		if err := database.Create(adminUser).Error; err != nil {
			log.Fatalf("❌ Failed to create admin user: %v", err)
		}

		log.Printf("✅ Successfully created admin user: %s (ID: %s)", email, adminUser.ID)
	}

	// Verify the admin user
	var verifyUser models.User
	if err := database.Where("email = ?", email).First(&verifyUser).Error; err != nil {
		log.Fatalf("❌ Failed to verify admin user: %v", err)
	}

	log.Println("\n" + strings.Repeat("=", 50))
	log.Println("ADMIN USER SETUP COMPLETE")
	log.Println(strings.Repeat("=", 50))
	log.Printf("📧 Email: %v", verifyUser.Email)
	log.Printf("👤 Name: %s", verifyUser.Name)
	log.Printf("🆔 User ID: %s", verifyUser.ID)
	log.Printf("🔑 Role: %s", verifyUser.Role)
	log.Printf("🔥 Firebase UID: %v", verifyUser.FirebaseUID)
	log.Printf("✅ Is Active: %t", verifyUser.IsActive)
	log.Printf("✅ Is Verified: %t", verifyUser.IsVerified)
	log.Println(strings.Repeat("=", 50))

	if verifyUser.FirebaseUID == nil || *verifyUser.FirebaseUID == "" {
		log.Println("⚠️  Note: Firebase UID is empty. User needs to login with Firebase first.")
		log.Println("   After Firebase login, the Firebase UID will be automatically linked.")
	}

	log.Println("🎉 Admin user is ready! You can now use admin endpoints.")
}
