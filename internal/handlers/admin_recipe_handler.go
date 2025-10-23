package handlers

import (
	"net/http"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// AdminRecipeHandler handles admin recipe-related HTTP requests
type AdminRecipeHandler struct {
	repoManager *repositories.RepositoryManager
}

// NewAdminRecipeHandler creates a new admin recipe handler
func NewAdminRecipeHandler(repoManager *repositories.RepositoryManager) *AdminRecipeHandler {
	return &AdminRecipeHandler{
		repoManager: repoManager,
	}
}

// GetPendingRecipeReviews handles GET /api/v1/admin/recipes/pending
// @Summary Get pending recipe reviews
// @Description Get a paginated list of pending recipe reviews for admin approval
// @Tags Admin Recipes
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(10)
// @Param status query string false "Filter by status (pending, approved, rejected)"
// @Success 200 {object} map[string]interface{} "Pending reviews retrieved"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /admin/recipes/pending [get]
func (h *AdminRecipeHandler) GetPendingRecipeReviews(c *gin.Context) {
	// Parse query parameters for pagination and filtering
	page := 1
	pageSize := 10

	if p := c.Query("page"); p != "" {
		if pageNum, err := strconv.Atoi(p); err == nil && pageNum > 0 {
			page = pageNum
		}
	}

	if ps := c.Query("page_size"); ps != "" {
		if size, err := strconv.Atoi(ps); err == nil && size > 0 && size <= 100 {
			pageSize = size
		}
	}

	// Check for status filter
	status := models.PendingRecipeReviewStatus(c.Query("status"))

	ctx := c.Request.Context()

	// Get pending reviews using the repository
	reviews, totalCount, err := h.repoManager.PendingRecipeReviewRepository.GetPendingReviews(ctx, page, pageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch pending reviews",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"reviews": reviews,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       totalCount,
			"total_pages": (totalCount + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetPendingRecipeReviewByID handles GET /api/v1/admin/recipes/pending/{id}
// @Summary Get pending recipe review by ID
// @Description Get a specific pending recipe review by ID
// @Tags Admin Recipes
// @Produce json
// @Param id path string true "Review ID"
// @Success 200 {object} map[string]interface{} "Pending review retrieved"
// @Failure 400 {object} map[string]interface{} "Review ID is required"
// @Failure 404 {object} map[string]interface{} "Review not found"
// @Router /admin/recipes/pending/{id} [get]
func (h *AdminRecipeHandler) GetPendingRecipeReviewByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Review ID is required"})
		return
	}

	ctx := c.Request.Context()

	review, err := h.repoManager.PendingRecipeReviewRepository.GetPendingReviewByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Review not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"review": review})
}

// ApproveRecipeReview handles POST /api/v1/admin/recipes/pending/{id}/approve
// @Summary Approve a recipe review
// @Description Approve a pending recipe review and add it to the main recipes table
// @Tags Admin Recipes
// @Produce json
// @Param id path string true "Review ID"
// @Success 200 {object} map[string]interface{} "Recipe approved and added"
// @Failure 400 {object} map[string]interface{} "Review ID is required"
// @Failure 404 {object} map[string]interface{} "Review not found"
// @Failure 500 {object} map[string]interface{} "Failed to approve recipe"
// @Router /admin/recipes/pending/{id}/approve [post]
func (h *AdminRecipeHandler) ApproveRecipeReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Review ID is required"})
		return
	}

	ctx := c.Request.Context()

	// Get the pending review
	review, err := h.repoManager.PendingRecipeReviewRepository.GetPendingReviewByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Review not found",
			"details": err.Error(),
		})
		return
	}

	// Check if already processed
	if review.Status != models.StatusPending {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Review has already been processed",
			"status": review.Status,
		})
		return
	}

	// Get admin user ID from context (set by middleware)
	adminUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin user ID not found"})
		return
	}

	adminID := adminUserID.(string)

	// Convert pending review to recipe
	recipe := review.ToRecipe()

	// Create the recipe in the main recipes table
	err = h.repoManager.RecipesRepository.CreateRecipe(ctx, recipe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create recipe",
			"details": err.Error(),
		})
		return
	}

	// Update the pending review status to approved
	review.Approve(adminID)
	err = h.repoManager.PendingRecipeReviewRepository.UpdatePendingReview(ctx, review)
	if err != nil {
		// Log error but don't fail the request since recipe was already created
		// In a production system, you might want to implement a rollback mechanism
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Recipe was created but failed to update review status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe approved and added successfully",
		"recipe":  recipe,
		"review":  review,
	})
}

// RejectRecipeReview handles POST /api/v1/admin/recipes/pending/{id}/reject
// @Summary Reject a recipe review
// @Description Reject a pending recipe review with a reason
// @Tags Admin Recipes
// @Accept json
// @Produce json
// @Param id path string true "Review ID"
// @Param request body map[string]string true "Rejection reason"
// @Success 200 {object} map[string]interface{} "Recipe rejected"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Review not found"
// @Failure 500 {object} map[string]interface{} "Failed to reject recipe"
// @Router /admin/recipes/pending/{id}/reject [post]
func (h *AdminRecipeHandler) RejectRecipeReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Review ID is required"})
		return
	}

	var request struct {
		Reason string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Get the pending review
	review, err := h.repoManager.PendingRecipeReviewRepository.GetPendingReviewByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Review not found",
			"details": err.Error(),
		})
		return
	}

	// Check if already processed
	if review.Status != models.StatusPending {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":  "Review has already been processed",
			"status": review.Status,
		})
		return
	}

	// Get admin user ID from context (set by middleware)
	adminUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Admin user ID not found"})
		return
	}

	adminID := adminUserID.(string)

	// Update the pending review status to rejected
	review.Reject(adminID, request.Reason)
	err = h.repoManager.PendingRecipeReviewRepository.UpdatePendingReview(ctx, review)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to reject recipe",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Recipe rejected successfully",
		"review":  review,
	})
}

// DeletePendingRecipeReview handles DELETE /api/v1/admin/recipes/pending/{id}
// @Summary Delete a pending recipe review
// @Description Permanently delete a pending recipe review
// @Tags Admin Recipes
// @Produce json
// @Param id path string true "Review ID"
// @Success 200 {object} map[string]interface{} "Review deleted"
// @Failure 400 {object} map[string]interface{} "Review ID is required"
// @Failure 500 {object} map[string]interface{} "Failed to delete review"
// @Router /admin/recipes/pending/{id} [delete]
func (h *AdminRecipeHandler) DeletePendingRecipeReview(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Review ID is required"})
		return
	}

	ctx := c.Request.Context()

	// Delete the pending review
	err := h.repoManager.PendingRecipeReviewRepository.DeletePendingReview(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete review",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Review deleted successfully"})
}
