package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PendingRecipeReviewRepository provides Postgres-backed persistence for pending submissions
type PendingRecipeReviewRepository struct {
	pgManager *db.PostgresManager
	logger    *zap.Logger
}

// NewPendingRecipeReviewRepository creates the repository using pgManager
func NewPendingRecipeReviewRepository(pgManager *db.PostgresManager, logger *zap.Logger) *PendingRecipeReviewRepository {
	return &PendingRecipeReviewRepository{
		pgManager: pgManager,
		logger:    logger,
	}
}

// getDB returns the GORM DB instance from pgManager
func (r *PendingRecipeReviewRepository) getDB(ctx context.Context) (*gorm.DB, error) {
	return r.pgManager.GetDB(ctx, false)
}

// CreatePendingReview stores a new pending recipe submission
func (r *PendingRecipeReviewRepository) CreatePendingReview(ctx context.Context, review *models.PendingRecipeReview) error {
	if review.ReviewID == "" && review.BaseModel != nil {
		review.ReviewID = review.GetID()
	}
	db, err := r.getDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := db.WithContext(ctx).Create(review).Error; err != nil {
		return fmt.Errorf("failed to create pending review: %w", err)
	}
	return nil
}

// GetPendingReviews returns paginated pending reviews optionally filtered by status
func (r *PendingRecipeReviewRepository) GetPendingReviews(ctx context.Context, page, pageSize int, status models.PendingRecipeReviewStatus) ([]*models.PendingRecipeReview, int64, error) {
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
	query := db.WithContext(ctx).Model(&models.PendingRecipeReview{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count pending reviews: %w", err)
	}

	var reviews []*models.PendingRecipeReview
	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&reviews).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch pending reviews: %w", err)
	}

	return reviews, total, nil
}

// GetPendingReviewByID fetches a pending review by its review_id or internal ID
func (r *PendingRecipeReviewRepository) GetPendingReviewByID(ctx context.Context, id string) (*models.PendingRecipeReview, error) {
	var review models.PendingRecipeReview
	db, err := r.getDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := db.WithContext(ctx).
		Where("review_id = ? OR id = ?", id, id).
		First(&review).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending review: %w", err)
	}
	return &review, nil
}

// UpdatePendingReview persists changes to a pending review
func (r *PendingRecipeReviewRepository) UpdatePendingReview(ctx context.Context, review *models.PendingRecipeReview) error {
	db, err := r.getDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := db.WithContext(ctx).Save(review).Error; err != nil {
		return fmt.Errorf("failed to update pending review: %w", err)
	}
	return nil
}

// DeletePendingReview removes a pending review permanently
func (r *PendingRecipeReviewRepository) DeletePendingReview(ctx context.Context, id string) error {
	db, err := r.getDB(ctx)
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}
	if err := db.WithContext(ctx).
		Where("review_id = ? OR id = ?", id, id).
		Delete(&models.PendingRecipeReview{}).Error; err != nil {
		return fmt.Errorf("failed to delete pending review: %w", err)
	}
	return nil
}
