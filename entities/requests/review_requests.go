package requests

import (
	"errors"
	"time"
)

// CreateReviewRequest represents a request to create a new product review
type CreateReviewRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Rating    int    `json:"rating" validate:"required,min=1,max=5"`
	Title     string `json:"title" validate:"required,min=5,max=100"`
	Comment   string `json:"comment" validate:"required,min=10,max=1000"`
}

// Validate validates the create review request
func (req *CreateReviewRequest) Validate() error {
	if req.ProductID == "" {
		return errors.New("product_id is required")
	}
	if req.Rating < 1 || req.Rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	if len(req.Title) < 5 || len(req.Title) > 100 {
		return errors.New("title must be between 5 and 100 characters")
	}
	if len(req.Comment) < 10 || len(req.Comment) > 1000 {
		return errors.New("comment must be between 10 and 1000 characters")
	}
	return nil
}

// UpdateReviewRequest represents a request to update an existing review
type UpdateReviewRequest struct {
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Title   string `json:"title" validate:"required,min=5,max=100"`
	Comment string `json:"comment" validate:"required,min=10,max=1000"`
}

// Validate validates the update review request
func (req *UpdateReviewRequest) Validate() error {
	if req.Rating < 1 || req.Rating > 5 {
		return errors.New("rating must be between 1 and 5")
	}
	if len(req.Title) < 5 || len(req.Title) > 100 {
		return errors.New("title must be between 5 and 100 characters")
	}
	if len(req.Comment) < 10 || len(req.Comment) > 1000 {
		return errors.New("comment must be between 10 and 1000 characters")
	}
	return nil
}

// GetProductReviewsRequest represents a request to get reviews for a product
type GetProductReviewsRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Page      int    `json:"page,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	SortBy    string `json:"sort_by,omitempty"` // "date", "rating", "likes"
	Order     string `json:"order,omitempty"`   // "asc", "desc"
	MinRating int    `json:"min_rating,omitempty"`
	MaxRating int    `json:"max_rating,omitempty"`
}

// Validate validates the get product reviews request
func (req *GetProductReviewsRequest) Validate() error {
	if req.ProductID == "" {
		return errors.New("product_id is required")
	}
	if req.Page < 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100 // Prevent excessive results
	}
	if req.SortBy == "" {
		req.SortBy = "date"
	}
	if req.Order == "" {
		req.Order = "desc"
	}
	if req.MinRating < 1 {
		req.MinRating = 1
	}
	if req.MaxRating > 5 || req.MaxRating < req.MinRating {
		req.MaxRating = 5
	}
	return nil
}

// GetUserReviewsRequest represents a request to get reviews by a user
type GetUserReviewsRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Page   int    `json:"page,omitempty"`
	Limit  int    `json:"limit,omitempty"`
	SortBy string `json:"sort_by,omitempty"` // "date", "rating", "likes"
	Order  string `json:"order,omitempty"`   // "asc", "desc"
}

// Validate validates the get user reviews request
func (req *GetUserReviewsRequest) Validate() error {
	if req.UserID == "" {
		return errors.New("user_id is required")
	}
	if req.Page < 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.SortBy == "" {
		req.SortBy = "date"
	}
	if req.Order == "" {
		req.Order = "desc"
	}
	return nil
}

// LikeReviewRequest represents a request to like/unlike a review
type LikeReviewRequest struct {
	ReviewID  string `json:"review_id" validate:"required"`
	ProductID string `json:"product_id" validate:"required"`
	Action    string `json:"action" validate:"required,oneof=like unlike"`
}

// Validate validates the like review request
func (req *LikeReviewRequest) Validate() error {
	if req.ReviewID == "" {
		return errors.New("review_id is required")
	}
	if req.ProductID == "" {
		return errors.New("product_id is required")
	}
	if req.Action != "like" && req.Action != "unlike" {
		return errors.New("action must be either 'like' or 'unlike'")
	}
	return nil
}

// DeleteReviewRequest represents a request to delete a review
type DeleteReviewRequest struct {
	ReviewID  string `json:"review_id" validate:"required"`
	ProductID string `json:"product_id" validate:"required"`
}

// Validate validates the delete review request
func (req *DeleteReviewRequest) Validate() error {
	if req.ReviewID == "" {
		return errors.New("review_id is required")
	}
	if req.ProductID == "" {
		return errors.New("product_id is required")
	}
	return nil
}

// GetReviewStatsRequest represents a request to get review statistics for a product
type GetReviewStatsRequest struct {
	ProductID string `json:"product_id" validate:"required"`
}

// Validate validates the get review stats request
func (req *GetReviewStatsRequest) Validate() error {
	if req.ProductID == "" {
		return errors.New("product_id is required")
	}
	return nil
}

// ReviewFilters represents filter options for review queries
type ReviewFilters struct {
	ProductID  string    `json:"product_id,omitempty"`
	UserID     string    `json:"user_id,omitempty"`
	MinRating  int       `json:"min_rating,omitempty"`
	MaxRating  int       `json:"max_rating,omitempty"`
	IsVerified *bool     `json:"is_verified,omitempty"`
	IsActive   *bool     `json:"is_active,omitempty"`
	StartDate  time.Time `json:"start_date,omitempty"`
	EndDate    time.Time `json:"end_date,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
	SortBy     string    `json:"sort_by,omitempty"`
	Order      string    `json:"order,omitempty"`
}

// ApplyDefaults applies default values to the filter
func (f *ReviewFilters) ApplyDefaults() {
	if f.Limit <= 0 {
		f.Limit = 10
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.SortBy == "" {
		f.SortBy = "review_date"
	}
	if f.Order == "" {
		f.Order = "desc"
	}
	if f.MinRating < 1 {
		f.MinRating = 1
	}
	if f.MaxRating > 5 || f.MaxRating < f.MinRating {
		f.MaxRating = 5
	}
}

// Validate validates the review filters
func (f *ReviewFilters) Validate() error {
	f.ApplyDefaults()

	if !f.EndDate.IsZero() && !f.StartDate.IsZero() && f.EndDate.Before(f.StartDate) {
		return errors.New("end_date must be after start_date")
	}

	validSortBy := []string{"review_date", "updated_date", "rating", "likes_count"}
	isValid := false
	for _, valid := range validSortBy {
		if f.SortBy == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("invalid sort_by value")
	}

	if f.Order != "asc" && f.Order != "desc" {
		return errors.New("order must be 'asc' or 'desc'")
	}

	return nil
}

// AdminReviewRequest represents admin-specific review management request
type AdminReviewRequest struct {
	ReviewID string `json:"review_id" validate:"required"`
	Action   string `json:"action" validate:"required,oneof=flag unflag approve reject force_delete hard_delete"`
	Reason   string `json:"reason,omitempty"`
}

// Validate validates the admin review request
func (req *AdminReviewRequest) Validate() error {
	if req.ReviewID == "" {
		return errors.New("review_id is required")
	}
	validActions := []string{"flag", "unflag", "approve", "reject", "force_delete", "hard_delete"}
	isValid := false
	for _, valid := range validActions {
		if req.Action == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("invalid action")
	}
	return nil
}

// AdminListReviewsRequest represents request for admin to list all reviews with filters
type AdminListReviewsRequest struct {
	Page      int       `json:"page,omitempty"`
	Limit     int       `json:"limit,omitempty"`
	SortBy    string    `json:"sort_by,omitempty"`
	Order     string    `json:"order,omitempty"`
	ProductID string    `json:"product_id,omitempty"`
	UserID    string    `json:"user_id,omitempty"`
	MinRating int       `json:"min_rating,omitempty"`
	MaxRating int       `json:"max_rating,omitempty"`
	IsFlagged *bool     `json:"is_flagged,omitempty"`
	IsActive  *bool     `json:"is_active,omitempty"`
	StartDate time.Time `json:"start_date,omitempty"`
	EndDate   time.Time `json:"end_date,omitempty"`
}

// Validate validates the admin list reviews request
func (req *AdminListReviewsRequest) Validate() error {
	if req.Page < 0 {
		req.Page = 1
	}
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Limit > 100 {
		req.Limit = 100
	}
	if req.SortBy == "" {
		req.SortBy = "review_date"
	}
	if req.Order == "" {
		req.Order = "desc"
	}
	if req.MinRating < 1 {
		req.MinRating = 1
	}
	if req.MaxRating > 5 || req.MaxRating < req.MinRating {
		req.MaxRating = 5
	}
	if !req.EndDate.IsZero() && !req.StartDate.IsZero() && req.EndDate.Before(req.StartDate) {
		return errors.New("end_date must be after start_date")
	}
	validSortBy := []string{"review_date", "updated_date", "rating", "likes_count", "user_id", "product_id"}
	isValid := false
	for _, valid := range validSortBy {
		if req.SortBy == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("invalid sort_by value")
	}
	if req.Order != "asc" && req.Order != "desc" {
		return errors.New("order must be 'asc' or 'desc'")
	}
	return nil
}

// AdminBulkDeleteFlaggedRequest represents request for bulk deletion of flagged reviews
type AdminBulkDeleteFlaggedRequest struct {
	Action string `json:"action" validate:"required,oneof=hard_delete force_delete"`
	Reason string `json:"reason,omitempty"`
}

// Validate validates the admin bulk delete request
func (req *AdminBulkDeleteFlaggedRequest) Validate() error {
	validActions := []string{"hard_delete", "force_delete"}
	isValid := false
	for _, valid := range validActions {
		if req.Action == valid {
			isValid = true
			break
		}
	}
	if !isValid {
		return errors.New("invalid action - must be 'hard_delete' or 'force_delete'")
	}
	return nil
}
