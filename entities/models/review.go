package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Review represents a product review stored in DynamoDB
type Review struct {
	*base.BaseModel `json:"base_model"`
	ProductID       string    `json:"product_id" dynamodbav:"product_id"`                                 // Primary key
	UserID          string    `json:"user_id" dynamodbav:"user_id"`                                       // Sort key
	UserName        string    `json:"user_name" dynamodbav:"user_name"`                                   // User's name (cached from users table)
	UserEmail       string    `json:"user_email,omitempty" dynamodbav:"user_email,omitempty"`             // User's email (optional)
	Rating          int       `json:"rating" dynamodbav:"rating"`                                         // 1-5 stars
	Title           string    `json:"title" dynamodbav:"title"`                                           // Review title
	Comment         string    `json:"comment" dynamodbav:"comment"`                                       // Review text
	LikesCount      int       `json:"likes_count" dynamodbav:"likes_count"`                               // Number of likes
	LikedBy         []string  `json:"liked_by" dynamodbav:"liked_by"`                                     // List of user IDs who liked this review
	IsVerified      bool      `json:"is_verified" dynamodbav:"is_verified"`                               // If the reviewer purchased the product
	IsActive        bool      `json:"is_active" dynamodbav:"is_active"`                                   // Active status
	IsFlagged       bool      `json:"is_flagged" dynamodbav:"is_flagged"`                                 // If the review has been flagged for review
	IsApproved      bool      `json:"is_approved" dynamodbav:"is_approved"`                               // If the review has been approved by admin
	FlagReason      string    `json:"flag_reason,omitempty" dynamodbav:"flag_reason,omitempty"`           // Reason for flagging
	FlaggedBy       string    `json:"flagged_by,omitempty" dynamodbav:"flagged_by,omitempty"`             // Admin who flagged the review
	FlaggedAt       time.Time `json:"flagged_at,omitempty" dynamodbav:"flagged_at,omitempty"`             // When the review was flagged
	ApprovedBy      string    `json:"approved_by,omitempty" dynamodbav:"approved_by,omitempty"`           // Admin who approved the review
	ApprovedAt      time.Time `json:"approved_at,omitempty" dynamodbav:"approved_at,omitempty"`           // When the review was approved
	RejectedBy      string    `json:"rejected_by,omitempty" dynamodbav:"rejected_by,omitempty"`           // Admin who rejected the review
	RejectedAt      time.Time `json:"rejected_at,omitempty" dynamodbav:"rejected_at,omitempty"`           // When the review was rejected
	RejectionReason string    `json:"rejection_reason,omitempty" dynamodbav:"rejection_reason,omitempty"` // Reason for rejection
	DeletedBy       string    `json:"deleted_by,omitempty" dynamodbav:"deleted_by,omitempty"`             // Admin who deleted the review
	DeletedAt       time.Time `json:"deleted_at,omitempty" dynamodbav:"deleted_at,omitempty"`             // When the review was deleted
	DeletionReason  string    `json:"deletion_reason,omitempty" dynamodbav:"deletion_reason,omitempty"`   // Reason for deletion
	ReviewDate      time.Time `json:"review_date" dynamodbav:"review_date"`                               // When the review was created
	UpdatedDate     time.Time `json:"updated_date" dynamodbav:"updated_date"`                             // When the review was last updated
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

// TableName returns the DynamoDB table name
func (r *Review) TableName() string {
	return "product_reviews"
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
