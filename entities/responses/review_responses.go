package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// ReviewResponse represents a review response with additional metadata
type ReviewResponse struct {
	ID          string    `json:"id"`
	ProductID   string    `json:"product_id"`
	UserID      string    `json:"user_id"`
	UserName    string    `json:"user_name"`
	UserEmail   string    `json:"user_email,omitempty"`
	Rating      int       `json:"rating"`
	Title       string    `json:"title"`
	Comment     string    `json:"comment"`
	LikesCount  int       `json:"likes_count"`
	IsUserLiked bool      `json:"is_user_liked"` // Whether the requesting user liked this review
	IsVerified  bool      `json:"is_verified"`   // Whether the reviewer purchased the product
	IsOwner     bool      `json:"is_owner"`      // Whether the requesting user is the review owner
	IsActive    bool      `json:"is_active"`
	ReviewDate  time.Time `json:"review_date"`
	UpdatedDate time.Time `json:"updated_date"`
	TimeAgo     string    `json:"time_ago"`   // Human readable time difference
	CanEdit     bool      `json:"can_edit"`   // Whether the requesting user can edit this review
	CanDelete   bool      `json:"can_delete"` // Whether the requesting user can delete this review

	// Admin-specific fields (only included for admin users)
	IsFlagged       bool      `json:"is_flagged,omitempty"`       // Whether the review has been flagged
	IsApproved      bool      `json:"is_approved,omitempty"`      // Whether the review has been approved
	FlagReason      string    `json:"flag_reason,omitempty"`      // Reason for flagging
	FlaggedBy       string    `json:"flagged_by,omitempty"`       // Admin who flagged the review
	FlaggedAt       time.Time `json:"flagged_at,omitempty"`       // When the review was flagged
	ApprovedBy      string    `json:"approved_by,omitempty"`      // Admin who approved the review
	ApprovedAt      time.Time `json:"approved_at,omitempty"`      // When the review was approved
	RejectedBy      string    `json:"rejected_by,omitempty"`      // Admin who rejected the review
	RejectedAt      time.Time `json:"rejected_at,omitempty"`      // When the review was rejected
	RejectionReason string    `json:"rejection_reason,omitempty"` // Reason for rejection
	DeletedBy       string    `json:"deleted_by,omitempty"`       // Admin who deleted the review
	DeletedAt       time.Time `json:"deleted_at,omitempty"`       // When the review was deleted
	DeletionReason  string    `json:"deletion_reason,omitempty"`  // Reason for deletion
}

// NewReviewResponse creates a new review response from a review model
func NewReviewResponse(review *models.Review, requestingUserID string, isAdmin bool) *ReviewResponse {
	if review == nil {
		return nil
	}

	response := &ReviewResponse{
		ID:          review.GetID(),
		ProductID:   review.ProductID,
		UserID:      review.UserID,
		UserName:    review.UserName,
		UserEmail:   review.UserEmail,
		Rating:      review.Rating,
		Title:       review.Title,
		Comment:     review.Comment,
		LikesCount:  review.LikesCount,
		IsUserLiked: review.HasUserLiked(requestingUserID),
		IsVerified:  review.IsVerified,
		IsOwner:     review.UserID == requestingUserID,
		IsActive:    review.IsActive,
		ReviewDate:  review.ReviewDate,
		UpdatedDate: review.UpdatedDate,
		TimeAgo:     formatTimeAgo(review.ReviewDate),
		CanEdit:     review.UserID == requestingUserID,            // Only owner can edit
		CanDelete:   review.UserID == requestingUserID || isAdmin, // Owner or admin can delete
	}

	// Include admin-specific fields only for admin users
	if isAdmin {
		response.IsFlagged = review.IsFlagged
		response.IsApproved = review.IsApproved
		response.FlagReason = review.FlagReason
		response.FlaggedBy = review.FlaggedBy
		response.FlaggedAt = review.FlaggedAt
		response.ApprovedBy = review.ApprovedBy
		response.ApprovedAt = review.ApprovedAt
		response.RejectedBy = review.RejectedBy
		response.RejectedAt = review.RejectedAt
		response.RejectionReason = review.RejectionReason
		response.DeletedBy = review.DeletedBy
		response.DeletedAt = review.DeletedAt
		response.DeletionReason = review.DeletionReason
	}

	return response
}

// ReviewStatsResponse represents aggregated review statistics for a product
type ReviewStatsResponse struct {
	ProductID         string            `json:"product_id"`
	TotalReviews      int               `json:"total_reviews"`
	AverageRating     float64           `json:"average_rating"`
	RatingBreakdown   map[string]int    `json:"rating_breakdown"` // "1": count, "2": count, etc.
	RecentReviews     []*ReviewResponse `json:"recent_reviews"`
	TopReviews        []*ReviewResponse `json:"top_reviews"` // Most liked reviews
	VerifiedReviews   int               `json:"verified_reviews"`
	UnverifiedReviews int               `json:"unverified_reviews"`
}

// NewReviewStatsResponse creates a new review stats response
func NewReviewStatsResponse(productID string, reviews []*models.Review, requestingUserID string, isAdmin bool) *ReviewStatsResponse {
	stats := &ReviewStatsResponse{
		ProductID:       productID,
		TotalReviews:    len(reviews),
		RatingBreakdown: make(map[string]int),
		RecentReviews:   []*ReviewResponse{},
		TopReviews:      []*ReviewResponse{},
	}

	if len(reviews) == 0 {
		return stats
	}

	// Calculate statistics
	var totalRating float64
	var verifiedCount, unverifiedCount int

	for _, review := range reviews {
		totalRating += float64(review.Rating)

		// Count rating breakdown
		ratingStr := string(rune(review.Rating + '0'))
		stats.RatingBreakdown[ratingStr]++

		// Count verified vs unverified
		if review.IsVerified {
			verifiedCount++
		} else {
			unverifiedCount++
		}
	}

	stats.AverageRating = totalRating / float64(len(reviews))
	stats.VerifiedReviews = verifiedCount
	stats.UnverifiedReviews = unverifiedCount

	// Get recent reviews (last 5)
	recentCount := 5
	if len(reviews) < recentCount {
		recentCount = len(reviews)
	}
	for i := 0; i < recentCount; i++ {
		stats.RecentReviews = append(stats.RecentReviews,
			NewReviewResponse(reviews[i], requestingUserID, isAdmin))
	}

	// Get top reviews (most liked, max 5)
	topReviews := make([]*models.Review, len(reviews))
	copy(topReviews, reviews)

	// Sort by likes count (bubble sort for small arrays)
	for i := 0; i < len(topReviews)-1; i++ {
		for j := 0; j < len(topReviews)-i-1; j++ {
			if topReviews[j].LikesCount < topReviews[j+1].LikesCount {
				topReviews[j], topReviews[j+1] = topReviews[j+1], topReviews[j]
			}
		}
	}

	topCount := 5
	if len(topReviews) < topCount {
		topCount = len(topReviews)
	}
	for i := 0; i < topCount; i++ {
		stats.TopReviews = append(stats.TopReviews,
			NewReviewResponse(topReviews[i], requestingUserID, isAdmin))
	}

	return stats
}

// PaginatedReviewsResponse represents a paginated list of reviews
type PaginatedReviewsResponse struct {
	Reviews     []*ReviewResponse `json:"reviews"`
	TotalCount  int               `json:"total_count"`
	Page        int               `json:"page"`
	Limit       int               `json:"limit"`
	TotalPages  int               `json:"total_pages"`
	HasNext     bool              `json:"has_next"`
	HasPrevious bool              `json:"has_previous"`
}

// NewPaginatedReviewsResponse creates a new paginated reviews response
func NewPaginatedReviewsResponse(reviews []*models.Review, totalCount, page, limit int, requestingUserID string, isAdmin bool) *PaginatedReviewsResponse {
	response := &PaginatedReviewsResponse{
		Reviews:     []*ReviewResponse{},
		TotalCount:  totalCount,
		Page:        page,
		Limit:       limit,
		TotalPages:  (totalCount + limit - 1) / limit,
		HasNext:     page*limit < totalCount,
		HasPrevious: page > 1,
	}

	// Convert reviews to responses
	for _, review := range reviews {
		reviewResponse := NewReviewResponse(review, requestingUserID, isAdmin)
		if reviewResponse != nil {
			response.Reviews = append(response.Reviews, reviewResponse)
		}
	}

	return response
}

// ReviewActionResponse represents the response after a review action (like, unlike, etc.)
type ReviewActionResponse struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	ReviewID    string `json:"review_id"`
	LikesCount  int    `json:"likes_count"`
	IsUserLiked bool   `json:"is_user_liked"`
}

// NewReviewActionResponse creates a new review action response
func NewReviewActionResponse(success bool, message, reviewID string, likesCount int, isUserLiked bool) *ReviewActionResponse {
	return &ReviewActionResponse{
		Success:     success,
		Message:     message,
		ReviewID:    reviewID,
		LikesCount:  likesCount,
		IsUserLiked: isUserLiked,
	}
}

// Helper function to format time ago
func formatTimeAgo(reviewTime time.Time) string {
	duration := time.Since(reviewTime)

	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return string(rune(minutes+'0')) + " minutes ago"
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return string(rune(hours+'0')) + " hours ago"
	} else if duration < 30*24*time.Hour {
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return string(rune(days+'0')) + " days ago"
	} else if duration < 365*24*time.Hour {
		months := int(duration.Hours() / (30 * 24))
		if months == 1 {
			return "1 month ago"
		}
		return string(rune(months+'0')) + " months ago"
	} else {
		years := int(duration.Hours() / (365 * 24))
		if years == 1 {
			return "1 year ago"
		}
		return string(rune(years+'0')) + " years ago"
	}
}

// BulkDeleteResponse represents the response for bulk delete operations
type BulkDeleteResponse struct {
	Message          string    `json:"message"`
	Action           string    `json:"action"`
	TotalFound       int       `json:"total_found"`
	TotalDeleted     int       `json:"total_deleted"`
	FailedDeletes    int       `json:"failed_deletes"`
	DeletedReviewIDs []string  `json:"deleted_review_ids,omitempty"`
	FailedReviewIDs  []string  `json:"failed_review_ids,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	ProcessedAt      time.Time `json:"processed_at"`
}

// NewBulkDeleteResponse creates a new bulk delete response
func NewBulkDeleteResponse(action string, totalFound, totalDeleted, failedDeletes int, deletedIDs, failedIDs []string, reason string) *BulkDeleteResponse {
	return &BulkDeleteResponse{
		Message:          "Bulk delete operation completed",
		Action:           action,
		TotalFound:       totalFound,
		TotalDeleted:     totalDeleted,
		FailedDeletes:    failedDeletes,
		DeletedReviewIDs: deletedIDs,
		FailedReviewIDs:  failedIDs,
		Reason:           reason,
		ProcessedAt:      time.Now(),
	}
}
