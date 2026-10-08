package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"github.com/lib/pq"
)

// Recipe represents a recipe in the system
type Recipe struct {
	*base.BaseModel `gorm:"embedded"`
	RecipeID        string            `json:"recipe_id" gorm:"type:varchar(64);uniqueIndex"`
	Name            string            `json:"name"`
	Category        string            `json:"category"`
	Rating          float64           `json:"rating" gorm:"default:0"`
	Reviews         int               `json:"reviews" gorm:"default:0"`
	Description     string            `json:"description" gorm:"type:text"`
	DemoDescription string            `json:"demo_description" gorm:"type:text"`
	PrepTime        string            `json:"prep_time"`
	Servings        int               `json:"servings"`
	Difficulty      string            `json:"difficulty"`
	Image           string            `json:"image"` // Recipe image URL from S3
	Ingredients     pq.StringArray    `json:"ingredients" gorm:"type:text[]"`
	Instructions    pq.StringArray    `json:"instructions" gorm:"type:text[]"`
	NutritionFacts  map[string]string `json:"nutrition_facts" gorm:"type:jsonb;serializer:json"`
	ProTips         pq.StringArray    `json:"pro_tips" gorm:"type:text[]"`
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
