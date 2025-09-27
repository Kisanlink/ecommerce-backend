package handlers

import (
	"net/http"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ReviewHandler handles HTTP requests for product reviews
type ReviewHandler struct {
	reviewService *services.ReviewService
	logger        *zap.Logger
}

// NewReviewHandler creates a new review handler
func NewReviewHandler(reviewService *services.ReviewService, logger *zap.Logger) *ReviewHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &ReviewHandler{
		reviewService: reviewService,
		logger:        logger,
	}
}

// CreateReview handles POST /products/:id/reviews
// @Summary Create a new product review
// @Description Create a new review for a product (authenticated users only)
// @Tags reviews
// @Accept json
// @Produce json
// @Param productId path string true "Product ID"
// @Param request body requests.CreateReviewRequest true "Review creation request"
// @Success 201 {object} responses.ReviewResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{productId}/reviews [post]
func (h *ReviewHandler) CreateReview(c *gin.Context) {
	// Get user ID from authentication context
	userID, exists := c.Get("firebase_uid")
	if !exists {
		userID, exists = c.Get("user_id")
		if !exists {
			h.logger.Error("User ID not found in context")
			c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Authentication required"})
			return
		}
	}

	userIDStr, ok := userID.(string)
	if !ok {
		h.logger.Error("Invalid user ID type in context")
		c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Invalid authentication"})
		return
	}

	// Get product ID from URL parameter
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Product ID is required"})
		return
	}

	// Parse request body
	var req requests.CreateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to parse create review request", zap.Error(err))
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid request format"})
		return
	}

	// Set product ID from URL parameter
	req.ProductID = productID

	// Create the review
	review, err := h.reviewService.CreateReview(c.Request.Context(), &req, userIDStr)
	if err != nil {
		h.logger.Error("Failed to create review", zap.Error(err), zap.String("user_id", userIDStr), zap.String("product_id", productID))
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, map[string]interface{}{
		"message": "Review created successfully",
		"data":    review,
	})
}

// GetProductReviews handles GET /products/:id/reviews
// @Summary Get all reviews for a product
// @Description Get paginated list of reviews for a specific product
// @Tags reviews
// @Accept json
// @Produce json
// @Param productId path string true "Product ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param sort_by query string false "Sort by: date, rating, likes" default("date")
// @Param order query string false "Sort order: asc, desc" default("desc")
// @Param min_rating query int false "Minimum rating filter" minimum(1) maximum(5)
// @Param max_rating query int false "Maximum rating filter" minimum(1) maximum(5)
// @Success 200 {object} responses.PaginatedReviewsResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{productId}/reviews [get]
func (h *ReviewHandler) GetProductReviews(c *gin.Context) {
	// Get product ID from URL parameter
	productID := c.Param("id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Product ID is required"})
		return
	}

	// Parse query parameters
	req := requests.GetProductReviewsRequest{
		ProductID: productID,
	}

	// Parse pagination parameters
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		req.Page = page
	} else {
		req.Page = 1
	}

	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "10")); err == nil {
		req.Limit = limit
	} else {
		req.Limit = 10
	}

	// Parse sorting parameters
	req.SortBy = c.DefaultQuery("sort_by", "date")
	req.Order = c.DefaultQuery("order", "desc")

	// Parse rating filters
	if minRating, err := strconv.Atoi(c.Query("min_rating")); err == nil {
		req.MinRating = minRating
	}
	if maxRating, err := strconv.Atoi(c.Query("max_rating")); err == nil {
		req.MaxRating = maxRating
	}

	// Get requesting user ID (optional for public endpoints)
	requestingUserID := ""
	if userID, exists := c.Get("firebase_uid"); exists {
		requestingUserID, _ = userID.(string)
	} else if userID, exists := c.Get("user_id"); exists {
		requestingUserID, _ = userID.(string)
	}

	// Get the reviews
	reviews, err := h.reviewService.GetProductReviews(c.Request.Context(), &req, requestingUserID)
	if err != nil {
		h.logger.Error("Failed to get product reviews", zap.Error(err), zap.String("product_id", productID))
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Reviews retrieved successfully",
		"data":    reviews,
	})
}

// GetUserReviews handles GET /users/:id/reviews
// @Summary Get all reviews by a user
// @Description Get paginated list of reviews written by a specific user
// @Tags reviews
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Param sort_by query string false "Sort by: date, rating, likes" default("date")
// @Param order query string false "Sort order: asc, desc" default("desc")
// @Success 200 {object} responses.PaginatedReviewsResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /users/{userId}/reviews [get]
func (h *ReviewHandler) GetUserReviews(c *gin.Context) {
	// Get user ID from URL parameter
	targetUserID := c.Param("id")
	if targetUserID == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "User ID is required"})
		return
	}

	// Parse query parameters
	req := requests.GetUserReviewsRequest{
		UserID: targetUserID,
	}

	// Parse pagination parameters
	if page, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil {
		req.Page = page
	} else {
		req.Page = 1
	}

	if limit, err := strconv.Atoi(c.DefaultQuery("limit", "10")); err == nil {
		req.Limit = limit
	} else {
		req.Limit = 10
	}

	// Parse sorting parameters
	req.SortBy = c.DefaultQuery("sort_by", "date")
	req.Order = c.DefaultQuery("order", "desc")

	// Get requesting user ID (optional)
	requestingUserID := ""
	if userID, exists := c.Get("firebase_uid"); exists {
		requestingUserID, _ = userID.(string)
	} else if userID, exists := c.Get("user_id"); exists {
		requestingUserID, _ = userID.(string)
	}

	// Get the reviews
	reviews, err := h.reviewService.GetUserReviews(c.Request.Context(), &req, requestingUserID)
	if err != nil {
		h.logger.Error("Failed to get user reviews", zap.Error(err), zap.String("user_id", targetUserID))
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "User reviews retrieved successfully",
		"data":    reviews,
	})
}

// UpdateReview handles PUT /products/:id/reviews/:reviewId
// @Summary Update a review
// @Description Update an existing review (only by review owner)
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param reviewId path string true "Review ID"
// @Param request body requests.UpdateReviewRequest true "Review update request"
// @Success 200 {object} responses.ReviewResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id}/reviews/{reviewId} [put]
func (h *ReviewHandler) UpdateReview(c *gin.Context) {
	// Get user ID from authentication context
	userID, exists := c.Get("firebase_uid")
	if !exists {
		userID, exists = c.Get("user_id")
		if !exists {
			h.logger.Error("User ID not found in context")
			c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Authentication required"})
			return
		}
	}

	userIDStr, ok := userID.(string)
	if !ok {
		h.logger.Error("Invalid user ID type in context")
		c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Invalid authentication"})
		return
	}

	// Get parameters
	productID := c.Param("id")      // Product ID comes from /products/:id/
	reviewID := c.Param("reviewId") // Review ID comes from /:reviewId
	if productID == "" || reviewID == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Product ID and Review ID are required"})
		return
	}

	// Parse request body
	var req requests.UpdateReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to parse update review request", zap.Error(err))
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid request format"})
		return
	}

	// Update the review
	review, err := h.reviewService.UpdateReview(c.Request.Context(), reviewID, &req, userIDStr)
	if err != nil {
		h.logger.Error("Failed to update review", zap.Error(err), zap.String("review_id", reviewID), zap.String("user_id", userIDStr))
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Review updated successfully",
		"data":    review,
	})
}

// DeleteReview handles DELETE /products/:id/reviews/:reviewId
// @Summary Delete a review
// @Description Delete a review (only by review owner or admin)
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param reviewId path string true "Review ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 403 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id}/reviews/{reviewId} [delete]
func (h *ReviewHandler) DeleteReview(c *gin.Context) {
	// Get user ID from authentication context
	userID, exists := c.Get("firebase_uid")
	if !exists {
		userID, exists = c.Get("user_id")
		if !exists {
			h.logger.Error("User ID not found in context")
			c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Authentication required"})
			return
		}
	}

	userIDStr, ok := userID.(string)
	if !ok {
		h.logger.Error("Invalid user ID type in context")
		c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Invalid authentication"})
		return
	}

	// Check if user is admin
	isAdmin := false
	if role, exists := c.Get("role"); exists {
		isAdmin = role == "admin"
	}

	// Get parameters
	productID := c.Param("id")      // Product ID comes from /products/:id/
	reviewID := c.Param("reviewId") // Review ID comes from /:reviewId
	if productID == "" || reviewID == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Product ID and Review ID are required"})
		return
	}

	// Create delete request
	req := requests.DeleteReviewRequest{
		ReviewID:  reviewID,
		ProductID: productID,
	}

	// Delete the review
	err := h.reviewService.DeleteReview(c.Request.Context(), &req, userIDStr, isAdmin)
	if err != nil {
		h.logger.Error("Failed to delete review", zap.Error(err), zap.String("review_id", reviewID), zap.String("user_id", userIDStr))
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Review deleted successfully",
	})
}

// LikeReview handles POST /products/:id/reviews/:reviewId/like
// @Summary Like or unlike a review
// @Description Like or unlike a review (authenticated users only)
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Param reviewId path string true "Review ID"
// @Param action body map[string]string true "Action: like or unlike"
// @Success 200 {object} responses.ReviewActionResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id}/reviews/{reviewId}/like [post]
func (h *ReviewHandler) LikeReview(c *gin.Context) {
	// Get user ID from authentication context
	userID, exists := c.Get("firebase_uid")
	if !exists {
		userID, exists = c.Get("user_id")
		if !exists {
			h.logger.Error("User ID not found in context")
			c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Authentication required"})
			return
		}
	}

	userIDStr, ok := userID.(string)
	if !ok {
		h.logger.Error("Invalid user ID type in context")
		c.JSON(http.StatusUnauthorized, map[string]interface{}{"error": "Invalid authentication"})
		return
	}

	// Get parameters
	productID := c.Param("id")      // Product ID comes from /products/:id/
	reviewID := c.Param("reviewId") // Review ID comes from /:reviewId
	if productID == "" || reviewID == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Product ID and Review ID are required"})
		return
	}

	// Parse action from body
	var actionBody map[string]string
	if err := c.ShouldBindJSON(&actionBody); err != nil {
		h.logger.Error("Failed to parse like action request", zap.Error(err))
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Invalid request format"})
		return
	}

	action, exists := actionBody["action"]
	if !exists {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Action is required (like or unlike)"})
		return
	}

	// Create like request
	req := requests.LikeReviewRequest{
		ReviewID:  reviewID,
		ProductID: productID,
		Action:    action,
	}

	// Process like/unlike
	response, err := h.reviewService.LikeReview(c.Request.Context(), &req, userIDStr)
	if err != nil {
		h.logger.Error("Failed to process review like", zap.Error(err), zap.String("review_id", reviewID), zap.String("user_id", userIDStr))
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetReviewStats handles GET /products/:id/reviews/stats
// @Summary Get review statistics for a product
// @Description Get aggregated review statistics including average rating, rating breakdown, etc.
// @Tags reviews
// @Accept json
// @Produce json
// @Param id path string true "Product ID"
// @Success 200 {object} responses.ReviewStatsResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /products/{id}/reviews/stats [get]
func (h *ReviewHandler) GetReviewStats(c *gin.Context) {
	// Get product ID from URL parameter
	productID := c.Param("id") // Product ID comes from /products/:id/
	if productID == "" {
		c.JSON(http.StatusBadRequest, map[string]interface{}{"error": "Product ID is required"})
		return
	}

	// Get requesting user ID (optional)
	requestingUserID := ""
	if userID, exists := c.Get("firebase_uid"); exists {
		requestingUserID, _ = userID.(string)
	} else if userID, exists := c.Get("user_id"); exists {
		requestingUserID, _ = userID.(string)
	}

	// Create stats request
	req := requests.GetReviewStatsRequest{
		ProductID: productID,
	}

	// Get review statistics
	stats, err := h.reviewService.GetReviewStats(c.Request.Context(), &req, requestingUserID)
	if err != nil {
		h.logger.Error("Failed to get review stats", zap.Error(err), zap.String("product_id", productID))
		c.JSON(http.StatusInternalServerError, map[string]interface{}{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Review statistics retrieved successfully",
		"data":    stats,
	})
}
