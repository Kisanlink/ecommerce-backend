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

	// FIXED: Removed redundant image columns - now handled by images table
	// FIXED: Removed inventory fields - now handled by inventory table

	// DEPRECATED: These fields are kept temporarily for backward compatibility
	// TODO: Remove these fields after migrating all services to use inventory system
	Stock       int    `json:"stock" gorm:"default:0"`                 // DEPRECATED: Use inventory service
	StockStatus string `json:"stock_status" gorm:"default:'In Stock'"` // DEPRECATED: Use inventory service
	ImageURL    string `json:"image_url"`                              // DEPRECATED: Use images table
	ImageURL1   string `json:"image_url_1"`                            // DEPRECATED: Use images table
	ImageURL2   string `json:"image_url_2"`                            // DEPRECATED: Use images table
	ImageURL3   string `json:"image_url_3"`                            // DEPRECATED: Use images table
	ImageURL4   string `json:"image_url_4"`                            // DEPRECATED: Use images table

	ReviewCount int     `json:"review_count" gorm:"default:0"` // Number of reviews
	Rating      float64 `json:"rating" gorm:"default:0"`       // Average rating
	IsActive    bool    `json:"is_active" gorm:"default:true"`
}

// NewProduct creates a new product with base model
func NewProduct(name, description, category string, price, actualPrice float64, reviewCount, discountPercent int, rating float64) *Product {
	baseModel := base.NewBaseModel("PROD", hash.Medium)

	return &Product{
		BaseModel:       baseModel,
		Name:            name,
		Description:     description,
		Category:        category,
		Price:           price,
		ActualPrice:     actualPrice,
		DiscountPercent: discountPercent,
		ReviewCount:     reviewCount,
		Rating:          rating,
		IsActive:        true,
	}
}

// TableName returns the table name for GORM
func (p *Product) TableName() string {
	return "products"
}

// NOTE: Image-related methods removed - images are now handled by the images table
// Use ImageRepository to manage product images
