package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Recipe represents a recipe in the system
type Recipe struct {
	*base.BaseModel `gorm:"embedded"`
	RecipeID        string            `json:"recipe_id" dynamodb:"recipee_id"` // Map to DynamoDB's recipee_id field
	Name            string            `json:"name" dynamodb:"name"`
	Category        string            `json:"category" dynamodb:"category"`
	Rating          float64           `json:"rating" dynamodb:"rating"`
	Reviews         int               `json:"reviews" dynamodb:"reviews"`
	Description     string            `json:"description" dynamodb:"description"`
	DemoDescription string            `json:"demo_description" dynamodb:"demo_description"`
	PrepTime        string            `json:"prep_time" dynamodb:"prep_time"`
	Servings        int               `json:"servings" dynamodb:"servings"`
	Difficulty      string            `json:"difficulty" dynamodb:"difficulty"`
	Image           string            `json:"image" dynamodb:"image"` // Recipe image URL from S3
	Ingredients     []string          `json:"ingredients" dynamodb:"ingredients"`
	Instructions    []string          `json:"instructions" dynamodb:"instructions"`
	NutritionFacts  map[string]string `json:"nutrition_facts" dynamodb:"nutrition_facts"`
	ProTips         []string          `json:"pro_tips" dynamodb:"pro_tips"`
}

// NewRecipe creates a new recipe with base model
func NewRecipe(name, description string, ingredients, instructions []string) *Recipe {
	baseModel := base.NewBaseModel("RECP", hash.Small)
	return &Recipe{
		BaseModel:       baseModel,
		RecipeID:        baseModel.GetID(), // Set RecipeID to the generated BaseModel ID
		Name:            name,
		Description:     description,
		DemoDescription: description, // Initialize demo_description with the same value as description
		Ingredients:     ingredients,
		Instructions:    instructions,
		Category:        "General",
		Difficulty:      "Medium",
		Servings:        1,
	}
}

// TableName returns the table name for GORM
func (r *Recipe) TableName() string {
	return "recipes"
}
