# KISANLINK-DB Request/Response Patterns for GitHub Copilot

> 📝 **Copilot Context**: This file contains request/response patterns for the KISANLINK-DB package.
> Keep this open when creating API handlers and request/response structures.

## Essential Imports
```go
import (
    "github.com/Kisanlink/kisanlink-db/pkg/base"
)
```

## Request Types Pattern

### Create Request
```go
type CreateProductRequest struct {
    base.CreateRequest                    // Embeds common fields
    Name        string  `json:"name" validate:"required"`
    SKU         string  `json:"sku" validate:"required"`
    Price       float64 `json:"price" validate:"required,min=0"`
    Category    string  `json:"category" validate:"required"`
    Description string  `json:"description,omitempty"`
}
```

### Update Request
```go
type UpdateProductRequest struct {
    base.UpdateRequest                    // Contains ID field
    Name        string  `json:"name" validate:"required"`
    SKU         string  `json:"sku" validate:"required"`
    Price       float64 `json:"price" validate:"required,min=0"`
    Category    string  `json:"category" validate:"required"`
    Description string  `json:"description,omitempty"`
}
```

### List Request with Filters
```go
type ListProductsRequest struct {
    base.ListRequest                      // Contains Page, PerPage fields
    Category    string `json:"category,omitempty"`
    MinPrice    float64 `json:"min_price,omitempty"`
    MaxPrice    float64 `json:"max_price,omitempty"`
    IsActive    *bool  `json:"is_active,omitempty"`  // Pointer for optional bool
    Search      string `json:"search,omitempty"`
}
```

### Get by ID Request
```go
type GetProductRequest struct {
    base.GetByIDRequest                   // Contains ID field
}
```

### Delete Request
```go
type DeleteProductRequest struct {
    base.DeleteRequest                    // Contains ID field
}
```

## Response Creation Patterns

### Success Response
```go
// Simple success
response := base.NewSuccessResponse("Operation completed successfully", data)

// Success with model
response := base.NewSuccessResponse("Product created successfully", product)

// Success without data
response := base.NewSuccessResponse("Product deleted successfully", nil)
```

### Error Responses
```go
// Validation error (400)
errObj := base.NewValidationError("Invalid input", "Name is required")
response := base.NewErrorResponse("Validation failed", errObj)

// Not found error (404)
errObj := base.NewNotFoundError("Product", productID)
response := base.NewErrorResponse("Product not found", errObj)

// Conflict error (409)
errObj := base.NewConflictError("Product", "SKU already exists")
response := base.NewErrorResponse("Duplicate SKU", errObj)

// Internal server error (500)
errObj := base.NewInternalServerError("Database connection failed", err.Error())
response := base.NewErrorResponse("Internal server error", errObj)
```

### Paginated Response
```go
// Create pagination info
pagination := base.NewPaginationInfo(
    req.Page,           // current page
    req.PerPage,        // items per page  
    int(totalCount),    // total items
)

// Create paginated response
response := base.NewPaginatedResponse("Products retrieved successfully", products, pagination)
```

## Complete Handler Examples

### Create Handler
```go
func (h *ProductHandler) CreateProduct(c *gin.Context) {
    var req CreateProductRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response := base.NewErrorResponse("Invalid request format", 
            base.NewValidationError("JSON parsing failed", err.Error()))
        c.JSON(400, response)
        return
    }
    
    // Set user ID from context
    req.SetUserID(getUserFromContext(c))
    
    response, err := h.service.CreateProduct(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, base.NewErrorResponse("Internal server error", 
            base.NewInternalServerError("Service error", err.Error())))
        return
    }
    
    // Determine status code based on response
    statusCode := 201
    if response.Success {
        c.JSON(statusCode, response)
    } else {
        c.JSON(400, response)
    }
}
```

### Get by ID Handler
```go
func (h *ProductHandler) GetProduct(c *gin.Context) {
    var req GetProductRequest
    req.ID = c.Param("id")
    
    if req.ID == "" {
        response := base.NewErrorResponse("Missing product ID", 
            base.NewValidationError("Invalid request", "Product ID is required"))
        c.JSON(400, response)
        return
    }
    
    response, err := h.service.GetProductByID(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, base.NewErrorResponse("Internal server error", 
            base.NewInternalServerError("Service error", err.Error())))
        return
    }
    
    if response.Success {
        c.JSON(200, response)
    } else {
        c.JSON(404, response)
    }
}
```

### List Handler with Pagination
```go
func (h *ProductHandler) GetProducts(c *gin.Context) {
    var req ListProductsRequest
    
    // Bind query parameters
    if err := c.ShouldBindQuery(&req); err != nil {
        response := base.NewErrorResponse("Invalid query parameters", 
            base.NewValidationError("Parameter binding failed", err.Error()))
        c.JSON(400, response)
        return
    }
    
    // Set default pagination if not provided
    if req.Page == 0 {
        req.Page = 1
    }
    if req.PerPage == 0 {
        req.PerPage = 20
    }
    
    response, err := h.service.GetProducts(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, base.NewErrorResponse("Internal server error", 
            base.NewInternalServerError("Service error", err.Error())))
        return
    }
    
    c.JSON(200, response)
}
```

### Update Handler
```go
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
    var req UpdateProductRequest
    req.ID = c.Param("id")
    
    if req.ID == "" {
        response := base.NewErrorResponse("Missing product ID", 
            base.NewValidationError("Invalid request", "Product ID is required"))
        c.JSON(400, response)
        return
    }
    
    if err := c.ShouldBindJSON(&req); err != nil {
        response := base.NewErrorResponse("Invalid request format", 
            base.NewValidationError("JSON parsing failed", err.Error()))
        c.JSON(400, response)
        return
    }
    
    // Set user ID from context
    req.SetUserID(getUserFromContext(c))
    
    response, err := h.service.UpdateProduct(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, base.NewErrorResponse("Internal server error", 
            base.NewInternalServerError("Service error", err.Error())))
        return
    }
    
    if response.Success {
        c.JSON(200, response)
    } else {
        c.JSON(400, response)
    }
}
```

### Delete Handler
```go
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
    var req DeleteProductRequest
    req.ID = c.Param("id")
    req.SetUserID(getUserFromContext(c))
    
    if req.ID == "" {
        response := base.NewErrorResponse("Missing product ID", 
            base.NewValidationError("Invalid request", "Product ID is required"))
        c.JSON(400, response)
        return
    }
    
    response, err := h.service.DeleteProduct(c.Request.Context(), &req)
    if err != nil {
        c.JSON(500, base.NewErrorResponse("Internal server error", 
            base.NewInternalServerError("Service error", err.Error())))
        return
    }
    
    c.JSON(200, response)
}
```

## Error Types Reference
```go
// 400 Bad Request
base.NewValidationError("message", "details")

// 401 Unauthorized  
base.NewUnauthorizedError("Authentication required")

// 403 Forbidden
base.NewForbiddenError("Access denied")

// 404 Not Found
base.NewNotFoundError("ModelName", "id")

// 409 Conflict
base.NewConflictError("ModelName", "reason")

// 429 Too Many Requests
base.NewTooManyRequestsError("Rate limit exceeded")

// 500 Internal Server Error
base.NewInternalServerError("message", "details")

// 503 Service Unavailable
base.NewServiceUnavailableError("Service temporarily unavailable")
```

## Validation Tags Reference
```go
// Required fields
`validate:"required"`

// String validation
`validate:"required,min=3,max=100"`
`validate:"email"`
`validate:"url"`
`validate:"alphanum"`

// Numeric validation
`validate:"required,min=0"`
`validate:"required,min=0,max=1000"`
`validate:"numeric"`

// Array validation
`validate:"required,dive,required"`  // Each array element required

// Custom validation
`validate:"required,oneof=active inactive"`
```

## Context Helper Functions
```go
// Extract user ID from context (implement based on your auth system)
func getUserFromContext(c *gin.Context) string {
    if userID, exists := c.Get("user_id"); exists {
        return userID.(string)
    }
    return "system"
}

// Extract user role from context
func getUserRoleFromContext(c *gin.Context) string {
    if role, exists := c.Get("user_role"); exists {
        return role.(string)
    }
    return "guest"
}
```

## Quick Handler Creation Checklist
- [ ] Define request struct with proper validation tags
- [ ] Bind request data (JSON for body, Query for params)
- [ ] Validate required fields and format
- [ ] Extract user context information
- [ ] Call service method with context
- [ ] Handle service errors appropriately
- [ ] Return proper HTTP status codes
- [ ] Use consistent response format