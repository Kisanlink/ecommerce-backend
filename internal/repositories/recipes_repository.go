package repositories

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
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

		recipe, err := r.dynamoItemToRecipe(item)
		if err != nil {
			r.logger.Error("failed to convert DynamoDB item to recipe",
				zap.Error(err),
				zap.Any("item_keys", getMapKeys(item)),
			)
			continue
		}

		r.logger.Debug("converted recipe",
			zap.String("recipe_id", recipe.RecipeID),
			zap.String("name", recipe.Name),
			zap.Int("nutrition_facts_count", len(recipe.NutritionFacts)),
			zap.Int("pro_tips_count", len(recipe.ProTips)),
		)

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

	recipe, err := r.dynamoItemToRecipe(result.Item)
	if err != nil {
		r.logger.Error("failed to convert DynamoDB item to recipe", zap.String("recipee_id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to convert recipe: %w", err)
	}

	r.logger.Info("successfully retrieved recipe",
		zap.String("recipee_id", id),
		zap.String("recipe_name", recipe.Name),
		zap.Int("nutrition_facts_count", len(recipe.NutritionFacts)),
		zap.Int("pro_tips_count", len(recipe.ProTips)),
	)
	return recipe, nil
}

// CreateRecipe creates a new recipe
func (r *RecipesRepository) CreateRecipe(ctx context.Context, recipe *models.Recipe) error {
	r.logger.Info("creating recipe", zap.String("name", recipe.Name))

	// Ensure recipe_id is set from BaseModel ID
	if recipe.RecipeID == "" && recipe.BaseModel != nil {
		recipe.RecipeID = recipe.GetID()
	}

	// Convert recipe to DynamoDB item
	item, err := r.recipeToDBItem(recipe)
	if err != nil {
		r.logger.Error("failed to convert recipe to DB item", zap.Error(err))
		return fmt.Errorf("failed to convert recipe to DB item: %w", err)
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

	// Convert recipe to DynamoDB item
	item, err := r.recipeToDBItem(recipe)
	if err != nil {
		r.logger.Error("failed to convert recipe to DB item", zap.Error(err))
		return fmt.Errorf("failed to convert recipe to DB item: %w", err)
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

// Helper function to convert DynamoDB item to Recipe model
func (r *RecipesRepository) dynamoItemToRecipe(item map[string]types.AttributeValue) (*models.Recipe, error) {
	recipe := &models.Recipe{}

	// Create a base model with a temporary ID
	baseModel := &base.BaseModel{}

	// Extract recipe ID from DynamoDB (recipee_id field)
	if recipeIDAttr, ok := item["recipee_id"]; ok {
		if recipeIDStr, ok := recipeIDAttr.(*types.AttributeValueMemberS); ok {
			recipe.RecipeID = recipeIDStr.Value
			baseModel.SetID(recipeIDStr.Value) // Use the DynamoDB ID as BaseModel ID
		}
	}

	recipe.BaseModel = baseModel

	// Extract basic string fields
	if nameAttr, ok := item["name"]; ok {
		if nameStr, ok := nameAttr.(*types.AttributeValueMemberS); ok {
			recipe.Name = nameStr.Value
		}
	}

	if categoryAttr, ok := item["category"]; ok {
		if categoryStr, ok := categoryAttr.(*types.AttributeValueMemberS); ok {
			recipe.Category = categoryStr.Value
		}
	}

	if descAttr, ok := item["description"]; ok {
		if descStr, ok := descAttr.(*types.AttributeValueMemberS); ok {
			recipe.Description = descStr.Value
		}
	}

	if demoDescAttr, ok := item["demo_description"]; ok {
		if demoDescStr, ok := demoDescAttr.(*types.AttributeValueMemberS); ok {
			recipe.DemoDescription = demoDescStr.Value
		}
	}

	if prepTimeAttr, ok := item["prep_time"]; ok {
		if prepTimeStr, ok := prepTimeAttr.(*types.AttributeValueMemberS); ok {
			recipe.PrepTime = prepTimeStr.Value
		}
	}

	if difficultyAttr, ok := item["difficulty"]; ok {
		if difficultyStr, ok := difficultyAttr.(*types.AttributeValueMemberS); ok {
			recipe.Difficulty = difficultyStr.Value
		}
	}

	// Extract image URL
	if imageAttr, ok := item["image"]; ok {
		if imageStr, ok := imageAttr.(*types.AttributeValueMemberS); ok {
			recipe.Image = imageStr.Value
		}
	}

	// Extract numeric fields
	if ratingAttr, ok := item["rating"]; ok {
		if ratingNum, ok := ratingAttr.(*types.AttributeValueMemberN); ok {
			if rating, err := strconv.ParseFloat(ratingNum.Value, 64); err == nil {
				recipe.Rating = rating
			}
		}
	}

	if reviewsAttr, ok := item["reviews"]; ok {
		if reviewsNum, ok := reviewsAttr.(*types.AttributeValueMemberN); ok {
			if reviews, err := strconv.Atoi(reviewsNum.Value); err == nil {
				recipe.Reviews = reviews
			}
		}
	}

	if servingsAttr, ok := item["servings"]; ok {
		if servingsNum, ok := servingsAttr.(*types.AttributeValueMemberN); ok {
			if servings, err := strconv.Atoi(servingsNum.Value); err == nil {
				recipe.Servings = servings
			}
		}
	}

	// Extract list fields (ingredients, instructions, pro_tips)
	if ingredientsAttr, ok := item["ingredients"]; ok {
		if ingredientsList, ok := ingredientsAttr.(*types.AttributeValueMemberL); ok {
			recipe.Ingredients = make([]string, 0, len(ingredientsList.Value))
			for _, ingredient := range ingredientsList.Value {
				if ingredientStr, ok := ingredient.(*types.AttributeValueMemberS); ok {
					recipe.Ingredients = append(recipe.Ingredients, ingredientStr.Value)
				}
			}
		}
	}

	if instructionsAttr, ok := item["instructions"]; ok {
		if instructionsList, ok := instructionsAttr.(*types.AttributeValueMemberL); ok {
			recipe.Instructions = make([]string, 0, len(instructionsList.Value))
			for _, instruction := range instructionsList.Value {
				if instructionStr, ok := instruction.(*types.AttributeValueMemberS); ok {
					recipe.Instructions = append(recipe.Instructions, instructionStr.Value)
				}
			}
		}
	}

	if proTipsAttr, ok := item["pro_tips"]; ok {
		if proTipsList, ok := proTipsAttr.(*types.AttributeValueMemberL); ok {
			recipe.ProTips = make([]string, 0, len(proTipsList.Value))
			for _, proTip := range proTipsList.Value {
				if proTipStr, ok := proTip.(*types.AttributeValueMemberS); ok {
					recipe.ProTips = append(recipe.ProTips, proTipStr.Value)
				}
			}
		}
	} else {
		recipe.ProTips = make([]string, 0) // Initialize empty slice
	}

	// Extract nutrition_facts map
	if nutritionAttr, ok := item["nutrition_facts"]; ok {
		if nutritionMap, ok := nutritionAttr.(*types.AttributeValueMemberM); ok {
			recipe.NutritionFacts = make(map[string]string)
			for key, value := range nutritionMap.Value {
				if valueStr, ok := value.(*types.AttributeValueMemberS); ok {
					recipe.NutritionFacts[key] = valueStr.Value
				}
			}
		}
	} else {
		recipe.NutritionFacts = make(map[string]string) // Initialize empty map
	}

	return recipe, nil
}

// Helper function to convert Recipe model to DynamoDB item
func (r *RecipesRepository) recipeToDBItem(recipe *models.Recipe) (map[string]types.AttributeValue, error) {
	item := make(map[string]types.AttributeValue)

	// Set the primary key (recipee_id)
	if recipe.RecipeID != "" {
		item["recipee_id"] = &types.AttributeValueMemberS{Value: recipe.RecipeID}
	}

	// Set string fields
	if recipe.Name != "" {
		item["name"] = &types.AttributeValueMemberS{Value: recipe.Name}
	}
	if recipe.Category != "" {
		item["category"] = &types.AttributeValueMemberS{Value: recipe.Category}
	}
	if recipe.Description != "" {
		item["description"] = &types.AttributeValueMemberS{Value: recipe.Description}
	}
	if recipe.DemoDescription != "" {
		item["demo_description"] = &types.AttributeValueMemberS{Value: recipe.DemoDescription}
	}
	if recipe.PrepTime != "" {
		item["prep_time"] = &types.AttributeValueMemberS{Value: recipe.PrepTime}
	}
	if recipe.Difficulty != "" {
		item["difficulty"] = &types.AttributeValueMemberS{Value: recipe.Difficulty}
	}
	if recipe.Image != "" {
		item["image"] = &types.AttributeValueMemberS{Value: recipe.Image}
	}

	// Set numeric fields
	item["rating"] = &types.AttributeValueMemberN{Value: fmt.Sprintf("%.1f", recipe.Rating)}
	item["reviews"] = &types.AttributeValueMemberN{Value: strconv.Itoa(recipe.Reviews)}
	item["servings"] = &types.AttributeValueMemberN{Value: strconv.Itoa(recipe.Servings)}

	// Set list fields
	if len(recipe.Ingredients) > 0 {
		ingredientsList := make([]types.AttributeValue, 0, len(recipe.Ingredients))
		for _, ingredient := range recipe.Ingredients {
			ingredientsList = append(ingredientsList, &types.AttributeValueMemberS{Value: ingredient})
		}
		item["ingredients"] = &types.AttributeValueMemberL{Value: ingredientsList}
	}

	if len(recipe.Instructions) > 0 {
		instructionsList := make([]types.AttributeValue, 0, len(recipe.Instructions))
		for _, instruction := range recipe.Instructions {
			instructionsList = append(instructionsList, &types.AttributeValueMemberS{Value: instruction})
		}
		item["instructions"] = &types.AttributeValueMemberL{Value: instructionsList}
	}

	if len(recipe.ProTips) > 0 {
		proTipsList := make([]types.AttributeValue, 0, len(recipe.ProTips))
		for _, proTip := range recipe.ProTips {
			proTipsList = append(proTipsList, &types.AttributeValueMemberS{Value: proTip})
		}
		item["pro_tips"] = &types.AttributeValueMemberL{Value: proTipsList}
	}

	// Set nutrition_facts map
	if len(recipe.NutritionFacts) > 0 {
		nutritionMap := make(map[string]types.AttributeValue)
		for key, value := range recipe.NutritionFacts {
			nutritionMap[key] = &types.AttributeValueMemberS{Value: value}
		}
		item["nutrition_facts"] = &types.AttributeValueMemberM{Value: nutritionMap}
	}

	return item, nil
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
