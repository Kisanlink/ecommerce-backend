package repositories

import (
	"context"
	"fmt"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

// PendingRecipeReviewRepository handles DynamoDB operations for pending recipe reviews
type PendingRecipeReviewRepository struct {
	client    *dynamodb.Client
	tableName string
	logger    *zap.Logger
}

// NewPendingRecipeReviewRepository creates a new pending recipe review repository
func NewPendingRecipeReviewRepository(logger *zap.Logger) (*PendingRecipeReviewRepository, error) {
	// Load AWS configuration
	cfg, err := config.LoadDefaultConfig(context.Background(),
		config.WithRegion(getEnvVar("AWS_REGION", "us-east-1")),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %w", err)
	}

	// Create DynamoDB client
	client := dynamodb.NewFromConfig(cfg)
	tableName := getEnvVar("DYNAMODB_PENDING_RECIPE_REVIEWS_TABLE", "pending_recipe_reviews")

	return &PendingRecipeReviewRepository{
		client:    client,
		tableName: tableName,
		logger:    logger,
	}, nil
}

// CreatePendingReview creates a new pending recipe review
func (r *PendingRecipeReviewRepository) CreatePendingReview(ctx context.Context, review *models.PendingRecipeReview) error {
	r.logger.Info("creating pending recipe review", zap.String("review_id", review.ReviewID))

	// Convert review to DynamoDB item
	item, err := r.pendingReviewToDBItem(review)
	if err != nil {
		r.logger.Error("failed to convert review to DB item", zap.Error(err))
		return fmt.Errorf("failed to convert review to DB item: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		r.logger.Error("failed to create pending review", zap.Error(err))
		return fmt.Errorf("failed to create pending review: %w", err)
	}

	r.logger.Info("successfully created pending review", zap.String("review_id", review.ReviewID))
	return nil
}

// GetPendingReviews retrieves all pending recipe reviews with optional filters
func (r *PendingRecipeReviewRepository) GetPendingReviews(ctx context.Context, page, pageSize int, status models.PendingRecipeReviewStatus) ([]*models.PendingRecipeReview, int64, error) {
	r.logger.Info("Getting pending recipe reviews",
		zap.Int("page", page),
		zap.Int("pageSize", pageSize),
		zap.String("status", string(status)),
	)

	// Build scan input
	scanInput := &dynamodb.ScanInput{
		TableName: aws.String(r.tableName),
	}

	// Apply status filter if provided
	if status != "" {
		scanInput.FilterExpression = aws.String("#status = :status")
		scanInput.ExpressionAttributeNames = map[string]string{
			"#status": "status",
		}
		scanInput.ExpressionAttributeValues = map[string]types.AttributeValue{
			":status": &types.AttributeValueMemberS{Value: string(status)},
		}
	}

	// Apply pagination
	if pageSize > 0 {
		scanInput.Limit = aws.Int32(int32(pageSize))
	}

	result, err := r.client.Scan(ctx, scanInput)
	if err != nil {
		r.logger.Error("failed to scan pending reviews", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to scan pending reviews: %w", err)
	}

	// Convert items to pending reviews
	var reviews []*models.PendingRecipeReview
	for _, item := range result.Items {
		review, err := r.dynamoItemToPendingReview(item)
		if err != nil {
			r.logger.Error("failed to convert DynamoDB item to pending review",
				zap.Error(err),
				zap.Any("item_keys", getMapKeys(item)),
			)
			continue
		}
		reviews = append(reviews, review)
	}

	// Get total count (this is approximate in DynamoDB)
	totalCount := int64(len(reviews))
	if result.Count > 0 {
		totalCount = int64(result.Count)
	}

	r.logger.Info("successfully retrieved pending reviews",
		zap.Int("count", len(reviews)),
		zap.Int64("total", totalCount),
	)

	return reviews, totalCount, nil
}

// GetPendingReviewByID retrieves a pending review by ID
func (r *PendingRecipeReviewRepository) GetPendingReviewByID(ctx context.Context, id string) (*models.PendingRecipeReview, error) {
	r.logger.Info("Getting pending review by ID", zap.String("review_id", id))

	key := map[string]types.AttributeValue{
		"review_id": &types.AttributeValueMemberS{Value: id},
	}

	result, err := r.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(r.tableName),
		Key:       key,
	})

	if err != nil {
		r.logger.Error("failed to get pending review", zap.String("review_id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to get pending review: %w", err)
	}

	if result.Item == nil {
		r.logger.Warn("pending review not found", zap.String("review_id", id))
		return nil, fmt.Errorf("pending review with ID %s not found", id)
	}

	review, err := r.dynamoItemToPendingReview(result.Item)
	if err != nil {
		r.logger.Error("failed to convert DynamoDB item to pending review", zap.String("review_id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to convert pending review: %w", err)
	}

	r.logger.Info("successfully retrieved pending review",
		zap.String("review_id", id),
		zap.String("status", string(review.Status)),
	)
	return review, nil
}

// UpdatePendingReview updates an existing pending review
func (r *PendingRecipeReviewRepository) UpdatePendingReview(ctx context.Context, review *models.PendingRecipeReview) error {
	r.logger.Info("updating pending review", zap.String("review_id", review.ReviewID))

	// Convert review to DynamoDB item
	item, err := r.pendingReviewToDBItem(review)
	if err != nil {
		r.logger.Error("failed to convert review to DB item", zap.Error(err))
		return fmt.Errorf("failed to convert review to DB item: %w", err)
	}

	_, err = r.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(r.tableName),
		Item:      item,
	})
	if err != nil {
		r.logger.Error("failed to update pending review", zap.Error(err))
		return fmt.Errorf("failed to update pending review: %w", err)
	}

	r.logger.Info("successfully updated pending review", zap.String("review_id", review.ReviewID))
	return nil
}

// DeletePendingReview deletes a pending review
func (r *PendingRecipeReviewRepository) DeletePendingReview(ctx context.Context, id string) error {
	r.logger.Info("deleting pending review", zap.String("review_id", id))

	key := map[string]types.AttributeValue{
		"review_id": &types.AttributeValueMemberS{Value: id},
	}

	_, err := r.client.DeleteItem(ctx, &dynamodb.DeleteItemInput{
		TableName: aws.String(r.tableName),
		Key:       key,
	})
	if err != nil {
		r.logger.Error("failed to delete pending review", zap.String("review_id", id), zap.Error(err))
		return fmt.Errorf("failed to delete pending review: %w", err)
	}

	r.logger.Info("successfully deleted pending review", zap.String("review_id", id))
	return nil
}

// Helper function to convert DynamoDB item to PendingRecipeReview model
func (r *PendingRecipeReviewRepository) dynamoItemToPendingReview(item map[string]types.AttributeValue) (*models.PendingRecipeReview, error) {
	review := &models.PendingRecipeReview{}

	// Create a base model with a temporary ID
	baseModel := &base.BaseModel{}

	// Extract review ID from DynamoDB
	if reviewIDAttr, ok := item["review_id"]; ok {
		if reviewIDStr, ok := reviewIDAttr.(*types.AttributeValueMemberS); ok {
			review.ReviewID = reviewIDStr.Value
			baseModel.SetID(reviewIDStr.Value)
		}
	}

	review.BaseModel = baseModel

	// Extract basic string fields
	if userIDAttr, ok := item["user_id"]; ok {
		if userIDStr, ok := userIDAttr.(*types.AttributeValueMemberS); ok {
			review.UserID = userIDStr.Value
		}
	}

	if statusAttr, ok := item["status"]; ok {
		if statusStr, ok := statusAttr.(*types.AttributeValueMemberS); ok {
			review.Status = models.PendingRecipeReviewStatus(statusStr.Value)
		}
	}

	if submittedAtAttr, ok := item["submitted_at"]; ok {
		if submittedAtStr, ok := submittedAtAttr.(*types.AttributeValueMemberS); ok {
			review.SubmittedAt = submittedAtStr.Value
		}
	}

	if reviewedAtAttr, ok := item["reviewed_at"]; ok {
		if reviewedAtStr, ok := reviewedAtAttr.(*types.AttributeValueMemberS); ok {
			review.ReviewedAt = reviewedAtStr.Value
		}
	}

	if reviewedByAttr, ok := item["reviewed_by"]; ok {
		if reviewedByStr, ok := reviewedByAttr.(*types.AttributeValueMemberS); ok {
			review.ReviewedBy = reviewedByStr.Value
		}
	}

	if rejectionReasonAttr, ok := item["rejection_reason"]; ok {
		if rejectionReasonStr, ok := rejectionReasonAttr.(*types.AttributeValueMemberS); ok {
			review.RejectionReason = rejectionReasonStr.Value
		}
	}

	// Extract recipe fields
	if nameAttr, ok := item["name"]; ok {
		if nameStr, ok := nameAttr.(*types.AttributeValueMemberS); ok {
			review.Name = nameStr.Value
		}
	}

	if categoryAttr, ok := item["category"]; ok {
		if categoryStr, ok := categoryAttr.(*types.AttributeValueMemberS); ok {
			review.Category = categoryStr.Value
		}
	}

	if descAttr, ok := item["description"]; ok {
		if descStr, ok := descAttr.(*types.AttributeValueMemberS); ok {
			review.Description = descStr.Value
		}
	}

	if demoDescAttr, ok := item["demo_description"]; ok {
		if demoDescStr, ok := demoDescAttr.(*types.AttributeValueMemberS); ok {
			review.DemoDescription = demoDescStr.Value
		}
	}

	if prepTimeAttr, ok := item["prep_time"]; ok {
		if prepTimeStr, ok := prepTimeAttr.(*types.AttributeValueMemberS); ok {
			review.PrepTime = prepTimeStr.Value
		}
	}

	if difficultyAttr, ok := item["difficulty"]; ok {
		if difficultyStr, ok := difficultyAttr.(*types.AttributeValueMemberS); ok {
			review.Difficulty = difficultyStr.Value
		}
	}

	if imageAttr, ok := item["image"]; ok {
		if imageStr, ok := imageAttr.(*types.AttributeValueMemberS); ok {
			review.Image = imageStr.Value
		}
	}

	// Extract numeric fields
	if servingsAttr, ok := item["servings"]; ok {
		if servingsNum, ok := servingsAttr.(*types.AttributeValueMemberN); ok {
			if servings, err := strconv.Atoi(servingsNum.Value); err == nil {
				review.Servings = servings
			}
		}
	}

	// Extract list fields (ingredients, instructions, pro_tips)
	if ingredientsAttr, ok := item["ingredients"]; ok {
		if ingredientsList, ok := ingredientsAttr.(*types.AttributeValueMemberL); ok {
			review.Ingredients = make([]string, 0, len(ingredientsList.Value))
			for _, ingredient := range ingredientsList.Value {
				if ingredientStr, ok := ingredient.(*types.AttributeValueMemberS); ok {
					review.Ingredients = append(review.Ingredients, ingredientStr.Value)
				}
			}
		}
	}

	if instructionsAttr, ok := item["instructions"]; ok {
		if instructionsList, ok := instructionsAttr.(*types.AttributeValueMemberL); ok {
			review.Instructions = make([]string, 0, len(instructionsList.Value))
			for _, instruction := range instructionsList.Value {
				if instructionStr, ok := instruction.(*types.AttributeValueMemberS); ok {
					review.Instructions = append(review.Instructions, instructionStr.Value)
				}
			}
		}
	}

	if proTipsAttr, ok := item["pro_tips"]; ok {
		if proTipsList, ok := proTipsAttr.(*types.AttributeValueMemberL); ok {
			review.ProTips = make([]string, 0, len(proTipsList.Value))
			for _, proTip := range proTipsList.Value {
				if proTipStr, ok := proTip.(*types.AttributeValueMemberS); ok {
					review.ProTips = append(review.ProTips, proTipStr.Value)
				}
			}
		}
	} else {
		review.ProTips = make([]string, 0)
	}

	// Extract nutrition_facts map
	if nutritionAttr, ok := item["nutrition_facts"]; ok {
		if nutritionMap, ok := nutritionAttr.(*types.AttributeValueMemberM); ok {
			review.NutritionFacts = make(map[string]string)
			for key, value := range nutritionMap.Value {
				if valueStr, ok := value.(*types.AttributeValueMemberS); ok {
					review.NutritionFacts[key] = valueStr.Value
				}
			}
		}
	} else {
		review.NutritionFacts = make(map[string]string)
	}

	return review, nil
}

// Helper function to convert PendingRecipeReview model to DynamoDB item
func (r *PendingRecipeReviewRepository) pendingReviewToDBItem(review *models.PendingRecipeReview) (map[string]types.AttributeValue, error) {
	item := make(map[string]types.AttributeValue)

	// Set the primary key (review_id)
	if review.ReviewID != "" {
		item["review_id"] = &types.AttributeValueMemberS{Value: review.ReviewID}
	}

	// Set string fields
	if review.UserID != "" {
		item["user_id"] = &types.AttributeValueMemberS{Value: review.UserID}
	}
	if review.Status != "" {
		item["status"] = &types.AttributeValueMemberS{Value: string(review.Status)}
	}
	if review.SubmittedAt != "" {
		item["submitted_at"] = &types.AttributeValueMemberS{Value: review.SubmittedAt}
	}
	if review.ReviewedAt != "" {
		item["reviewed_at"] = &types.AttributeValueMemberS{Value: review.ReviewedAt}
	}
	if review.ReviewedBy != "" {
		item["reviewed_by"] = &types.AttributeValueMemberS{Value: review.ReviewedBy}
	}
	if review.RejectionReason != "" {
		item["rejection_reason"] = &types.AttributeValueMemberS{Value: review.RejectionReason}
	}

	// Set recipe fields
	if review.Name != "" {
		item["name"] = &types.AttributeValueMemberS{Value: review.Name}
	}
	if review.Category != "" {
		item["category"] = &types.AttributeValueMemberS{Value: review.Category}
	}
	if review.Description != "" {
		item["description"] = &types.AttributeValueMemberS{Value: review.Description}
	}
	if review.DemoDescription != "" {
		item["demo_description"] = &types.AttributeValueMemberS{Value: review.DemoDescription}
	}
	if review.PrepTime != "" {
		item["prep_time"] = &types.AttributeValueMemberS{Value: review.PrepTime}
	}
	if review.Difficulty != "" {
		item["difficulty"] = &types.AttributeValueMemberS{Value: review.Difficulty}
	}
	if review.Image != "" {
		item["image"] = &types.AttributeValueMemberS{Value: review.Image}
	}

	// Set numeric fields
	item["servings"] = &types.AttributeValueMemberN{Value: strconv.Itoa(review.Servings)}

	// Set list fields
	if len(review.Ingredients) > 0 {
		ingredientsList := make([]types.AttributeValue, 0, len(review.Ingredients))
		for _, ingredient := range review.Ingredients {
			ingredientsList = append(ingredientsList, &types.AttributeValueMemberS{Value: ingredient})
		}
		item["ingredients"] = &types.AttributeValueMemberL{Value: ingredientsList}
	}

	if len(review.Instructions) > 0 {
		instructionsList := make([]types.AttributeValue, 0, len(review.Instructions))
		for _, instruction := range review.Instructions {
			instructionsList = append(instructionsList, &types.AttributeValueMemberS{Value: instruction})
		}
		item["instructions"] = &types.AttributeValueMemberL{Value: instructionsList}
	}

	if len(review.ProTips) > 0 {
		proTipsList := make([]types.AttributeValue, 0, len(review.ProTips))
		for _, proTip := range review.ProTips {
			proTipsList = append(proTipsList, &types.AttributeValueMemberS{Value: proTip})
		}
		item["pro_tips"] = &types.AttributeValueMemberL{Value: proTipsList}
	}

	// Set nutrition_facts map
	if len(review.NutritionFacts) > 0 {
		nutritionMap := make(map[string]types.AttributeValue)
		for key, value := range review.NutritionFacts {
			nutritionMap[key] = &types.AttributeValueMemberS{Value: value}
		}
		item["nutrition_facts"] = &types.AttributeValueMemberM{Value: nutritionMap}
	}

	return item, nil
}
