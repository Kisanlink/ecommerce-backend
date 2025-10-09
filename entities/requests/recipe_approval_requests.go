package requests

// SubmitRecipeRequest represents a request to submit a recipe for review
type SubmitRecipeRequest struct {
	Name            string            `json:"name" binding:"required"`
	Category        string            `json:"category" binding:"required"`
	Description     string            `json:"description" binding:"required"`
	DemoDescription string            `json:"demo_description,omitempty"`
	PrepTime        string            `json:"prep_time" binding:"required"`
	Servings        int               `json:"servings" binding:"required,min=1"`
	Difficulty      string            `json:"difficulty" binding:"required,oneof=Easy Medium Hard"`
	Image           string            `json:"image,omitempty"`
	Ingredients     []string          `json:"ingredients" binding:"required,min=1"`
	Instructions    []string          `json:"instructions" binding:"required,min=1"`
	NutritionFacts  map[string]string `json:"nutrition_facts,omitempty"`
	ProTips         []string          `json:"pro_tips,omitempty"`
}

// RejectRecipeRequest represents a request to reject a recipe
type RejectRecipeRequest struct {
	Reason string `json:"reason" binding:"required,min=10"`
}

// PendingRecipeReviewResponse represents a response for pending recipe review
type PendingRecipeReviewResponse struct {
	ReviewID        string `json:"review_id"`
	UserID          string `json:"user_id"`
	Status          string `json:"status"`
	SubmittedAt     string `json:"submitted_at"`
	ReviewedAt      string `json:"reviewed_at,omitempty"`
	ReviewedBy      string `json:"reviewed_by,omitempty"`
	RejectionReason string `json:"rejection_reason,omitempty"`

	// Recipe data
	Name            string            `json:"name"`
	Category        string            `json:"category"`
	Description     string            `json:"description"`
	DemoDescription string            `json:"demo_description"`
	PrepTime        string            `json:"prep_time"`
	Servings        int               `json:"servings"`
	Difficulty      string            `json:"difficulty"`
	Image           string            `json:"image"`
	Ingredients     []string          `json:"ingredients"`
	Instructions    []string          `json:"instructions"`
	NutritionFacts  map[string]string `json:"nutrition_facts"`
	ProTips         []string          `json:"pro_tips"`
}

// RecipeSubmissionResponse represents a response for recipe submission
type RecipeSubmissionResponse struct {
	Message string                      `json:"message"`
	Review  PendingRecipeReviewResponse `json:"review"`
}

// RecipeApprovalResponse represents a response for recipe approval
type RecipeApprovalResponse struct {
	Message string                      `json:"message"`
	Recipe  interface{}                 `json:"recipe"`
	Review  PendingRecipeReviewResponse `json:"review"`
}

// RecipeRejectionResponse represents a response for recipe rejection
type RecipeRejectionResponse struct {
	Message string                      `json:"message"`
	Review  PendingRecipeReviewResponse `json:"review"`
}
