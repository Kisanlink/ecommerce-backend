package models

import (
	"fmt"
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// StockStatus represents the stock status of a product
type StockStatus string

const (
	StockStatusInStock      StockStatus = "In Stock"
	StockStatusLowStock     StockStatus = "Low Stock"
	StockStatusOutOfStock   StockStatus = "Out of Stock"
	StockStatusDiscontinued StockStatus = "Discontinued"
)

// Inventory represents inventory information for a product
type Inventory struct {
	*base.BaseModel `gorm:"embedded"`
	ProductID       string `json:"product_id" gorm:"not null;index"`

	// Stock Information
	CurrentStock   int `json:"current_stock" gorm:"not null;default:0"`
	ReservedStock  int `json:"reserved_stock" gorm:"default:0"`
	AvailableStock int `json:"available_stock" gorm:"-"` // Computed field

	// Stock Management
	MinStockLevel int `json:"min_stock_level" gorm:"default:10"`
	MaxStockLevel int `json:"max_stock_level" gorm:"default:1000"`
	ReorderPoint  int `json:"reorder_point" gorm:"default:20"`

	// Stock Status
	StockStatus     StockStatus `json:"stock_status" gorm:"default:'In Stock'"`
	LastRestockedAt *time.Time  `json:"last_restocked_at"`
	LastSoldAt      *time.Time  `json:"last_sold_at"`

	// Audit Fields
	UpdatedBy string `json:"updated_by"`
}

// NewInventory creates a new inventory record
func NewInventory(productID string, currentStock, minStockLevel, maxStockLevel, reorderPoint int) *Inventory {
	baseModel := base.NewBaseModel("INV", hash.Medium)

	// Calculate stock status
	stockStatus := StockStatusInStock
	if currentStock <= 0 {
		stockStatus = StockStatusOutOfStock
	} else if currentStock <= minStockLevel {
		stockStatus = StockStatusLowStock
	}

	return &Inventory{
		BaseModel:     baseModel,
		ProductID:     productID,
		CurrentStock:  currentStock,
		ReservedStock: 0,
		MinStockLevel: minStockLevel,
		MaxStockLevel: maxStockLevel,
		ReorderPoint:  reorderPoint,
		StockStatus:   stockStatus,
	}
}

// TableName returns the table name for GORM
func (i *Inventory) TableName() string {
	return "inventory"
}

// GetAvailableStock calculates and returns available stock
func (i *Inventory) GetAvailableStock() int {
	return i.CurrentStock - i.ReservedStock
}

// UpdateStock updates the current stock and recalculates status
func (i *Inventory) UpdateStock(newStock int, updatedBy string) {
	i.CurrentStock = newStock
	i.UpdatedBy = updatedBy

	// Recalculate stock status
	if newStock <= 0 {
		i.StockStatus = StockStatusOutOfStock
	} else if newStock <= i.MinStockLevel {
		i.StockStatus = StockStatusLowStock
	} else {
		i.StockStatus = StockStatusInStock
	}
}

// ReserveStock reserves stock for pending orders
func (i *Inventory) ReserveStock(quantity int) error {
	if i.GetAvailableStock() < quantity {
		return fmt.Errorf("insufficient stock: available %d, requested %d", i.GetAvailableStock(), quantity)
	}
	i.ReservedStock += quantity
	return nil
}

// ReleaseStock releases reserved stock
func (i *Inventory) ReleaseStock(quantity int) error {
	if i.ReservedStock < quantity {
		return fmt.Errorf("cannot release more stock than reserved: reserved %d, requested %d", i.ReservedStock, quantity)
	}
	i.ReservedStock -= quantity
	return nil
}

// SellStock processes a sale and updates stock
func (i *Inventory) SellStock(quantity int, updatedBy string) error {
	if i.GetAvailableStock() < quantity {
		return fmt.Errorf("insufficient stock for sale: available %d, requested %d", i.GetAvailableStock(), quantity)
	}

	i.CurrentStock -= quantity
	i.UpdatedBy = updatedBy
	now := time.Now()
	i.LastSoldAt = &now

	// Recalculate stock status
	if i.CurrentStock <= 0 {
		i.StockStatus = StockStatusOutOfStock
	} else if i.CurrentStock <= i.MinStockLevel {
		i.StockStatus = StockStatusLowStock
	} else {
		i.StockStatus = StockStatusInStock
	}

	return nil
}

// Restock adds stock to inventory
func (i *Inventory) Restock(quantity int, updatedBy string) error {
	if quantity <= 0 {
		return fmt.Errorf("restock quantity must be positive")
	}

	i.CurrentStock += quantity
	i.UpdatedBy = updatedBy
	now := time.Now()
	i.LastRestockedAt = &now

	// Recalculate stock status
	if i.CurrentStock <= 0 {
		i.StockStatus = StockStatusOutOfStock
	} else if i.CurrentStock <= i.MinStockLevel {
		i.StockStatus = StockStatusLowStock
	} else {
		i.StockStatus = StockStatusInStock
	}

	return nil
}

// IsLowStock checks if the product is low on stock
func (i *Inventory) IsLowStock() bool {
	return i.CurrentStock <= i.MinStockLevel
}

// IsOutOfStock checks if the product is out of stock
func (i *Inventory) IsOutOfStock() bool {
	return i.CurrentStock <= 0
}

// NeedsReorder checks if the product needs to be reordered
func (i *Inventory) NeedsReorder() bool {
	return i.CurrentStock <= i.ReorderPoint
}

// IsValidStockStatus checks if the stock status is valid
func (s StockStatus) IsValidStockStatus() bool {
	switch s {
	case StockStatusInStock, StockStatusLowStock, StockStatusOutOfStock, StockStatusDiscontinued:
		return true
	default:
		return false
	}
}
