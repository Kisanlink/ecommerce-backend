package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
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
	ReviewID        string                    `json:"review_id" dynamodb:"review_id"` // Primary key
	UserID          string                    `json:"user_id" dynamodb:"user_id"`     // User who submitted the recipe
	Status          PendingRecipeReviewStatus `json:"status" dynamodb:"status"`       // pending, approved, rejected
	SubmittedAt     string                    `json:"submitted_at" dynamodb:"submitted_at"`
	ReviewedAt      string                    `json:"reviewed_at" dynamodb:"reviewed_at,omitempty"`
	ReviewedBy      string                    `json:"reviewed_by" dynamodb:"reviewed_by,omitempty"` // Admin who reviewed
	RejectionReason string                    `json:"rejection_reason" dynamodb:"rejection_reason,omitempty"`

	// Recipe data (same structure as Recipe model)
	Name            string            `json:"name" dynamodb:"name"`
	Category        string            `json:"category" dynamodb:"category"`
	Description     string            `json:"description" dynamodb:"description"`
	DemoDescription string            `json:"demo_description" dynamodb:"demo_description"`
	PrepTime        string            `json:"prep_time" dynamodb:"prep_time"`
	Servings        int               `json:"servings" dynamodb:"servings"`
	Difficulty      string            `json:"difficulty" dynamodb:"difficulty"`
	Image           string            `json:"image" dynamodb:"image"`
	Ingredients     []string          `json:"ingredients" dynamodb:"ingredients"`
	Instructions    []string          `json:"instructions" dynamodb:"instructions"`
	NutritionFacts  map[string]string `json:"nutrition_facts" dynamodb:"nutrition_facts"`
	ProTips         []string          `json:"pro_tips" dynamodb:"pro_tips"`
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
