package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"go.uber.org/zap"
)

// ReviewService handles all review-related business logic
type ReviewService struct {
	dbMgr          db.DBManager
	userService    *UserService
	profileService *ProfileService
	dynamoClient   *dynamodb.Client
	logger         *zap.Logger
	tableName      string
}

// NewReviewService creates a new review service
func NewReviewService(dbMgr db.DBManager, userService *UserService, profileService *ProfileService, logger *zap.Logger) *ReviewService {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	service := &ReviewService{
		dbMgr:          dbMgr,
		userService:    userService,
		profileService: profileService,
		logger:         logger,
		tableName:      "product-reviews",
	}

	// Initialize DynamoDB client
	if err := service.initDynamoDB(); err != nil {
		logger.Warn("Failed to initialize DynamoDB client", zap.Error(err))
	}

	return service
}

// initDynamoDB initializes the DynamoDB client using AWS SDK's default credential chain
func (s *ReviewService) initDynamoDB() error {
	region := os.Getenv("DB_DYNAMO_REGION")
	if region == "" {
		region = os.Getenv("AWS_REGION")
	}
	if region == "" {
		return errors.New("no AWS region specified")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
	)
	if err != nil {
		return fmt.Errorf("unable to load AWS config: %w", err)
	}

	s.dynamoClient = dynamodb.NewFromConfig(cfg)
	s.logger.Info("DynamoDB client initialized successfully", zap.String("region", region))
	return nil
}

// getUserDisplayName gets user's display name with proper fallback logic
func (s *ReviewService) getUserDisplayName(ctx context.Context, userID string) (string, error) {
	// First, try to get the name from Users table
	userResponse, err := s.userService.GetUserByID(ctx, userID)
	if err == nil && userResponse.Success {
		if userData, ok := userResponse.Data.(map[string]interface{}); ok {
			if userName, ok := userData["name"].(string); ok && strings.TrimSpace(userName) != "" {
				return userName, nil
			}
		}
	}

	// If Users table doesn't have a name, try Profile table
	if s.profileService != nil {
		profile, err := s.profileService.GetProfileByUserID(ctx, userID)
		if err == nil && profile != nil {
			// Construct name from FirstName and LastName
			firstName := strings.TrimSpace(profile.FirstName)
			lastName := strings.TrimSpace(profile.LastName)

			if firstName != "" && lastName != "" {
				return firstName + " " + lastName, nil
			} else if firstName != "" {
				return firstName, nil
			} else if lastName != "" {
				return lastName, nil
			}
		}
	}

	// If both fail, return an error asking user to set up their name
	return "", errors.New("please set up your display name first. Visit /api/v1/profiles to update your profile information")
}

// CreateReview creates a new product review
func (s *ReviewService) CreateReview(ctx context.Context, req *requests.CreateReviewRequest, userID string) (*responses.ReviewResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get user's display name with proper fallback logic
	userName, err := s.getUserDisplayName(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user display name: %w", err)
	}

	// Get user email for the review
	var userEmail string
	userResponse, err := s.userService.GetUserByID(ctx, userID)
	if err == nil && userResponse.Success {
		if userData, ok := userResponse.Data.(map[string]interface{}); ok {
			userEmail, _ = userData["email"].(string)
		}
	}

	// Check if user has already reviewed this product
	existingReview, err := s.GetReviewByUserAndProduct(ctx, userID, req.ProductID)
	if err == nil && existingReview != nil && existingReview.IsActive {
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
// DeleteReview handles review deletion with proper DynamoDB key lookup
func (s *ReviewService) DeleteReview(ctx context.Context, req *requests.DeleteReviewRequest, userID string, isAdmin bool) error {
	if err := req.Validate(); err != nil {
		s.logger.Error("DeleteReview validation failed", zap.Error(err), zap.String("review_id", req.ReviewID))
		return fmt.Errorf("validation failed: %w", err)
	}

	s.logger.Info("Attempting to delete review",
		zap.String("review_id", req.ReviewID),
		zap.String("product_id", req.ProductID),
		zap.String("user_id", userID),
		zap.Bool("is_admin", isAdmin))

	// Get the review by its ID (this will use scan to find it)
	review, err := s.getReviewByID(ctx, req.ReviewID)
	if err != nil {
		s.logger.Error("Failed to get review for deletion", zap.Error(err), zap.String("review_id", req.ReviewID))
		return fmt.Errorf("failed to get review: %w", err)
	}

	// Verify that the review belongs to the specified product
	if review.ProductID != req.ProductID {
		s.logger.Error("Product ID mismatch",
			zap.String("review_product_id", review.ProductID),
			zap.String("request_product_id", req.ProductID),
			zap.String("review_id", req.ReviewID))
		return errors.New("product ID mismatch")
	}

	// Check permissions - user can delete own review or admin can delete any
	if review.UserID != userID && !isAdmin {
		s.logger.Error("Unauthorized delete attempt",
			zap.String("review_user_id", review.UserID),
			zap.String("requesting_user_id", userID),
			zap.Bool("is_admin", isAdmin),
			zap.String("review_id", req.ReviewID))
		return errors.New("unauthorized: can only delete own reviews or admin access required")
	}

	// Soft delete by setting is_active = false
	review.IsActive = false
	review.UpdatedDate = time.Now()

	// Save updated review to DynamoDB
	if err := s.saveReviewToDynamoDB(ctx, review); err != nil {
		s.logger.Error("Failed to save deleted review to DynamoDB", zap.Error(err), zap.String("review_id", req.ReviewID))
		return fmt.Errorf("failed to delete review: %w", err)
	}

	// Update product statistics
	if err := s.updateProductReviewStats(ctx, review.ProductID); err != nil {
		s.logger.Warn("Failed to update product review stats after deletion", zap.Error(err), zap.String("product_id", review.ProductID))
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
		s.logger.Error("LikeReview validation failed", zap.Error(err), zap.String("review_id", req.ReviewID))
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	s.logger.Info("Attempting to like/unlike review",
		zap.String("review_id", req.ReviewID),
		zap.String("product_id", req.ProductID),
		zap.String("user_id", userID),
		zap.String("action", req.Action))

	// Get the review by its ID (this will use scan to find it)
	review, err := s.getReviewByID(ctx, req.ReviewID)
	if err != nil {
		s.logger.Error("Failed to get review for like/unlike", zap.Error(err), zap.String("review_id", req.ReviewID))
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	// Verify product ID matches
	if review.ProductID != req.ProductID {
		s.logger.Error("Product ID mismatch in like request",
			zap.String("review_product_id", review.ProductID),
			zap.String("request_product_id", req.ProductID),
			zap.String("review_id", req.ReviewID))
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

	s.logger.Info("Like/Unlike action processed",
		zap.String("review_id", req.ReviewID),
		zap.String("action", req.Action),
		zap.Bool("success", success),
		zap.Int("new_likes_count", review.LikesCount))

	// Save updated review to DynamoDB (even if no change, for consistency)
	if err := s.saveReviewToDynamoDB(ctx, review); err != nil {
		s.logger.Error("Failed to save review after like/unlike", zap.Error(err), zap.String("review_id", req.ReviewID))
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	s.logger.Info("Review like action completed",
		zap.String("review_id", req.ReviewID),
		zap.String("user_id", userID),
		zap.String("action", req.Action),
		zap.Bool("success", success))

	return &responses.ReviewActionResponse{
		Success:     success,
		Message:     message,
		ReviewID:    review.ID,
		LikesCount:  review.LikesCount,
		IsUserLiked: review.HasUserLiked(userID),
	}, nil
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
	if s.dynamoClient == nil {
		return nil, errors.New("DynamoDB not available")
	}

	// Log the review ID we're looking for
	s.logger.Info("Looking for review by ID", zap.String("review_id", reviewID))

	// Use scan with filter - we need this because we don't have a GSI on id field
	input := &dynamodb.ScanInput{
		TableName:        aws.String(s.tableName),
		FilterExpression: aws.String("ID = :review_id AND is_active = :is_active"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":review_id": &types.AttributeValueMemberS{Value: reviewID},
			":is_active": &types.AttributeValueMemberBOOL{Value: true},
		},
	}

	result, err := s.dynamoClient.Scan(ctx, input)
	if err != nil {
		s.logger.Error("Failed to scan for review", zap.Error(err), zap.String("review_id", reviewID))
		return nil, fmt.Errorf("failed to scan for review: %w", err)
	}

	s.logger.Info("Scan result", zap.Int32("item_count", result.Count), zap.String("review_id", reviewID))

	if result.Count == 0 || len(result.Items) == 0 {
		s.logger.Warn("Review not found", zap.String("review_id", reviewID))
		return nil, errors.New("review not found")
	}

	review := &models.Review{}
	if err := attributevalue.UnmarshalMap(result.Items[0], review); err != nil {
		s.logger.Error("Failed to unmarshal review", zap.Error(err), zap.String("review_id", reviewID))
		return nil, fmt.Errorf("failed to unmarshal review: %w", err)
	}

	s.logger.Info("Successfully found review", zap.String("review_id", reviewID), zap.String("product_id", review.ProductID), zap.String("user_id", review.UserID))
	return review, nil
}

// GetReviewByUserAndProduct gets an active review by user and product (for checking duplicates)
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
	if err != nil {
		return nil, err
	}

	// Check if the review is active (not deleted)
	if !review.IsActive {
		return nil, errors.New("review not found")
	}

	return review, nil
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
