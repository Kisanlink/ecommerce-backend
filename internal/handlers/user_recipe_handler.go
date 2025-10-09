package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// UserRecipeHandler handles user recipe-related HTTP requests
type UserRecipeHandler struct {
	repoManager *repositories.RepositoryManager
}

// NewUserRecipeHandler creates a new user recipe handler
func NewUserRecipeHandler(repoManager *repositories.RepositoryManager) *UserRecipeHandler {
	return &UserRecipeHandler{
		repoManager: repoManager,
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
