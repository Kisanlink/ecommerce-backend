# Swagger Documentation Setup

## Overview

This project uses Swagger/OpenAPI documentation with Scalar UI for interactive API documentation.

## Dependencies

The following packages have been added to `go.mod`:

- `github.com/swaggo/swag` - Swagger code generation
- `github.com/swaggo/gin-swagger` - Gin integration
- `github.com/swaggo/files` - Static file serving
- `github.com/MarceloPetrucio/go-scalar-api-reference` - Scalar UI

## Setup

1. **Install swag CLI tool**:
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. **Generate documentation**:
   ```bash
   swag init -g cmd/server/main.go -o docs/
   ```

3. **Start the server**:
   ```bash
   go run cmd/server/main.go
   ```

## Access Documentation

- **Swagger UI with Scalar**: http://localhost:8080/docs
- **Raw OpenAPI JSON**: http://localhost:8080/docs/swagger.json

## Adding Documentation

Add Swagger annotations to your handlers:

```go
// @Summary Create a new product
// @Description Create a new product (Admin only)
// @Tags Products
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.CreateProductRequest true "Product information"
// @Success 201 {object} map[string]interface{} "Product created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
    // Handler implementation
}
```

## Available Tags

- **Authentication**: User auth endpoints
- **Products**: Product management
- **Cart**: Shopping cart operations
- **Orders**: Order processing
- **Payment**: Payment integration
- **Users**: User management
- **Favorites**: User favorites
- **Recipes**: Recipe management
- **Delivery**: Delivery tracking

## Security

The API uses Firebase Authentication. Include the token in the Authorization header:

```
Authorization: Bearer <firebase_id_token>
```
