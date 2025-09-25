package responses

import (
	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// InventoryResponse represents a complete inventory response
type InventoryResponse struct {
	ID              string `json:"id"`
	ProductID       string `json:"product_id"`
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
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	UpdatedBy       string `json:"updated_by"`
}

// InventoryCreateResponse represents response after creating inventory
type InventoryCreateResponse struct {
	Inventory *InventoryResponse `json:"inventory"`
	Message   string             `json:"message"`
}

// InventoryUpdateResponse represents response after updating inventory
type InventoryUpdateResponse struct {
	Inventory *InventoryResponse `json:"inventory"`
	Message   string             `json:"message"`
}

// InventoryDeleteResponse represents response after deleting inventory
type InventoryDeleteResponse struct {
	Message string `json:"message"`
}

// StockUpdateResponse represents response after updating stock
type StockUpdateResponse struct {
	ProductID     string `json:"product_id"`
	PreviousStock int    `json:"previous_stock"`
	NewStock      int    `json:"new_stock"`
	StockStatus   string `json:"stock_status"`
	Message       string `json:"message"`
}

// StockReserveResponse represents response after reserving stock
type StockReserveResponse struct {
	ProductID      string `json:"product_id"`
	ReservedAmount int    `json:"reserved_amount"`
	AvailableStock int    `json:"available_stock"`
	Message        string `json:"message"`
}

// StockReleaseResponse represents response after releasing stock
type StockReleaseResponse struct {
	ProductID      string `json:"product_id"`
	ReleasedAmount int    `json:"released_amount"`
	AvailableStock int    `json:"available_stock"`
	Message        string `json:"message"`
}

// StockSellResponse represents response after selling stock
type StockSellResponse struct {
	ProductID      string `json:"product_id"`
	SoldAmount     int    `json:"sold_amount"`
	RemainingStock int    `json:"remaining_stock"`
	StockStatus    string `json:"stock_status"`
	Message        string `json:"message"`
}

// StockRestockResponse represents response after restocking
type StockRestockResponse struct {
	ProductID       string `json:"product_id"`
	RestockedAmount int    `json:"restocked_amount"`
	NewStock        int    `json:"new_stock"`
	StockStatus     string `json:"stock_status"`
	Message         string `json:"message"`
}

// InventoryListPaginatedResponse represents paginated list of inventory
type InventoryListPaginatedResponse struct {
	Inventory  []InventoryResponse  `json:"inventory"`
	Pagination *base.PaginationInfo `json:"pagination"`
	Message    string               `json:"message"`
}

// LowStockItemsResponse represents response for low stock items
type LowStockItemsResponse struct {
	Items   []InventoryResponse `json:"items"`
	Count   int                 `json:"count"`
	Message string              `json:"message"`
}

// OutOfStockItemsResponse represents response for out of stock items
type OutOfStockItemsResponse struct {
	Items   []InventoryResponse `json:"items"`
	Count   int                 `json:"count"`
	Message string              `json:"message"`
}

// ItemsNeedingReorderResponse represents response for items needing reorder
type ItemsNeedingReorderResponse struct {
	Items   []InventoryResponse `json:"items"`
	Count   int                 `json:"count"`
	Message string              `json:"message"`
}

// NewInventoryResponse creates an InventoryResponse from Inventory model
func NewInventoryResponse(inventory *models.Inventory) InventoryResponse {
	response := InventoryResponse{
		ID:             inventory.ID,
		ProductID:      inventory.ProductID,
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
		CreatedAt:      inventory.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      inventory.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedBy:      inventory.UpdatedBy,
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

// NewInventoryCreateResponse creates an InventoryCreateResponse
func NewInventoryCreateResponse(inventory *models.Inventory) InventoryCreateResponse {
	inventoryResp := NewInventoryResponse(inventory)
	return InventoryCreateResponse{
		Inventory: &inventoryResp,
		Message:   "Inventory created successfully",
	}
}

// NewInventoryUpdateResponse creates an InventoryUpdateResponse
func NewInventoryUpdateResponse(inventory *models.Inventory) InventoryUpdateResponse {
	inventoryResp := NewInventoryResponse(inventory)
	return InventoryUpdateResponse{
		Inventory: &inventoryResp,
		Message:   "Inventory updated successfully",
	}
}

// NewInventoryDeleteResponse creates an InventoryDeleteResponse
func NewInventoryDeleteResponse() InventoryDeleteResponse {
	return InventoryDeleteResponse{
		Message: "Inventory deleted successfully",
	}
}

// NewStockUpdateResponse creates a StockUpdateResponse
func NewStockUpdateResponse(productID string, previousStock, newStock int, stockStatus string) StockUpdateResponse {
	return StockUpdateResponse{
		ProductID:     productID,
		PreviousStock: previousStock,
		NewStock:      newStock,
		StockStatus:   stockStatus,
		Message:       "Stock updated successfully",
	}
}

// NewStockReserveResponse creates a StockReserveResponse
func NewStockReserveResponse(productID string, reservedAmount, availableStock int) StockReserveResponse {
	return StockReserveResponse{
		ProductID:      productID,
		ReservedAmount: reservedAmount,
		AvailableStock: availableStock,
		Message:        "Stock reserved successfully",
	}
}

// NewStockReleaseResponse creates a StockReleaseResponse
func NewStockReleaseResponse(productID string, releasedAmount, availableStock int) StockReleaseResponse {
	return StockReleaseResponse{
		ProductID:      productID,
		ReleasedAmount: releasedAmount,
		AvailableStock: availableStock,
		Message:        "Stock released successfully",
	}
}

// NewStockSellResponse creates a StockSellResponse
func NewStockSellResponse(productID string, soldAmount, remainingStock int, stockStatus string) StockSellResponse {
	return StockSellResponse{
		ProductID:      productID,
		SoldAmount:     soldAmount,
		RemainingStock: remainingStock,
		StockStatus:    stockStatus,
		Message:        "Stock sold successfully",
	}
}

// NewStockRestockResponse creates a StockRestockResponse
func NewStockRestockResponse(productID string, restockedAmount, newStock int, stockStatus string) StockRestockResponse {
	return StockRestockResponse{
		ProductID:       productID,
		RestockedAmount: restockedAmount,
		NewStock:        newStock,
		StockStatus:     stockStatus,
		Message:         "Stock restocked successfully",
	}
}

// NewInventoryListPaginatedResponse creates an InventoryListPaginatedResponse
func NewInventoryListPaginatedResponse(inventories []InventoryResponse, pagination *base.PaginationInfo) InventoryListPaginatedResponse {
	return InventoryListPaginatedResponse{
		Inventory:  inventories,
		Pagination: pagination,
		Message:    "Inventory retrieved successfully",
	}
}

// NewLowStockItemsResponse creates a LowStockItemsResponse
func NewLowStockItemsResponse(inventories []*models.Inventory) LowStockItemsResponse {
	items := make([]InventoryResponse, len(inventories))
	for i, inv := range inventories {
		items[i] = NewInventoryResponse(inv)
	}

	return LowStockItemsResponse{
		Items:   items,
		Count:   len(items),
		Message: "Low stock items retrieved successfully",
	}
}

// NewOutOfStockItemsResponse creates an OutOfStockItemsResponse
func NewOutOfStockItemsResponse(inventories []*models.Inventory) OutOfStockItemsResponse {
	items := make([]InventoryResponse, len(inventories))
	for i, inv := range inventories {
		items[i] = NewInventoryResponse(inv)
	}

	return OutOfStockItemsResponse{
		Items:   items,
		Count:   len(items),
		Message: "Out of stock items retrieved successfully",
	}
}

// NewItemsNeedingReorderResponse creates an ItemsNeedingReorderResponse
func NewItemsNeedingReorderResponse(inventories []*models.Inventory) ItemsNeedingReorderResponse {
	items := make([]InventoryResponse, len(inventories))
	for i, inv := range inventories {
		items[i] = NewInventoryResponse(inv)
	}

	return ItemsNeedingReorderResponse{
		Items:   items,
		Count:   len(items),
		Message: "Items needing reorder retrieved successfully",
	}
}
