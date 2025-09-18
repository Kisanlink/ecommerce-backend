package main

import (
	"context"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	_ "github.com/lib/pq"
)

func main() {
	log.Println("Starting Users and Profiles normalization migration...")

	// Get database connection string
	dsn := getDatabaseDSN()

	// Connect to database
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	log.Println("Database connection established successfully")

	// Check if migration should be run
	if len(os.Args) > 1 && os.Args[1] == "rollback" {
		runRollbackMigration(db)
	} else {
		runNormalizationMigration(db)
	}
}

func runNormalizationMigration(db *sql.DB) {
	log.Println("Running Users and Profiles normalization migration...")

	// Read migration file
	migrationFile := filepath.Join("migrations", "normalize_users_profiles_tables.sql")
	sqlContent, err := ioutil.ReadFile(migrationFile)
	if err != nil {
		log.Fatalf("Failed to read migration file %s: %v", migrationFile, err)
	}

	// Execute migration
	ctx := context.Background()
	_, err = db.ExecContext(ctx, string(sqlContent))
	if err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	log.Println("Users and Profiles normalization migration completed successfully!")
	log.Println("✅ Removed duplicate email and phone fields from profiles table")
	log.Println("✅ Added proper constraints and indexes")
	log.Println("✅ Updated table comments and documentation")
}

func runRollbackMigration(db *sql.DB) {
	log.Println("Running Users and Profiles rollback migration...")

	// Read rollback file
	rollbackFile := filepath.Join("migrations", "rollback_normalize_users_profiles_tables.sql")
	sqlContent, err := ioutil.ReadFile(rollbackFile)
	if err != nil {
		log.Fatalf("Failed to read rollback file %s: %v", rollbackFile, err)
	}

	// Execute rollback
	ctx := context.Background()
	_, err = db.ExecContext(ctx, string(sqlContent))
	if err != nil {
		log.Fatalf("Rollback failed: %v", err)
	}

	log.Println("Users and Profiles rollback migration completed successfully!")
	log.Println("⚠️  Restored duplicate email and phone fields to profiles table")
	log.Println("⚠️  Denormalized state restored")
}

func getDatabaseDSN() string {
	// Try environment variables first
	if dsn := os.Getenv("POSTGRES_DSN"); dsn != "" {
		log.Printf("Using POSTGRES_DSN from environment")
		return dsn
	}

	// Try individual environment variables
	host := getEnvOrDefault("DB_POSTGRES_HOST", "localhost")
	port := getEnvOrDefault("DB_POSTGRES_PORT", "5432")
	user := getEnvOrDefault("DB_POSTGRES_USER", "postgres")
	password := getEnvOrDefault("DB_POSTGRES_PASSWORD", "12345678")
	dbname := getEnvOrDefault("DB_POSTGRES_DBNAME", "DemoDB")
	sslmode := getEnvOrDefault("DB_POSTGRES_SSLMODE", "disable")

	// Construct DSN
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Kolkata",
		host, user, password, dbname, port, sslmode)

	log.Printf("Using database connection: host=%s, port=%s, user=%s, dbname=%s", host, port, user, dbname)
	return dsn
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
