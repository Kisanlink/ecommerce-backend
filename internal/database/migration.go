package database

import (
	"context"
	"fmt"
	"log"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"gorm.io/gorm"
)

// Migration represents database migration operations
type Migration struct {
	db        *gorm.DB
	dbManager *db.DatabaseManager
}

// NewMigration creates a new migration instance
func NewMigration(gormDB *gorm.DB) *Migration {
	dbManager := db.NewDatabaseManager()
	return &Migration{
		db:        gormDB,
		dbManager: dbManager,
	}
}

// RunMigrations runs all database migrations using GORM auto migrate
func (m *Migration) RunMigrations() error {
	log.Println("Starting database migrations...")

	// PostgreSQL migrations using GORM
	if err := m.runPostgresMigrations(); err != nil {
		return fmt.Errorf("PostgreSQL migrations failed: %w", err)
	}

	// DynamoDB table creation
	if err := m.runDynamoDBMigrations(); err != nil {
		log.Printf("Warning: DynamoDB migrations failed: %v", err)
		// Don't fail the entire migration if DynamoDB is not available
	}

	log.Println("Database migrations completed successfully!")
	return nil
}

// runPostgresMigrations runs PostgreSQL migrations using GORM
func (m *Migration) runPostgresMigrations() error {
	// List of all models to migrate in dependency order
	modelsToMigrate := []interface{}{
		&models.User{},    // ✅ Authentication data only
		&models.Profile{}, // ✅ Personal info only (no duplicate fields)
		&models.Address{},
		&models.Product{},
		&models.Image{},         // Add Image model for auto-migration
		&models.Inventory{},     // Add Inventory model for auto-migration
		&models.OrderCustomer{}, // NEW: Order customer normalization
		&models.OrderAddress{},  // NEW: Order address normalization
		&models.Cart{},
		&models.CartItem{},
		&models.Favorite{},
		&models.Order{},
		&models.OrderItem{},
		&models.Recipe{},
		&models.Coupon{},     // Add Coupon model for auto-migration
		&models.CouponRule{}, // Add CouponRule model for auto-migration
	}

	// Run auto migration for each model
	for _, model := range modelsToMigrate {
		if err := m.migrateModel(model); err != nil {
			return fmt.Errorf("failed to migrate model %T: %w", model, err)
		}
	}

	// Run custom migrations for schema changes
	if err := m.runCustomMigrations(); err != nil {
		return fmt.Errorf("custom migrations failed: %w", err)
	}

	return nil
}

// runCustomMigrations runs custom SQL migrations for schema changes
func (m *Migration) runCustomMigrations() error {
	log.Println("Running custom migrations for schema changes...")

	// Check if users and profiles normalization has been applied
	if err := m.checkAndApplyUsersProfilesNormalization(); err != nil {
		return fmt.Errorf("users and profiles normalization failed: %w", err)
	}

	return nil
}

// checkAndApplyUsersProfilesNormalization checks and applies users/profiles normalization
func (m *Migration) checkAndApplyUsersProfilesNormalization() error {
	// Check if profiles table still has email column (denormalized state)
	var emailColumnExists bool
	err := m.db.Raw(`
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns 
			WHERE table_name = 'profiles' AND column_name = 'email'
		)
	`).Scan(&emailColumnExists).Error

	if err != nil {
		return fmt.Errorf("failed to check profiles table structure: %w", err)
	}

	if emailColumnExists {
		log.Println("⚠️  Profiles table still has duplicate fields (email, phone)")
		log.Println("📋 To apply normalization, run: go run cmd/migrate-users-profiles/main.go")
		log.Println("📋 To rollback later, run: go run cmd/migrate-users-profiles/main.go rollback")
	} else {
		log.Println("✅ Users and Profiles tables are already normalized")
	}

	return nil
}

// runDynamoDBMigrations creates DynamoDB tables
func (m *Migration) runDynamoDBMigrations() error {
	if m.dbManager == nil {
		return fmt.Errorf("database manager not available")
	}

	// Connect to DynamoDB
	ctx := context.Background()
	if err := m.dbManager.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to DynamoDB: %w", err)
	}
	defer m.dbManager.Close()

	dynamoManager := m.dbManager.GetDynamoManager()
	if dynamoManager == nil {
		return fmt.Errorf("DynamoDB manager not available")
	}

	return dynamoManager.WithClient(ctx, func(client *dynamodb.Client) error {
		return m.createReviewsTable(ctx, client)
	})
}

// createReviewsTable creates the product_reviews table in DynamoDB
func (m *Migration) createReviewsTable(ctx context.Context, client *dynamodb.Client) error {
	tableName := "product_reviews"

	// Check if table already exists
	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err == nil {
		log.Printf("DynamoDB table %s already exists, skipping creation", tableName)
		return nil
	}

	log.Printf("Creating DynamoDB table: %s", tableName)

	// Create table with composite primary key (product_id, user_id)
	input := &dynamodb.CreateTableInput{
		TableName: aws.String(tableName),
		KeySchema: []types.KeySchemaElement{
			{
				AttributeName: aws.String("product_id"),
				KeyType:       types.KeyTypeHash, // Partition key
			},
			{
				AttributeName: aws.String("user_id"),
				KeyType:       types.KeyTypeRange, // Sort key
			},
		},
		AttributeDefinitions: []types.AttributeDefinition{
			{
				AttributeName: aws.String("product_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("user_id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
			{
				AttributeName: aws.String("id"),
				AttributeType: types.ScalarAttributeTypeS,
			},
		},
		BillingMode: types.BillingModePayPerRequest,
		GlobalSecondaryIndexes: []types.GlobalSecondaryIndex{
			{
				IndexName: aws.String("ReviewIDIndex"),
				KeySchema: []types.KeySchemaElement{
					{
						AttributeName: aws.String("id"),
						KeyType:       types.KeyTypeHash,
					},
				},
				Projection: &types.Projection{
					ProjectionType: types.ProjectionTypeKeysOnly,
				},
			},
		},
		Tags: []types.Tag{
			{
				Key:   aws.String("Environment"),
				Value: aws.String("development"),
			},
			{
				Key:   aws.String("Application"),
				Value: aws.String("ecommerce-backend"),
			},
			{
				Key:   aws.String("Purpose"),
				Value: aws.String("product-reviews"),
			},
		},
	}

	_, err = client.CreateTable(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create DynamoDB table %s: %w", tableName, err)
	}

	log.Printf("DynamoDB table %s created successfully", tableName)
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

	// List of all models to drop in reverse dependency order
	modelsToDrop := []interface{}{
		&models.OrderItem{},
		&models.Order{},
		&models.OrderAddress{},  // NEW: Order address normalization
		&models.OrderCustomer{}, // NEW: Order customer normalization
		&models.Favorite{},
		&models.CartItem{},
		&models.Cart{},
		&models.Image{},     // Add Image model to drop list
		&models.Inventory{}, // Add Inventory model to drop list
		&models.Product{},
		&models.Address{},
		&models.Profile{}, // ✅ Normalized profile table
		&models.User{},    // ✅ Normalized user table
		&models.Recipe{},
		&models.CouponRule{}, // Add CouponRule model to drop list
		&models.Coupon{},     // Add Coupon model to drop list
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
		&models.User{},    // ✅ Normalized user table
		&models.Profile{}, // ✅ Normalized profile table
		&models.Address{},
		&models.Product{},
		&models.Image{},         // Add Image model to status check
		&models.Inventory{},     // Add Inventory model to status check
		&models.OrderCustomer{}, // NEW: Order customer normalization
		&models.OrderAddress{},  // NEW: Order address normalization
		&models.Cart{},
		&models.CartItem{},
		&models.Favorite{},
		&models.Order{},
		&models.OrderItem{},
		&models.Recipe{},
		&models.Coupon{}, // Add Coupon model to status check
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
