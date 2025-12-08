package services

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// ProductService handles product business logic
type ProductService struct {
	repoManager *repositories.RepositoryManager
}

// NewProductService creates a new product service
func NewProductService(repoManager *repositories.RepositoryManager) *ProductService {
	service := &ProductService{
		repoManager: repoManager,
	}

	// Initialize the ID generator with existing product IDs
	service.initializeIDGenerator()

	return service
}

// initializeIDGenerator initializes the global ID generator with existing product IDs
func (s *ProductService) initializeIDGenerator() {
	ctx := context.Background()

	// Get all existing products to extract their IDs
	filter := base.NewFilterBuilder().
		Page(1, 10000). // Get all products
		Build()

	result, err := s.repoManager.ProductRepository.Find(ctx, filter)
	if err != nil {
		// If we can't get existing products, start from 0
		fmt.Printf("Warning: Could not initialize ID generator: %v\n", err)
		return
	}

	// Extract all existing IDs
	existingIDs := make([]string, len(result))
	for i, product := range result {
		existingIDs[i] = product.GetID()
	}

	// Initialize the global generator with existing IDs
	hash.InitializeGlobalCountersFromDatabase("PROD", existingIDs, hash.Medium)
	fmt.Printf("Initialized ID generator with %d existing product IDs\n", len(existingIDs))
}

// CreateProduct creates a new product
func (s *ProductService) CreateProduct(ctx context.Context, req *requests.CreateProductRequest) (*base.BaseResponse, error) {
	// Create product model (without images and stock)
	product := models.NewProduct(
		req.Name,
		req.Description,
		req.Category,
		req.Price,
		req.ActualPrice,
		req.ReviewCount,
		req.DiscountPercent,
		req.Rating,
	)

	// Set created by from request context
	product.SetCreatedBy(req.GetUserID())

	// Save product to database
	if err := s.repoManager.ProductRepository.Create(ctx, product); err != nil {
		return base.NewErrorResponse("Failed to create product",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create images if provided
	var images []*models.Image
	imageUrls := req.GetEffectiveImages()
	for i, imageURL := range imageUrls {
		// For legacy image URLs, we'll use the URL as the S3 key
		// In a real scenario, these would be uploaded to S3 first
		image := models.NewImage(product.ID, imageURL, imageURL, models.ImageTypeProduct)
		image.SetCreatedBy(req.GetUserID())

		// Set first image as primary
		if i == 0 {
			image.SetAsPrimary()
		}

		// Set sort order
		image.SetSortOrder(i)

		if err := s.repoManager.ImageRepository.Create(ctx, image); err != nil {
			// Log error but don't fail the product creation
			fmt.Printf("Warning: Failed to create image %d for product %s: %v\n", i, product.ID, err)
		} else {
			images = append(images, image)
		}
	}

	// Get inventory configuration from environment or request
	config.LoadEnv()
	inventoryConfig := config.GetInventoryEnvConfig()

	// Get inventory defaults from request or use environment defaults
	minStock, maxStock, reorderPoint := req.GetInventoryDefaults(
		inventoryConfig.DefaultMinStockLevel,
		inventoryConfig.DefaultMaxStockLevel,
		inventoryConfig.DefaultReorderPoint,
	)

	// Create inventory record for the new product
	inventory := models.NewInventory(
		product.ID,
		0,            // Initial stock is 0
		minStock,     // Min stock level (configurable)
		maxStock,     // Max stock level (configurable)
		reorderPoint, // Reorder point (configurable)
	)
	inventory.SetCreatedBy(req.GetUserID())

	if err := s.repoManager.InventoryRepository.Create(ctx, inventory); err != nil {
		// Log error but don't fail the product creation
		fmt.Printf("Warning: Failed to create inventory for product %s: %v\n", product.ID, err)
		inventory = nil
	}

	// Create response with product, images, and inventory
	productResp := responses.NewProductCreateResponse(product, images, inventory)
	return base.NewSuccessResponse("Product created successfully", productResp), nil
}

// GetProductByID retrieves a product by ID
func (s *ProductService) GetProductByID(ctx context.Context, req *requests.GetProductRequest) (*base.BaseResponse, error) {
	// Build filter for active products only (not soft deleted)
	filter := base.NewFilterBuilder().
		Where("id", base.OpEqual, req.ID).
		Where("is_active", base.OpEqual, true).
		Where("deleted_at", base.OpIsNull, nil).
		Build()

	product, err := s.repoManager.ProductRepository.FindOne(ctx, filter)
	if err != nil {
		return base.NewErrorResponse("Product not found",
			base.NewNotFoundError("Product", req.ID)), nil
	}

	// Get images from the images table
	images, err := s.repoManager.ImageRepository.GetActiveImagesByProductID(ctx, req.ID)
	if err != nil {
		// Log error but don't fail the request - product can still be returned without images
		fmt.Printf("Warning: Failed to get images for product %s: %v\n", req.ID, err)
		images = []*models.Image{}
	}

	// Get inventory data from inventory table
	var inventory *models.Inventory
	inventory, err = s.repoManager.InventoryRepository.GetByProductID(ctx, req.ID)
	if err != nil {
		// Log warning but don't fail - product can still be returned without inventory
		fmt.Printf("Warning: Failed to get inventory for product %s: %v\n", req.ID, err)
		inventory = nil
	}

	// Create product response with images and inventory
	productResp := responses.NewProductResponse(product, images, inventory)
	return base.NewSuccessResponse("Product retrieved successfully", productResp), nil
}

// GetProducts retrieves products with filtering and pagination
func (s *ProductService) GetProducts(ctx context.Context, req *requests.ListProductsRequest) (*base.PaginatedResponse, error) {
	// Build base filter for active products (not soft deleted)
	filterBuilder := base.NewFilterBuilder().
		Where("is_active", base.OpEqual, true).
		Where("deleted_at", base.OpIsNull, nil).
		Sort("created_at", "desc").
		Page(req.Page, req.PerPage)

	// Add optional filters
	if req.Category != "" {
		filterBuilder.Where("category", base.OpEqual, req.Category)
	}

	if req.MinPrice > 0 {
		filterBuilder.Where("price", base.OpGreaterEqual, req.MinPrice)
	}

	if req.MaxPrice > 0 {
		filterBuilder.Where("price", base.OpLessEqual, req.MaxPrice)
	}

	if req.Search != "" {
		// Create conditions for each field to search
		nameCondition := base.FilterCondition{
			Field:    "name",
			Operator: base.OpContains,
			Value:    req.Search,
		}
		descCondition := base.FilterCondition{
			Field:    "description",
			Operator: base.OpContains,
			Value:    req.Search,
		}
		categoryCondition := base.FilterCondition{
			Field:    "category",
			Operator: base.OpContains,
			Value:    req.Search,
		}

		// Add OR group for search conditions
		filterBuilder.Or(nameCondition, descCondition, categoryCondition)
	}

	filter := filterBuilder.Build()

	// Execute query
	products, err := s.repoManager.ProductRepository.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve products: %w", err)
	}

	// Get total count
	count, err := s.repoManager.ProductRepository.CountWithFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count products: %w", err)
	}

	// Convert products to list responses with images and inventory
	var productListResponses []responses.ProductListResponse
	for _, product := range products {
		// Get primary image
		var primaryImage *models.Image
		primaryImage, err = s.repoManager.ImageRepository.GetPrimaryImageByProductID(ctx, product.ID)
		if err != nil {
			// No primary image found, that's okay
			primaryImage = nil
		}

		// Get inventory data from inventory table
		var inventory *models.Inventory
		inventory, err = s.repoManager.InventoryRepository.GetByProductID(ctx, product.ID)
		if err != nil {
			// Log warning but don't fail - product can still be returned without inventory
			fmt.Printf("Warning: Failed to get inventory for product %s: %v\n", product.ID, err)
			inventory = nil
		}

		productListResp := responses.NewProductListResponse(product, primaryImage, inventory)
		productListResponses = append(productListResponses, productListResp)
	}

	pagination := base.NewPaginationInfo(req.Page, req.PerPage, int(count))
	productListPaginatedResp := responses.NewProductListPaginatedResponse(productListResponses, pagination)
	return base.NewPaginatedResponse("Products retrieved successfully", productListPaginatedResp, pagination), nil
}

// UpdateProduct updates an existing product
func (s *ProductService) UpdateProduct(ctx context.Context, req *requests.UpdateProductRequest) (*base.BaseResponse, error) {
	// Get existing product
	existing, err := s.repoManager.ProductRepository.GetByID(ctx, req.ID, &models.Product{})
	if err != nil {
		return base.NewErrorResponse("Product not found",
			base.NewNotFoundError("Product", req.ID)), nil
	}

	// Update basic fields
	existing.Name = req.Name
	existing.Description = req.Description
	existing.Category = req.Category
	existing.Price = req.Price
	existing.ActualPrice = req.ActualPrice
	existing.DiscountPercent = req.DiscountPercent
	existing.ReviewCount = req.ReviewCount
	existing.Rating = req.Rating
	existing.SetUpdatedBy(req.GetUserID())

	// Save product changes
	if err := s.repoManager.ProductRepository.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	// Handle image updates
	imageUrls := req.GetEffectiveImages()

	// Delete existing images
	if err := s.repoManager.ImageRepository.DeleteImagesByProductID(ctx, req.ID); err != nil {
		fmt.Printf("Warning: Failed to delete existing images for product %s: %v\n", req.ID, err)
	}

	// Create new images
	var images []*models.Image
	for i, imageURL := range imageUrls {
		// For legacy image URLs, we'll use the URL as the S3 key
		// In a real scenario, these would be uploaded to S3 first
		image := models.NewImage(req.ID, imageURL, imageURL, models.ImageTypeProduct)
		image.SetCreatedBy(req.GetUserID())

		// Set first image as primary
		if i == 0 {
			image.SetAsPrimary()
		}

		// Set sort order
		image.SetSortOrder(i)

		if err := s.repoManager.ImageRepository.Create(ctx, image); err != nil {
			fmt.Printf("Warning: Failed to create image %d for product %s: %v\n", i, req.ID, err)
		} else {
			images = append(images, image)
		}
	}

	// Create response with updated product and images (inventory handled separately)
	productResp := responses.NewProductUpdateResponse(existing, images, nil)
	return base.NewSuccessResponse("Product updated successfully", productResp), nil
}

// DeleteProduct soft deletes a product
func (s *ProductService) DeleteProduct(ctx context.Context, req *requests.DeleteProductRequest) (*base.BaseResponse, error) {
	// Check if product exists
	exists, err := s.repoManager.ProductRepository.Exists(ctx, req.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to check product existence: %w", err)
	}

	if !exists {
		return base.NewErrorResponse("Product not found",
			base.NewNotFoundError("Product", req.ID)), nil
	}

	// Soft delete product
	if err := s.repoManager.ProductRepository.SoftDelete(ctx, req.ID, req.GetUserID()); err != nil {
		return nil, fmt.Errorf("failed to delete product: %w", err)
	}

	// Also soft delete related images
	if err := s.repoManager.ImageRepository.DeleteImagesByProductID(ctx, req.ID); err != nil {
		fmt.Printf("Warning: Failed to delete images for product %s: %v\n", req.ID, err)
	}

	// Also delete related inventory record
	if err := s.repoManager.InventoryRepository.DeleteByProductID(ctx, req.ID); err != nil {
		fmt.Printf("Warning: Failed to delete inventory for product %s: %v\n", req.ID, err)
	}

	// Create response
	productResp := responses.NewProductDeleteResponse()
	return base.NewSuccessResponse("Product deleted successfully", productResp), nil
}

// UploadProductImage uploads a product image to R2 and creates image record
func (s *ProductService) UploadProductImage(ctx context.Context, req *requests.UploadProductImageRequest, file multipart.File, header *multipart.FileHeader, r2Service *R2Service) (*base.BaseResponse, error) {
	// Check if product exists and is not soft deleted
	filter := base.NewFilterBuilder().
		Where("id", base.OpEqual, req.ProductID).
		Where("is_active", base.OpEqual, true).
		Where("deleted_at", base.OpIsNull, nil).
		Build()

	_, err := s.repoManager.ProductRepository.FindOne(ctx, filter)
	if err != nil {
		return base.NewErrorResponse("Product not found",
			base.NewNotFoundError("Product", req.ProductID)), nil
	}

	// Check if R2 service is available
	if r2Service == nil || !r2Service.IsConfigured() {
		return base.NewErrorResponse("Storage service not available",
			base.NewInternalServerError("R2 service error", "R2 service is not configured")), nil
	}

	// Upload image to R2
	r2Key, err := r2Service.UploadProductImage(ctx, req.ProductID, file, header)
	if err != nil {
		return base.NewErrorResponse("Failed to upload image",
			base.NewInternalServerError("R2 upload error", err.Error())), nil
	}

	// Get the public URL from R2 service
	imageURL := r2Service.GetProductImageURL(r2Key)

	// Create image record
	image := models.NewImage(req.ProductID, imageURL, r2Key, models.ImageTypeProduct)
	image.AltText = req.AltText
	image.Caption = req.Caption
	image.FileSize = header.Size
	image.MimeType = header.Header.Get("Content-Type")
	image.SetCreatedBy(req.GetUserID())

	// Check if this is the first image for the product
	existingImages, err := s.repoManager.ImageRepository.GetActiveImagesByProductID(ctx, req.ProductID)
	if err != nil {
		// Log error but continue - this shouldn't fail the upload
		fmt.Printf("Warning: Failed to check existing images for product %s: %v\n", req.ProductID, err)
	}

	// Set as primary if it's the first image
	if len(existingImages) == 0 {
		image.SetAsPrimary()
	} else {
		// Set sort order as the next available
		image.SetSortOrder(len(existingImages))
	}

	// Save image to database
	if err := s.repoManager.ImageRepository.Create(ctx, image); err != nil {
		// If database save fails, try to delete from R2
		if deleteErr := r2Service.DeleteProductImage(ctx, r2Key); deleteErr != nil {
			fmt.Printf("Warning: Failed to delete uploaded image from R2: %v\n", deleteErr)
		}
		return base.NewErrorResponse("Failed to save image record",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response (r2Key stored in s3_key field for backward compatibility)
	imageResp := responses.NewImageUploadResponse(image, r2Key)
	return base.NewSuccessResponse("Image uploaded successfully", imageResp), nil
}

// DeleteProductImage deletes a product image from both database and R2
func (s *ProductService) DeleteProductImage(ctx context.Context, req *requests.DeleteProductImageRequest, r2Service *R2Service) (*base.BaseResponse, error) {
	// Get image record
	var image models.Image
	imagePtr, err := s.repoManager.ImageRepository.GetByID(ctx, req.ImageID, &image)
	if err != nil {
		return base.NewErrorResponse("Image not found",
			base.NewNotFoundError("Image", req.ImageID)), nil
	}
	image = *imagePtr

	// Verify the image belongs to the specified product
	if image.ProductID != req.ProductID {
		return base.NewErrorResponse("Image does not belong to the specified product",
			base.NewValidationError("Product mismatch", "Image product ID does not match")), nil
	}

	// Use the stored R2 key, fallback to image URL for backward compatibility
	r2Key := image.R2Key
	if r2Key == "" {
		r2Key = image.ImageURL
	}

	// Delete from R2
	if r2Service != nil && r2Service.IsConfigured() {
		if err := r2Service.DeleteProductImage(ctx, r2Key); err != nil {
			fmt.Printf("Warning: Failed to delete image from R2: %v\n", err)
			// Continue with database deletion even if R2 deletion fails
		}
	}

	// Delete from database
	if err := s.repoManager.ImageRepository.Delete(ctx, req.ImageID); err != nil {
		return base.NewErrorResponse("Failed to delete image",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// If this was the primary image, set another image as primary
	if image.IsPrimary {
		remainingImages, err := s.repoManager.ImageRepository.GetActiveImagesByProductID(ctx, req.ProductID)
		if err == nil && len(remainingImages) > 0 {
			// Set the first remaining image as primary
			if err := s.repoManager.ImageRepository.SetPrimaryImage(ctx, req.ProductID, remainingImages[0].ID); err != nil {
				fmt.Printf("Warning: Failed to set new primary image: %v\n", err)
			}
		}
	}

	return base.NewSuccessResponse("Image deleted successfully", nil), nil
}

// SetPrimaryImage sets an image as the primary image for a product
func (s *ProductService) SetPrimaryImage(ctx context.Context, req *requests.SetPrimaryImageRequest) (*base.BaseResponse, error) {
	// Get image record
	var image models.Image
	imagePtr, err := s.repoManager.ImageRepository.GetByID(ctx, req.ImageID, &image)
	if err != nil {
		return base.NewErrorResponse("Image not found",
			base.NewNotFoundError("Image", req.ImageID)), nil
	}
	image = *imagePtr

	// Verify the image belongs to the specified product
	if image.ProductID != req.ProductID {
		return base.NewErrorResponse("Image does not belong to the specified product",
			base.NewValidationError("Product mismatch", "Image product ID does not match")), nil
	}

	// Set as primary image
	if err := s.repoManager.ImageRepository.SetPrimaryImage(ctx, req.ProductID, req.ImageID); err != nil {
		return base.NewErrorResponse("Failed to set primary image",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Get updated image
	updatedImage, err := s.repoManager.ImageRepository.GetByID(ctx, req.ImageID, &image)
	if err != nil {
		return base.NewErrorResponse("Failed to retrieve updated image",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response
	imageResp := responses.NewImageResponse(updatedImage)
	return base.NewSuccessResponse("Primary image set successfully", imageResp), nil
}

// GetProductImages retrieves all images for a product
func (s *ProductService) GetProductImages(ctx context.Context, req *requests.GetProductImagesRequest) (*base.BaseResponse, error) {
	// Check if product exists and is not soft deleted
	filter := base.NewFilterBuilder().
		Where("id", base.OpEqual, req.ProductID).
		Where("is_active", base.OpEqual, true).
		Where("deleted_at", base.OpIsNull, nil).
		Build()

	_, err := s.repoManager.ProductRepository.FindOne(ctx, filter)
	if err != nil {
		return base.NewErrorResponse("Product not found",
			base.NewNotFoundError("Product", req.ProductID)), nil
	}

	// Get all active images for the product
	images, err := s.repoManager.ImageRepository.GetActiveImagesByProductID(ctx, req.ProductID)
	if err != nil {
		return base.NewErrorResponse("Failed to retrieve images",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response
	imagesResp := responses.NewProductImagesResponse(req.ProductID, images)
	return base.NewSuccessResponse("Images retrieved successfully", imagesResp), nil
}
