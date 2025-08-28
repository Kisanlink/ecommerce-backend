package requests

import "github.com/Kisanlink/kisanlink-db/pkg/base"

// CreateRecipeRequest represents a request to create a recipe
type CreateRecipeRequest struct {
	*base.BaseRequest `json:",inline"`
	Name              string            `json:"name" binding:"required"`
	Category          string            `json:"category" binding:"required"`
	Description       string            `json:"description" binding:"required"`
	DemoDescription   string            `json:"demo_description"`
	PrepTime          string            `json:"prep_time"`
	Servings          int               `json:"servings" binding:"required,min=1"`
	Difficulty        string            `json:"difficulty" binding:"required"`
	Image             string            `json:"image"`
	Ingredients       []string          `json:"ingredients" binding:"required"`
	Instructions      []string          `json:"instructions" binding:"required"`
	Rating            float64           `json:"rating"`
	Reviews           int               `json:"reviews"`
	NutritionFacts    map[string]string `json:"nutrition_facts"`
	ProTips           []string          `json:"pro_tips"`
}

// GetRecipeRequest represents a request to get a recipe by ID
type GetRecipeRequest struct {
	*base.BaseRequest `json:",inline"`
	ID                string `uri:"id" binding:"required"`
}

// UpdateRecipeRequest represents a request to update a recipe
type UpdateRecipeRequest struct {
	*base.BaseRequest `json:",inline"`
	ID                string            `uri:"id" binding:"required"`
	Name              string            `json:"name" binding:"required"`
	Category          string            `json:"category" binding:"required"`
	Description       string            `json:"description" binding:"required"`
	DemoDescription   string            `json:"demo_description"`
	PrepTime          string            `json:"prep_time"`
	Servings          int               `json:"servings" binding:"required,min=1"`
	Difficulty        string            `json:"difficulty" binding:"required"`
	Image             string            `json:"image"`
	Ingredients       []string          `json:"ingredients" binding:"required"`
	Instructions      []string          `json:"instructions" binding:"required"`
	Rating            float64           `json:"rating"`
	Reviews           int               `json:"reviews"`
	NutritionFacts    map[string]string `json:"nutrition_facts"`
	ProTips           []string          `json:"pro_tips"`
}

// DeleteRecipeRequest represents a request to delete a recipe
type DeleteRecipeRequest struct {
	*base.BaseRequest `json:",inline"`
	ID                string `uri:"id" binding:"required"`
}

// ListRecipesRequest represents a request to list recipes with filtering and pagination
type ListRecipesRequest struct {
	*base.BaseRequest `json:",inline"`
	Page              int    `form:"page" json:"page"`
	PerPage           int    `form:"per_page" json:"per_page"`
	Category          string `form:"category" json:"category"`
	Search            string `form:"search" json:"search"`
	Ingredient        string `form:"ingredient" json:"ingredient"`
}

// RecipeListResponse represents a simplified response for listing recipes
type RecipeListResponse struct {
	Name            string  `json:"name"`
	DemoDescription string  `json:"demo_description"`
	Reviews         int     `json:"reviews"`
	Rating          float64 `json:"rating"`
	PrepTime        string  `json:"prep_time"`
	Difficulty      string  `json:"difficulty"`
	Image           string  `json:"image"`
}
