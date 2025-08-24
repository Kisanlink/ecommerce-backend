# KISANLINK-DB Repository Patterns for GitHub Copilot

> 📝 **Copilot Context**: This file contains repository and database patterns for the KISANLINK-DB package.
> Keep this open when implementing services and database operations.

## Essential Imports
```go
import (
    "context"
    "github.com/Kisanlink/kisanlink-db/pkg/base"
    "github.com/Kisanlink/kisanlink-db/pkg/db"
)
```

## Repository Creation Pattern
```go
// Standard repository setup - ALWAYS follow this pattern
type YourService struct {
    repo   base.FilterableRepository[*YourModel]
    dbMgr  *db.DatabaseManager
}

func NewYourService() *YourService {
    dbMgr := db.NewDatabaseManager()
    repo := base.NewBaseFilterableRepository[*YourModel]()
    
    // Connect repository to database manager
    if err := dbMgr.Connect(context.Background()); err == nil {
        repo.SetDBManager(dbMgr.GetPostgresManager())
    }
    
    return &YourService{
        repo:  repo,
        dbMgr: dbMgr,
    }
}

// ALWAYS implement Close method
func (s *YourService) Close() error {
    return s.dbMgr.Close()
}
```

## Basic CRUD Operations
```go
// Create operation
func (s *YourService) Create(ctx context.Context, model *YourModel) error {
    model.SetCreatedBy(getUserFromContext(ctx))
    return s.repo.Create(ctx, model)
}

// Get by ID operation
func (s *YourService) GetByID(ctx context.Context, id string) (*YourModel, error) {
    return s.repo.GetByID(ctx, id, &YourModel{})
}

// Update operation
func (s *YourService) Update(ctx context.Context, model *YourModel) error {
    model.SetUpdatedBy(getUserFromContext(ctx))
    return s.repo.Update(ctx, model)
}

// Soft delete operation (preferred)
func (s *YourService) Delete(ctx context.Context, id, userID string) error {
    return s.repo.SoftDelete(ctx, id, userID)
}

// List with pagination
func (s *YourService) List(ctx context.Context, limit, offset int) ([]*YourModel, error) {
    return s.repo.List(ctx, limit, offset)
}

// Count total records
func (s *YourService) Count(ctx context.Context) (int64, error) {
    return s.repo.Count(ctx)
}

// Check existence
func (s *YourService) Exists(ctx context.Context, id string) (bool, error) {
    return s.repo.Exists(ctx, id)
}
```

## Advanced Filtering Patterns
```go
// Complex filter example
func (s *ProductService) GetActiveProducts(ctx context.Context, category string, minPrice float64) ([]*Product, error) {
    filter := base.NewFilterBuilder().
        Where("is_active", base.OpEqual, true).
        Where("category", base.OpEqual, category).
        Where("price", base.OpGreaterEqual, minPrice).
        Sort("created_at", "desc").
        Page(1, 20).
        Build()
    
    return s.repo.Find(ctx, filter)
}

// Search with multiple conditions
func (s *UserService) SearchUsers(ctx context.Context, searchTerm string, isActive bool) ([]*User, error) {
    filter := base.NewFilterBuilder().
        Where("is_active", base.OpEqual, isActive).
        Group(func(gb *base.GroupBuilder) {
            gb.Where("email", base.OpContains, searchTerm).
               Or("username", base.OpContains, searchTerm).
               Or("first_name", base.OpContains, searchTerm)
        }).
        Sort("username", "asc").
        Build()
    
    return s.repo.Find(ctx, filter)
}
```

## Filter Operators Reference
```go
// Comparison operators
base.OpEqual        // =
base.OpNotEqual     // !=
base.OpGreaterThan  // >
base.OpGreaterEqual // >=
base.OpLessThan     // <
base.OpLessEqual    // <=

// String operators
base.OpContains     // ILIKE %value%
base.OpStartsWith   // ILIKE value%
base.OpEndsWith     // ILIKE %value
base.OpLike         // LIKE with custom wildcards
base.OpNotLike      // NOT LIKE

// Array/Null operators
base.OpIn           // IN (value1, value2, ...)
base.OpNotIn        // NOT IN (value1, value2, ...)
base.OpIsNull       // IS NULL
base.OpIsNotNull    // IS NOT NULL

// Date operators
base.OpDateEqual    // DATE(field) = DATE(value)
base.OpDateBefore   // DATE(field) < DATE(value)
base.OpDateAfter    // DATE(field) > DATE(value)
base.OpDateBetween  // DATE(field) BETWEEN date1 AND date2
```

## Service Method Patterns

### Create with Validation
```go
func (s *ProductService) CreateProduct(ctx context.Context, req *CreateProductRequest) (*base.BaseResponse, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return base.NewErrorResponse("Validation failed", err), nil
    }
    
    // Create model
    product := NewProduct(req.Name, req.SKU, req.Price, req.Category)
    product.SetCreatedBy(req.GetUserID())
    
    // Save to database
    if err := s.repo.Create(ctx, product); err != nil {
        return nil, err
    }
    
    return base.NewSuccessResponse("Product created successfully", product), nil
}
```

### Get with Error Handling
```go
func (s *ProductService) GetProductByID(ctx context.Context, req *GetProductRequest) (*base.BaseResponse, error) {
    filter := base.NewFilterBuilder().
        Where("id", base.OpEqual, req.ID).
        Where("is_active", base.OpEqual, true).
        Build()
    
    product, err := s.repo.FindOne(ctx, filter)
    if err != nil {
        return base.NewErrorResponse("Product not found", 
            base.NewNotFoundError("Product", req.ID)), nil
    }
    
    return base.NewSuccessResponse("Product retrieved", product), nil
}
```

### List with Pagination
```go
func (s *ProductService) GetProducts(ctx context.Context, req *ListProductsRequest) (*base.PaginatedResponse, error) {
    filter := base.NewFilterBuilder().
        Where("is_active", base.OpEqual, true).
        Sort("created_at", "desc").
        Page(req.Page, req.PerPage)
    
    // Add optional filters
    if req.Category != "" {
        filter.Where("category", base.OpEqual, req.Category)
    }
    
    // Execute query
    products, err := s.repo.Find(ctx, filter.Build())
    if err != nil {
        return nil, err
    }
    
    // Get total count
    count, err := s.repo.CountWithFilter(ctx, filter.Build())
    if err != nil {
        return nil, err
    }
    
    pagination := base.NewPaginationInfo(req.Page, req.PerPage, int(count))
    return base.NewPaginatedResponse("Products retrieved", products, pagination), nil
}
```

### Update with Existence Check
```go
func (s *ProductService) UpdateProduct(ctx context.Context, req *UpdateProductRequest) (*base.BaseResponse, error) {
    // Validate request
    if err := req.Validate(); err != nil {
        return base.NewErrorResponse("Validation failed", err), nil
    }
    
    // Get existing record
    existing, err := s.repo.GetByID(ctx, req.ID, &Product{})
    if err != nil {
        return base.NewErrorResponse("Product not found", 
            base.NewNotFoundError("Product", req.ID)), nil
    }
    
    // Update fields
    existing.Name = req.Name
    existing.Price = req.Price
    existing.Category = req.Category
    existing.SetUpdatedBy(req.GetUserID())
    
    // Save changes
    if err := s.repo.Update(ctx, existing); err != nil {
        return nil, err
    }
    
    return base.NewSuccessResponse("Product updated successfully", existing), nil
}
```

## Database Manager Usage
```go
// Environment-based setup (recommended)
dbManager := db.NewDatabaseManager()

// Manual configuration
config := &db.Config{
    PrimaryBackend:    db.BackendGorm,
    PostgresHost:      "localhost",
    PostgresPort:      "5432",
    PostgresUser:      "postgres",
    PostgresPassword:  "password",
    PostgresDBName:    "your_database",
    PostgresSSLMode:   "disable",
    PostgresMaxConns:  10,
    PostgresIdleConns: 5,
}
dbManager := db.NewDatabaseManagerWithConfig(config)

// Connection and cleanup
ctx := context.Background()
if err := dbManager.Connect(ctx); err != nil {
    log.Fatal("Failed to connect:", err)
}
defer dbManager.Close()
```

## Transaction Pattern
```go
func (s *OrderService) CreateOrderWithItems(ctx context.Context, order *Order, items []*OrderItem) error {
    postgresManager := s.dbMgr.GetPostgresManager()
    if postgresManager == nil {
        return errors.New("postgres manager not available")
    }
    
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
```

## Context Helper Function
```go
// Extract user ID from context
func getUserFromContext(ctx context.Context) string {
    if userID, ok := ctx.Value("user_id").(string); ok {
        return userID
    }
    return "system"
}