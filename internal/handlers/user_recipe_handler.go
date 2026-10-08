package handlers

import (
	"net/http"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// UserRecipeHandler handles user recipe-related HTTP requests
type UserRecipeHandler struct {
	repoManager *repositories.RepositoryManager
	r2Service   *services.R2Service
}

// NewUserRecipeHandler creates a new user recipe handler
func NewUserRecipeHandler(repoManager *repositories.RepositoryManager, r2Service *services.R2Service) *UserRecipeHandler {
	return &UserRecipeHandler{
		repoManager: repoManager,
		r2Service:   r2Service,
	}
}

// SubmitRecipeForReview handles POST /api/v1/recipes/submit
// @Summary Submit a recipe for review
// @Description Submit a recipe for admin approval
// @Tags User Recipes
// @Accept json
// @Produce json
// @Param request body models.Recipe true "Recipe to submit"
// @Success 201 {object} map[string]interface{} "Recipe submitted for review"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to submit recipe"
// @Router /recipes/submit [post]
func (h *UserRecipeHandler) SubmitRecipeForReview(c *gin.Context) {
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user ID from context (set by middleware)
	userID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	userIDStr := userID.(string)

	// Create new recipe with proper constructor
	newRecipe := models.NewRecipe(recipe.Name, recipe.Description, recipe.Ingredients, recipe.Instructions)

	// Set additional fields from the request
	newRecipe.Category = recipe.Category
	newRecipe.PrepTime = recipe.PrepTime
	newRecipe.Difficulty = recipe.Difficulty
	newRecipe.Servings = recipe.Servings
	newRecipe.Image = recipe.Image
	newRecipe.DemoDescription = recipe.DemoDescription
	newRecipe.NutritionFacts = recipe.NutritionFacts
	newRecipe.ProTips = recipe.ProTips

	// If demo_description is not provided, use description as default
	if newRecipe.DemoDescription == "" {
		newRecipe.DemoDescription = newRecipe.Description
	}

	// Create pending recipe review
	pendingReview := models.NewPendingRecipeReview(userIDStr, newRecipe)

	ctx := c.Request.Context()

	// Save to pending reviews table
	err := h.repoManager.PendingRecipeReviewRepository.CreatePendingReview(ctx, pendingReview)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to submit recipe for review",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Recipe submitted for review successfully",
		"review":  pendingReview,
	})
}

// GetUserSubmittedRecipes handles GET /api/v1/recipes/my-submissions
// @Summary Get user's submitted recipes
// @Description Get a list of recipes submitted by the current user
// @Tags User Recipes
// @Produce json
// @Success 200 {object} map[string]interface{} "User's submitted recipes"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to fetch submissions"
// @Router /recipes/my-submissions [get]
func (h *UserRecipeHandler) GetUserSubmittedRecipes(c *gin.Context) {
	// Get user ID from context (set by middleware)
	userID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	userIDStr := userID.(string)

	ctx := c.Request.Context()

	// Get all reviews for this user (we'll need to add this method to the repository)
	// For now, we'll get all pending reviews and filter by user ID
	reviews, _, err := h.repoManager.PendingRecipeReviewRepository.GetPendingReviews(ctx, 1, 100, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch user submissions",
			"details": err.Error(),
		})
		return
	}

	// Filter reviews by user ID
	var userReviews []*models.PendingRecipeReview
	for _, review := range reviews {
		if review.UserID == userIDStr {
			userReviews = append(userReviews, review)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"submissions": userReviews,
		"count":       len(userReviews),
	})
}

// UploadRecipeImage handles POST /api/v1/recipes/images/upload
// @Summary Upload recipe image
// @Description Upload an image for a recipe and get the R2 URL
// @Tags User Recipes
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Recipe image file"
// @Success 201 {object} map[string]interface{} "Image uploaded successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to upload image"
// @Router /recipes/images/upload [post]
func (h *UserRecipeHandler) UploadRecipeImage(c *gin.Context) {
	// Get user ID from context (set by middleware)
	userID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	userIDStr := userID.(string)

	// Get the uploaded file
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided"})
		return
	}
	defer file.Close()

	// Validate file type
	contentType := header.Header.Get("Content-Type")
	if contentType != "image/jpeg" && contentType != "image/png" && contentType != "image/webp" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type. Only JPEG, PNG, and WebP images are allowed"})
		return
	}

	// Validate file size (max 10MB)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size too large. Maximum size is 10MB"})
		return
	}

	ctx := c.Request.Context()

	// Check if R2 service is available
	if h.r2Service == nil || !h.r2Service.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "R2 service not available",
			"details": "R2 storage service is not configured",
		})
		return
	}

	// Upload to R2
	r2Key, err := h.r2Service.UploadRecipeImage(ctx, userIDStr, file, header)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to upload image",
			"details": err.Error(),
		})
		return
	}

	// Generate public URL for the uploaded image
	imageURL, err := h.r2Service.GetRecipeImageURL(ctx, r2Key, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to generate image URL",
			"details": err.Error(),
		})
		return
	}

	// Generate a unique image ID for tracking
	imageID := "img_" + time.Now().Format("20060102150405") + "_" + userIDStr[:8]

	c.JSON(http.StatusCreated, gin.H{
		"message":   "Image uploaded successfully",
		"image_id":  imageID,
		"image_url": imageURL,
		"s3_key":    r2Key, // Keep s3_key for backward compatibility
		"r2_key":    r2Key,
		"filename":  header.Filename,
		"size":      header.Size,
		"type":      contentType,
	})
}

// DeleteRecipeImage handles DELETE /api/v1/recipes/images/{s3_key}
// @Summary Delete recipe image
// @Description Delete an uploaded recipe image from R2
// @Tags User Recipes
// @Produce json
// @Param s3_key path string true "S3 key of the image to delete"
// @Success 200 {object} map[string]interface{} "Image deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Failed to delete image"
// @Router /recipes/images/{s3_key} [delete]
func (h *UserRecipeHandler) DeleteRecipeImage(c *gin.Context) {
	// Get user ID from context (set by middleware)
	userID, exists := c.Get("firebase_uid")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found"})
		return
	}

	r2Key := c.Param("s3_key") // Parameter name kept as s3_key for backward compatibility
	if r2Key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "R2 key is required"})
		return
	}

	// Basic security check - ensure the R2 key belongs to the user
	userIDStr := userID.(string)
	if !contains(r2Key, userIDStr) {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own images"})
		return
	}

	// Check if R2 service is available
	if h.r2Service == nil || !h.r2Service.IsConfigured() {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "R2 service not available",
			"details": "R2 storage service is not configured",
		})
		return
	}

	ctx := c.Request.Context()

	// Delete from R2
	err := h.r2Service.DeleteRecipeImage(ctx, r2Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete image",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Image deleted successfully",
		"s3_key":  r2Key, // Keep s3_key for backward compatibility
		"r2_key":  r2Key,
	})
}

// Helper function to check if a string contains another string
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) && (s[:len(substr)] == substr ||
			s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr))))
}

// Helper function to check substring
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
