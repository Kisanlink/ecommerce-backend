package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// ImageType represents the type of image
type ImageType string

const (
	ImageTypeProduct   ImageType = "product"
	ImageTypeThumbnail ImageType = "thumbnail"
	ImageTypeGallery   ImageType = "gallery"
	ImageTypeBanner    ImageType = "banner"
	ImageTypeCategory  ImageType = "category"
	ImageTypeVideo     ImageType = "video"
)

// Image represents an image in the system
type Image struct {
	*base.BaseModel `gorm:"embedded"`
	ProductID       string    `json:"product_id" gorm:"not null;index"`
	ImageURL        string    `json:"image_url" gorm:"not null"`
	R2Key           string    `json:"r2_key" gorm:"column:s3_key"` // R2 object key for direct access (column kept as s3_key for backward compatibility)
	ImageType       ImageType `json:"image_type" gorm:"default:'product'"`
	AltText         string    `json:"alt_text" gorm:"size:255"`
	Caption         string    `json:"caption" gorm:"size:500"`
	SortOrder       int       `json:"sort_order" gorm:"default:0"`
	IsActive        bool      `json:"is_active" gorm:"default:true"`
	IsPrimary       bool      `json:"is_primary" gorm:"default:false"`
	FileSize        int64     `json:"file_size"`                 // Size in bytes
	Width           int       `json:"width"`                     // Image width in pixels
	Height          int       `json:"height"`                    // Image height in pixels
	MimeType        string    `json:"mime_type" gorm:"size:100"` // e.g., image/jpeg, image/png
}

// NewImage creates a new image with base model
func NewImage(productID, imageURL, r2Key string, imageType ImageType) *Image {
	baseModel := base.NewBaseModel("IMG", hash.Large)
	return &Image{
		BaseModel: baseModel,
		ProductID: productID,
		ImageURL:  imageURL,
		R2Key:     r2Key,
		ImageType: imageType,
		IsActive:  true,
		IsPrimary: false,
		SortOrder: 0,
	}
}

// NewPrimaryImage creates a new primary image
func NewPrimaryImage(productID, imageURL, r2Key string) *Image {
	image := NewImage(productID, imageURL, r2Key, ImageTypeProduct)
	image.IsPrimary = true
	image.SortOrder = 0
	return image
}

// TableName returns the table name for GORM
func (i *Image) TableName() string {
	return "images"
}

// IsValidImageType checks if the image type is valid
func (it ImageType) IsValidImageType() bool {
	switch it {
	case ImageTypeProduct, ImageTypeThumbnail, ImageTypeGallery, ImageTypeBanner, ImageTypeCategory, ImageTypeVideo:
		return true
	default:
		return false
	}
}

// SetAsPrimary sets this image as the primary image for the product
func (i *Image) SetAsPrimary() {
	i.IsPrimary = true
	i.SortOrder = 0
}

// SetSortOrder sets the sort order for the image
func (i *Image) SetSortOrder(order int) {
	i.SortOrder = order
}
