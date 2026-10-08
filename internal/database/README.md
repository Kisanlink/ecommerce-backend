# Database Migration System

This package provides a comprehensive database migration system using GORM's auto-migrate functionality for PostgreSQL.

## Features

- **Auto Migration**: Automatically creates and updates database tables based on Go struct definitions
- **Model Support**: Supports all ecommerce models including User, Product, Cart, Order, etc.
- **Migration Status**: Check which tables exist and which are missing
- **Safe Operations**: Drop tables with confirmation prompts
- **Command Line Tool**: Standalone migration tool for development and deployment

## Models Supported

The migration system automatically handles the following models:

- `User` - User accounts and authentication
- `Profile` - User profile information
- `Address` - User addresses (home, work, shipping, billing)
- `Product` - Ecommerce products
- `Cart` - Shopping cart and cart items
- `Favorite` - User favorite products
- `Order` - Customer orders and order items
- `Recipe` - Recipe information (also supports DynamoDB)

## Usage

### 1. Automatic Migration on Server Start

```go
import "github.com/Kisanlink/ecommerce-backend/internal/database"

// Connect and run migrations automatically
if err := database.ConnectAndMigrate(); err != nil {
    log.Fatalf("Database setup failed: %v", err)
}
```

### 2. Manual Migration Control

```go
import "github.com/Kisanlink/ecommerce-backend/internal/database"

// Connect first
if err := database.ConnectPostgres(); err != nil {
    log.Fatalf("Connection failed: %v", err)
}

// Run migrations when ready
if err := database.RunMigrationsWithDB(); err != nil {
    log.Fatalf("Migration failed: %v", err)
}
```

### 3. Check Migration Status

```go
status, err := database.GetMigrationStatusWithDB()
if err != nil {
    log.Fatalf("Failed to get status: %v", err)
}

for tableName, exists := range status {
    if exists {
        fmt.Printf("✅ %s exists\n", tableName)
    } else {
        fmt.Printf("❌ %s missing\n", tableName)
    }
}
```

### 4. Drop All Tables (Development Only)

```go
// ⚠️ WARNING: This will delete all data!
if err := database.DropAllTablesWithDB(); err != nil {
    log.Fatalf("Failed to drop tables: %v", err)
}
```

## Command Line Tool

A standalone migration tool is available at `cmd/migrate/main.go`:

### Run Migrations
```bash
go run cmd/migrate/main.go
```

### Check Status
```bash
go run cmd/migrate/main.go -action status
```

### Drop All Tables (with confirmation)
```bash
go run cmd/migrate/main.go -action drop
```

### Show Help
```bash
go run cmd/migrate/main.go -help
```

## Environment Variables

Required environment variable:
- `POSTGRES_DSN` - PostgreSQL connection string

Example:
```bash
export POSTGRES_DSN="postgresql://neondb_owner:password@ep-your-neon-host.aws.neon.tech/neondb?sslmode=require&channel_binding=require"
```

## Migration Order

The system automatically handles migration order to respect foreign key constraints:

1. **Independent Tables**: User, Product, Recipe
2. **Dependent Tables**: Profile, Address, Cart, Favorite
3. **Relationship Tables**: CartItem, Order, OrderItem

## GORM Features Used

- **Auto Migrate**: Automatically creates tables and adds missing columns
- **Embedded Structs**: Properly handles embedded BaseModel
- **Foreign Keys**: Automatically creates foreign key relationships
- **Indexes**: Creates indexes based on GORM tags
- **Constraints**: Applies NOT NULL, UNIQUE, and DEFAULT constraints

## Best Practices

1. **Development**: Use `ConnectAndMigrate()` for automatic setup
2. **Production**: Run migrations manually before starting the server
3. **Testing**: Use the command line tool to reset database state
4. **Backup**: Always backup data before running destructive operations
5. **Version Control**: Keep model changes in version control for team coordination

## Troubleshooting

### Common Issues

1. **Connection Failed**: Check `POSTGRES_DSN` environment variable
2. **Permission Denied**: Ensure database user has CREATE TABLE privileges
3. **Migration Errors**: Check model struct definitions for syntax errors
4. **Foreign Key Issues**: Verify that referenced tables exist

### Debug Mode

Enable detailed logging by setting the log level:
```go
log.SetFlags(log.LstdFlags | log.Lshortfile)
```

## Integration with Existing Code

This migration system can work alongside the existing database plugin system. You can:

1. Use this for development and testing
2. Use the existing system for production
3. Run both systems in parallel for validation

## Example Integration

```go
// In your main.go or server initialization
func initDatabase() error {
    // Option 1: Use new migration system
    if os.Getenv("USE_NEW_MIGRATIONS") == "true" {
        return database.ConnectAndMigrate()
    }
    
    // Option 2: Use existing system
    return existingDatabaseSetup()
}
```
