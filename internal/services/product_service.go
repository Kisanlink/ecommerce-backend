package services

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
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
	// Get effective images from request
	imageUrls := req.GetEffectiveImages()
	primaryImage := ""
	if len(imageUrls) > 0 {
		primaryImage = imageUrls[0]
	}

	// Create product model
	product := models.NewProduct(
		req.Name,
		req.Description,
		req.Category,
		primaryImage, // Use first image as primary
		req.Price,
		req.ActualPrice,
		req.Stock,
		req.ReviewCount,
		req.DiscountPercent,
		req.Rating,
	)

	// Set all images using the new method
	product.SetImages(imageUrls)

	// Set created by from request context
	product.SetCreatedBy(req.GetUserID())

	// Save to database using repository
	if err := s.repoManager.ProductRepository.Create(ctx, product); err != nil {
		return base.NewErrorResponse("Failed to create product",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	return base.NewSuccessResponse("Product created successfully", product), nil
}

// GetProductByID retrieves a product by ID
func (s *ProductService) GetProductByID(ctx context.Context, req *requests.GetProductRequest) (*base.BaseResponse, error) {
	// Build filter for active products only
	filter := base.NewFilterBuilder().
		Where("id", base.OpEqual, req.ID).
		Where("is_active", base.OpEqual, true).
		Build()

	product, err := s.repoManager.ProductRepository.FindOne(ctx, filter)
	if err != nil {
		return base.NewErrorResponse("Product not found",
			base.NewNotFoundError("Product", req.ID)), nil
	}

	// Get additional images from the images table
	images, err := s.repoManager.ImageRepository.GetActiveImagesByProductID(ctx, req.ID)
	if err != nil {
		// Log error but don't fail the request - product can still be returned without additional images
		// You might want to add logging here
		images = []*models.Image{}
	}

	// Create enhanced product response with images
	productWithImages := map[string]interface{}{
		"product":     product,
		"images":      images,
		"image_count": len(images),
	}

	return base.NewSuccessResponse("Product retrieved successfully", productWithImages), nil
}

// GetProducts retrieves products with filtering and pagination
func (s *ProductService) GetProducts(ctx context.Context, req *requests.ListProductsRequest) (*base.PaginatedResponse, error) {
	// Build base filter for active products
	filterBuilder := base.NewFilterBuilder().
		Where("is_active", base.OpEqual, true).
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

	if req.InStock != nil && *req.InStock {
		filterBuilder.Where("stock", base.OpGreaterThan, 0)
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

	pagination := base.NewPaginationInfo(req.Page, req.PerPage, int(count))
	return base.NewPaginatedResponse("Products retrieved successfully", products, pagination), nil
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
	existing.Stock = req.Stock
	existing.ReviewCount = req.ReviewCount
	existing.Rating = req.Rating

	// Handle image updates
	imageUrls := req.GetEffectiveImages()
	existing.SetImages(imageUrls)

	// Update primary ImageURL for backward compatibility
	if len(imageUrls) > 0 {
		existing.ImageURL = imageUrls[0]
	} else {
		existing.ImageURL = ""
	}

	// Update stock status based on stock
	if existing.Stock <= 0 {
		existing.StockStatus = "Out of Stock"
	} else {
		existing.StockStatus = "In Stock"
	}

	existing.SetUpdatedBy(req.GetUserID())

	// Save changes
	if err := s.repoManager.ProductRepository.Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update product: %w", err)
	}

	return base.NewSuccessResponse("Product updated successfully", existing), nil
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

	// Soft delete
	if err := s.repoManager.ProductRepository.SoftDelete(ctx, req.ID, req.GetUserID()); err != nil {
		return nil, fmt.Errorf("failed to delete product: %w", err)
	}

	return base.NewSuccessResponse("Product deleted successfully", nil), nil
}
