package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"go.uber.org/zap"
)

// ReviewService handles all review-related business logic
type ReviewService struct {
	reviewRepo     *repositories.ReviewRepository
	userService    *UserService
	profileService *ProfileService
	logger         *zap.Logger
}

// NewReviewService creates a new review service
func NewReviewService(reviewRepo *repositories.ReviewRepository, userService *UserService, profileService *ProfileService, logger *zap.Logger) *ReviewService {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &ReviewService{
		reviewRepo:     reviewRepo,
		userService:    userService,
		profileService: profileService,
		logger:         logger,
	}
}

// getUserDisplayName gets user's display name with proper fallback logic
func (s *ReviewService) getUserDisplayName(ctx context.Context, userID string) (string, error) {
	s.logger.Info("Getting user display name", zap.String("user_id", userID))

	// First, try to get the name from Profile table (most up-to-date)
	if s.profileService != nil {
		profile, err := s.profileService.GetProfileByUserID(ctx, userID)
		if err == nil && profile != nil {
			// Construct name from FirstName and LastName
			firstName := strings.TrimSpace(profile.FirstName)
			lastName := strings.TrimSpace(profile.LastName)

			// Check if firstName is the default "User" value - treat it as empty
			if firstName == "User" && lastName == "" {
				// Default value, treat as empty and check Users table
				s.logger.Info("Profile has default 'User' name, checking Users table", zap.String("user_id", userID))
			} else {
				// Build name from profile: first_name + last_name (if both exist)
				// If only first_name exists, use just first_name
				// If only last_name exists, use just last_name
				if firstName != "" && lastName != "" {
					fullName := firstName + " " + lastName
					s.logger.Info("Found user name from Profile table",
						zap.String("user_id", userID),
						zap.String("user_name", fullName))
					return fullName, nil
				} else if firstName != "" && firstName != "User" {
					// Use first name only if it's not the default "User"
					s.logger.Info("Found first name from Profile table",
						zap.String("user_id", userID),
						zap.String("user_name", firstName))
					return firstName, nil
				} else if lastName != "" {
					// Use last name if first name is empty or default
					s.logger.Info("Found last name from Profile table",
						zap.String("user_id", userID),
						zap.String("user_name", lastName))
					return lastName, nil
				} else {
					s.logger.Info("No valid name found in Profile table", zap.String("user_id", userID))
				}
			}
		} else {
			s.logger.Warn("Failed to get profile from Profile table",
				zap.String("user_id", userID),
				zap.Error(err))
		}
	} else {
		s.logger.Warn("Profile service not available", zap.String("user_id", userID))
	}

	// If Profile table doesn't have a valid name, try Users table as fallback
	userResponse, err := s.userService.GetUserByID(ctx, userID)
	if err == nil && userResponse.Success {
		if userData, ok := userResponse.Data.(map[string]interface{}); ok {
			if userName, ok := userData["name"].(string); ok && strings.TrimSpace(userName) != "" {
				s.logger.Info("Found user name from Users table (fallback)",
					zap.String("user_id", userID),
					zap.String("user_name", userName))
				return userName, nil
			} else {
				s.logger.Info("No name found in Users table", zap.String("user_id", userID))
			}
		}
	} else {
		s.logger.Warn("Failed to get user from Users table",
			zap.String("user_id", userID),
			zap.Error(err))
	}

	// If both fail, return a fallback name instead of error
	// This allows reviews to be created even if user hasn't set up their profile
	s.logger.Info("Using fallback name 'User'", zap.String("user_id", userID))
	return "User", nil
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

	s.logger.Info("Using user name for review creation",
		zap.String("user_id", userID),
		zap.String("user_name", userName))

	// Get user email for the review
	var userEmail string
	userResponse, err := s.userService.GetUserByID(ctx, userID)
	if err == nil && userResponse.Success {
		if userData, ok := userResponse.Data.(map[string]interface{}); ok {
			userEmail, _ = userData["email"].(string)
		}
	}

	// Check if user has already reviewed this product
	existingReview, err := s.reviewRepo.GetReviewByUserAndProduct(ctx, userID, req.ProductID)
	if err == nil && existingReview != nil && existingReview.IsActive {
		s.logger.Warn("User already has an active review for this product",
			zap.String("user_id", userID),
			zap.String("product_id", req.ProductID),
			zap.String("existing_review_id", existingReview.GetID()))
		return nil, errors.New("user has already reviewed this product")
	} else if err != nil {
		// This is expected when no review exists or review is inactive (deleted)
		s.logger.Info("No active review found for user and product (user can create new review)",
			zap.String("user_id", userID),
			zap.String("product_id", req.ProductID),
			zap.Error(err))
	} else if existingReview != nil && !existingReview.IsActive {
		s.logger.Info("Found inactive (deleted) review - user can create new review",
			zap.String("user_id", userID),
			zap.String("product_id", req.ProductID),
			zap.String("deleted_review_id", existingReview.GetID()))
	}

	// TODO: Check if user has purchased this product (for verification)
	isVerified := false // For now, set to false. Later integrate with order service

	// Create the review
	review := models.NewReview(req.ProductID, userID, userName, userEmail, req.Title, req.Comment, req.Rating, isVerified)

	// Save to PostgreSQL
	if err := s.reviewRepo.CreateReview(ctx, review); err != nil {
		s.logger.Error("Failed to save review to PostgreSQL", zap.Error(err))
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

	// TODO: Implement proper admin check - for now, assume non-admin
	isAdmin := false

	// Get reviews from repository (repository handles pagination, sorting, and filtering)
	reviews, totalCount, err := s.reviewRepo.GetProductReviews(ctx, req.ProductID, req.Page, req.Limit, req.MinRating, req.MaxRating, req.SortBy, req.Order, isAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to get product reviews: %w", err)
	}

	// Refresh user names from PostgreSQL for all reviews
	if err := s.refreshUserNamesFromPostgres(ctx, reviews); err != nil {
		s.logger.Warn("Failed to refresh user names from PostgreSQL", zap.Error(err))
		// Continue without failing - user names might show as stored in DB
	}

	return responses.NewPaginatedReviewsResponse(reviews, int(totalCount), req.Page, req.Limit, requestingUserID, isAdmin), nil
}

// GetUserReviews retrieves all reviews by a specific user
func (s *ReviewService) GetUserReviews(ctx context.Context, req *requests.GetUserReviewsRequest, requestingUserID string) (*responses.PaginatedReviewsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// TODO: Implement proper admin check - for now, assume non-admin
	isAdmin := false

	// Get reviews from repository (repository handles pagination, sorting, and filtering)
	reviews, totalCount, err := s.reviewRepo.GetUserReviews(ctx, req.UserID, req.Page, req.Limit, req.SortBy, req.Order, isAdmin)
	if err != nil {
		return nil, fmt.Errorf("failed to get user reviews: %w", err)
	}

	return responses.NewPaginatedReviewsResponse(reviews, int(totalCount), req.Page, req.Limit, requestingUserID, isAdmin), nil
}

// UpdateReview updates an existing review
func (s *ReviewService) UpdateReview(ctx context.Context, reviewID string, req *requests.UpdateReviewRequest, userID string) (*responses.ReviewResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get the existing review first
	review, err := s.reviewRepo.GetReviewByID(ctx, reviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	// Check if user owns this review
	if review.UserID != userID {
		return nil, errors.New("unauthorized: can only update own reviews")
	}

	// Update the review
	review.Update(req.Title, req.Comment, req.Rating)

	// Save updated review to PostgreSQL
	if err := s.reviewRepo.UpdateReview(ctx, review); err != nil {
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

	// Get the review by its ID
	review, err := s.reviewRepo.GetReviewByID(ctx, req.ReviewID)
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

	// Soft delete by setting is_active = false (repository handles this)
	if err := s.reviewRepo.DeleteReview(ctx, req.ReviewID); err != nil {
		s.logger.Error("Failed to delete review", zap.Error(err), zap.String("review_id", req.ReviewID))
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

	// Get the review by its ID
	review, err := s.reviewRepo.GetReviewByID(ctx, req.ReviewID)
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

	// Save updated review to PostgreSQL (even if no change, for consistency)
	if err := s.reviewRepo.UpdateReview(ctx, review); err != nil {
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

	// Fetch users from PostgreSQL using userService and profileService
	userMap := make(map[string]string)
	for _, userID := range userIDs {
		// First try to get name from Profile table (most up-to-date)
		if s.profileService != nil {
			profile, err := s.profileService.GetProfileByUserID(ctx, userID)
			if err == nil && profile != nil {
				// Construct name from FirstName and LastName
				firstName := strings.TrimSpace(profile.FirstName)
				lastName := strings.TrimSpace(profile.LastName)

				if firstName != "" && lastName != "" {
					userMap[userID] = firstName + " " + lastName
					continue // Found name, move to next user
				} else if firstName != "" {
					userMap[userID] = firstName
					continue // Found name, move to next user
				} else if lastName != "" {
					userMap[userID] = lastName
					continue // Found name, move to next user
				}
			}
		}

		// If Profile table doesn't have a name, try Users table as fallback
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
			// Keep existing name or set better default if empty
			if review.UserName == "" || review.UserName == "Anonymous User" {
				review.UserName = "User"
			}
		}
	}

	s.logger.Debug("Refreshed user names from PostgreSQL",
		zap.Int("total_reviews", len(reviews)),
		zap.Int("users_found", len(userMap)))

	return nil
}

// AdminListReviews retrieves all reviews with admin-level filtering and access
func (s *ReviewService) AdminListReviews(ctx context.Context, req *requests.AdminListReviewsRequest) (*responses.PaginatedReviewsResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Build repository params
	params := &repositories.AdminListReviewsParams{
		Page:      req.Page,
		PageSize:  req.Limit,
		IsActive:  req.IsActive,
		ProductID: req.ProductID,
		UserID:    req.UserID,
		MinRating: req.MinRating,
		MaxRating: req.MaxRating,
		IsFlagged: req.IsFlagged,
		StartDate: req.StartDate,
		EndDate:   req.EndDate,
		SortBy:    req.SortBy,
		Order:     req.Order,
	}

	// Get reviews from repository
	reviews, totalCount, err := s.reviewRepo.AdminListReviews(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get admin reviews: %w", err)
	}

	// Refresh user names from PostgreSQL
	if err := s.refreshUserNamesFromPostgres(ctx, reviews); err != nil {
		s.logger.Warn("Failed to refresh user names from PostgreSQL", zap.Error(err))
	}

	// Admin has full access to all review data
	return responses.NewPaginatedReviewsResponse(reviews, int(totalCount), req.Page, req.Limit, "", true), nil
}

// AdminManageReview handles admin actions on reviews (flag, unflag, approve, reject, force delete)
func (s *ReviewService) AdminManageReview(ctx context.Context, req *requests.AdminReviewRequest, adminUserID string) (*responses.ReviewResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	s.logger.Info("Admin managing review",
		zap.String("review_id", req.ReviewID),
		zap.String("action", req.Action),
		zap.String("admin_user_id", adminUserID),
		zap.String("reason", req.Reason))

	// Get the review
	review, err := s.reviewRepo.GetReviewByID(ctx, req.ReviewID)
	if err != nil {
		return nil, fmt.Errorf("failed to get review: %w", err)
	}

	// Apply admin action
	switch req.Action {
	case "flag":
		review.IsFlagged = true
		review.FlagReason = req.Reason
		review.FlaggedBy = adminUserID
		review.FlaggedAt = time.Now()
	case "unflag":
		review.IsFlagged = false
		review.FlagReason = ""
		review.FlaggedBy = ""
		review.FlaggedAt = time.Time{}
	case "approve":
		review.IsApproved = true
		review.ApprovedBy = adminUserID
		review.ApprovedAt = time.Now()
	case "reject":
		review.IsApproved = false
		review.RejectedBy = adminUserID
		review.RejectedAt = time.Now()
		review.RejectionReason = req.Reason
	case "force_delete":
		review.IsActive = false
		review.DeletedBy = adminUserID
		review.DeletedAt = time.Now()
		review.DeletionReason = req.Reason
	case "hard_delete":
		// For hard delete, we'll delete the review from DynamoDB entirely
		// We'll handle this after the switch statement
		// Just log the action for now
		s.logger.Info("Hard delete requested for review",
			zap.String("review_id", req.ReviewID),
			zap.String("admin_user_id", adminUserID),
			zap.String("reason", req.Reason))
	default:
		return nil, errors.New("invalid admin action")
	}

	// Handle hard delete differently - actually delete from PostgreSQL
	if req.Action == "hard_delete" {
		// Hard delete the review from PostgreSQL
		if err := s.reviewRepo.HardDeleteReview(ctx, req.ReviewID); err != nil {
			return nil, fmt.Errorf("failed to hard delete review: %w", err)
		}

		// Update product statistics after hard deletion
		if err := s.updateProductReviewStats(ctx, review.ProductID); err != nil {
			s.logger.Warn("Failed to update product review stats after hard deletion", zap.Error(err))
		}

		s.logger.Info("Admin hard delete completed",
			zap.String("review_id", req.ReviewID),
			zap.String("action", req.Action),
			zap.String("admin_user_id", adminUserID))

		// Return the deleted review info (since it's no longer in DB)
		return responses.NewReviewResponse(review, "", true), nil
	}

	// For all other actions, update the review
	review.UpdatedDate = time.Now()

	// Save updated review
	if err := s.reviewRepo.UpdateReview(ctx, review); err != nil {
		return nil, fmt.Errorf("failed to update review: %w", err)
	}

	// Update product statistics if review was soft deleted
	if req.Action == "force_delete" {
		if err := s.updateProductReviewStats(ctx, review.ProductID); err != nil {
			s.logger.Warn("Failed to update product review stats after admin deletion", zap.Error(err))
		}
	}

	s.logger.Info("Admin review action completed",
		zap.String("review_id", req.ReviewID),
		zap.String("action", req.Action),
		zap.String("admin_user_id", adminUserID))

	// Return the updated review
	return responses.NewReviewResponse(review, "", true), nil
}

// AdminGetFlaggedReviews retrieves all flagged reviews for admin review
func (s *ReviewService) AdminGetFlaggedReviews(ctx context.Context, page, limit int) (*responses.PaginatedReviewsResponse, error) {
	// Set defaults
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	// Get flagged reviews from repository
	reviews, totalCount, err := s.reviewRepo.GetFlaggedReviews(ctx, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get flagged reviews: %w", err)
	}

	// Refresh user names from PostgreSQL
	if err := s.refreshUserNamesFromPostgres(ctx, reviews); err != nil {
		s.logger.Warn("Failed to refresh user names from PostgreSQL", zap.Error(err))
	}

	return responses.NewPaginatedReviewsResponse(reviews, int(totalCount), page, limit, "", true), nil
}

// AdminBulkDeleteFlaggedReviews deletes all flagged reviews at once
func (s *ReviewService) AdminBulkDeleteFlaggedReviews(ctx context.Context, req *requests.AdminBulkDeleteFlaggedRequest, adminUserID string) (*responses.BulkDeleteResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	s.logger.Info("Admin bulk delete flagged reviews started",
		zap.String("action", req.Action),
		zap.String("admin_user_id", adminUserID),
		zap.String("reason", req.Reason))

	// First, get all flagged reviews
	flaggedReviews, err := s.reviewRepo.GetAllFlaggedReviews(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get flagged reviews: %w", err)
	}

	totalFound := len(flaggedReviews)
	if totalFound == 0 {
		s.logger.Info("No flagged reviews found for bulk deletion")
		return responses.NewBulkDeleteResponse(req.Action, 0, 0, 0, []string{}, []string{}, req.Reason), nil
	}

	s.logger.Info("Found flagged reviews for bulk deletion",
		zap.Int("total_found", totalFound),
		zap.String("action", req.Action))

	var deletedIDs []string
	var failedIDs []string
	var productStatsUpdates = make(map[string]bool) // Track which products need stats updates

	// Process each flagged review
	for _, review := range flaggedReviews {
		reviewID := review.GetID()

		if req.Action == "hard_delete" {
			// Hard delete - remove from database entirely
			if err := s.reviewRepo.HardDeleteReview(ctx, reviewID); err != nil {
				s.logger.Error("Failed to hard delete review in bulk operation",
					zap.String("review_id", reviewID),
					zap.Error(err))
				failedIDs = append(failedIDs, reviewID)
			} else {
				s.logger.Info("Successfully hard deleted review in bulk operation",
					zap.String("review_id", reviewID))
				deletedIDs = append(deletedIDs, reviewID)
				productStatsUpdates[review.ProductID] = true
			}
		} else if req.Action == "force_delete" {
			// Force delete - soft delete (mark as inactive)
			review.IsActive = false
			review.DeletedBy = adminUserID
			review.DeletedAt = time.Now()
			review.DeletionReason = req.Reason
			review.UpdatedDate = time.Now()

			if err := s.reviewRepo.UpdateReview(ctx, review); err != nil {
				s.logger.Error("Failed to force delete review in bulk operation",
					zap.String("review_id", reviewID),
					zap.Error(err))
				failedIDs = append(failedIDs, reviewID)
			} else {
				s.logger.Info("Successfully force deleted review in bulk operation",
					zap.String("review_id", reviewID))
				deletedIDs = append(deletedIDs, reviewID)
				productStatsUpdates[review.ProductID] = true
			}
		}
	}

	// Update product statistics for affected products
	for productID := range productStatsUpdates {
		if err := s.updateProductReviewStats(ctx, productID); err != nil {
			s.logger.Warn("Failed to update product review stats after bulk deletion",
				zap.String("product_id", productID),
				zap.Error(err))
		}
	}

	totalDeleted := len(deletedIDs)
	failedDeletes := len(failedIDs)

	s.logger.Info("Admin bulk delete flagged reviews completed",
		zap.String("action", req.Action),
		zap.String("admin_user_id", adminUserID),
		zap.Int("total_found", totalFound),
		zap.Int("total_deleted", totalDeleted),
		zap.Int("failed_deletes", failedDeletes))

	return responses.NewBulkDeleteResponse(req.Action, totalFound, totalDeleted, failedDeletes, deletedIDs, failedIDs, req.Reason), nil
}
