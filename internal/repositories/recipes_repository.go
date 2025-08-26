package repositories

import (
	"context"
	"fmt"
	"os"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

// RecipesRepository handles DynamoDB operations for recipes
type RecipesRepository struct {
	client    *dynamodb.Client
	tableName string
	logger    *zap.Logger
}

// NewRecipesRepository creates a new recipes repository
func NewRecipesRepository(logger *zap.Logger) (*RecipesRepository, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(getEnvVar("AWS_REGION", "us-east-1")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)
	tableName := getEnvVar("DYNAMODB_RECIPES_TABLE", "recipes")

	return &RecipesRepository{
		client:    client,
		tableName: tableName,
		logger:    logger,
	}, nil
}

// GetAllRecipes retrieves all recipes with optional filters
func (r *RecipesRepository) GetAllRecipes(ctx context.Context, page, pageSize int, category, search, ingredient string) ([]*models.Recipe, int64, error) {
	r.logger.Info("Getting all recipes",
		zap.Int("page", page),
		zap.Int("pageSize", pageSize),
		zap.String("category", category),
		zap.String("search", search),
		zap.String("ingredient", ingredient),
	)

	// Build scan input
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	}

	// Apply filters
	var filterExpressions []string
	var expressionAttributeNames map[string]string
	var expressionAttributeValues map[string]types.AttributeValue

	// Add category filter
	if category != "" {
		if expressionAttributeNames == nil {
			expressionAttributeNames = make(map[string]string)
		}
		if expressionAttributeValues == nil {
			expressionAttributeValues = make(map[string]types.AttributeValue)
		}
		expressionAttributeNames["#category"] = "category"
		expressionAttributeValues[":category"] = &types.AttributeValueMemberS{Value: category}
		filterExpressions = append(filterExpressions, "#category = :category")
	}

	// Add search filter (name contains)
	if search != "" {
		if expressionAttributeNames == nil {
			expressionAttributeNames = make(map[string]string)
		}
		if expressionAttributeValues == nil {
			expressionAttributeValues = make(map[string]types.AttributeValue)
		}
		expressionAttributeNames["#name"] = "name"
		expressionAttributeValues[":search"] = &types.AttributeValueMemberS{Value: search}
		filterExpressions = append(filterExpressions, "contains(#name, :search)")
	}

	// Add ingredient filter
	if ingredient != "" {
		if expressionAttributeNames == nil {
			expressionAttributeNames = make(map[string]string)
		}
		if expressionAttributeValues == nil {
			expressionAttributeValues = make(map[string]types.AttributeValue)
		}
		expressionAttributeNames["#ingredients"] = "ingredients"
		expressionAttributeValues[":ingredient"] = &types.AttributeValueMemberS{Value: ingredient}
		filterExpressions = append(filterExpressions, "contains(#ingredients, :ingredient)")
	}

	// Apply filters if any
	if len(filterExpressions) > 0 {
		scanInput.FilterExpression = aws.String(fmt.Sprintf("(%s)", joinStrings(filterExpressions, ") AND (")))
		if expressionAttributeNames != nil {
			scanInput.ExpressionAttributeNames = expressionAttributeNames
		}
		if expressionAttributeValues != nil {
			scanInput.ExpressionAttributeValues = expressionAttributeValues
		}
	}

	// Apply pagination
	if pageSize > 0 {
		scanInput.Limit = aws.Int32(int32(pageSize))
	}

	result, err := r.client.Scan(ctx, scanInput)
	if err != nil {
		r.logger.Error("failed to scan recipes", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to scan recipes: %w", err)
	}

	// Convert items to recipes
	var recipes []*models.Recipe
	for _, item := range result.Items {
		// Log raw item for debugging
		r.logger.Debug("raw DynamoDB item", zap.Any("item_keys", getMapKeys(item)))

		recipe := &models.Recipe{}
		if err := attributevalue.UnmarshalMap(item, recipe); err != nil {
			r.logger.Error("failed to unmarshal recipe",
				zap.Error(err),
				zap.Any("item_keys", getMapKeys(item)),
			)

			// Try to unmarshal to a generic map to see the actual structure
			var rawRecipe map[string]interface{}
			if unmarshalErr := attributevalue.UnmarshalMap(item, &rawRecipe); unmarshalErr == nil {
				r.logger.Error("raw recipe structure", zap.Any("raw_recipe", rawRecipe))
			}
			continue
		}

		// Fix the ID mapping issue - ensure recipe_id is populated
		if recipe.RecipeID == "" && recipe.BaseModel != nil {
			recipe.RecipeID = recipe.GetID()
		}

		// Handle null nutrition_facts and pro_tips
		if recipe.NutritionFacts == nil {
			recipe.NutritionFacts = make(map[string]string)
		}
		if recipe.ProTips == nil {
			recipe.ProTips = make([]string, 0)
		}

		recipes = append(recipes, recipe)
	}

	// Get total count (this is approximate in DynamoDB)
	totalCount := int64(len(recipes))
	if result.Count > 0 {
		totalCount = int64(result.Count)
	}

	r.logger.Info("successfully retrieved recipes",
		zap.Int("count", len(recipes)),
		zap.Int64("total", totalCount),
	)

	return recipes, totalCount, nil
}

// GetRecipeByID retrieves a recipe by ID
func (r *RecipesRepository) GetRecipeByID(ctx context.Context, id string) (*models.Recipe, error) {
	r.logger.Info("Getting recipe by ID", zap.String("recipee_id", id))

	// Use recipee_id as the primary key based on your DynamoDB structure
	key := map[string]types.AttributeValue{
		"recipee_id": &types.AttributeValueMemberS{Value: id},
	}

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key:       key,
	})

	if err != nil {
		r.logger.Error("failed to get recipe", zap.String("recipee_id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get recipe: %w", err)
	}

	if result.Item == nil {
		r.logger.Warn("recipe not found", zap.String("recipee_id", id))
		return nil, fmt.Errorf("recipe with ID %s not found", id)
	}

	recipe := &models.Recipe{}
	if err := attributevalue.UnmarshalMap(result.Item, recipe); err != nil {
		r.logger.Error("failed to unmarshal recipe", zap.String("recipee_id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal recipe: %w", err)
	}

	// Fix the ID mapping issue - ensure recipe_id is populated
	if recipe.RecipeID == "" && recipe.BaseModel != nil {
		recipe.RecipeID = recipe.GetID()
	}

	// Handle null nutrition_facts and pro_tips
	if recipe.NutritionFacts == nil {
		recipe.NutritionFacts = make(map[string]string)
	}
	if recipe.ProTips == nil {
		recipe.ProTips = make([]string, 0)
	}

	r.logger.Info("successfully retrieved recipe", zap.String("recipee_id", id))
	return recipe, nil
}

// CreateRecipe creates a new recipe
func (r *RecipesRepository) CreateRecipe(ctx context.Context, recipe *models.Recipe) error {
	r.logger.Info("creating recipe", zap.String("name", recipe.Name))

	// Ensure recipe_id is set from BaseModel ID
	if recipe.RecipeID == "" && recipe.BaseModel != nil {
		recipe.RecipeID = recipe.GetID()
	}

	// Ensure nutrition_facts and pro_tips are not null
	if recipe.NutritionFacts == nil {
		recipe.NutritionFacts = make(map[string]string)
	}
	if recipe.ProTips == nil {
		recipe.ProTips = make([]string, 0)
	}

	// Marshal recipe to DynamoDB item
	item, err := attributevalue.MarshalMap(recipe)
	if err != nil {
		r.logger.Error("failed to marshal recipe", zap.Error(err))
		return fmt.Errorf("failed to marshal recipe: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		r.logger.Error("failed to create recipe", zap.Error(err))
		return fmt.Errorf("failed to create recipe: %w", err)
	}

	r.logger.Info("successfully created recipe", zap.String("name", recipe.Name), zap.String("recipee_id", recipe.RecipeID))
	return nil
}

// UpdateRecipe updates an existing recipe
func (r *RecipesRepository) UpdateRecipe(ctx context.Context, recipe *models.Recipe) error {
	r.logger.Info("updating recipe", zap.String("recipee_id", recipe.RecipeID))

	// Ensure recipe_id is set from BaseModel ID
	if recipe.RecipeID == "" && recipe.BaseModel != nil {
		recipe.RecipeID = recipe.GetID()
	}

	// Ensure nutrition_facts and pro_tips are not null
	if recipe.NutritionFacts == nil {
		recipe.NutritionFacts = make(map[string]string)
	}
	if recipe.ProTips == nil {
		recipe.ProTips = make([]string, 0)
	}

	// Marshal recipe to DynamoDB item
	item, err := attributevalue.MarshalMap(recipe)
	if err != nil {
		r.logger.Error("failed to marshal recipe", zap.Error(err))
		return fmt.Errorf("failed to marshal recipe: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		r.logger.Error("failed to update recipe", zap.Error(err))
		return fmt.Errorf("failed to update recipe: %w", err)
	}

	r.logger.Info("successfully updated recipe", zap.String("recipee_id", recipe.RecipeID))
	return nil
}

// DeleteRecipe deletes a recipe
func (r *RecipesRepository) DeleteRecipe(ctx context.Context, id string) error {
	r.logger.Info("deleting recipe", zap.String("recipee_id", id))

	key := map[string]types.AttributeValue{
		"recipee_id": &types.AttributeValueMemberS{Value: id},
	}

	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key:       key,
	})
	if err != nil {
		r.logger.Error("failed to delete recipe", zap.String("recipee_id", id), zap.Error(err))
		return fmt.Errorf("failed to delete recipe: %w", err)
	}

	r.logger.Info("successfully deleted recipe", zap.String("recipee_id", id))
	return nil
}

// Helper function to get environment variables with defaults
func getEnvVar(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Helper function to join strings
func joinStrings(strs []string, separator string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return strs[0]
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += separator + strs[i]
	}
	return result
}

// Helper function to get map keys for debugging
func getMapKeys(m map[string]types.AttributeValue) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
