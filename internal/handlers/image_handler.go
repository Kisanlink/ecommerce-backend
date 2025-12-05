package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ImageHandler handles image-related HTTP requests
type ImageHandler struct {
	imageService services.ImageServiceInterface
	authContext  *auth.AuthContext
	logger       *zap.Logger
}

// NewImageHandler creates a new image handler
func NewImageHandler(imageService services.ImageServiceInterface, authContext *auth.AuthContext, logger *zap.Logger) *ImageHandler {
	return &ImageHandler{
		imageService: imageService,
		authContext:  authContext,
		logger:       logger,
	}
}

// CreateImageRequest represents the request to create an image
type CreateImageRequest struct {
	ProductID string `json:"product_id" binding:"required"`
	ImageURL  string `json:"image_url" binding:"required"`
	ImageType string `json:"image_type" binding:"required"`
	AltText   string `json:"alt_text"`
	Caption   string `json:"caption"`
	SortOrder int    `json:"sort_order"`
	IsPrimary bool   `json:"is_primary"`
	IsActive  *bool  `json:"is_active"`
	MimeType  string `json:"mime_type"`
	FileSize  int64  `json:"file_size"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
}

// UpdateImageRequest represents the request to update an image
type UpdateImageRequest struct {
	ImageURL  string `json:"image_url"`
	ImageType string `json:"image_type"`
	AltText   string `json:"alt_text"`
	Caption   string `json:"caption"`
	SortOrder int    `json:"sort_order"`
	IsActive  *bool  `json:"is_active"`
}

// AddMultipleImagesRequest represents the request to add multiple images
type AddMultipleImagesRequest struct {
	ProductID string   `json:"product_id" binding:"required"`
	ImageURLs []string `json:"image_urls" binding:"required"`
	ImageType string   `json:"image_type" binding:"required"`
}

// ReorderImagesRequest represents the request to reorder images
type ReorderImagesRequest struct {
	ImageOrders map[string]int `json:"image_orders" binding:"required"`
}

// CreateImage creates a new image for a product
// POST /api/v1/images
// @Summary Create a new image
// @Description Create a new image for a product
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body handlers.CreateImageRequest true "Create image payload"
// @Success 201 {object} map[string]interface{} "Image created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /images [post]
func (h *ImageHandler) CreateImage(c *gin.Context) {
	requestID := c.GetString("request_id")
	h.logger.Info("Creating new image",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path))

	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User authentication failed for image creation",
			zap.String("request_id", requestID),
			zap.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid image creation request data",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate image type
	imageType := models.ImageType(req.ImageType)
	if !imageType.IsValidImageType() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image type"})
		return
	}

	// Create the image
	image, err := h.imageService.CreateImage(c.Request.Context(), req.ProductID, req.ImageURL, imageType, req.AltText, req.Caption)
	if err != nil {
		h.logger.Error("Failed to create image",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create image"})
		return
	}

	// Apply additional fields if provided
	needsUpdate := false
	if req.SortOrder != 0 {
		image.SortOrder = req.SortOrder
		needsUpdate = true
	}
	if req.IsPrimary {
		image.IsPrimary = req.IsPrimary
		needsUpdate = true
	}
	if req.IsActive != nil {
		image.IsActive = *req.IsActive
		needsUpdate = true
	}
	if req.MimeType != "" {
		image.MimeType = req.MimeType
		needsUpdate = true
	}
	if req.FileSize != 0 {
		image.FileSize = req.FileSize
		needsUpdate = true
	}
	if req.Width != 0 {
		image.Width = req.Width
		needsUpdate = true
	}
	if req.Height != 0 {
		image.Height = req.Height
		needsUpdate = true
	}

	// Update if any additional fields were provided
	if needsUpdate {
		if err := h.imageService.UpdateImage(c.Request.Context(), image); err != nil {
			h.logger.Warn("Failed to update image with additional fields",
				zap.String("request_id", requestID),
				zap.String("image_id", image.ID),
				zap.Error(err))
		}
	}

	h.logger.Info("Image created successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("image_id", image.ID),
		zap.String("product_id", req.ProductID))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Image created successfully",
		"image":   image,
	})
}

// GetImage retrieves an image by ID
// GET /api/v1/images/:id
// @Summary Get image by ID
// @Description Get image details by ID
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Image ID"
// @Success 200 {object} map[string]interface{} "Image retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid image ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Image not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /images/{id} [get]
func (h *ImageHandler) GetImage(c *gin.Context) {
	requestID := c.GetString("request_id")
	imageID := c.Param("id")

	h.logger.Info("Retrieving image",
		zap.String("request_id", requestID),
		zap.String("image_id", imageID))

	if imageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image ID is required"})
		return
	}

	image, err := h.imageService.GetImageByID(c.Request.Context(), imageID)
	if err != nil {
		h.logger.Error("Failed to get image",
			zap.String("request_id", requestID),
			zap.String("image_id", imageID),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"image": image,
	})
}

// UpdateImage updates an existing image
// PUT /api/v1/images/:id
// @Summary Update image
// @Description Update image details
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Image ID"
// @Param request body handlers.UpdateImageRequest true "Update image payload"
// @Success 200 {object} map[string]interface{} "Image updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Image not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /images/{id} [put]
func (h *ImageHandler) UpdateImage(c *gin.Context) {
	requestID := c.GetString("request_id")
	imageID := c.Param("id")

	h.logger.Info("Updating image",
		zap.String("request_id", requestID),
		zap.String("image_id", imageID))

	if imageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image ID is required"})
		return
	}

	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req UpdateImageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid image update request data",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Get existing image
	image, err := h.imageService.GetImageByID(c.Request.Context(), imageID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	// Update fields if provided
	if req.ImageURL != "" {
		image.ImageURL = req.ImageURL
	}
	if req.ImageType != "" {
		imageType := models.ImageType(req.ImageType)
		if imageType.IsValidImageType() {
			image.ImageType = imageType
		}
	}
	if req.AltText != "" {
		image.AltText = req.AltText
	}
	if req.Caption != "" {
		image.Caption = req.Caption
	}
	if req.SortOrder >= 0 {
		image.SortOrder = req.SortOrder
	}
	if req.IsActive != nil {
		image.IsActive = *req.IsActive
	}

	err = h.imageService.UpdateImage(c.Request.Context(), image)
	if err != nil {
		h.logger.Error("Failed to update image",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("image_id", imageID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update image"})
		return
	}

	h.logger.Info("Image updated successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("image_id", imageID))

	c.JSON(http.StatusOK, gin.H{
		"message": "Image updated successfully",
		"image":   image,
	})
}

// DeleteImage deletes an image
// DELETE /api/v1/images/:id
// @Summary Delete image
// @Description Delete an image
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Image ID"
// @Success 200 {object} map[string]interface{} "Image deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid image ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Image not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /images/{id} [delete]
func (h *ImageHandler) DeleteImage(c *gin.Context) {
	requestID := c.GetString("request_id")
	imageID := c.Param("id")

	h.logger.Info("Deleting image",
		zap.String("request_id", requestID),
		zap.String("image_id", imageID))

	if imageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Image ID is required"})
		return
	}

	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err := h.imageService.DeleteImage(c.Request.Context(), imageID)
	if err != nil {
		h.logger.Error("Failed to delete image",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("image_id", imageID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete image"})
		return
	}

	h.logger.Info("Image deleted successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("image_id", imageID))

	c.JSON(http.StatusOK, gin.H{
		"message": "Image deleted successfully",
	})
}

// GetImagesByProductID retrieves all images for a specific product
// GET /api/v1/products/:productId/images
// @Summary Get images by product ID
// @Description Get all images for a specific product
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param productId path string true "Product ID"
// @Param type query string false "Image type filter"
// @Param active query bool false "Filter by active status"
// @Success 200 {object} map[string]interface{} "Images retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid product ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{productId}/images [get]
func (h *ImageHandler) GetImagesByProductID(c *gin.Context) {
	requestID := c.GetString("request_id")
	productID := c.Param("productId")
	imageType := c.Query("type")
	activeOnly := c.Query("active") == "true"

	h.logger.Info("Retrieving images by product ID",
		zap.String("request_id", requestID),
		zap.String("product_id", productID),
		zap.String("image_type", imageType),
		zap.Bool("active_only", activeOnly))

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	var images []*models.Image
	var err error

	if imageType != "" {
		// Filter by image type
		imgType := models.ImageType(imageType)
		if !imgType.IsValidImageType() {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image type"})
			return
		}
		images, err = h.imageService.GetImagesByProductIDAndType(c.Request.Context(), productID, imgType)
	} else if activeOnly {
		// Get only active images
		images, err = h.imageService.GetActiveImagesByProductID(c.Request.Context(), productID)
	} else {
		// Get all images
		images, err = h.imageService.GetImagesByProductID(c.Request.Context(), productID)
	}

	if err != nil {
		h.logger.Error("Failed to get images by product ID",
			zap.String("request_id", requestID),
			zap.String("product_id", productID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve images"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"images": images,
		"count":  len(images),
	})
}

// GetPrimaryImageByProductID retrieves the primary image for a specific product
// GET /api/v1/products/:productId/images/primary
// @Summary Get primary image by product ID
// @Description Get the primary image for a specific product
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param productId path string true "Product ID"
// @Success 200 {object} map[string]interface{} "Primary image retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid product ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Primary image not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{productId}/images/primary [get]
func (h *ImageHandler) GetPrimaryImageByProductID(c *gin.Context) {
	requestID := c.GetString("request_id")
	productID := c.Param("productId")

	h.logger.Info("Retrieving primary image by product ID",
		zap.String("request_id", requestID),
		zap.String("product_id", productID))

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	image, err := h.imageService.GetPrimaryImageByProductID(c.Request.Context(), productID)
	if err != nil {
		h.logger.Error("Failed to get primary image",
			zap.String("request_id", requestID),
			zap.String("product_id", productID),
			zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Primary image not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"image": image,
	})
}

// SetPrimaryImage sets an image as the primary image for a product
// PUT /api/v1/products/:productId/images/:imageId/primary
// @Summary Set primary image
// @Description Set an image as the primary image for a product
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param productId path string true "Product ID"
// @Param imageId path string true "Image ID"
// @Success 200 {object} map[string]interface{} "Primary image set successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Image not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{productId}/images/{imageId}/primary [put]
func (h *ImageHandler) SetPrimaryImage(c *gin.Context) {
	requestID := c.GetString("request_id")
	productID := c.Param("productId")
	imageID := c.Param("imageId")

	h.logger.Info("Setting primary image",
		zap.String("request_id", requestID),
		zap.String("product_id", productID),
		zap.String("image_id", imageID))

	if productID == "" || imageID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID and Image ID are required"})
		return
	}

	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	err := h.imageService.SetPrimaryImage(c.Request.Context(), productID, imageID)
	if err != nil {
		h.logger.Error("Failed to set primary image",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.String("image_id", imageID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set primary image"})
		return
	}

	h.logger.Info("Primary image set successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("product_id", productID),
		zap.String("image_id", imageID))

	c.JSON(http.StatusOK, gin.H{
		"message": "Primary image set successfully",
	})
}

// AddMultipleImages adds multiple images to a product
// POST /api/v1/products/:productId/images/bulk
// @Summary Add multiple images
// @Description Add multiple images to a product
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param productId path string true "Product ID"
// @Param request body handlers.AddMultipleImagesRequest true "Add multiple images payload"
// @Success 201 {object} map[string]interface{} "Images added successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{productId}/images/bulk [post]
func (h *ImageHandler) AddMultipleImages(c *gin.Context) {
	requestID := c.GetString("request_id")
	productID := c.Param("productId")

	h.logger.Info("Adding multiple images",
		zap.String("request_id", requestID),
		zap.String("product_id", productID))

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req AddMultipleImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid bulk image creation request data",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate image type
	imageType := models.ImageType(req.ImageType)
	if !imageType.IsValidImageType() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image type"})
		return
	}

	// Add multiple images
	images, err := h.imageService.AddMultipleImages(c.Request.Context(), productID, req.ImageURLs, imageType)
	if err != nil {
		h.logger.Error("Failed to add multiple images",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add images"})
		return
	}

	h.logger.Info("Multiple images added successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("product_id", productID),
		zap.Int("created_count", len(images)))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Images added successfully",
		"images":  images,
		"count":   len(images),
	})
}

// ReorderImages reorders images for a product
// PUT /api/v1/products/:productId/images/reorder
// @Summary Reorder images
// @Description Reorder images for a product
// @Tags Images
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param productId path string true "Product ID"
// @Param request body handlers.ReorderImagesRequest true "Reorder images payload"
// @Success 200 {object} map[string]interface{} "Images reordered successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /products/{productId}/images/reorder [put]
func (h *ImageHandler) ReorderImages(c *gin.Context) {
	requestID := c.GetString("request_id")
	productID := c.Param("productId")

	h.logger.Info("Reordering images",
		zap.String("request_id", requestID),
		zap.String("product_id", productID))

	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req ReorderImagesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid reorder images request data",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	err := h.imageService.ReorderImages(c.Request.Context(), productID, req.ImageOrders)
	if err != nil {
		h.logger.Error("Failed to reorder images",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder images"})
		return
	}

	h.logger.Info("Images reordered successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("product_id", productID))

	c.JSON(http.StatusOK, gin.H{
		"message": "Images reordered successfully",
	})
}
