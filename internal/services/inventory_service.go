package services

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// InventoryService handles inventory business logic
type InventoryService struct {
	repoManager *repositories.RepositoryManager
}

// NewInventoryService creates a new inventory service
func NewInventoryService(repoManager *repositories.RepositoryManager) *InventoryService {
	service := &InventoryService{
		repoManager: repoManager,
	}

	// Initialize the ID generator with existing inventory IDs
	service.initializeIDGenerator()

	return service
}

// initializeIDGenerator initializes the global ID generator with existing inventory IDs
func (s *InventoryService) initializeIDGenerator() {
	ctx := context.Background()

	// Get all existing inventory records to extract their IDs
	filter := base.NewFilterBuilder().
		Page(1, 10000). // Get all inventory records
		Build()

	result, err := s.repoManager.InventoryRepository.Find(ctx, filter)
	if err != nil {
		// If we can't get existing inventory, start from 0
		fmt.Printf("Warning: Could not initialize inventory ID generator: %v\n", err)
		return
	}

	// Extract all existing IDs
	existingIDs := make([]string, len(result))
	for i, inventory := range result {
		existingIDs[i] = inventory.GetID()
	}

	// Initialize the global generator with existing IDs
	hash.InitializeGlobalCountersFromDatabase("INV", existingIDs, hash.Medium)
	fmt.Printf("Initialized inventory ID generator with %d existing inventory IDs\n", len(existingIDs))
}

// CreateInventory creates a new inventory record
func (s *InventoryService) CreateInventory(ctx context.Context, req *requests.CreateInventoryRequest) (*base.BaseResponse, error) {
	// Check if product exists
	exists, err := s.repoManager.ProductRepository.Exists(ctx, req.ProductID)
	if err != nil {
		return base.NewErrorResponse("Failed to check product existence",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	if !exists {
		return base.NewErrorResponse("Product not found",
			base.NewNotFoundError("Product", req.ProductID)), nil
	}

	// Check if inventory already exists for this product
	_, err = s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	if err == nil {
		return base.NewErrorResponse("Inventory already exists for this product",
			base.NewValidationError("Duplicate inventory", "Inventory already exists for product "+req.ProductID)), nil
	}

	// Create inventory model
	inventory := models.NewInventory(
		req.ProductID,
		req.CurrentStock,
		req.MinStockLevel,
		req.MaxStockLevel,
		req.ReorderPoint,
	)

	// Set created by from request context
	inventory.SetCreatedBy(req.GetUserID())

	// Save to database
	if err := s.repoManager.InventoryRepository.Create(ctx, inventory); err != nil {
		return base.NewErrorResponse("Failed to create inventory",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response
	inventoryResp := responses.NewInventoryCreateResponse(inventory)
	return base.NewSuccessResponse("Inventory created successfully", inventoryResp), nil
}

// GetInventoryByProductID retrieves inventory by product ID
func (s *InventoryService) GetInventoryByProductID(ctx context.Context, req *requests.GetInventoryRequest) (*base.BaseResponse, error) {
	inventory, err := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	if err != nil {
		return base.NewErrorResponse("Inventory not found",
			base.NewNotFoundError("Inventory", req.ProductID)), nil
	}

	// Create response
	inventoryResp := responses.NewInventoryResponse(inventory)
	return base.NewSuccessResponse("Inventory retrieved successfully", inventoryResp), nil
}

// UpdateInventory updates an existing inventory record
func (s *InventoryService) UpdateInventory(ctx context.Context, req *requests.UpdateInventoryRequest) (*base.BaseResponse, error) {
	// Get existing inventory
	inventory, err := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	if err != nil {
		return base.NewErrorResponse("Inventory not found",
			base.NewNotFoundError("Inventory", req.ProductID)), nil
	}

	// Update fields
	inventory.CurrentStock = req.CurrentStock
	inventory.MinStockLevel = req.MinStockLevel
	inventory.MaxStockLevel = req.MaxStockLevel
	inventory.ReorderPoint = req.ReorderPoint
	inventory.SetUpdatedBy(req.GetUserID())

	// Recalculate stock status
	inventory.UpdateStock(req.CurrentStock, req.GetUserID())

	// Save changes
	if err := s.repoManager.InventoryRepository.Update(ctx, inventory); err != nil {
		return base.NewErrorResponse("Failed to update inventory",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response
	inventoryResp := responses.NewInventoryUpdateResponse(inventory)
	return base.NewSuccessResponse("Inventory updated successfully", inventoryResp), nil
}

// DeleteInventory deletes an inventory record
func (s *InventoryService) DeleteInventory(ctx context.Context, req *requests.DeleteInventoryRequest) (*base.BaseResponse, error) {
	// Check if inventory exists
	_, err := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	if err != nil {
		return base.NewErrorResponse("Inventory not found",
			base.NewNotFoundError("Inventory", req.ProductID)), nil
	}

	// Delete inventory
	if err := s.repoManager.InventoryRepository.DeleteByProductID(ctx, req.ProductID); err != nil {
		return base.NewErrorResponse("Failed to delete inventory",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response
	inventoryResp := responses.NewInventoryDeleteResponse()
	return base.NewSuccessResponse("Inventory deleted successfully", inventoryResp), nil
}

// ListInventory retrieves inventory records with filtering and pagination
func (s *InventoryService) ListInventory(ctx context.Context, req *requests.ListInventoryRequest) (*base.PaginatedResponse, error) {
	// Build base filter
	filterBuilder := base.NewFilterBuilder().
		Sort("created_at", "desc").
		Page(req.Page, req.PerPage)

	// Add optional filters
	if req.ProductID != "" {
		filterBuilder.Where("product_id", base.OpEqual, req.ProductID)
	}

	if req.StockStatus != "" {
		filterBuilder.Where("stock_status", base.OpEqual, req.StockStatus)
	}

	if req.LowStock != nil && *req.LowStock {
		filterBuilder.Where("stock_status", base.OpEqual, string(models.StockStatusLowStock))
	}

	if req.OutOfStock != nil && *req.OutOfStock {
		filterBuilder.Where("stock_status", base.OpEqual, string(models.StockStatusOutOfStock))
	}

	if req.NeedsReorder != nil && *req.NeedsReorder {
		// This would need a custom query or computed field
		// For now, we'll filter by low stock status
		filterBuilder.Where("stock_status", base.OpEqual, string(models.StockStatusLowStock))
	}

	filter := filterBuilder.Build()

	// Execute query
	inventories, err := s.repoManager.InventoryRepository.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve inventory: %w", err)
	}

	// Get total count
	count, err := s.repoManager.InventoryRepository.CountWithFilter(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to count inventory: %w", err)
	}

	// Convert to response format
	inventoryResponses := make([]responses.InventoryResponse, len(inventories))
	for i, inventory := range inventories {
		inventoryResponses[i] = responses.NewInventoryResponse(inventory)
	}

	pagination := base.NewPaginationInfo(req.Page, req.PerPage, int(count))
	inventoryListResp := responses.NewInventoryListPaginatedResponse(inventoryResponses, pagination)
	return base.NewPaginatedResponse("Inventory retrieved successfully", inventoryListResp, pagination), nil
}

// UpdateStock updates the stock for a product
func (s *InventoryService) UpdateStock(ctx context.Context, req *requests.UpdateStockRequest) (*base.BaseResponse, error) {
	// Get existing inventory
	inventory, err := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	if err != nil {
		return base.NewErrorResponse("Inventory not found",
			base.NewNotFoundError("Inventory", req.ProductID)), nil
	}

	previousStock := inventory.CurrentStock

	// Update stock
	if err := s.repoManager.InventoryRepository.UpdateStock(ctx, req.ProductID, req.NewStock, req.GetUserID()); err != nil {
		return base.NewErrorResponse("Failed to update stock",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Get updated inventory for status
	updatedInventory, _ := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	stockStatus := string(updatedInventory.StockStatus)

	// Create response
	stockResp := responses.NewStockUpdateResponse(req.ProductID, previousStock, req.NewStock, stockStatus)
	return base.NewSuccessResponse("Stock updated successfully", stockResp), nil
}

// ReserveStock reserves stock for a product
func (s *InventoryService) ReserveStock(ctx context.Context, req *requests.ReserveStockRequest) (*base.BaseResponse, error) {
	// Reserve stock
	if err := s.repoManager.InventoryRepository.ReserveStock(ctx, req.ProductID, req.Quantity); err != nil {
		return base.NewErrorResponse("Failed to reserve stock",
			base.NewValidationError("Insufficient stock", err.Error())), nil
	}

	// Get updated inventory for available stock
	inventory, _ := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	availableStock := inventory.GetAvailableStock()

	// Create response
	stockResp := responses.NewStockReserveResponse(req.ProductID, req.Quantity, availableStock)
	return base.NewSuccessResponse("Stock reserved successfully", stockResp), nil
}

// ReleaseStock releases reserved stock for a product
func (s *InventoryService) ReleaseStock(ctx context.Context, req *requests.ReleaseStockRequest) (*base.BaseResponse, error) {
	// Release stock
	if err := s.repoManager.InventoryRepository.ReleaseStock(ctx, req.ProductID, req.Quantity); err != nil {
		return base.NewErrorResponse("Failed to release stock",
			base.NewValidationError("Invalid release", err.Error())), nil
	}

	// Get updated inventory for available stock
	inventory, _ := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	availableStock := inventory.GetAvailableStock()

	// Create response
	stockResp := responses.NewStockReleaseResponse(req.ProductID, req.Quantity, availableStock)
	return base.NewSuccessResponse("Stock released successfully", stockResp), nil
}

// SellStock processes a sale and updates stock
func (s *InventoryService) SellStock(ctx context.Context, req *requests.SellStockRequest) (*base.BaseResponse, error) {
	// Sell stock
	if err := s.repoManager.InventoryRepository.SellStock(ctx, req.ProductID, req.Quantity, req.GetUserID()); err != nil {
		return base.NewErrorResponse("Failed to sell stock",
			base.NewValidationError("Insufficient stock", err.Error())), nil
	}

	// Get updated inventory
	inventory, _ := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	remainingStock := inventory.CurrentStock
	stockStatus := string(inventory.StockStatus)

	// Create response
	stockResp := responses.NewStockSellResponse(req.ProductID, req.Quantity, remainingStock, stockStatus)
	return base.NewSuccessResponse("Stock sold successfully", stockResp), nil
}

// Restock adds stock to inventory
func (s *InventoryService) Restock(ctx context.Context, req *requests.RestockRequest) (*base.BaseResponse, error) {
	// Restock
	if err := s.repoManager.InventoryRepository.Restock(ctx, req.ProductID, req.Quantity, req.GetUserID()); err != nil {
		return base.NewErrorResponse("Failed to restock",
			base.NewValidationError("Invalid restock", err.Error())), nil
	}

	// Get updated inventory
	inventory, _ := s.repoManager.InventoryRepository.GetByProductID(ctx, req.ProductID)
	newStock := inventory.CurrentStock
	stockStatus := string(inventory.StockStatus)

	// Create response
	stockResp := responses.NewStockRestockResponse(req.ProductID, req.Quantity, newStock, stockStatus)
	return base.NewSuccessResponse("Stock restocked successfully", stockResp), nil
}

// GetLowStockItems retrieves all items with low stock
func (s *InventoryService) GetLowStockItems(ctx context.Context, req *requests.GetLowStockItemsRequest) (*base.BaseResponse, error) {
	inventories, err := s.repoManager.InventoryRepository.GetLowStockItems(ctx)
	if err != nil {
		return base.NewErrorResponse("Failed to get low stock items",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response
	lowStockResp := responses.NewLowStockItemsResponse(inventories)
	return base.NewSuccessResponse("Low stock items retrieved successfully", lowStockResp), nil
}

// GetOutOfStockItems retrieves all out of stock items
func (s *InventoryService) GetOutOfStockItems(ctx context.Context, req *requests.GetOutOfStockItemsRequest) (*base.BaseResponse, error) {
	inventories, err := s.repoManager.InventoryRepository.GetOutOfStockItems(ctx)
	if err != nil {
		return base.NewErrorResponse("Failed to get out of stock items",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response
	outOfStockResp := responses.NewOutOfStockItemsResponse(inventories)
	return base.NewSuccessResponse("Out of stock items retrieved successfully", outOfStockResp), nil
}

// GetItemsNeedingReorder retrieves items that need to be reordered
func (s *InventoryService) GetItemsNeedingReorder(ctx context.Context, req *requests.GetItemsNeedingReorderRequest) (*base.BaseResponse, error) {
	inventories, err := s.repoManager.InventoryRepository.GetItemsNeedingReorder(ctx)
	if err != nil {
		return base.NewErrorResponse("Failed to get items needing reorder",
			base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create response
	reorderResp := responses.NewItemsNeedingReorderResponse(inventories)
	return base.NewSuccessResponse("Items needing reorder retrieved successfully", reorderResp), nil
}
