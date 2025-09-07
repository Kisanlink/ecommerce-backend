package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Kisanlink/ecommerce-backend/internal/database"
)

func main() {
	// Example 1: Basic connection and migration
	fmt.Println("=== Example 1: Basic Migration ===")

	// Set environment variable (in real usage, this would be in .env file)
	os.Setenv("POSTGRES_DSN", "host=localhost user=postgres password=password dbname=ecommerce port=5432 sslmode=disable")

	// Connect and run migrations
	if err := database.ConnectAndMigrate(); err != nil {
		log.Printf("Migration failed: %v", err)
	} else {
		fmt.Println("✅ Database setup completed successfully!")
	}

	// Example 2: Check migration status
	fmt.Println("\n=== Example 2: Check Migration Status ===")

	status, err := database.GetMigrationStatusWithDB()
	if err != nil {
		log.Printf("Failed to get status: %v", err)
	} else {
		fmt.Println("📊 Migration Status:")
		for tableName, exists := range status {
			icon := "✅"
			statusText := "EXISTS"
			if !exists {
				icon = "❌"
				statusText = "MISSING"
			}
			fmt.Printf("  %s %s: %s\n", icon, tableName, statusText)
		}
	}

	// Example 3: Manual migration control
	fmt.Println("\n=== Example 3: Manual Migration Control ===")

	// You can also run migrations manually
	if err := database.RunMigrationsWithDB(); err != nil {
		log.Printf("Manual migration failed: %v", err)
	} else {
		fmt.Println("✅ Manual migration completed!")
	}

	// Example 4: Get database instance
	fmt.Println("\n=== Example 4: Get Database Instance ===")

	db := database.GetDB()
	if db != nil {
		fmt.Println("✅ Database instance retrieved successfully!")

		// You can use the db instance for custom operations
		var tableCount int64
		if err := db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tableCount).Error; err != nil {
			log.Printf("Failed to count tables: %v", err)
		} else {
			fmt.Printf("📊 Total tables in database: %d\n", tableCount)
		}
	} else {
		fmt.Println("❌ Database instance is nil")
	}

	fmt.Println("\n=== Migration Examples Completed ===")
	fmt.Println("💡 Check the README.md for more detailed usage instructions")
}
