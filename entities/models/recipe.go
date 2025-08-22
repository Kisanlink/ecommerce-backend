package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Recipe represents a recipe in the system
type Recipe struct {
	*base.BaseModel `gorm:"embedded"`
	Title           string `json:"title" gorm:"not null"`
	Description     string `json:"description"`
	Ingredients     string `json:"ingredients"`
	Steps           string `json:"steps"`
	ImageURL        string `json:"image_url"`
}

// NewRecipe creates a new recipe with base model
func NewRecipe(title, description, ingredients, steps, imageURL string) *Recipe {
	baseModel := base.NewBaseModel("RECP", hash.Small)
	return &Recipe{
		BaseModel:   baseModel,
		Title:       title,
		Description: description,
		Ingredients: ingredients,
		Steps:       steps,
		ImageURL:    imageURL,
	}
}

// TableName returns the table name for GORM
func (r *Recipe) TableName() string {
	return "recipes"
}
