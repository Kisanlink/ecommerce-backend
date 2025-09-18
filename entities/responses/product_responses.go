package responses

import (
	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// ProductImageResponse represents an image in product response
type ProductImageResponse struct {
	ID        string `json:"id"`
	ImageURL  string `json:"image_url"`
	ImageType string `json:"image_type"`
	AltText   string `json:"alt_text"`
	Caption   string `json:"caption"`
	SortOrder int    `json:"sort_order"`
	IsPrimary bool   `json:"is_primary"`
	IsActive  bool   `json:"is_active"`
	FileSize  int64  `json:"file_size"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	MimeType  string `json:"mime_type"`
}

// ProductInventoryResponse represents inventory information in product response
type ProductInventoryResponse struct {
	ID              string `json:"id"`
	CurrentStock    int    `json:"current_stock"`
	ReservedStock   int    `json:"reserved_stock"`
	AvailableStock  int    `json:"available_stock"`
	MinStockLevel   int    `json:"min_stock_level"`
	MaxStockLevel   int    `json:"max_stock_level"`
	ReorderPoint    int    `json:"reorder_point"`
	StockStatus     string `json:"stock_status"`
	LastRestockedAt string `json:"last_restocked_at,omitempty"`
	LastSoldAt      string `json:"last_sold_at,omitempty"`
	IsLowStock      bool   `json:"is_low_stock"`
	IsOutOfStock    bool   `json:"is_out_of_stock"`
	NeedsReorder    bool   `json:"needs_reorder"`
}

// ProductResponse represents a complete product response with images and inventory
type ProductResponse struct {
	// Basic product information
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	Price           float64 `json:"price"`
	ActualPrice     float64 `json:"actual_price"`
	DiscountPercent int     `json:"discount_percent"`
	ReviewCount     int     `json:"review_count"`
	Rating          float64 `json:"rating"`
	IsActive        bool    `json:"is_active"`

	// Timestamps
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	// Related data
	Images    []ProductImageResponse    `json:"images"`
	Inventory *ProductInventoryResponse `json:"inventory,omitempty"`

	// Computed fields
	ImageCount int  `json:"image_count"`
	HasImages  bool `json:"has_images"`
}

// ProductListResponse represents a product in list view (without full details)
type ProductListResponse struct {
	// Basic product information
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Category        string  `json:"category"`
	Price           float64 `json:"price"`
	ActualPrice     float64 `json:"actual_price"`
	DiscountPercent int     `json:"discount_percent"`
	ReviewCount     int     `json:"review_count"`
	Rating          float64 `json:"rating"`
	IsActive        bool    `json:"is_active"`

	// Primary image only for list view
	PrimaryImage *ProductImageResponse `json:"primary_image,omitempty"`

	// Basic inventory info for list view
	StockStatus    string `json:"stock_status"`
	AvailableStock int    `json:"available_stock"`
	IsInStock      bool   `json:"is_in_stock"`

	// Timestamps
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// ProductCreateResponse represents response after creating a product
type ProductCreateResponse struct {
	Product *ProductResponse `json:"product"`
	Message string           `json:"message"`
}

// ProductUpdateResponse represents response after updating a product
type ProductUpdateResponse struct {
	Product *ProductResponse `json:"product"`
	Message string           `json:"message"`
}

// ProductDeleteResponse represents response after deleting a product
type ProductDeleteResponse struct {
	Message string `json:"message"`
}

// ProductListPaginatedResponse represents paginated list of products
type ProductListPaginatedResponse struct {
	Products   []ProductListResponse `json:"products"`
	Pagination *base.PaginationInfo  `json:"pagination"`
	Message    string                `json:"message"`
}

// NewProductImageResponse creates a ProductImageResponse from Image model
func NewProductImageResponse(image *models.Image) ProductImageResponse {
	return ProductImageResponse{
		ID:        image.ID,
		ImageURL:  image.ImageURL,
		ImageType: string(image.ImageType),
		AltText:   image.AltText,
		Caption:   image.Caption,
		SortOrder: image.SortOrder,
		IsPrimary: image.IsPrimary,
		IsActive:  image.IsActive,
		FileSize:  image.FileSize,
		Width:     image.Width,
		Height:    image.Height,
		MimeType:  image.MimeType,
	}
}

// NewProductInventoryResponse creates a ProductInventoryResponse from Inventory model
func NewProductInventoryResponse(inventory *models.Inventory) ProductInventoryResponse {
	response := ProductInventoryResponse{
		ID:             inventory.ID,
		CurrentStock:   inventory.CurrentStock,
		ReservedStock:  inventory.ReservedStock,
		AvailableStock: inventory.GetAvailableStock(),
		MinStockLevel:  inventory.MinStockLevel,
		MaxStockLevel:  inventory.MaxStockLevel,
		ReorderPoint:   inventory.ReorderPoint,
		StockStatus:    string(inventory.StockStatus),
		IsLowStock:     inventory.IsLowStock(),
		IsOutOfStock:   inventory.IsOutOfStock(),
		NeedsReorder:   inventory.NeedsReorder(),
	}

	// Format timestamps
	if inventory.LastRestockedAt != nil {
		response.LastRestockedAt = inventory.LastRestockedAt.Format("2006-01-02T15:04:05Z")
	}
	if inventory.LastSoldAt != nil {
		response.LastSoldAt = inventory.LastSoldAt.Format("2006-01-02T15:04:05Z")
	}

	return response
}

// NewProductResponse creates a ProductResponse from Product model with related data
func NewProductResponse(product *models.Product, images []*models.Image, inventory *models.Inventory) ProductResponse {
	response := ProductResponse{
		ID:              product.ID,
		Name:            product.Name,
		Description:     product.Description,
		Category:        product.Category,
		Price:           product.Price,
		ActualPrice:     product.ActualPrice,
		DiscountPercent: product.DiscountPercent,
		ReviewCount:     product.ReviewCount,
		Rating:          product.Rating,
		IsActive:        product.IsActive,
		CreatedAt:       product.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       product.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		ImageCount:      len(images),
		HasImages:       len(images) > 0,
	}

	// Convert images
	response.Images = make([]ProductImageResponse, len(images))
	for i, image := range images {
		response.Images[i] = NewProductImageResponse(image)
	}

	// Add inventory if provided
	if inventory != nil {
		inventoryResp := NewProductInventoryResponse(inventory)
		response.Inventory = &inventoryResp
	}

	return response
}

// NewProductListResponse creates a ProductListResponse from Product model with primary image and basic inventory
func NewProductListResponse(product *models.Product, primaryImage *models.Image, inventory *models.Inventory) ProductListResponse {
	response := ProductListResponse{
		ID:              product.ID,
		Name:            product.Name,
		Description:     product.Description,
		Category:        product.Category,
		Price:           product.Price,
		ActualPrice:     product.ActualPrice,
		DiscountPercent: product.DiscountPercent,
		ReviewCount:     product.ReviewCount,
		Rating:          product.Rating,
		IsActive:        product.IsActive,
		CreatedAt:       product.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:       product.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Add primary image if available
	if primaryImage != nil {
		imageResp := NewProductImageResponse(primaryImage)
		response.PrimaryImage = &imageResp
	}

	// Add basic inventory info if available
	if inventory != nil {
		response.StockStatus = string(inventory.StockStatus)
		response.AvailableStock = inventory.GetAvailableStock()
		response.IsInStock = !inventory.IsOutOfStock()
	}

	return response
}

// NewProductCreateResponse creates a ProductCreateResponse
func NewProductCreateResponse(product *models.Product, images []*models.Image, inventory *models.Inventory) ProductCreateResponse {
	productResp := NewProductResponse(product, images, inventory)
	return ProductCreateResponse{
		Product: &productResp,
		Message: "Product created successfully",
	}
}

// NewProductUpdateResponse creates a ProductUpdateResponse
func NewProductUpdateResponse(product *models.Product, images []*models.Image, inventory *models.Inventory) ProductUpdateResponse {
	productResp := NewProductResponse(product, images, inventory)
	return ProductUpdateResponse{
		Product: &productResp,
		Message: "Product updated successfully",
	}
}

// NewProductDeleteResponse creates a ProductDeleteResponse
func NewProductDeleteResponse() ProductDeleteResponse {
	return ProductDeleteResponse{
		Message: "Product deleted successfully",
	}
}

// NewProductListPaginatedResponse creates a ProductListPaginatedResponse
func NewProductListPaginatedResponse(products []ProductListResponse, pagination *base.PaginationInfo) ProductListPaginatedResponse {
	return ProductListPaginatedResponse{
		Products:   products,
		Pagination: pagination,
		Message:    "Products retrieved successfully",
	}
}
