package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/database"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ReviewRepository provides Postgres-backed CRUD for reviews
type ReviewRepository struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewReviewRepository creates a new review repository backed by Neon/Postgres
func NewReviewRepository(logger *zap.Logger) *ReviewRepository {
	return &ReviewRepository{
		db:     database.GetDB(),
		logger: logger,
	}
}

// CreateReview persists a new review
func (r *ReviewRepository) CreateReview(ctx context.Context, review *models.Review) error {
	if err := r.db.WithContext(ctx).Create(review).Error; err != nil {
		return fmt.Errorf("failed to create review: %w", err)
	}
	return nil
}

// GetReviewByID fetches a review by its ID
func (r *ReviewRepository) GetReviewByID(ctx context.Context, reviewID string) (*models.Review, error) {
	var review models.Review
	err := r.db.WithContext(ctx).
		Where("id = ? AND is_active = ?", reviewID, true).
		First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("review not found")
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}
	return &review, nil
}

// GetReviewByUserAndProduct gets an active review by user and product (for checking duplicates)
func (r *ReviewRepository) GetReviewByUserAndProduct(ctx context.Context, userID, productID string) (*models.Review, error) {
	var review models.Review
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND product_id = ? AND is_active = ?", userID, productID, true).
		First(&review).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("review not found")
		}
		return nil, fmt.Errorf("failed to get review: %w", err)
	}
	return &review, nil
}

// UpdateReview updates an existing review
func (r *ReviewRepository) UpdateReview(ctx context.Context, review *models.Review) error {
	if err := r.db.WithContext(ctx).Save(review).Error; err != nil {
		return fmt.Errorf("failed to update review: %w", err)
	}
	return nil
}

// DeleteReview soft deletes a review by setting is_active = false
func (r *ReviewRepository) DeleteReview(ctx context.Context, reviewID string) error {
	result := r.db.WithContext(ctx).
		Model(&models.Review{}).
		Where("id = ?", reviewID).
		Updates(map[string]interface{}{
			"is_active":    false,
			"updated_date": time.Now(),
		})
	if result.Error != nil {
		return fmt.Errorf("failed to delete review: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("review not found")
	}
	return nil
}

// HardDeleteReview permanently deletes a review
func (r *ReviewRepository) HardDeleteReview(ctx context.Context, reviewID string) error {
	result := r.db.WithContext(ctx).
		Where("id = ?", reviewID).
		Delete(&models.Review{})
	if result.Error != nil {
		return fmt.Errorf("failed to hard delete review: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("review not found")
	}
	return nil
}

// GetProductReviews retrieves all active reviews for a specific product with pagination and filters
func (r *ReviewRepository) GetProductReviews(ctx context.Context, productID string, page, pageSize int, minRating, maxRating int, sortBy, order string, isAdmin bool) ([]*models.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&models.Review{}).
		Where("product_id = ? AND is_active = ?", productID, true)

	// Apply rating filters
	if minRating > 1 {
		query = query.Where("rating >= ?", minRating)
	}
	if maxRating < 5 {
		query = query.Where("rating <= ?", maxRating)
	}

	// For non-admin users, filter out flagged reviews
	if !isAdmin {
		query = query.Where("is_flagged = ?", false)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	// Apply sorting
	switch sortBy {
	case "rating":
		if order == "desc" {
			query = query.Order("rating DESC")
		} else {
			query = query.Order("rating ASC")
		}
	case "likes":
		if order == "desc" {
			query = query.Order("likes_count DESC")
		} else {
			query = query.Order("likes_count ASC")
		}
	case "date":
		fallthrough
	default:
		if order == "desc" {
			query = query.Order("review_date DESC")
		} else {
			query = query.Order("review_date ASC")
		}
	}

	// Apply pagination
	var reviews []*models.Review
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&reviews).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch reviews: %w", err)
	}

	return reviews, total, nil
}

// GetUserReviews retrieves all active reviews by a specific user with pagination
func (r *ReviewRepository) GetUserReviews(ctx context.Context, userID string, page, pageSize int, sortBy, order string, isAdmin bool) ([]*models.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&models.Review{}).
		Where("user_id = ? AND is_active = ?", userID, true)

	// For non-admin users, filter out flagged reviews
	if !isAdmin {
		query = query.Where("is_flagged = ?", false)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	// Apply sorting
	switch sortBy {
	case "rating":
		if order == "desc" {
			query = query.Order("rating DESC")
		} else {
			query = query.Order("rating ASC")
		}
	case "likes":
		if order == "desc" {
			query = query.Order("likes_count DESC")
		} else {
			query = query.Order("likes_count ASC")
		}
	case "date":
		fallthrough
	default:
		if order == "desc" {
			query = query.Order("review_date DESC")
		} else {
			query = query.Order("review_date ASC")
		}
	}

	// Apply pagination
	var reviews []*models.Review
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&reviews).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch reviews: %w", err)
	}

	return reviews, total, nil
}

// AdminListReviews retrieves all reviews with admin-level filtering
func (r *ReviewRepository) AdminListReviews(ctx context.Context, req *AdminListReviewsParams) ([]*models.Review, int64, error) {
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&models.Review{})

	// Apply filters
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	if req.ProductID != "" {
		query = query.Where("product_id = ?", req.ProductID)
	}

	if req.UserID != "" {
		query = query.Where("user_id = ?", req.UserID)
	}

	if req.MinRating > 1 {
		query = query.Where("rating >= ?", req.MinRating)
	}

	if req.MaxRating < 5 {
		query = query.Where("rating <= ?", req.MaxRating)
	}

	if req.IsFlagged != nil {
		query = query.Where("is_flagged = ?", *req.IsFlagged)
	}

	if !req.StartDate.IsZero() {
		query = query.Where("review_date >= ?", req.StartDate)
	}

	if !req.EndDate.IsZero() {
		query = query.Where("review_date <= ?", req.EndDate)
	}

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count reviews: %w", err)
	}

	// Apply sorting
	switch req.SortBy {
	case "rating":
		if req.Order == "desc" {
			query = query.Order("rating DESC")
		} else {
			query = query.Order("rating ASC")
		}
	case "likes":
		if req.Order == "desc" {
			query = query.Order("likes_count DESC")
		} else {
			query = query.Order("likes_count ASC")
		}
	case "date":
		fallthrough
	default:
		if req.Order == "desc" {
			query = query.Order("review_date DESC")
		} else {
			query = query.Order("review_date ASC")
		}
	}

	// Apply pagination
	var reviews []*models.Review
	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&reviews).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch reviews: %w", err)
	}

	return reviews, total, nil
}

// GetFlaggedReviews retrieves all flagged reviews
func (r *ReviewRepository) GetFlaggedReviews(ctx context.Context, page, pageSize int) ([]*models.Review, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	query := r.db.WithContext(ctx).Model(&models.Review{}).
		Where("is_flagged = ?", true)

	// Get total count
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count flagged reviews: %w", err)
	}

	// Sort by flag date (most recent first)
	query = query.Order("flagged_at DESC")

	// Apply pagination
	var reviews []*models.Review
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Find(&reviews).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch flagged reviews: %w", err)
	}

	return reviews, total, nil
}

// GetAllFlaggedReviews retrieves all flagged reviews without pagination (for bulk operations)
func (r *ReviewRepository) GetAllFlaggedReviews(ctx context.Context) ([]*models.Review, error) {
	var reviews []*models.Review
	if err := r.db.WithContext(ctx).
		Where("is_flagged = ?", true).
		Find(&reviews).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch flagged reviews: %w", err)
	}
	return reviews, nil
}

// AdminListReviewsParams holds parameters for admin review listing
type AdminListReviewsParams struct {
	Page      int
	PageSize  int
	IsActive  *bool
	ProductID string
	UserID    string
	MinRating int
	MaxRating int
	IsFlagged *bool
	StartDate time.Time
	EndDate   time.Time
	SortBy    string
	Order     string
}
