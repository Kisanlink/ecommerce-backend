package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/gin-gonic/gin"
)

// ProductHandler handles product-related HTTP requests
type ProductHandler struct {
	productService *services.ProductService
}

// NewProductHandler creates a new product handler
func NewProductHandler(productService *services.ProductService) *ProductHandler {
	return &ProductHandler{
		productService: productService,
	}
}

// CreateProduct handles POST /api/v1/products
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req requests.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set user ID from context
	req.SetUserID(getUserFromContext(c))

	response, err := h.productService.CreateProduct(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusCreated, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// GetProduct handles GET /api/v1/products/:id
func (h *ProductHandler) GetProduct(c *gin.Context) {
	var req requests.GetProductRequest
	req.ID = c.Param("id")

	if req.ID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response, err := h.productService.GetProductByID(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusNotFound, response)
	}
}

// GetProducts handles GET /api/v1/products
func (h *ProductHandler) GetProducts(c *gin.Context) {
	var req requests.ListProductsRequest

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		response := base.NewErrorResponse("Invalid query parameters",
			base.NewValidationError("Parameter binding failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set default pagination if not provided
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PerPage == 0 {
		req.PerPage = 20
	}

	response, err := h.productService.GetProducts(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateProduct handles PUT /api/v1/products/:id
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	var req requests.UpdateProductRequest
	req.ID = c.Param("id")

	if req.ID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set user ID from context
	req.SetUserID(getUserFromContext(c))

	response, err := h.productService.UpdateProduct(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	if response.Success {
		c.JSON(http.StatusOK, response)
	} else {
		c.JSON(http.StatusBadRequest, response)
	}
}

// DeleteProduct handles DELETE /api/v1/products/:id
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	var req requests.DeleteProductRequest
	req.ID = c.Param("id")
	req.SetUserID(getUserFromContext(c))

	if req.ID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response, err := h.productService.DeleteProduct(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetProductsByCategory handles GET /api/v1/products/category/:category
func (h *ProductHandler) GetProductsByCategory(c *gin.Context) {
	var req requests.ListProductsRequest
	req.Category = c.Param("category")

	// Bind query parameters for pagination
	if err := c.ShouldBindQuery(&req); err != nil {
		response := base.NewErrorResponse("Invalid query parameters",
			base.NewValidationError("Parameter binding failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set default pagination if not provided
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PerPage == 0 {
		req.PerPage = 20
	}

	response, err := h.productService.GetProducts(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// SearchProducts handles GET /api/v1/products/search
func (h *ProductHandler) SearchProducts(c *gin.Context) {
	var req requests.ListProductsRequest
	req.Search = c.Query("q")

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		response := base.NewErrorResponse("Invalid query parameters",
			base.NewValidationError("Parameter binding failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set default pagination if not provided
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PerPage == 0 {
		req.PerPage = 20
	}

	response, err := h.productService.GetProducts(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// Helper function to extract user ID from context
func getUserFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return "system"
}
