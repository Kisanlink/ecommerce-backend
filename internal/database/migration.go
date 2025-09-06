package database

import (
	"fmt"
	"log"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"gorm.io/gorm"
)

// Migration represents database migration operations
type Migration struct {
	db *gorm.DB
}

// NewMigration creates a new migration instance
func NewMigration(db *gorm.DB) *Migration {
	return &Migration{db: db}
}

// RunMigrations runs all database migrations using GORM auto migrate
func (m *Migration) RunMigrations() error {
	log.Println("Starting database migrations...")

	// List of all models to migrate
	modelsToMigrate := []interface{}{
		&models.User{},
		&models.Profile{},
		&models.Address{},
		&models.Product{},
		&models.Cart{},
		&models.CartItem{},
		&models.Favorite{},
		&models.Order{},
		&models.OrderItem{},
		&models.Recipe{},
	}

	// Run auto migration for each model
	for _, model := range modelsToMigrate {
		if err := m.migrateModel(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
	}

	log.Println("Database migrations completed successfully!")
	return nil
}

// migrateModel runs auto migration for a single model
func (m *Migration) migrateModel(model interface{}) error {
	log.Printf("Migrating model: %T", model)

	if err := m.db.AutoMigrate(model); err != nil {
		return fmt.Errorf("auto migration failed for %T: %w", model, err)
	}

	log.Printf("Successfully migrated model: %T", model)
	return nil
}

// RunMigrationsWithDB runs migrations using the global DB instance
func RunMigrationsWithDB() error {
	if DB == nil {
		return fmt.Errorf("database connection not established")
	}

	migration := NewMigration(DB)
	return migration.RunMigrations()
}

// DropAllTables drops all tables (use with caution - only for development/testing)
func (m *Migration) DropAllTables() error {
	log.Println("Warning: Dropping all tables...")

	// List of all models to drop
	modelsToDrop := []interface{}{
		&models.OrderItem{},
		&models.Order{},
		&models.Favorite{},
		&models.CartItem{},
		&models.Cart{},
		&models.Product{},
		&models.Address{},
		&models.Profile{},
		&models.User{},
		&models.Recipe{},
	}

	// Drop tables in reverse order to handle foreign key constraints
	for _, model := range modelsToDrop {
		if err := m.db.Migrator().DropTable(model); err != nil {
			return fmt.Errorf("failed to drop table for %T: %w", model, err)
		}
		log.Printf("Dropped table for model: %T", model)
	}

	log.Println("All tables dropped successfully!")
	return nil
}

// DropAllTablesWithDB drops all tables using the global DB instance
func DropAllTablesWithDB() error {
	if DB == nil {
		return fmt.Errorf("database connection not established")
	}

	migration := NewMigration(DB)
	return migration.DropAllTables()
}

// GetMigrationStatus returns the current migration status
func (m *Migration) GetMigrationStatus() (map[string]bool, error) {
	status := make(map[string]bool)

	// Check if each table exists
	modelsToCheck := []interface{}{
		&models.User{},
		&models.Profile{},
		&models.Address{},
		&models.Product{},
		&models.Cart{},
		&models.CartItem{},
		&models.Favorite{},
		&models.Order{},
		&models.OrderItem{},
		&models.Recipe{},
	}

	for _, model := range modelsToCheck {
		tableName := m.db.NamingStrategy.TableName(m.db.Statement.Table)
		exists := m.db.Migrator().HasTable(model)
		status[tableName] = exists
	}

	return status, nil
}

// GetMigrationStatusWithDB gets migration status using the global DB instance
func GetMigrationStatusWithDB() (map[string]bool, error) {
	if DB == nil {
		return nil, fmt.Errorf("database connection not established")
	}

	migration := NewMigration(DB)
	return migration.GetMigrationStatus()
}
