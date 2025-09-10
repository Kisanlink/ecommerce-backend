package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
)

// ImageRepositoryInterface defines the contract for image operations
type ImageRepositoryInterface interface {
	Create(ctx context.Context, image *models.Image) error
	GetByID(ctx context.Context, id string, image *models.Image) (*models.Image, error)
	Update(ctx context.Context, image *models.Image) error
	Delete(ctx context.Context, id string) error
	GetImagesByProductID(ctx context.Context, productID string) ([]*models.Image, error)
	GetPrimaryImageByProductID(ctx context.Context, productID string) (*models.Image, error)
	GetImagesByProductIDAndType(ctx context.Context, productID string, imageType models.ImageType) ([]*models.Image, error)
	SetPrimaryImage(ctx context.Context, productID, imageID string) error
	UpdateImageOrder(ctx context.Context, imageID string, sortOrder int) error
	GetActiveImagesByProductID(ctx context.Context, productID string) ([]*models.Image, error)
	DeleteImagesByProductID(ctx context.Context, productID string) error
}

// ImageRepository handles image database operations
type ImageRepository struct {
	*base.BaseFilterableRepository[*models.Image]
	dbManager db.DBManager
}

// NewImageRepository creates a new image repository
func NewImageRepository(dbManager db.DBManager) ImageRepositoryInterface {
	repo := &ImageRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Image](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Create creates a new image in the database
func (r *ImageRepository) Create(ctx context.Context, image *models.Image) error {
	return r.dbManager.Create(ctx, image)
}

// GetByID retrieves an image by its ID
func (r *ImageRepository) GetByID(ctx context.Context, id string, image *models.Image) (*models.Image, error) {
	err := r.dbManager.GetByID(ctx, id, image)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	return image, nil
}

// Update updates an existing image
func (r *ImageRepository) Update(ctx context.Context, image *models.Image) error {
	return r.dbManager.Update(ctx, image)
}

// Delete deletes an image by ID
func (r *ImageRepository) Delete(ctx context.Context, id string) error {
	var image models.Image
	return r.dbManager.Delete(ctx, id, &image)
}

// GetImagesByProductID retrieves all images for a specific product
func (r *ImageRepository) GetImagesByProductID(ctx context.Context, productID string) ([]*models.Image, error) {
	var images []*models.Image
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "product_id", Operator: base.OpEqual, Value: productID},
			},
			Logic: base.LogicAnd,
		},
	}
	err := r.dbManager.List(ctx, filter, &images)
	if err != nil {
		return nil, fmt.Errorf("failed to get images by product ID: %w", err)
	}
	return images, nil
}

// GetPrimaryImageByProductID retrieves the primary image for a specific product
func (r *ImageRepository) GetPrimaryImageByProductID(ctx context.Context, productID string) (*models.Image, error) {
	var images []*models.Image
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "product_id", Operator: base.OpEqual, Value: productID},
				{Field: "is_primary", Operator: base.OpEqual, Value: true},
			},
			Logic: base.LogicAnd,
		},
		Limit: 1,
	}
	err := r.dbManager.List(ctx, filter, &images)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary image: %w", err)
	}
	if len(images) == 0 {
		return nil, fmt.Errorf("no primary image found")
	}
	return images[0], nil
}

// GetImagesByProductIDAndType retrieves images for a specific product and type
func (r *ImageRepository) GetImagesByProductIDAndType(ctx context.Context, productID string, imageType models.ImageType) ([]*models.Image, error) {
	var images []*models.Image
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "product_id", Operator: base.OpEqual, Value: productID},
				{Field: "image_type", Operator: base.OpEqual, Value: imageType},
			},
			Logic: base.LogicAnd,
		},
	}
	err := r.dbManager.List(ctx, filter, &images)
	if err != nil {
		return nil, fmt.Errorf("failed to get images by product ID and type: %w", err)
	}
	return images, nil
}

// SetPrimaryImage sets an image as the primary image for a product
func (r *ImageRepository) SetPrimaryImage(ctx context.Context, productID, imageID string) error {
	// Get all images for the product
	images, err := r.GetImagesByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get images for product: %w", err)
	}

	// Unset all primary images
	for _, img := range images {
		if img.IsPrimary {
			img.IsPrimary = false
			if err := r.Update(ctx, img); err != nil {
				return fmt.Errorf("failed to unset primary image: %w", err)
			}
		}
	}

	// Set the specified image as primary
	var targetImage models.Image
	if err := r.dbManager.GetByID(ctx, imageID, &targetImage); err != nil {
		return fmt.Errorf("failed to get target image: %w", err)
	}
	targetImage.IsPrimary = true
	if err := r.Update(ctx, &targetImage); err != nil {
		return fmt.Errorf("failed to set primary image: %w", err)
	}

	return nil
}

// UpdateImageOrder updates the sort order of an image
func (r *ImageRepository) UpdateImageOrder(ctx context.Context, imageID string, sortOrder int) error {
	var image models.Image
	if err := r.dbManager.GetByID(ctx, imageID, &image); err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}
	image.SortOrder = sortOrder
	return r.Update(ctx, &image)
}

// GetActiveImagesByProductID retrieves only active images for a specific product
func (r *ImageRepository) GetActiveImagesByProductID(ctx context.Context, productID string) ([]*models.Image, error) {
	var images []*models.Image
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "product_id", Operator: base.OpEqual, Value: productID},
				{Field: "is_active", Operator: base.OpEqual, Value: true},
			},
			Logic: base.LogicAnd,
		},
	}
	err := r.dbManager.List(ctx, filter, &images)
	if err != nil {
		return nil, fmt.Errorf("failed to get active images by product ID: %w", err)
	}
	return images, nil
}

// DeleteImagesByProductID deletes all images for a specific product
func (r *ImageRepository) DeleteImagesByProductID(ctx context.Context, productID string) error {
	images, err := r.GetImagesByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get images for product: %w", err)
	}

	for _, img := range images {
		if err := r.Delete(ctx, img.ID); err != nil {
			return fmt.Errorf("failed to delete image %s: %w", img.ID, err)
		}
	}

	return nil
}
