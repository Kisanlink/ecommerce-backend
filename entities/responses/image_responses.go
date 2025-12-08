package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// ImageResponse represents a single image response
type ImageResponse struct {
	ID        string           `json:"id"`
	ProductID string           `json:"product_id"`
	ImageURL  string           `json:"image_url"`
	R2Key     string           `json:"r2_key"`
	ImageType models.ImageType `json:"image_type"`
	AltText   string           `json:"alt_text"`
	Caption   string           `json:"caption"`
	SortOrder int              `json:"sort_order"`
	IsActive  bool             `json:"is_active"`
	IsPrimary bool             `json:"is_primary"`
	FileSize  int64            `json:"file_size"`
	Width     int              `json:"width"`
	Height    int              `json:"height"`
	MimeType  string           `json:"mime_type"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// ProductImagesResponse represents response for product images
type ProductImagesResponse struct {
	ProductID string          `json:"product_id"`
	Images    []ImageResponse `json:"images"`
	Count     int             `json:"count"`
}

// ImageUploadResponse represents response for image upload
type ImageUploadResponse struct {
	ImageID   string `json:"image_id"`
	ProductID string `json:"product_id"`
	ImageURL  string `json:"image_url"`
	R2Key     string `json:"r2_key"`
	IsPrimary bool   `json:"is_primary"`
}

// NewImageResponse creates a new image response from model
func NewImageResponse(image *models.Image) ImageResponse {
	return ImageResponse{
		ID:        image.ID,
		ProductID: image.ProductID,
		ImageURL:  image.ImageURL,
		R2Key:     image.R2Key,
		ImageType: image.ImageType,
		AltText:   image.AltText,
		Caption:   image.Caption,
		SortOrder: image.SortOrder,
		IsActive:  image.IsActive,
		IsPrimary: image.IsPrimary,
		FileSize:  image.FileSize,
		Width:     image.Width,
		Height:    image.Height,
		MimeType:  image.MimeType,
		CreatedAt: image.CreatedAt,
		UpdatedAt: image.UpdatedAt,
	}
}

// NewProductImagesResponse creates a new product images response
func NewProductImagesResponse(productID string, images []*models.Image) ProductImagesResponse {
	imageResponses := make([]ImageResponse, len(images))
	for i, image := range images {
		imageResponses[i] = NewImageResponse(image)
	}

	return ProductImagesResponse{
		ProductID: productID,
		Images:    imageResponses,
		Count:     len(images),
	}
}

// NewImageUploadResponse creates a new image upload response
func NewImageUploadResponse(image *models.Image, r2Key string) ImageUploadResponse {
	return ImageUploadResponse{
		ImageID:   image.ID,
		ProductID: image.ProductID,
		ImageURL:  image.ImageURL,
		R2Key:     r2Key,
		IsPrimary: image.IsPrimary,
	}
}
