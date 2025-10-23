package requests

import "github.com/Kisanlink/kisanlink-db/pkg/base"

// UploadProductImageRequest represents request for uploading a product image
type UploadProductImageRequest struct {
	base.CreateRequest
	ProductID string `form:"product_id" validate:"required"`
	AltText   string `form:"alt_text"`
	Caption   string `form:"caption"`
}

// DeleteProductImageRequest represents request for deleting a product image
type DeleteProductImageRequest struct {
	base.DeleteRequest
	ProductID string `json:"product_id" validate:"required"`
	ImageID   string `json:"image_id" validate:"required"`
}

// SetPrimaryImageRequest represents request for setting a primary image
type SetPrimaryImageRequest struct {
	base.UpdateRequest
	ProductID string `json:"product_id" validate:"required"`
	ImageID   string `json:"image_id" validate:"required"`
}

// UpdateImageOrderRequest represents request for updating image order
type UpdateImageOrderRequest struct {
	base.UpdateRequest
	ProductID string `json:"product_id" validate:"required"`
	ImageID   string `json:"image_id" validate:"required"`
	SortOrder int    `json:"sort_order" validate:"min=0"`
}

// GetProductImagesRequest represents request for getting product images
type GetProductImagesRequest struct {
	base.GetByIDRequest
	ProductID string `json:"product_id" validate:"required"`
}

