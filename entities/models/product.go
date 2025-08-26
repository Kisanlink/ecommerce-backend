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
	Category        string  `json:"category"`
	Price           float64 `json:"price" gorm:"not null"`             // Current/discounted price
	ActualPrice     float64 `json:"actual_price" gorm:"not null"`      // Original price
	DiscountPercent int     `json:"discount_percent" gorm:"default:0"` // Discount percentage
	ImageURL        string  `json:"image_url"`
	Stock           int     `json:"stock" gorm:"default:0"`
	StockStatus     string  `json:"stock_status" gorm:"default:'In Stock'"` // Stock status
	ReviewCount     int     `json:"review_count" gorm:"default:0"`          // Number of reviews
	Rating          float64 `json:"rating" gorm:"default:0"`                // Average rating
	IsActive        bool    `json:"is_active" gorm:"default:true"`
}

// NewProduct creates a new product with base model
func NewProduct(name, description, category, imageURL string, price, actualPrice float64, stock, reviewCount, discountPercent int, rating float64) *Product {
	baseModel := base.NewBaseModel("PROD", hash.Medium)
	stockStatus := "In Stock"
	if stock <= 0 {
		stockStatus = "Out of Stock"
	}

	return &Product{
		BaseModel:       baseModel,
		Name:            name,
		Description:     description,
		Category:        category,
		Price:           price,
		ActualPrice:     actualPrice,
		DiscountPercent: discountPercent,
		ImageURL:        imageURL,
		Stock:           stock,
		StockStatus:     stockStatus,
		ReviewCount:     reviewCount,
		Rating:          rating,
		IsActive:        true,
	}
}

// TableName returns the table name for GORM
func (p *Product) TableName() string {
	return "products"
}
