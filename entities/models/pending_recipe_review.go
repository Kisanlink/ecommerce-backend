package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"github.com/lib/pq"
)

// PendingRecipeReviewStatus represents the status of a recipe review
type PendingRecipeReviewStatus string

const (
	StatusPending  PendingRecipeReviewStatus = "pending"
	StatusApproved PendingRecipeReviewStatus = "approved"
	StatusRejected PendingRecipeReviewStatus = "rejected"
)

// PendingRecipeReview represents a recipe submitted by a user that needs admin approval
type PendingRecipeReview struct {
	*base.BaseModel `gorm:"embedded"`
	ReviewID        string                    `json:"review_id" gorm:"type:varchar(64);uniqueIndex"` // Primary key
	UserID          string                    `json:"user_id" gorm:"type:varchar(64);index"`         // User who submitted the recipe
	Status          PendingRecipeReviewStatus `json:"status" gorm:"type:varchar(32);index"`          // pending, approved, rejected
	SubmittedAt     string                    `json:"submitted_at"`
	ReviewedAt      string                    `json:"reviewed_at,omitempty"`
	ReviewedBy      string                    `json:"reviewed_by,omitempty"` // Admin who reviewed
	RejectionReason string                    `json:"rejection_reason,omitempty"`

	// Recipe data (same structure as Recipe model)
	Name            string            `json:"name"`
	Category        string            `json:"category"`
	Description     string            `json:"description" gorm:"type:text"`
	DemoDescription string            `json:"demo_description" gorm:"type:text"`
	PrepTime        string            `json:"prep_time"`
	Servings        int               `json:"servings"`
	Difficulty      string            `json:"difficulty"`
	Image           string            `json:"image"`
	Ingredients     pq.StringArray    `json:"ingredients" gorm:"type:text[]"`
	Instructions    pq.StringArray    `json:"instructions" gorm:"type:text[]"`
	NutritionFacts  map[string]string `json:"nutrition_facts" gorm:"type:jsonb;serializer:json"`
	ProTips         pq.StringArray    `json:"pro_tips" gorm:"type:text[]"`
}

// NewPendingRecipeReview creates a new pending recipe review
func NewPendingRecipeReview(userID string, recipe *Recipe) *PendingRecipeReview {
	baseModel := base.NewBaseModel("PRR", hash.Small)
	return &PendingRecipeReview{
		BaseModel:       baseModel,
		ReviewID:        baseModel.GetID(),
		UserID:          userID,
		Status:          StatusPending,
		SubmittedAt:     baseModel.GetCreatedAt().Format("2006-01-02T15:04:05Z07:00"),
		Name:            recipe.Name,
		Category:        recipe.Category,
		Description:     recipe.Description,
		DemoDescription: recipe.DemoDescription,
		PrepTime:        recipe.PrepTime,
		Servings:        recipe.Servings,
		Difficulty:      recipe.Difficulty,
		Image:           recipe.Image,
		Ingredients:     recipe.Ingredients,
		Instructions:    recipe.Instructions,
		NutritionFacts:  recipe.NutritionFacts,
		ProTips:         recipe.ProTips,
	}
}

// ToRecipe converts a pending recipe review to a Recipe model
func (prr *PendingRecipeReview) ToRecipe() *Recipe {
	baseModel := base.NewBaseModel("RECP", hash.Small)
	return &Recipe{
		BaseModel:       baseModel,
		RecipeID:        baseModel.GetID(),
		Name:            prr.Name,
		Category:        prr.Category,
		Rating:          0.0, // Initialize with default values
		Reviews:         0,
		Description:     prr.Description,
		DemoDescription: prr.DemoDescription,
		PrepTime:        prr.PrepTime,
		Servings:        prr.Servings,
		Difficulty:      prr.Difficulty,
		Image:           prr.Image,
		Ingredients:     prr.Ingredients,
		Instructions:    prr.Instructions,
		NutritionFacts:  prr.NutritionFacts,
		ProTips:         prr.ProTips,
	}
}

// Approve marks the recipe review as approved
func (prr *PendingRecipeReview) Approve(reviewedBy string) {
	prr.Status = StatusApproved
	prr.ReviewedBy = reviewedBy
	prr.ReviewedAt = base.NewBaseModel("", hash.Small).GetCreatedAt().Format("2006-01-02T15:04:05Z07:00")
}

// Reject marks the recipe review as rejected with a reason
func (prr *PendingRecipeReview) Reject(reviewedBy, reason string) {
	prr.Status = StatusRejected
	prr.ReviewedBy = reviewedBy
	prr.RejectionReason = reason
	prr.ReviewedAt = base.NewBaseModel("", hash.Small).GetCreatedAt().Format("2006-01-02T15:04:05Z07:00")
}

// TableName returns the table name for GORM
func (prr *PendingRecipeReview) TableName() string {
	return "pending_recipe_reviews"
}
