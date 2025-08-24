# KISANLINK-DB Database & Backends Guide for GitHub Copilot

> 📝 **Copilot Context**: This file contains all database backend patterns and configurations for the KISANLINK-DB package.
>


## Essential Imports
```go
import (
    "context"
    "github.com/Kisanlink/kisanlink-db/pkg/db"
    "gorm.io/gorm"
    "github.com/aws/aws-sdk-go-v2/service/dynamodb"
    "github.com/authzed/authzed-go/v1"
)
```
## Quick Database Setup Checklist
- [ ] Choose primary backend (`DB_PRIMARY_BACKEND`)
- [ ] Set connection environment variables
- [ ] Initialize `db.NewDatabaseManager()`
- [ ] Call `dbManager.Connect(ctx)` on startup
- [ ] Check connection status with `IsConnected()`
- [ ] Use appropriate `WithTransaction()` or `WithClient()` patterns
- [ ] Implement proper error handling for each backend
- [ ] Add `defer dbManager.Close()` for cleanup
- [ ] Test connection with health check endpoints
- [ ] Monitor connection pools and performance
## Database Manager Setup Pattern
```go
// Standard database manager initialization - ALWAYS use this pattern
func NewDatabaseService() *DatabaseService {
    dbMgr := db.NewDatabaseManager()
    
    // Connect to all configured backends
    ctx := context.Background()
    if err := dbMgr.Connect(ctx); err != nil {
        log.Fatal("Failed to connect to databases:", err)
    }
    
    return &DatabaseService{
        dbMgr: dbMgr,
    }
}

// ALWAYS implement cleanup
func (s *DatabaseService) Close() error {
    return s.dbMgr.Close()
}
```

## Supported Backend Types
```go
// Available backend types
db.BackendInMemory  // In-memory storage (testing/development)
db.BackendGorm      // PostgreSQL via GORM (primary relational DB)
db.BackendDynamo    // AWS DynamoDB (NoSQL document store)
db.BackendSpiceDB   // SpiceDB (authorization/permissions)
db.BackendS3        // AWS S3 (file/blob storage)
```

## Configuration Patterns

### Environment-Based Setup (Recommended)
```go
// Uses environment variables - most flexible approach
dbManager := db.NewDatabaseManager()

// Required environment variables will be automatically loaded
ctx := context.Background()
err := dbManager.Connect(ctx)
if err != nil {
    log.Fatal("Database connection failed:", err)
}
defer dbManager.Close()
```

### Manual Configuration Setup
```go
// Manual configuration for specific requirements
config := &db.Config{
    // Primary backend selection
    PrimaryBackend: db.BackendGorm,
    
    // PostgreSQL configuration
    PostgresHost:      "localhost",
    PostgresPort:      "5432",
    PostgresUser:      "postgres",
    PostgresPassword:  "your_password",
    PostgresDBName:    "your_database",
    PostgresSSLMode:   "disable",
    PostgresMaxConns:  25,
    PostgresIdleConns: 5,
    PostgresReadReplicas: []string{"replica1:5432", "replica2:5432"},
    
    // DynamoDB configuration
    DynamoRegion: "us-east-1",
    DynamoTable:  "your_table",
    
    // SpiceDB configuration
    SpiceDBEndpoint: "localhost:50051",
    SpiceDBToken:    "your_token",
    
    // S3 configuration
    S3Region: "us-east-1",
    S3Bucket: "your_bucket",
    
    // Logging
    LogLevel: "info",
}

dbManager := db.NewDatabaseManagerWithConfig(config)
```

## PostgreSQL (GORM) Patterns

### Basic Operations
```go
// Get PostgreSQL manager
postgresManager := dbManager.GetPostgresManager()
if postgresManager == nil {
    return errors.New("PostgreSQL not available")
}

// Direct database operations
func (s *YourService) CreateWithGorm(ctx context.Context, model *YourModel) error {
    return postgresManager.WithTransaction(ctx, func(tx *gorm.DB) error {
        return tx.Create(model).Error
    })
}

// Read-only operations (uses read replicas if available)
func (s *YourService) GetWithReadReplica(ctx context.Context, id string) (*YourModel, error) {
    var model YourModel
    err := postgresManager.WithReadOnly(ctx, func(db *gorm.DB) error {
        return db.Where("id = ?", id).First(&model).Error
    })
    return &model, err
}
```

### Transaction Patterns
```go
// Single transaction
func (s *OrderService) CreateOrderWithItems(ctx context.Context, order *Order, items []*OrderItem) error {
    postgresManager := s.dbMgr.GetPostgresManager()
    
    return postgresManager.WithTransaction(ctx, func(tx *gorm.DB) error {
        // Create order
        if err := tx.Create(order).Error; err != nil {
            return err
        }
        
        // Create order items
        for _, item := range items {
            item.OrderID = order.GetID()
            if err := tx.Create(item).Error; err != nil {
                return err
            }
        }
        
        return nil
    })
}

// Complex transaction with rollback handling
func (s *InventoryService) TransferStock(ctx context.Context, fromID, toID string, quantity int) error {
    postgresManager := s.dbMgr.GetPostgresManager()
    
    return postgresManager.WithTransaction(ctx, func(tx *gorm.DB) error {
        // Lock rows for update
        var fromProduct, toProduct Product
        
        if err := tx.Set("gorm:query_option", "FOR UPDATE").
            Where("id = ?", fromID).First(&fromProduct).Error; err != nil {
            return err
        }
        
        if err := tx.Set("gorm:query_option", "FOR UPDATE").
            Where("id = ?", toID).First(&toProduct).Error; err != nil {
            return err
        }
        
        // Check availability
        if fromProduct.Stock < quantity {
            return errors.New("insufficient stock")
        }
        
        // Update stock
        fromProduct.Stock -= quantity
        toProduct.Stock += quantity
        
        if err := tx.Save(&fromProduct).Error; err != nil {
            return err
        }
        
        return tx.Save(&toProduct).Error
    })
}
```

### Advanced GORM Operations
```go
// Raw SQL queries
func (s *ReportService) GetSalesReport(ctx context.Context, startDate, endDate time.Time) ([]SalesReport, error) {
    postgresManager := s.dbMgr.GetPostgresManager()
    var reports []SalesReport
    
    err := postgresManager.WithReadOnly(ctx, func(db *gorm.DB) error {
        return db.Raw(`
            SELECT 
                DATE(created_at) as date,
                COUNT(*) as orders,
                SUM(total_amount) as revenue
            FROM orders 
            WHERE created_at BETWEEN ? AND ?
            GROUP BY DATE(created_at)
            ORDER BY date DESC
        `, startDate, endDate).Scan(&reports).Error
    })
    
    return reports, err
}

// Batch operations
func (s *ProductService) BulkUpdatePrices(ctx context.Context, updates []PriceUpdate) error {
    postgresManager := s.dbMgr.GetPostgresManager()
    
    return postgresManager.WithTransaction(ctx, func(tx *gorm.DB) error {
        for _, update := range updates {
            if err := tx.Model(&Product{}).
                Where("id = ?", update.ProductID).
                Update("price", update.NewPrice).Error; err != nil {
                return err
            }
        }
        return nil
    })
}
```

## DynamoDB Patterns

### Basic Operations
```go
// Get DynamoDB manager
dynamoManager := dbManager.GetDynamoManager()
if dynamoManager == nil {
    return errors.New("DynamoDB not available")
}

// Put item
func (s *DocumentService) SaveDocument(ctx context.Context, doc *Document) error {
    return dynamoManager.WithClient(ctx, func(client *dynamodb.Client) error {
        item, err := attributevalue.MarshalMap(doc)
        if err != nil {
            return err
        }
        
        _, err = client.PutItem(ctx, &dynamodb.PutItemInput{
            TableName: aws.String("documents"),
            Item:      item,
        })
        return err
    })
}

// Get item
func (s *DocumentService) GetDocument(ctx context.Context, id string) (*Document, error) {
    dynamoManager := s.dbMgr.GetDynamoManager()
    var doc Document
    
    err := dynamoManager.WithClient(ctx, func(client *dynamodb.Client) error {
        result, err := client.GetItem(ctx, &dynamodb.GetItemInput{
            TableName: aws.String("documents"),
            Key: map[string]types.AttributeValue{
                "id": &types.AttributeValueMemberS{Value: id},
            },
        })
        if err != nil {
            return err
        }
        
        return attributevalue.UnmarshalMap(result.Item, &doc)
    })
    
    return &doc, err
}

// Query with pagination
func (s *DocumentService) GetUserDocuments(ctx context.Context, userID string, limit int) ([]Document, error) {
    dynamoManager := s.dbMgr.GetDynamoManager()
    var docs []Document
    
    err := dynamoManager.WithClient(ctx, func(client *dynamodb.Client) error {
        result, err := client.Query(ctx, &dynamodb.QueryInput{
            TableName:              aws.String("documents"),
            KeyConditionExpression: aws.String("user_id = :uid"),
            ExpressionAttributeValues: map[string]types.AttributeValue{
                ":uid": &types.AttributeValueMemberS{Value: userID},
            },
            Limit: aws.Int32(int32(limit)),
        })
        if err != nil {
            return err
        }
        
        return attributevalue.UnmarshalListOfMaps(result.Items, &docs)
    })
    
    return docs, err
}
```

## SpiceDB (Authorization) Patterns

### Basic Permission Checks
```go
// Get SpiceDB manager
spiceManager := dbManager.GetSpiceManager()
if spiceManager == nil {
    return errors.New("SpiceDB not available")
}

// Check permission
func (s *AuthService) CheckPermission(ctx context.Context, userID, resource, action string) (bool, error) {
    var allowed bool
    
    err := spiceManager.WithClient(ctx, func(client *authzed.Client) error {
        resp, err := client.CheckPermission(ctx, &v1.CheckPermissionRequest{
            Resource: &v1.ObjectReference{
                ObjectType: "document",
                ObjectId:   resource,
            },
            Permission: action,
            Subject: &v1.SubjectReference{
                Object: &v1.ObjectReference{
                    ObjectType: "user",
                    ObjectId:   userID,
                },
            },
        })
        if err != nil {
            return err
        }
        
        allowed = resp.Permissionship == v1.CheckPermissionResponse_PERMISSIONSHIP_HAS_PERMISSION
        return nil
    })
    
    return allowed, err
}

// Write relationships
func (s *AuthService) GrantAccess(ctx context.Context, userID, resource, relation string) error {
    return spiceManager.WithClient(ctx, func(client *authzed.Client) error {
        _, err := client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
            Updates: []*v1.RelationshipUpdate{
                {
                    Operation: v1.RelationshipUpdate_OPERATION_TOUCH,
                    Relationship: &v1.Relationship{
                        Resource: &v1.ObjectReference{
                            ObjectType: "document",
                            ObjectId:   resource,
                        },
                        Relation: relation,
                        Subject: &v1.SubjectReference{
                            Object: &v1.ObjectReference{
                                ObjectType: "user",
                                ObjectId:   userID,
                            },
                        },
                    },
                },
            },
        })
        return err
    })
}
```

## S3 Storage Patterns

### File Operations
```go
// Get S3 manager
s3Manager := dbManager.GetS3Manager()
if s3Manager == nil {
    return errors.New("S3 not available")
}

// Upload file
func (s *FileService) UploadFile(ctx context.Context, key string, data []byte) error {
    return s3Manager.WithClient(ctx, func(client *s3.Client) error {
        _, err := client.PutObject(ctx, &s3.PutObjectInput{
            Bucket: aws.String("your-bucket"),
            Key:    aws.String(key),
            Body:   bytes.NewReader(data),
        })
        return err
    })
}

// Download file
func (s *FileService) DownloadFile(ctx context.Context, key string) ([]byte, error) {
    s3Manager := s.dbMgr.GetS3Manager()
    var data []byte
    
    err := s3Manager.WithClient(ctx, func(client *s3.Client) error {
        result, err := client.GetObject(ctx, &s3.GetObjectInput{
            Bucket: aws.String("your-bucket"),
            Key:    aws.String(key),
        })
        if err != nil {
            return err
        }
        defer result.Body.Close()
        
        data, err = io.ReadAll(result.Body)
        return err
    })
    
    return data, err
}
```

## Environment Variables Reference
```bash
# Primary backend selection
DB_PRIMARY_BACKEND=gorm

# PostgreSQL settings
DB_POSTGRES_HOST=localhost
DB_POSTGRES_PORT=5432
DB_POSTGRES_USER=postgres
DB_POSTGRES_PASSWORD=your_password
DB_POSTGRES_DBNAME=your_database
DB_POSTGRES_SSLMODE=disable
DB_POSTGRES_MAX_CONNS=25
DB_POSTGRES_IDLE_CONNS=5
DB_POSTGRES_READ_REPLICAS=replica1:5432,replica2:5432

# DynamoDB settings
DB_DYNAMO_REGION=us-east-1
DB_DYNAMO_TABLE=your_table
DB_DYNAMO_ACCESS_KEY_ID=your_access_key
DB_DYNAMO_SECRET_ACCESS_KEY=your_secret_key

# SpiceDB settings
DB_SPICEDB_ENDPOINT=localhost:50051
DB_SPICEDB_TOKEN=your_token
DB_SPICEDB_INSECURE=true

# S3 settings
DB_S3_REGION=us-east-1
DB_S3_BUCKET=your_bucket
DB_S3_ACCESS_KEY_ID=your_access_key
DB_S3_SECRET_ACCESS_KEY=your_secret_key

# Logging
DB_LOG_LEVEL=info
```

## Connection Status & Health Checks
```go
// Check individual backend connections
func (s *HealthService) CheckDatabaseHealth(ctx context.Context) map[string]bool {
    dbMgr := s.dbMgr
    
    return map[string]bool{
        "postgres": dbMgr.IsConnected(db.BackendGorm),
        "dynamo":   dbMgr.IsConnected(db.BackendDynamo),
        "spicedb":  dbMgr.IsConnected(db.BackendSpiceDB),
        "s3":       dbMgr.IsConnected(db.BackendS3),
        "memory":   dbMgr.IsConnected(db.BackendInMemory),
    }
}

// Get all available managers
func (s *DatabaseService) GetAvailableBackends() []string {
    managers := s.dbMgr.GetAllManagers()
    backends := make([]string, 0, len(managers))
    
    for backend := range managers {
        backends = append(backends, backend.String())
    }
    
    return backends
}

// Graceful shutdown
func (s *DatabaseService) Shutdown(ctx context.Context) error {
    // Close all connections
    if err := s.dbMgr.Close(); err != nil {
        log.Printf("Error closing database connections: %v", err)
        return err
    }
    
    log.Println("All database connections closed successfully")
    return nil
}
```

## Multi-Backend Service Pattern
```go
// Service using multiple backends
type HybridService struct {
    dbMgr *db.DatabaseManager
}

func (s *HybridService) SaveUserProfile(ctx context.Context, profile *UserProfile) error {
    // Save structured data to PostgreSQL
    postgresManager := s.dbMgr.GetPostgresManager()
    if postgresManager != nil {
        err := postgresManager.WithTransaction(ctx, func(tx *gorm.DB) error {
            return tx.Create(profile).Error
        })
        if err != nil {
            return err
        }
    }
    
    // Save document/metadata to DynamoDB
    dynamoManager := s.dbMgr.GetDynamoManager()
    if dynamoManager != nil {
        metadata := profile.ToMetadata()
        err := dynamoManager.WithClient(ctx, func(client *dynamodb.Client) error {
            item, _ := attributevalue.MarshalMap(metadata)
            _, err := client.PutItem(ctx, &dynamodb.PutItemInput{
                TableName: aws.String("user_metadata"),
                Item:      item,
            })
            return err
        })
        if err != nil {
            log.Printf("Failed to save to DynamoDB: %v", err)
            // Don't fail the operation if DynamoDB fails
        }
    }
    
    // Set up permissions in SpiceDB
    spiceManager := s.dbMgr.GetSpiceManager()
    if spiceManager != nil {
        err := spiceManager.WithClient(ctx, func(client *authzed.Client) error {
            _, err := client.WriteRelationships(ctx, &v1.WriteRelationshipsRequest{
                Updates: []*v1.RelationshipUpdate{
                    {
                        Operation: v1.RelationshipUpdate_OPERATION_TOUCH,
                        Relationship: &v1.Relationship{
                            Resource: &v1.ObjectReference{
                                ObjectType: "profile",
                                ObjectId:   profile.GetID(),
                            },
                            Relation: "owner",
                            Subject: &v1.SubjectReference{
                                Object: &v1.ObjectReference{
                                    ObjectType: "user",
                                    ObjectId:   profile.UserID,
                                },
                            },
                        },
                    },
                },
            })
            return err
        })
        if err != nil {
            log.Printf("Failed to set permissions: %v", err)
        }
    }
    
    return nil
}
```
