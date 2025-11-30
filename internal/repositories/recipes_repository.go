package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// RecipesRepository provides Postgres-backed CRUD for recipes
type RecipesRepository struct {
	pgManager *db.PostgresManager
	logger    *zap.Logger
}

// NewRecipesRepository creates a new recipes repository backed by Neon/Postgres
func NewRecipesRepository(pgManager *db.PostgresManager, logger *zap.Logger) *RecipesRepository {
	return &RecipesRepository{
		pgManager: pgManager,
		logger:    logger,
	}
}

// getDB returns the GORM DB instance from pgManager
func (r *RecipesRepository) getDB(ctx context.Context) (*gorm.DB, error) {
	return r.pgManager.GetDB(ctx, false)
}

// GetAllRecipes retrieves recipes with optional filters and pagination
func (r *RecipesRepository) GetAllRecipes(ctx context.Context, page, pageSize int, category, search, ingredient string) ([]*models.Recipe, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	db, err := r.getDB(ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get database connection: %w", err)
	}
	query := db.WithContext(ctx).Model(&models.Recipe{})

	if category != "" {
		query = query.Where("category = ?", category)
	}
	if search != "" {
		pattern := fmt.Sprintf("%%%s%%", search)
		query = query.Where("(name ILIKE ? OR description ILIKE ?)", pattern, pattern)
	}
	if ingredient != "" {
		query = query.Where("? = ANY(ingredients)", ingredient)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count recipes: %w", err)
	}

	var recipes []*models.Recipe
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&recipes).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch recipes: %w", err)
	}

	return recipes, total, nil
}

// GetRecipeByID fetches a recipe by its public recipe ID or internal ID
func (r *RecipesRepository) GetRecipeByID(ctx context.Context, id string) (*models.Recipe, error) {
	var recipe models.Recipe
	db, err := r.getDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	err = db.WithContext(ctx).
		Where("recipe_id = ? OR id = ?", id, id).
		First(&recipe).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}
	return &recipe, nil
}

// CreateRecipe persists a new recipe
func (r *RecipesRepository) CreateRecipe(ctx context.Context, recipe *models.Recipe) error {
	if recipe.RecipeID == "" && recipe.BaseModel != nil {
		recipe.RecipeID = recipe.GetID()
	}
	db, err := r.getDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := db.WithContext(ctx).Create(recipe).Error; err != nil {
		return fmt.Errorf("failed to create recipe: %w", err)
	}
	return nil
}

// UpdateRecipe updates an existing recipe
func (r *RecipesRepository) UpdateRecipe(ctx context.Context, recipe *models.Recipe) error {
	if recipe.RecipeID == "" && recipe.BaseModel != nil {
		recipe.RecipeID = recipe.GetID()
	}
	db, err := r.getDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := db.WithContext(ctx).Save(recipe).Error; err != nil {
		return fmt.Errorf("failed to update recipe: %w", err)
	}
	return nil
}

// DeleteRecipe hard deletes a recipe by ID
func (r *RecipesRepository) DeleteRecipe(ctx context.Context, id string) error {
	db, err := r.getDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := db.WithContext(ctx).
		Where("recipe_id = ? OR id = ?", id, id).
		Delete(&models.Recipe{}).Error; err != nil {
		return fmt.Errorf("failed to delete recipe: %w", err)
	}
	return nil
}
