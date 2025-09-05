package requests

import "github.com/Kisanlink/kisanlink-db/pkg/base"

// CreateProductRequest represents request for creating a product
type CreateProductRequest struct {
	base.CreateRequest
	Name            string   `json:"name" validate:"required"`
	Description     string   `json:"description" validate:"required"`
	Category        string   `json:"category" validate:"required"`
	Price           float64  `json:"price" validate:"required,min=0"`
	ActualPrice     float64  `json:"actual_price" validate:"required,min=0"`
	DiscountPercent int      `json:"discount_percent" validate:"min=0,max=100"`
	ImageURL        string   `json:"image_url,omitempty"`        // Primary image (backward compatibility)
	ImageURLs       []string `json:"image_urls,omitempty"`       // Array of up to 4 image URLs
	Stock           int      `json:"stock" validate:"min=0"`
	ReviewCount     int      `json:"review_count" validate:"min=0"`
	Rating          float64  `json:"rating" validate:"min=0,max=5"`
}

// UpdateProductRequest represents request for updating a product
type UpdateProductRequest struct {
	base.UpdateRequest
	Name            string   `json:"name" validate:"required"`
	Description     string   `json:"description" validate:"required"`
	Category        string   `json:"category" validate:"required"`
	Price           float64  `json:"price" validate:"required,min=0"`
	ActualPrice     float64  `json:"actual_price" validate:"required,min=0"`
	DiscountPercent int      `json:"discount_percent" validate:"min=0,max=100"`
	ImageURL        string   `json:"image_url,omitempty"`        // Primary image (backward compatibility)
	ImageURLs       []string `json:"image_urls,omitempty"`       // Array of up to 4 image URLs
	Stock           int      `json:"stock" validate:"min=0"`
	ReviewCount     int      `json:"review_count" validate:"min=0"`
	Rating          float64  `json:"rating" validate:"min=0,max=5"`
}

// GetProductRequest represents request for getting a product by ID
type GetProductRequest struct {
	base.GetByIDRequest
}

// ListProductsRequest represents request for listing products with filters
type ListProductsRequest struct {
	base.ListRequest
	Category string  `json:"category,omitempty"`
	MinPrice float64 `json:"min_price,omitempty"`
	MaxPrice float64 `json:"max_price,omitempty"`
	IsActive *bool   `json:"is_active,omitempty"`
	Search   string  `json:"search,omitempty"`
	InStock  *bool   `json:"in_stock,omitempty"`
}

// DeleteProductRequest represents request for deleting a product
type DeleteProductRequest struct {
	base.DeleteRequest
}

// GetEffectiveImages returns the effective image URLs, prioritizing ImageURLs array over single ImageURL
func (r *CreateProductRequest) GetEffectiveImages() []string {
	if len(r.ImageURLs) > 0 {
		// Limit to maximum of 4 images
		if len(r.ImageURLs) > 4 {
			return r.ImageURLs[:4]
		}
		return r.ImageURLs
	}
	
	// Fallback to single ImageURL for backward compatibility
	if r.ImageURL != "" {
		return []string{r.ImageURL}
	}
	
	return []string{}
}

// GetEffectiveImages returns the effective image URLs, prioritizing ImageURLs array over single ImageURL
func (r *UpdateProductRequest) GetEffectiveImages() []string {
	if len(r.ImageURLs) > 0 {
		// Limit to maximum of 4 images
		if len(r.ImageURLs) > 4 {
			return r.ImageURLs[:4]
		}
		return r.ImageURLs
	}
	
	// Fallback to single ImageURL for backward compatibility
	if r.ImageURL != "" {
		return []string{r.ImageURL}
	}
	
	return []string{}
}
