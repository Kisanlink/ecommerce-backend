package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"github.com/lib/pq"
)

// Review represents a product review stored in Neon/PostgreSQL
type Review struct {
	*base.BaseModel `json:"base_model" gorm:"embedded"`
	ProductID       string         `json:"product_id" gorm:"type:varchar(64);index"`
	UserID          string         `json:"user_id" gorm:"type:varchar(64);index"`
	UserName        string         `json:"user_name"`
	UserEmail       string         `json:"user_email,omitempty"`
	Rating          int            `json:"rating"`
	Title           string         `json:"title"`
	Comment         string         `json:"comment" gorm:"type:text"`
	LikesCount      int            `json:"likes_count" gorm:"default:0"`
	LikedBy         pq.StringArray `json:"liked_by" gorm:"type:text[]"`
	IsVerified      bool           `json:"is_verified" gorm:"default:false"`
	IsActive        bool           `json:"is_active" gorm:"default:true;index"`
	IsFlagged       bool           `json:"is_flagged" gorm:"default:false;index"`
	IsApproved      bool           `json:"is_approved" gorm:"default:true"`
	FlagReason      string         `json:"flag_reason,omitempty"`
	FlaggedBy       string         `json:"flagged_by,omitempty"`
	FlaggedAt       time.Time      `json:"flagged_at,omitempty"`
	ApprovedBy      string         `json:"approved_by,omitempty"`
	ApprovedAt      time.Time      `json:"approved_at,omitempty"`
	RejectedBy      string         `json:"rejected_by,omitempty"`
	RejectedAt      time.Time      `json:"rejected_at,omitempty"`
	RejectionReason string         `json:"rejection_reason,omitempty"`
	DeletedBy       string         `json:"deleted_by,omitempty"`
	DeletedAt       time.Time      `json:"deleted_at,omitempty"`
	DeletionReason  string         `json:"deletion_reason,omitempty"`
	ReviewDate      time.Time      `json:"review_date"`
	UpdatedDate     time.Time      `json:"updated_date"`
}

// NewReview creates a new review with base model
func NewReview(productID, userID, userName, userEmail, title, comment string, rating int, isVerified bool) *Review {
	baseModel := base.NewBaseModel("REV", hash.Medium)

	return &Review{
		BaseModel:   baseModel,
		ProductID:   productID,
		UserID:      userID,
		UserName:    userName,
		UserEmail:   userEmail,
		Rating:      rating,
		Title:       title,
		Comment:     comment,
		LikesCount:  0,
		LikedBy:     []string{},
		IsVerified:  isVerified,
		IsActive:    true,
		IsFlagged:   false,
		IsApproved:  true, // Reviews are approved by default
		ReviewDate:  time.Now(),
		UpdatedDate: time.Now(),
	}
}

// TableName returns the PostgreSQL table name
func (r *Review) TableName() string {
	return "reviews"
}

// AddLike adds a like from a user if not already liked
func (r *Review) AddLike(userID string) bool {
	// Check if user already liked this review
	for _, likedUserID := range r.LikedBy {
		if likedUserID == userID {
			return false // Already liked
		}
	}

	// Add the like
	r.LikedBy = append(r.LikedBy, userID)
	r.LikesCount = len(r.LikedBy)
	r.UpdatedDate = time.Now()
	return true
}

// RemoveLike removes a like from a user if it exists
func (r *Review) RemoveLike(userID string) bool {
	// Find and remove the user ID
	for i, likedUserID := range r.LikedBy {
		if likedUserID == userID {
			// Remove the user ID from the slice
			r.LikedBy = append(r.LikedBy[:i], r.LikedBy[i+1:]...)
			r.LikesCount = len(r.LikedBy)
			r.UpdatedDate = time.Now()
			return true
		}
	}
	return false // User hadn't liked this review
}

// HasUserLiked checks if a specific user has liked this review
func (r *Review) HasUserLiked(userID string) bool {
	for _, likedUserID := range r.LikedBy {
		if likedUserID == userID {
			return true
		}
	}
	return false
}

// Update updates the review content
func (r *Review) Update(title, comment string, rating int) {
	r.Title = title
	r.Comment = comment
	r.Rating = rating
	r.UpdatedDate = time.Now()
}

// Validate validates the review data
func (r *Review) Validate() error {
	if r.ProductID == "" {
		return base.NewValidationError("ProductID", "product_id is required")
	}
	if r.UserID == "" {
		return base.NewValidationError("UserID", "user_id is required")
	}
	if r.Rating < 1 || r.Rating > 5 {
		return base.NewValidationError("Rating", "rating must be between 1 and 5")
	}
	if len(r.Comment) < 10 {
		return base.NewValidationError("Comment", "comment must be at least 10 characters long")
	}
	if len(r.Title) < 5 {
		return base.NewValidationError("Title", "title must be at least 5 characters long")
	}
	return nil
}

// GetAge returns how old this review is
func (r *Review) GetAge() time.Duration {
	return time.Since(r.ReviewDate)
}

// IsRecent checks if the review was created within the specified duration
func (r *Review) IsRecent(duration time.Duration) bool {
	return r.GetAge() <= duration
}
