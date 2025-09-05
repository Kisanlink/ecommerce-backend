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
	ImageURL        string  `json:"image_url"`                         // Primary image (for backward compatibility)
	ImageURL1       string  `json:"image_url_1"`                       // Additional image 1
	ImageURL2       string  `json:"image_url_2"`                       // Additional image 2
	ImageURL3       string  `json:"image_url_3"`                       // Additional image 3
	ImageURL4       string  `json:"image_url_4"`                       // Additional image 4
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

// GetImages returns all non-empty image URLs as a slice
func (p *Product) GetImages() []string {
	var images []string
	
	// Add primary image first if exists
	if p.ImageURL != "" {
		images = append(images, p.ImageURL)
	}
	
	// Add additional images
	if p.ImageURL1 != "" {
		images = append(images, p.ImageURL1)
	}
	if p.ImageURL2 != "" {
		images = append(images, p.ImageURL2)
	}
	if p.ImageURL3 != "" {
		images = append(images, p.ImageURL3)
	}
	if p.ImageURL4 != "" {
		images = append(images, p.ImageURL4)
	}
	
	return images
}

// SetImages sets up to 4 image URLs from a slice
func (p *Product) SetImages(imageUrls []string) {
	// Reset all image fields
	p.ImageURL = ""
	p.ImageURL1 = ""
	p.ImageURL2 = ""
	p.ImageURL3 = ""
	p.ImageURL4 = ""
	
	// Set images based on slice length
	if len(imageUrls) > 0 {
		p.ImageURL = imageUrls[0]
	}
	if len(imageUrls) > 1 {
		p.ImageURL1 = imageUrls[1]
	}
	if len(imageUrls) > 2 {
		p.ImageURL2 = imageUrls[2]
	}
	if len(imageUrls) > 3 {
		p.ImageURL3 = imageUrls[3]
	}
	if len(imageUrls) > 4 {
		p.ImageURL4 = imageUrls[4]
	}
}
