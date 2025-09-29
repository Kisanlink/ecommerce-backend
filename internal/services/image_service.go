package services

import (
	"context"
	"fmt"
	"sort"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"go.uber.org/zap"
)

// ImageServiceInterface defines the contract for image operations
type ImageServiceInterface interface {
	CreateImage(ctx context.Context, productID, imageURL string, imageType models.ImageType, altText, caption string) (*models.Image, error)
	GetImageByID(ctx context.Context, imageID string) (*models.Image, error)
	UpdateImage(ctx context.Context, image *models.Image) error
	DeleteImage(ctx context.Context, imageID string) error
	GetImagesByProductID(ctx context.Context, productID string) ([]*models.Image, error)
	GetActiveImagesByProductID(ctx context.Context, productID string) ([]*models.Image, error)
	GetPrimaryImageByProductID(ctx context.Context, productID string) (*models.Image, error)
	GetImagesByProductIDAndType(ctx context.Context, productID string, imageType models.ImageType) ([]*models.Image, error)
	SetPrimaryImage(ctx context.Context, productID, imageID string) error
	UpdateImageOrder(ctx context.Context, imageID string, sortOrder int) error
	AddMultipleImages(ctx context.Context, productID string, imageURLs []string, imageType models.ImageType) ([]*models.Image, error)
	DeleteImagesByProductID(ctx context.Context, productID string) error
	ReorderImages(ctx context.Context, productID string, imageOrders map[string]int) error
}

// ImageService implements image business logic
type ImageService struct {
	imageRepo   repositories.ImageRepositoryInterface
	productRepo repositories.ProductRepositoryInterface
	logger      *zap.Logger
}

// NewImageService creates a new image service
func NewImageService(imageRepo repositories.ImageRepositoryInterface, productRepo repositories.ProductRepositoryInterface, logger *zap.Logger) ImageServiceInterface {
	return &ImageService{
		imageRepo:   imageRepo,
		productRepo: productRepo,
		logger:      logger,
	}
}

// CreateImage creates a new image for a product
func (s *ImageService) CreateImage(ctx context.Context, productID, imageURL string, imageType models.ImageType, altText, caption string) (*models.Image, error) {
	// Validate that the product exists
	var product models.Product
	_, err := s.productRepo.GetByID(ctx, productID, &product)
	if err != nil {
		return nil, fmt.Errorf("product not found: %w", err)
	}

	// Create the image
	// For legacy image URLs, we'll use the URL as the S3 key
	image := models.NewImage(productID, imageURL, imageURL, imageType)
	image.AltText = altText
	image.Caption = caption

	// If this is the first image for the product, make it primary
	existingImages, err := s.imageRepo.GetImagesByProductID(ctx, productID)
	if err != nil {
		s.logger.Warn("Failed to check existing images", zap.Error(err))
	} else if len(existingImages) == 0 {
		image.SetAsPrimary()
	}

	// Set sort order based on existing images
	if len(existingImages) > 0 {
		image.SetSortOrder(len(existingImages))
	}

	err = s.imageRepo.Create(ctx, image)
	if err != nil {
		return nil, fmt.Errorf("failed to create image: %w", err)
	}

	s.logger.Info("Image created successfully",
		zap.String("image_id", image.ID),
		zap.String("product_id", productID),
		zap.String("image_type", string(imageType)))

	return image, nil
}

// GetImageByID retrieves an image by its ID
func (s *ImageService) GetImageByID(ctx context.Context, imageID string) (*models.Image, error) {
	var image models.Image
	result, err := s.imageRepo.GetByID(ctx, imageID, &image)
	if err != nil {
		return nil, fmt.Errorf("failed to get image: %w", err)
	}
	return result, nil
}

// UpdateImage updates an existing image
func (s *ImageService) UpdateImage(ctx context.Context, image *models.Image) error {
	if image == nil || image.ID == "" {
		return fmt.Errorf("invalid image to update")
	}

	err := s.imageRepo.Update(ctx, image)
	if err != nil {
		return fmt.Errorf("failed to update image: %w", err)
	}

	s.logger.Info("Image updated successfully", zap.String("image_id", image.ID))
	return nil
}

// DeleteImage deletes an image
func (s *ImageService) DeleteImage(ctx context.Context, imageID string) error {
	// Get the image first to check if it's primary
	image, err := s.GetImageByID(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to get image for deletion: %w", err)
	}

	// Delete the image
	err = s.imageRepo.Delete(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}

	// If the deleted image was primary, set another image as primary
	if image.IsPrimary {
		remainingImages, err := s.imageRepo.GetImagesByProductID(ctx, image.ProductID)
		if err == nil && len(remainingImages) > 0 {
			// Set the first remaining image as primary
			err = s.imageRepo.SetPrimaryImage(ctx, image.ProductID, remainingImages[0].ID)
			if err != nil {
				s.logger.Warn("Failed to set new primary image after deletion",
					zap.String("product_id", image.ProductID),
					zap.Error(err))
			}
		}
	}

	s.logger.Info("Image deleted successfully", zap.String("image_id", imageID))
	return nil
}

// GetImagesByProductID retrieves all images for a specific product
func (s *ImageService) GetImagesByProductID(ctx context.Context, productID string) ([]*models.Image, error) {
	images, err := s.imageRepo.GetImagesByProductID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get images by product ID: %w", err)
	}

	// Sort images by sort order and creation date
	sort.Slice(images, func(i, j int) bool {
		if images[i].SortOrder != images[j].SortOrder {
			return images[i].SortOrder < images[j].SortOrder
		}
		return images[i].CreatedAt.Before(images[j].CreatedAt)
	})

	return images, nil
}

// GetActiveImagesByProductID retrieves only active images for a specific product
func (s *ImageService) GetActiveImagesByProductID(ctx context.Context, productID string) ([]*models.Image, error) {
	images, err := s.imageRepo.GetActiveImagesByProductID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active images by product ID: %w", err)
	}
	return images, nil
}

// GetPrimaryImageByProductID retrieves the primary image for a specific product
func (s *ImageService) GetPrimaryImageByProductID(ctx context.Context, productID string) (*models.Image, error) {
	image, err := s.imageRepo.GetPrimaryImageByProductID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("failed to get primary image: %w", err)
	}
	return image, nil
}

// GetImagesByProductIDAndType retrieves images for a specific product and type
func (s *ImageService) GetImagesByProductIDAndType(ctx context.Context, productID string, imageType models.ImageType) ([]*models.Image, error) {
	images, err := s.imageRepo.GetImagesByProductIDAndType(ctx, productID, imageType)
	if err != nil {
		return nil, fmt.Errorf("failed to get images by product ID and type: %w", err)
	}

	// Sort images by sort order and creation date
	sort.Slice(images, func(i, j int) bool {
		if images[i].SortOrder != images[j].SortOrder {
			return images[i].SortOrder < images[j].SortOrder
		}
		return images[i].CreatedAt.Before(images[j].CreatedAt)
	})

	return images, nil
}

// SetPrimaryImage sets an image as the primary image for a product
func (s *ImageService) SetPrimaryImage(ctx context.Context, productID, imageID string) error {
	// Verify the image belongs to the product
	image, err := s.GetImageByID(ctx, imageID)
	if err != nil {
		return fmt.Errorf("failed to get image: %w", err)
	}

	if image.ProductID != productID {
		return fmt.Errorf("image does not belong to the specified product")
	}

	err = s.imageRepo.SetPrimaryImage(ctx, productID, imageID)
	if err != nil {
		return fmt.Errorf("failed to set primary image: %w", err)
	}

	s.logger.Info("Primary image set successfully",
		zap.String("image_id", imageID),
		zap.String("product_id", productID))

	return nil
}

// UpdateImageOrder updates the sort order of an image
func (s *ImageService) UpdateImageOrder(ctx context.Context, imageID string, sortOrder int) error {
	err := s.imageRepo.UpdateImageOrder(ctx, imageID, sortOrder)
	if err != nil {
		return fmt.Errorf("failed to update image order: %w", err)
	}

	s.logger.Info("Image order updated successfully",
		zap.String("image_id", imageID),
		zap.Int("sort_order", sortOrder))

	return nil
}

// AddMultipleImages adds multiple images to a product
func (s *ImageService) AddMultipleImages(ctx context.Context, productID string, imageURLs []string, imageType models.ImageType) ([]*models.Image, error) {
	var createdImages []*models.Image

	for i, imageURL := range imageURLs {
		image, err := s.CreateImage(ctx, productID, imageURL, imageType, "", "")
		if err != nil {
			// Log error but continue with other images
			s.logger.Error("Failed to create image",
				zap.String("product_id", productID),
				zap.String("image_url", imageURL),
				zap.Error(err))
			continue
		}

		// Set sort order based on position in the array
		image.SetSortOrder(i)
		err = s.UpdateImage(ctx, image)
		if err != nil {
			s.logger.Warn("Failed to update image sort order",
				zap.String("image_id", image.ID),
				zap.Error(err))
		}

		createdImages = append(createdImages, image)
	}

	s.logger.Info("Multiple images added successfully",
		zap.String("product_id", productID),
		zap.Int("created_count", len(createdImages)),
		zap.Int("requested_count", len(imageURLs)))

	return createdImages, nil
}

// DeleteImagesByProductID deletes all images for a specific product
func (s *ImageService) DeleteImagesByProductID(ctx context.Context, productID string) error {
	err := s.imageRepo.DeleteImagesByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to delete images by product ID: %w", err)
	}

	s.logger.Info("All images deleted for product", zap.String("product_id", productID))
	return nil
}

// ReorderImages reorders images for a product
func (s *ImageService) ReorderImages(ctx context.Context, productID string, imageOrders map[string]int) error {
	for imageID, sortOrder := range imageOrders {
		err := s.UpdateImageOrder(ctx, imageID, sortOrder)
		if err != nil {
			s.logger.Error("Failed to update image order",
				zap.String("image_id", imageID),
				zap.Int("sort_order", sortOrder),
				zap.Error(err))
			// Continue with other images even if one fails
		}
	}

	s.logger.Info("Images reordered successfully", zap.String("product_id", productID))
	return nil
}
