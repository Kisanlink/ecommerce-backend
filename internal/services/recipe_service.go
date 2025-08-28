package services

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// RecipeService handles recipe business logic
type RecipeService struct {
	repoManager *repositories.RepositoryManager
	recipesRepo *repositories.RecipesRepository
}

// NewRecipeService creates a new recipe service
func NewRecipeService(repoManager *repositories.RepositoryManager, recipesRepo *repositories.RecipesRepository) *RecipeService {
	service := &RecipeService{
		repoManager: repoManager,
		recipesRepo: recipesRepo,
	}

	return service
}

// CreateRecipe creates a new recipe
func (s *RecipeService) CreateRecipe(ctx context.Context, req *requests.CreateRecipeRequest) (*base.BaseResponse, error) {
	// Create recipe model
	recipe := models.NewRecipe(req.Name, req.Description, req.Ingredients, req.Instructions)

	// Set additional fields
	recipe.Category = req.Category
	recipe.PrepTime = req.PrepTime
	recipe.Servings = req.Servings
	recipe.Difficulty = req.Difficulty
	recipe.Image = req.Image                     // Set the image field
	recipe.DemoDescription = req.DemoDescription // Set the demo_description field
	recipe.Rating = req.Rating
	recipe.Reviews = req.Reviews
	recipe.NutritionFacts = req.NutritionFacts
	recipe.ProTips = req.ProTips

	// If demo_description is not provided, use description as default
	if recipe.DemoDescription == "" {
		recipe.DemoDescription = recipe.Description
	}

	// Set the recipe_id from the base model ID
	recipe.RecipeID = recipe.GetID()

	// Set created by from request context
	recipe.SetCreatedBy(req.GetUserID())

	// Save to DynamoDB using the recipes repository
	if err := s.recipesRepo.CreateRecipe(ctx, recipe); err != nil {
		return base.NewErrorResponse("Failed to create recipe",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	return base.NewSuccessResponse("Recipe created successfully", recipe), nil
}

// GetRecipeByID retrieves a recipe by ID
func (s *RecipeService) GetRecipeByID(ctx context.Context, req *requests.GetRecipeRequest) (*base.BaseResponse, error) {
	recipe, err := s.recipesRepo.GetRecipeByID(ctx, req.ID)
	if err != nil {
		return base.NewErrorResponse("Recipe not found",
			base.NewNotFoundError("Recipe", req.ID)), nil
	}

	// Ensure recipe_id is set properly
	if recipe.RecipeID == "" {
		recipe.RecipeID = recipe.GetID()
	}

	return base.NewSuccessResponse("Recipe retrieved successfully", recipe), nil
}

// GetRecipes retrieves recipes with filtering and pagination
func (s *RecipeService) GetRecipes(ctx context.Context, req *requests.ListRecipesRequest) (*base.PaginatedResponse, error) {
	// Calculate offset for pagination
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PerPage
	if pageSize < 1 {
		pageSize = 10
	}

	// Get recipes from DynamoDB with filters
	recipes, totalCount, err := s.recipesRepo.GetAllRecipes(
		ctx,
		page,
		pageSize,
		req.Category,
		req.Search,
		req.Ingredient,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve recipes: %w", err)
	}

	// Ensure all recipes have proper recipe_id and handle null nutrition_facts and pro_tips
	for _, recipe := range recipes {
		if recipe.RecipeID == "" {
			recipe.RecipeID = recipe.GetID()
		}

		// Ensure nutrition_facts and pro_tips are not null
		if recipe.NutritionFacts == nil {
			recipe.NutritionFacts = make(map[string]string)
		}
		if recipe.ProTips == nil {
			recipe.ProTips = make([]string, 0)
		}
	}

	pagination := base.NewPaginationInfo(page, pageSize, int(totalCount))
	return base.NewPaginatedResponse("Recipes retrieved successfully", recipes, pagination), nil
}

// UpdateRecipe updates an existing recipe
func (s *RecipeService) UpdateRecipe(ctx context.Context, req *requests.UpdateRecipeRequest) (*base.BaseResponse, error) {
	// Get existing recipe
	existing, err := s.recipesRepo.GetRecipeByID(ctx, req.ID)
	if err != nil {
		return base.NewErrorResponse("Recipe not found",
			base.NewNotFoundError("Recipe", req.ID)), nil
	}

	// Update fields
	existing.Name = req.Name
	existing.Description = req.Description
	existing.DemoDescription = req.DemoDescription // Set the demo_description field
	existing.Category = req.Category
	existing.PrepTime = req.PrepTime
	existing.Servings = req.Servings
	existing.Difficulty = req.Difficulty
	existing.Image = req.Image // Set the image field
	existing.Ingredients = req.Ingredients
	existing.Instructions = req.Instructions
	existing.Rating = req.Rating
	existing.Reviews = req.Reviews
	existing.NutritionFacts = req.NutritionFacts
	existing.ProTips = req.ProTips

	// If demo_description is not provided, use description as default
	if existing.DemoDescription == "" {
		existing.DemoDescription = existing.Description
	}

	// Ensure recipe_id is set
	if existing.RecipeID == "" {
		existing.RecipeID = existing.GetID()
	}

	existing.SetUpdatedBy(req.GetUserID())

	// Save changes
	if err := s.recipesRepo.UpdateRecipe(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update recipe: %w", err)
	}

	return base.NewSuccessResponse("Recipe updated successfully", existing), nil
}

// DeleteRecipe deletes a recipe
func (s *RecipeService) DeleteRecipe(ctx context.Context, req *requests.DeleteRecipeRequest) (*base.BaseResponse, error) {
	// Check if recipe exists
	_, err := s.recipesRepo.GetRecipeByID(ctx, req.ID)
	if err != nil {
		return base.NewErrorResponse("Recipe not found",
			base.NewNotFoundError("Recipe", req.ID)), nil
	}

	// Delete from DynamoDB
	if err := s.recipesRepo.DeleteRecipe(ctx, req.ID); err != nil {
		return nil, fmt.Errorf("failed to delete recipe: %w", err)
	}

	return base.NewSuccessResponse("Recipe deleted successfully", nil), nil
}
