package handlers

import (
	"net/http"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
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
	// Also check for category from URL parameter (for /category/:category route)
	if categoryParam := c.Param("category"); categoryParam != "" {
		category = categoryParam
	}
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

	// Convert to simplified response format
	simplifiedRecipes := make([]requests.RecipeListResponse, len(recipes))
	for i, recipe := range recipes {
		simplifiedRecipes[i] = requests.RecipeListResponse{
			ID:              recipe.ID, // Include the ID field
			Name:            recipe.Name,
			DemoDescription: recipe.DemoDescription,
			Reviews:         recipe.Reviews,
			Rating:          recipe.Rating,
			PrepTime:        recipe.PrepTime,
			Difficulty:      recipe.Difficulty,
			Image:           recipe.Image,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"recipes": simplifiedRecipes,
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
	newRecipe.Image = recipe.Image                     // Set the image field
	newRecipe.DemoDescription = recipe.DemoDescription // Set the demo_description field
	newRecipe.NutritionFacts = recipe.NutritionFacts
	newRecipe.ProTips = recipe.ProTips

	// If demo_description is not provided, use description as default
	if newRecipe.DemoDescription == "" {
		newRecipe.DemoDescription = newRecipe.Description
	}

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

	var updateRequest map[string]interface{}
	if err := c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()

	// First, get the existing recipe
	existingRecipe, err := h.repoManager.RecipesRepository.GetRecipeByID(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":   "Recipe not found",
			"details": err.Error(),
		})
		return
	}

	// Update only the provided fields, keeping unedited fields unchanged
	if name, ok := updateRequest["name"].(string); ok {
		existingRecipe.Name = name
	}
	if description, ok := updateRequest["description"].(string); ok {
		existingRecipe.Description = description
	}
	if demoDescription, ok := updateRequest["demo_description"].(string); ok {
		existingRecipe.DemoDescription = demoDescription
	}
	if category, ok := updateRequest["category"].(string); ok {
		existingRecipe.Category = category
	}
	if prepTime, ok := updateRequest["prep_time"].(string); ok {
		existingRecipe.PrepTime = prepTime
	}
	if difficulty, ok := updateRequest["difficulty"].(string); ok {
		existingRecipe.Difficulty = difficulty
	}
	if servings, ok := updateRequest["servings"].(float64); ok {
		existingRecipe.Servings = int(servings)
	}
	if image, ok := updateRequest["image"].(string); ok {
		existingRecipe.Image = image
	}
	if rating, ok := updateRequest["rating"].(float64); ok {
		existingRecipe.Rating = rating
	}
	if reviews, ok := updateRequest["reviews"].(float64); ok {
		existingRecipe.Reviews = int(reviews)
	}
	if ingredients, ok := updateRequest["ingredients"].([]interface{}); ok {
		stringIngredients := make([]string, len(ingredients))
		for i, ingredient := range ingredients {
			if str, ok := ingredient.(string); ok {
				stringIngredients[i] = str
			}
		}
		existingRecipe.Ingredients = stringIngredients
	}
	if instructions, ok := updateRequest["instructions"].([]interface{}); ok {
		stringInstructions := make([]string, len(instructions))
		for i, instruction := range instructions {
			if str, ok := instruction.(string); ok {
				stringInstructions[i] = str
			}
		}
		existingRecipe.Instructions = stringInstructions
	}
	if proTips, ok := updateRequest["pro_tips"].([]interface{}); ok {
		stringProTips := make([]string, len(proTips))
		for i, tip := range proTips {
			if str, ok := tip.(string); ok {
				stringProTips[i] = str
			}
		}
		existingRecipe.ProTips = stringProTips
	}
	if nutritionFacts, ok := updateRequest["nutrition_facts"].(map[string]interface{}); ok {
		stringNutritionFacts := make(map[string]string)
		for key, value := range nutritionFacts {
			if str, ok := value.(string); ok {
				stringNutritionFacts[key] = str
			}
		}
		existingRecipe.NutritionFacts = stringNutritionFacts
	}

	// Update in DynamoDB
	err = h.repoManager.RecipesRepository.UpdateRecipe(ctx, existingRecipe)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to update recipe",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recipe": existingRecipe})
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
