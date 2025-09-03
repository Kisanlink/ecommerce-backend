package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/Kisanlink/ecommerce-backend/internal/database"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Parse command line flags
	var (
		action = flag.String("action", "migrate", "Action to perform: migrate, drop, status")
		help   = flag.Bool("help", false, "Show help message")
	)
	flag.Parse()

	if *help {
		showHelp()
		return
	}

	// Connect to database
	if err := database.ConnectPostgres(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Perform the requested action
	switch *action {
	case "migrate":
		if err := database.RunMigrationsWithDB(); err != nil {
			log.Fatalf("Migration failed: %v", err)
		}
		log.Println("✅ Database migrations completed successfully!")

	case "drop":
		fmt.Print("⚠️  WARNING: This will drop ALL tables. Are you sure? (yes/no): ")
		var confirmation string
		fmt.Scanln(&confirmation)
		
		if confirmation == "yes" {
			if err := database.DropAllTablesWithDB(); err != nil {
				log.Fatalf("Failed to drop tables: %v", err)
			}
			log.Println("✅ All tables dropped successfully!")
		} else {
			log.Println("❌ Operation cancelled")
		}

	case "status":
		status, err := database.GetMigrationStatusWithDB()
		if err != nil {
			log.Fatalf("Failed to get migration status: %v", err)
		}
		
		fmt.Println("\n📊 Database Migration Status:")
		fmt.Println("================================")
		for tableName, exists := range status {
			statusIcon := "✅"
			statusText := "EXISTS"
			if !exists {
				statusIcon = "❌"
				statusText = "MISSING"
			}
			fmt.Printf("%s %s: %s\n", statusIcon, tableName, statusText)
		}

	default:
		log.Fatalf("Unknown action: %s. Use -help to see available options", *action)
	}
}

func showHelp() {
	fmt.Println(`
🔄 Database Migration Tool

Usage:
  go run cmd/migrate/main.go [flags]

Flags:
  -action string    Action to perform (default "migrate")
  -help            Show this help message

Available Actions:
  migrate          Run database migrations (create/update tables)
  drop             Drop all tables (⚠️  DESTRUCTIVE - use with caution)
  status           Show current migration status

Examples:
  go run cmd/migrate/main.go                    # Run migrations
  go run cmd/migrate/main.go -action status     # Check migration status
  go run cmd/migrate/main.go -action drop       # Drop all tables

Environment Variables:
  POSTGRES_DSN     PostgreSQL connection string
`)
}
