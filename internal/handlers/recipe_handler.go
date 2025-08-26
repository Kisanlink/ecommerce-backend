package handlers

import (
	"net/http"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// RecipeHandler handles recipe-related HTTP requests
type RecipeHandler struct {
	repoManager *repositories.RepositoryManager
}

// NewRecipeHandler creates a new recipe handler
func NewRecipeHandler(repoManager *repositories.RepositoryManager) *RecipeHandler {
	return &RecipeHandler{
		repoManager: repoManager,
	}
}

func (h *RecipeHandler) GetRecipes(c *gin.Context) {
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

	// Check for search/filter parameters
	category := c.Query("category")
	search := c.Query("search")
	ingredient := c.Query("ingredient")

	ctx := c.Request.Context()

	// Get recipes using the new repository method
	recipes, totalCount, err := h.repoManager.RecipesRepository.GetAllRecipes(ctx, page, pageSize, category, search, ingredient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to fetch recipes",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"recipes": recipes,
		"pagination": gin.H{
			"page":        page,
			"page_size":   pageSize,
			"total":       totalCount,
			"total_pages": (totalCount + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

func (h *RecipeHandler) GetRecipeByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Recipe ID is required"})
		return
	}

	ctx := c.Request.Context()

	recipe, err := h.repoManager.RecipesRepository.GetRecipeByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Recipe not found",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recipe": recipe})
}

func (h *RecipeHandler) CreateRecipe(c *gin.Context) {
	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Create new recipe with proper constructor using the new model structure
	newRecipe := models.NewRecipe(recipe.Name, recipe.Description, recipe.Ingredients, recipe.Instructions)

	// Set additional fields from the request
	newRecipe.RecipeID = recipe.RecipeID
	newRecipe.Category = recipe.Category
	newRecipe.Rating = recipe.Rating
	newRecipe.Reviews = recipe.Reviews
	newRecipe.PrepTime = recipe.PrepTime
	newRecipe.Difficulty = recipe.Difficulty
	newRecipe.Servings = recipe.Servings
	newRecipe.NutritionFacts = recipe.NutritionFacts
	newRecipe.ProTips = recipe.ProTips

	ctx := c.Request.Context()

	// Save to DynamoDB
	err := h.repoManager.RecipesRepository.CreateRecipe(ctx, newRecipe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to create recipe",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"recipe": newRecipe})
}

func (h *RecipeHandler) UpdateRecipe(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Recipe ID is required"})
		return
	}

	var recipe models.Recipe
	if err := c.ShouldBindJSON(&recipe); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set the RecipeID from the URL parameter
	recipe.RecipeID = id

	ctx := c.Request.Context()

	// Update in DynamoDB
	err := h.repoManager.RecipesRepository.UpdateRecipe(ctx, &recipe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update recipe",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recipe": recipe})
}

func (h *RecipeHandler) DeleteRecipe(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Recipe ID is required"})
		return
	}

	ctx := c.Request.Context()

	// Soft delete in DynamoDB
	err := h.repoManager.RecipesRepository.DeleteRecipe(ctx, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to delete recipe",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recipe deleted successfully"})
}
