package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Product represents a product in the ecommerce system
type Product struct {
	*base.BaseModel `gorm:"embedded"`
	Name            string  `json:"name" gorm:"not null"`
	Description     string  `json:"description"`
	Price           float64 `json:"price" gorm:"not null"`
	Category        string  `json:"category"`
	ImageURL        string  `json:"image_url"`
	Stock           int     `json:"stock" gorm:"default:0"`
	IsActive        bool    `json:"is_active" gorm:"default:true"`
}

// NewProduct creates a new product with base model
func NewProduct(name, description, category, imageURL string, price float64, stock int) *Product {
	baseModel := base.NewBaseModel("PROD", hash.Medium)
	return &Product{
		BaseModel:   baseModel,
		Name:        name,
		Description: description,
		Price:       price,
		Category:    category,
		ImageURL:    imageURL,
		Stock:       stock,
		IsActive:    true,
	}
}

// TableName returns the table name for GORM
func (p *Product) TableName() string {
	return "products"
}
