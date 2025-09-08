package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

// ReviewService handles all review-related business logic
type ReviewService struct {
	dbMgr        db.DBManager
	userService  *UserService
	dynamoClient *dynamodb.Client
	logger       *zap.Logger
	tableName    string
}

// NewReviewService creates a new review service
func NewReviewService(dbMgr db.DBManager, userService *UserService, logger *zap.Logger) *ReviewService {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	service := &ReviewService{
		dbMgr:       dbMgr,
		userService: userService,
		logger:      logger,
		tableName:   "product-reviews",
	}

	// Initialize DynamoDB client
	if err := service.initDynamoDB(); err != nil {
		logger.Warn("Failed to initialize DynamoDB client", zap.Error(err))
	}

	return service
}

// initDynamoDB initializes the DynamoDB client with AWS credentials from environment
func (s *ReviewService) initDynamoDB() error {
	region := os.Getenv("DB_DYNAMO_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
		if region == "" {
			return errors.New("no AWS region specified")
		}
	}

	accessKey := os.Getenv("DB_DYNAMO_ACCESS_KEY_ID")
	if accessKey == "" {
		accessKey = os.Getenv("AWS_ACCESS_KEY_ID")
	}

	secretKey := os.Getenv("DB_DYNAMO_SECRET_ACCESS_KEY")
	if secretKey == "" {
		secretKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}

	if accessKey == "" || secretKey == "" {
		return errors.New("AWS credentials not found")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"",
		)),
	)
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	s.dynamoClient = dynamodb.NewFromConfig(cfg)
	s.logger.Info("DynamoDB client initialized successfully", zap.String("region", region))
	return nil
}

// CreateReview creates a new product review
func (s *ReviewService) CreateReview(ctx context.Context, req *requests.CreateReviewRequest, userID string) (*responses.ReviewResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get user information to cache in the review
	var userName, userEmail string

	userResponse, err := s.userService.GetUserByID(ctx, userID)
	if err != nil {
		s.logger.Warn("Failed to get user information, using defaults", zap.Error(err), zap.String("user_id", userID))
		userName = "Anonymous User"
		userEmail = ""
	} else if !userResponse.Success {
		s.logger.Warn("User service returned unsuccessful response, using defaults", zap.String("user_id", userID))
		userName = "Anonymous User"
		userEmail = ""
	} else {
		// Extract user data from the response
		if userData, ok := userResponse.Data.(map[string]interface{}); ok {
			userName, _ = userData["name"].(string)
			userEmail, _ = userData["email"].(string)
		}

		// Fallback if name is empty
		if userName == "" {
			userName = "Anonymous User"
		}
	}

	// Check if user has already reviewed this product
	existingReview, err := s.GetReviewByUserAndProduct(ctx, userID, req.ProductID)
	if err == nil && existingReview != nil {
		return nil, errors.New("user has already reviewed this product")
	}

	// TODO: Check if user has purchased this product (for verification)
	isVerified := false // For now, set to false. Later integrate with order service

	// Create the review
	review := models.NewReview(req.ProductID, userID, userName, userEmail, req.Title, req.Comment, req.Rating, isVerified)

	// Save to DynamoDB
	if err := s.saveReviewToDynamoDB(ctx, review); err != nil {
		s.logger.Error("Failed to save review to DynamoDB", zap.Error(err))
		return nil, fmt.Errorf("failed to save review: %w", err)
	}

	// Update product statistics
	if err := s.updateProductReviewStats(ctx, req.ProductID); err != nil {
		s.logger.Warn("Failed to update product review stats", zap.Error(err))
		// Don't fail the operation if stats update fails
	}

	s.logger.Info("Review created successfully",
		zap.String("review_id", review.GetID()),
		zap.String("product_id", req.ProductID),
		zap.String("user_id", userID))

	return responses.NewReviewResponse(review, userID, false), nil
}

// GetProductReviews retrieves all reviews for a specific product with pagination
func (s *ReviewService) GetProductReviews(ctx context.Context, req *requests.GetProductReviewsRequest, requestingUserID string) (*responses.PaginatedReviewsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if s.dynamoClient == nil {
		return nil, errors.New("DynamoDB not available")
	}

	var reviews []*models.Review
	var totalCount int

	// Query reviews for the product
	input := &dynamodb.QueryInput{
		TableName:              aws.String(s.tableName),
		KeyConditionExpression: aws.String("product_id = :product_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":product_id": &types.AttributeValueMemberS{Value: req.ProductID},
		},
	}

	// Add filter expressions for rating range
	if req.MinRating > 1 || req.MaxRating < 5 {
		filterExpressions := []string{}
		if req.MinRating > 1 {
			input.ExpressionAttributeValues[":min_rating"] = &types.AttributeValueMemberN{Value: strconv.Itoa(req.MinRating)}
			filterExpressions = append(filterExpressions, "rating >= :min_rating")
		}
		if req.MaxRating < 5 {
			input.ExpressionAttributeValues[":max_rating"] = &types.AttributeValueMemberN{Value: strconv.Itoa(req.MaxRating)}
			filterExpressions = append(filterExpressions, "rating <= :max_rating")
		}
		if len(filterExpressions) > 0 {
			input.FilterExpression = aws.String(filterExpressions[0])
			for i := 1; i < len(filterExpressions); i++ {
				*input.FilterExpression += " AND " + filterExpressions[i]
			}
		}
	}

	// Add active filter
	if input.FilterExpression == nil {
		input.FilterExpression = aws.String("is_active = :is_active")
	} else {
		*input.FilterExpression += " AND is_active = :is_active"
	}
	input.ExpressionAttributeValues[":is_active"] = &types.AttributeValueMemberBOOL{Value: true}

	result, err := s.dynamoClient.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query reviews: %w", err)
	}

	// Unmarshal all items first
	allReviews := make([]*models.Review, 0)
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &allReviews); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reviews: %w", err)
	}

	totalCount = len(allReviews)

	// Sort reviews based on request
	s.sortReviews(allReviews, req.SortBy, req.Order)

	// Apply pagination
	startIndex := (req.Page - 1) * req.Limit
	endIndex := startIndex + req.Limit

	if startIndex >= len(allReviews) {
		reviews = []*models.Review{}
	} else {
		if endIndex > len(allReviews) {
			endIndex = len(allReviews)
		}
		reviews = allReviews[startIndex:endIndex]
	}

	// Refresh user names from PostgreSQL for all reviews
	if err := s.refreshUserNamesFromPostgres(ctx, reviews); err != nil {
		s.logger.Warn("Failed to refresh user names from PostgreSQL", zap.Error(err))
		// Continue without failing - user names might show as stored in DynamoDB
	}

	// Check if requesting user is admin (simplified for now)
	isAdmin := false // TODO: Implement proper admin check

	return responses.NewPaginatedReviewsResponse(reviews, totalCount, req.Page, req.Limit, requestingUserID, isAdmin), nil
}

// GetUserReviews retrieves all reviews by a specific user
func (s *ReviewService) GetUserReviews(ctx context.Context, req *requests.GetUserReviewsRequest, requestingUserID string) (*responses.PaginatedReviewsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	if s.dynamoClient == nil {
		return nil, errors.New("DynamoDB not available")
	}

	var reviews []*models.Review
	var totalCount int

	// Use scan with filter for user reviews (since user_id is sort key)
	input := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("user_id = :user_id AND is_active = :is_active"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":user_id":   &types.AttributeValueMemberS{Value: req.UserID},
			":is_active": &types.AttributeValueMemberBOOL{Value: true},
		},
	}

	result, err := s.dynamoClient.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan user reviews: %w", err)
	}

	// Unmarshal all items
	allReviews := make([]*models.Review, 0)
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &allReviews); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reviews: %w", err)
	}

	totalCount = len(allReviews)

	// Sort reviews
	s.sortReviews(allReviews, req.SortBy, req.Order)

	// Apply pagination
	startIndex := (req.Page - 1) * req.Limit
	endIndex := startIndex + req.Limit

	if startIndex >= len(allReviews) {
		reviews = []*models.Review{}
	} else {
		if endIndex > len(allReviews) {
			endIndex = len(allReviews)
		}
		reviews = allReviews[startIndex:endIndex]
	}

	isAdmin := false // TODO: Implement proper admin check

	return responses.NewPaginatedReviewsResponse(reviews, totalCount, req.Page, req.Limit, requestingUserID, isAdmin), nil
}

// UpdateReview updates an existing review
func (s *ReviewService) UpdateReview(ctx context.Context, reviewID string, req *requests.UpdateReviewRequest, userID string) (*responses.ReviewResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get the existing review first
	review, err := s.getReviewByID(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	// Check if user owns this review
	if review.UserID != userID {
		return nil, errors.New("unauthorized: can only update own reviews")
	}

	// Update the review
	review.Update(req.Title, req.Comment, req.Rating)

	// Save updated review to DynamoDB
	if err := s.saveReviewToDynamoDB(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	// Update product statistics
	if err := s.updateProductReviewStats(ctx, review.ProductID); err != nil {
		s.logger.Warn("Failed to update product review stats", zap.Error(err))
	}

	s.logger.Info("Review updated successfully",
		zap.String("review_id", reviewID),
		zap.String("user_id", userID))

	return responses.NewReviewResponse(review, userID, false), nil
}

// DeleteReview deletes a review (soft delete by setting is_active = false)
func (s *ReviewService) DeleteReview(ctx context.Context, req *requests.DeleteReviewRequest, userID string, isAdmin bool) error {
	if err := req.Validate(); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Get the existing review first
	review, err := s.getReviewByID(ctx, req.ReviewID)
	if err != nil {
		return fmt.Errorf("failed to get review: %w", err)
	}

	// Check permissions
	if review.UserID != userID && !isAdmin {
		return errors.New("unauthorized: can only delete own reviews or admin access required")
	}

	// Soft delete by setting is_active = false
	review.IsActive = false
	review.UpdatedDate = time.Now()

	// Save updated review to DynamoDB
	if err := s.saveReviewToDynamoDB(ctx, review); err != nil {
		return fmt.Errorf("failed to delete review: %w", err)
	}

	// Update product statistics
	if err := s.updateProductReviewStats(ctx, review.ProductID); err != nil {
		s.logger.Warn("Failed to update product review stats", zap.Error(err))
	}

	s.logger.Info("Review deleted successfully",
		zap.String("review_id", req.ReviewID),
		zap.String("user_id", userID),
		zap.Bool("is_admin", isAdmin))

	return nil
}

// LikeReview handles like/unlike actions on a review
func (s *ReviewService) LikeReview(ctx context.Context, req *requests.LikeReviewRequest, userID string) (*responses.ReviewActionResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get the existing review
	review, err := s.getReviewByID(ctx, req.ReviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	// Verify product ID matches
	if review.ProductID != req.ProductID {
		return nil, errors.New("product ID mismatch")
	}

	var success bool
	var message string

	// Handle like/unlike action
	if req.Action == "like" {
		success = review.AddLike(userID)
		if success {
			message = "Review liked successfully"
		} else {
			message = "Review already liked"
		}
	} else { // unlike
		success = review.RemoveLike(userID)
		if success {
			message = "Review unliked successfully"
		} else {
			message = "Review was not liked"
		}
	}

	// Save updated review to DynamoDB (even if no change, for consistency)
	if err := s.saveReviewToDynamoDB(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	s.logger.Info("Review like action completed",
		zap.String("review_id", req.ReviewID),
		zap.String("user_id", userID),
		zap.String("action", req.Action),
		zap.Bool("success", success))

	return responses.NewReviewActionResponse(
		success,
		message,
		req.ReviewID,
		review.LikesCount,
		review.HasUserLiked(userID),
	), nil
}

// GetReviewStats gets aggregated review statistics for a product
func (s *ReviewService) GetReviewStats(ctx context.Context, req *requests.GetReviewStatsRequest, requestingUserID string) (*responses.ReviewStatsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get all reviews for the product
	getReviewsReq := &requests.GetProductReviewsRequest{
		ProductID: req.ProductID,
		Page:      1,
		Limit:     1000, // Get all reviews for stats
		SortBy:    "date",
		Order:     "desc",
	}

	reviewsResponse, err := s.GetProductReviews(ctx, getReviewsReq, requestingUserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get product reviews: %w", err)
	}

	// Convert review responses back to models for stats calculation
	reviews := make([]*models.Review, 0, len(reviewsResponse.Reviews))
	for _, reviewResponse := range reviewsResponse.Reviews {
		// Reconstruct basic review model for stats
		review := &models.Review{
			ProductID:  reviewResponse.ProductID,
			UserID:     reviewResponse.UserID,
			UserName:   reviewResponse.UserName,
			Rating:     reviewResponse.Rating,
			LikesCount: reviewResponse.LikesCount,
			IsVerified: reviewResponse.IsVerified,
			ReviewDate: reviewResponse.ReviewDate,
		}
		review.SetID(reviewResponse.ID)
		reviews = append(reviews, review)
	}

	isAdmin := false // TODO: Implement proper admin check

	return responses.NewReviewStatsResponse(req.ProductID, reviews, requestingUserID, isAdmin), nil
}

// Private helper methods

// saveReviewToDynamoDB saves a review to DynamoDB
func (s *ReviewService) saveReviewToDynamoDB(ctx context.Context, review *models.Review) error {
	if s.dynamoClient == nil {
		return errors.New("DynamoDB not available")
	}

	item, err := attributevalue.MarshalMap(review)
	if err != nil {
		return fmt.Errorf("failed to marshal review: %w", err)
	}

	_, err = s.dynamoClient.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(s.tableName),
		Item:      item,
	})
	return err
}

// getReviewByID retrieves a review by its ID
func (s *ReviewService) getReviewByID(ctx context.Context, reviewID string) (*models.Review, error) {
	// Since we need both product_id and user_id for DynamoDB key,
	// we'll need to implement a global secondary index or scan
	// For now, we'll use scan with filter (not optimal for production)

	if s.dynamoClient == nil {
		return nil, errors.New("DynamoDB not available")
	}

	input := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("id = :review_id"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":review_id": &types.AttributeValueMemberS{Value: reviewID},
		},
		Limit: aws.Int32(1),
	}

	result, err := s.dynamoClient.Scan(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for review: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, errors.New("review not found")
	}

	review := &models.Review{}
	err = attributevalue.UnmarshalMap(result.Items[0], review)
	return review, err
}

// GetReviewByUserAndProduct gets a review by user and product (for checking duplicates)
func (s *ReviewService) GetReviewByUserAndProduct(ctx context.Context, userID, productID string) (*models.Review, error) {
	if s.dynamoClient == nil {
		return nil, errors.New("DynamoDB not available")
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(s.tableName),
		Key: map[string]types.AttributeValue{
			"product_id": &types.AttributeValueMemberS{Value: productID},
			"user_id":    &types.AttributeValueMemberS{Value: userID},
		},
	}

	result, err := s.dynamoClient.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	if len(result.Item) == 0 {
		return nil, errors.New("review not found")
	}

	review := &models.Review{}
	err = attributevalue.UnmarshalMap(result.Item, review)
	return review, err
}

// sortReviews sorts a slice of reviews based on the given criteria
func (s *ReviewService) sortReviews(reviews []*models.Review, sortBy, order string) {
	sort.Slice(reviews, func(i, j int) bool {
		var result bool

		switch sortBy {
		case "rating":
			result = reviews[i].Rating < reviews[j].Rating
		case "likes":
			result = reviews[i].LikesCount < reviews[j].LikesCount
		case "date":
			fallthrough
		default:
			result = reviews[i].ReviewDate.Before(reviews[j].ReviewDate)
		}

		if order == "desc" {
			result = !result
		}

		return result
	})
}

// updateProductReviewStats updates the product's review statistics
func (s *ReviewService) updateProductReviewStats(ctx context.Context, productID string) error {
	// This would integrate with the product service to update review count and average rating
	// For now, we'll just log the update
	s.logger.Info("Updating product review stats", zap.String("product_id", productID))

	// TODO: Implement product statistics update
	// 1. Get all active reviews for the product
	// 2. Calculate new average rating and review count
	// 3. Update the product record in PostgreSQL

	return nil
}

// refreshUserNamesFromPostgres fetches user names directly from PostgreSQL and updates the reviews
func (s *ReviewService) refreshUserNamesFromPostgres(ctx context.Context, reviews []*models.Review) error {
	if len(reviews) == 0 {
		return nil
	}

	// Collect all unique user IDs
	userIDs := make([]string, 0)
	userIDSet := make(map[string]bool)
	for _, review := range reviews {
		if !userIDSet[review.UserID] {
			userIDs = append(userIDs, review.UserID)
			userIDSet[review.UserID] = true
		}
	}

	// Fetch users from PostgreSQL using userService
	userMap := make(map[string]string)
	for _, userID := range userIDs {
		userResponse, err := s.userService.GetUserByID(ctx, userID)
		if err == nil && userResponse.Success {
			// Extract user data from the response
			if userData, ok := userResponse.Data.(map[string]interface{}); ok {
				if userName, ok := userData["name"].(string); ok && userName != "" {
					userMap[userID] = userName
				}
			}
		}
	}

	// Update review user names
	for _, review := range reviews {
		if userName, exists := userMap[review.UserID]; exists {
			review.UserName = userName
		} else {
			// Keep existing name or set default if empty
			if review.UserName == "" || review.UserName == "Anonymous User" {
				review.UserName = "Anonymous User"
			}
		}
	}

	s.logger.Debug("Refreshed user names from PostgreSQL",
		zap.Int("total_reviews", len(reviews)),
		zap.Int("users_found", len(userMap)))

	return nil
}
