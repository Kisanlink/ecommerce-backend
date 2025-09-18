package requests

import "github.com/Kisanlink/kisanlink-db/pkg/base"

// CreateInventoryRequest represents request for creating inventory
type CreateInventoryRequest struct {
	base.CreateRequest
	ProductID     string `json:"product_id" validate:"required"`
	CurrentStock  int    `json:"current_stock" validate:"min=0"`
	MinStockLevel int    `json:"min_stock_level" validate:"min=0"`
	MaxStockLevel int    `json:"max_stock_level" validate:"min=0"`
	ReorderPoint  int    `json:"reorder_point" validate:"min=0"`
}

// UpdateInventoryRequest represents request for updating inventory
type UpdateInventoryRequest struct {
	base.UpdateRequest
	ProductID     string `json:"product_id" validate:"required"`
	CurrentStock  int    `json:"current_stock" validate:"min=0"`
	MinStockLevel int    `json:"min_stock_level" validate:"min=0"`
	MaxStockLevel int    `json:"max_stock_level" validate:"min=0"`
	ReorderPoint  int    `json:"reorder_point" validate:"min=0"`
}

// GetInventoryRequest represents request for getting inventory by product ID
type GetInventoryRequest struct {
	ProductID string `json:"product_id" validate:"required"`
}

// UpdateStockRequest represents request for updating stock
type UpdateStockRequest struct {
	base.UpdateRequest
	ProductID string `json:"product_id" validate:"required"`
	NewStock  int    `json:"new_stock" validate:"min=0"`
}

// ReserveStockRequest represents request for reserving stock
type ReserveStockRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"min=1"`
}

// ReleaseStockRequest represents request for releasing reserved stock
type ReleaseStockRequest struct {
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"min=1"`
}

// SellStockRequest represents request for selling stock
type SellStockRequest struct {
	base.UpdateRequest
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"min=1"`
}

// RestockRequest represents request for restocking
type RestockRequest struct {
	base.UpdateRequest
	ProductID string `json:"product_id" validate:"required"`
	Quantity  int    `json:"quantity" validate:"min=1"`
}

// DeleteInventoryRequest represents request for deleting inventory
type DeleteInventoryRequest struct {
	base.DeleteRequest
	ProductID string `json:"product_id" validate:"required"`
}

// ListInventoryRequest represents request for listing inventory with filters
type ListInventoryRequest struct {
	base.ListRequest
	ProductID    string `json:"product_id,omitempty"`
	StockStatus  string `json:"stock_status,omitempty"`
	LowStock     *bool  `json:"low_stock,omitempty"`
	OutOfStock   *bool  `json:"out_of_stock,omitempty"`
	NeedsReorder *bool  `json:"needs_reorder,omitempty"`
}

// GetLowStockItemsRequest represents request for getting low stock items
type GetLowStockItemsRequest struct {
	base.ListRequest
}

// GetOutOfStockItemsRequest represents request for getting out of stock items
type GetOutOfStockItemsRequest struct {
	base.ListRequest
}

// GetItemsNeedingReorderRequest represents request for getting items needing reorder
type GetItemsNeedingReorderRequest struct {
	base.ListRequest
}
