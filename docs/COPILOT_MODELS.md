# KISANLINK-DB Model Patterns for GitHub Copilot

> 📝 **Copilot Context**: This file contains model creation patterns for the KISANLINK-DB package..

## Essential Imports
```go
import (
    "github.com/Kisanlink/kisanlink-db/pkg/base"
    "github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)
```

## Model Pattern Template
```go
// Standard model structure - ALWAYS follow this pattern
type YourModel struct {
    *base.BaseModel `gorm:"embedded"`
    Name            string `json:"name" gorm:"not null"`
    Description     string `json:"description"`
    IsActive        bool   `json:"is_active" gorm:"default:true"`
}

// Constructor pattern - ALWAYS include
func NewYourModel(name, description string) *YourModel {
    return &YourModel{
        BaseModel:   base.NewBaseModel("YMOD", hash.Small),
        Name:        name,
        Description: description,
        IsActive:    true,
    }
}

// TableName method - REQUIRED for GORM
func (m *YourModel) TableName() string {
    return "your_models"
}
```

## Hash Size Selection Guide
```go
// Choose based on expected table size:
hash.Tiny    // 4 digits  - "YMOD1234"     - < 1K records
hash.Small   // 6 digits  - "YMOD123456"   - < 100K records  
hash.Medium  // 8 digits  - "YMOD12345678" - < 10M records
hash.Large   // 10 digits - "YMOD1234567890" - < 1B records
hash.XLarge  // 12 digits - "YMOD123456789012" - > 1B records
```

## Common Model Examples

### User Model
```go
type User struct {
    *base.BaseModel `gorm:"embedded"`
    Email           string `json:"email" gorm:"uniqueIndex;not null"`
    Username        string `json:"username" gorm:"uniqueIndex;not null"`
    FirstName       string `json:"first_name"`
    LastName        string `json:"last_name"`
    IsActive        bool   `json:"is_active" gorm:"default:true"`
    Role            string `json:"role" gorm:"default:'user'"`
}

func NewUser(email, username, firstName, lastName string) *User {
    return &User{
        BaseModel: base.NewBaseModel("USER", hash.Medium),
        Email:     email,
        Username:  username,
        FirstName: firstName,
        LastName:  lastName,
        IsActive:  true,
        Role:      "user",
    }
}

func (u *User) TableName() string {
    return "users"
}
```

### Product Model
```go
type Product struct {
    *base.BaseModel `gorm:"embedded"`
    Name            string  `json:"name" gorm:"not null"`
    SKU             string  `json:"sku" gorm:"uniqueIndex"`
    Price           float64 `json:"price" gorm:"type:decimal(10,2)"`
    Category        string  `json:"category"`
    Stock           int     `json:"stock" gorm:"default:0"`
    IsActive        bool    `json:"is_active" gorm:"default:true"`
}

func NewProduct(name, sku string, price float64, category string) *Product {
    return &Product{
        BaseModel: base.NewBaseModel("PROD", hash.Medium),
        Name:      name,
        SKU:       sku,
        Price:     price,
        Category:  category,
        Stock:     0,
        IsActive:  true,
    }
}

func (p *Product) TableName() string {
    return "products"
}
```

### Order Model
```go
type Order struct {
    *base.BaseModel `gorm:"embedded"`
    UserID          string  `json:"user_id" gorm:"not null;index"`
    TotalAmount     float64 `json:"total_amount" gorm:"type:decimal(10,2)"`
    Status          string  `json:"status" gorm:"default:'pending'"`
    ShippingAddress string  `json:"shipping_address"`
    OrderDate       time.Time `json:"order_date"`
}

func NewOrder(userID string, totalAmount float64, shippingAddress string) *Order {
    return &Order{
        BaseModel:       base.NewBaseModel("ORD", hash.Large),
        UserID:          userID,
        TotalAmount:     totalAmount,
        Status:          "pending",
        ShippingAddress: shippingAddress,
        OrderDate:       time.Now(),
    }
}

func (o *Order) TableName() string {
    return "orders"
}
```

## Essential Base Model Methods
```go
// ID operations
model.GetID()           // Get generated ID
model.SetID("NEW123456") // Set custom ID (rare usage)

// Audit fields - ALWAYS set these in services
model.SetCreatedBy(userID)  // Set creator
model.SetUpdatedBy(userID)  // Set updater

// Timestamps - automatically managed
model.GetCreatedAt()    // Get creation time
model.GetUpdatedAt()    // Get last update time

// Soft delete - preferred over hard delete
model.IsDeleted()       // Check deletion status
model.SetDeletedBy(&userID) // Set deleter
```

## GORM Tags Reference
```go
// Common GORM tags to use:
`gorm:"not null"`           // Required field
`gorm:"default:true"`       // Default value
`gorm:"uniqueIndex"`        // Unique constraint
`gorm:"index"`              // Database index
`gorm:"type:varchar(100)"`  // Specific SQL type
`gorm:"type:decimal(10,2)"` // Decimal with precision
`gorm:"embedded"`           // Embed BaseModel (REQUIRED)
```

## JSON Tags Best Practices
```go
// Standard JSON naming
Name        string `json:"name"`
FirstName   string `json:"first_name"`   // snake_case for JSON
IsActive    bool   `json:"is_active"`

// Optional fields in requests
Category    string `json:"category,omitempty"`
Description string `json:"description,omitempty"`
```

## Model Validation Patterns
```go
// Add validation tags for request structs
type CreateUserRequest struct {
    Email     string `json:"email" validate:"required,email"`
    Username  string `json:"username" validate:"required,min=3,max=50"`
    FirstName string `json:"first_name" validate:"required"`
    LastName  string `json:"last_name" validate:"required"`
}
```
## Quick Model Creation Checklist
- [ ] Embed `*base.BaseModel` with `gorm:"embedded"`
- [ ] Add constructor with `base.NewBaseModel(PREFIX, hash.Size)`
- [ ] Implement `TableName() string` method
- [ ] Use appropriate GORM tags
- [ ] Use snake_case for JSON tags
- [ ] Choose correct hash size
- [ ] Set default values in constructor
- [ ] Add proper validation tags for requests