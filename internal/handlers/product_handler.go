package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/gin-gonic/gin"
)

// ProductHandler handles product-related HTTP requests
type ProductHandler struct {
	productService *services.ProductService
	r2Service      *services.R2Service
}

// NewProductHandler creates a new product handler
func NewProductHandler(productService *services.ProductService, r2Service *services.R2Service) *ProductHandler {
	return &ProductHandler{
		productService: productService,
		r2Service:      r2Service,
	}
}

// CreateProduct handles POST /api/v1/products

// @Summary Create a new product
// @Description Create a new product (Admin only)
// @Tags Admin - Products
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.CreateProductRequest true "Product information"
// @Success 201 {object} map[string]interface{} "Product created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products [post]
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var req requests.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := base.NewErrorResponse("Invalid request format",
			base.NewValidationError("JSON parsing failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Set user ID from context
	req.SetUserID(utils.GetUserFromContext(c))

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
// @Summary Get product by ID
// @Description Get a specific product by its ID
// @Tags Products
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} map[string]interface{} "Product retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid product ID"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{id} [get]
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
// @Summary Get all products
// @Description Get a paginated list of all products with optional filtering
// @Tags Products
// @Accept json
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Param category query string false "Filter by category"
// @Param search query string false "Search term"
// @Param min_price query number false "Minimum price filter"
// @Param max_price query number false "Maximum price filter"
// @Success 200 {object} map[string]interface{} "Products retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid query parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products [get]
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
	req.SetUserID(utils.GetUserFromContext(c))

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
	req.SetUserID(utils.GetUserFromContext(c))

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
// @Summary Search products
// @Description Search products by name, description, or other fields
// @Tags Products
// @Accept json
// @Produce json
// @Param q query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Param per_page query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{} "Search results retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid query parameters"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/search [get]
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

// UploadProductImage handles POST /api/v1/products/:id/images
// @Summary Upload product image
// @Description Upload an image for a product (Admin only)
// @Tags Admin - Products
// @Accept multipart/form-data
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Product ID"
// @Param image formData file true "Image file"
// @Param alt_text formData string false "Alt text for the image"
// @Param caption formData string false "Caption for the image"
// @Success 201 {object} map[string]interface{} "Image uploaded successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{id}/images [post]
func (h *ProductHandler) UploadProductImage(c *gin.Context) {
	productID := c.Param("id")
	if productID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		response := base.NewErrorResponse("No image file provided",
			base.NewValidationError("File upload failed", err.Error()))
		c.JSON(http.StatusBadRequest, response)
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		response := base.NewErrorResponse("Invalid file type",
			base.NewValidationError("File type", "Only JPEG, PNG, and WebP images are allowed"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate file size (max 10MB)
	if header.Size > 10*1024*1024 {
		response := base.NewErrorResponse("File too large",
			base.NewValidationError("File size", "Maximum file size is 10MB"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Create request
	req := &requests.UploadProductImageRequest{
		ProductID: productID,
		AltText:   c.PostForm("alt_text"),
		Caption:   c.PostForm("caption"),
	}
	req.SetUserID(utils.GetUserFromContext(c))

	// Use the R2 service from the handler
	response, err := h.productService.UploadProductImage(c.Request.Context(), req, file, header, h.r2Service)
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

// DeleteProductImage handles DELETE /api/v1/products/:id/images/:imageId
// @Summary Delete product image
// @Description Delete a product image (Admin only)
// @Tags Admin - Products
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Product ID"
// @Param imageId path string true "Image ID"
// @Success 200 {object} map[string]interface{} "Image deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 404 {object} map[string]interface{} "Image not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{id}/images/{imageId} [delete]
func (h *ProductHandler) DeleteProductImage(c *gin.Context) {
	productID := c.Param("id")
	imageID := c.Param("imageId")

	if productID == "" || imageID == "" {
		response := base.NewErrorResponse("Missing product ID or image ID",
			base.NewValidationError("Invalid request", "Product ID and Image ID are required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	req := &requests.DeleteProductImageRequest{
		ProductID: productID,
		ImageID:   imageID,
	}
	req.SetUserID(utils.GetUserFromContext(c))

	response, err := h.productService.DeleteProductImage(c.Request.Context(), req, h.r2Service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// SetPrimaryImage handles PUT /api/v1/products/:id/images/:imageId/primary
// @Summary Set primary image
// @Description Set an image as the primary image for a product (Admin only)
// @Tags Admin - Products
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Product ID"
// @Param imageId path string true "Image ID"
// @Success 200 {object} map[string]interface{} "Primary image set successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 404 {object} map[string]interface{} "Image not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{id}/images/{imageId}/primary [put]
func (h *ProductHandler) SetPrimaryImage(c *gin.Context) {
	productID := c.Param("id")
	imageID := c.Param("imageId")

	if productID == "" || imageID == "" {
		response := base.NewErrorResponse("Missing product ID or image ID",
			base.NewValidationError("Invalid request", "Product ID and Image ID are required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	req := &requests.SetPrimaryImageRequest{
		ProductID: productID,
		ImageID:   imageID,
	}
	req.SetUserID(utils.GetUserFromContext(c))

	response, err := h.productService.SetPrimaryImage(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetProductImages handles GET /api/v1/products/:id/images
// @Summary Get product images
// @Description Get all images for a product (Admin only)
// @Tags Admin - Products
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Product ID"
// @Success 200 {object} map[string]interface{} "Images retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Unauthorized - Admin access required"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{id}/images [get]
func (h *ProductHandler) GetProductImages(c *gin.Context) {
	productID := c.Param("id")

	if productID == "" {
		response := base.NewErrorResponse("Missing product ID",
			base.NewValidationError("Invalid request", "Product ID is required"))
		c.JSON(http.StatusBadRequest, response)
		return
	}

	req := &requests.GetProductImagesRequest{
		ProductID: productID,
	}

	response, err := h.productService.GetProductImages(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, base.NewErrorResponse("Internal server error",
			base.NewInternalServerError("Service error", err.Error())))
		return
	}

	c.JSON(http.StatusOK, response)
}
