package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
)

// InventoryRepositoryInterface defines the contract for inventory operations
type InventoryRepositoryInterface interface {
	Create(ctx context.Context, inventory *models.Inventory) error
	GetByID(ctx context.Context, id string, inventory *models.Inventory) (*models.Inventory, error)
	GetByProductID(ctx context.Context, productID string) (*models.Inventory, error)
	Update(ctx context.Context, inventory *models.Inventory) error
	Delete(ctx context.Context, id string) error
	DeleteByProductID(ctx context.Context, productID string) error
	Find(ctx context.Context, filter *base.Filter) ([]*models.Inventory, error)
	CountWithFilter(ctx context.Context, filter *base.Filter) (int64, error)
	FindOne(ctx context.Context, filter *base.Filter) (*models.Inventory, error)
	GetLowStockItems(ctx context.Context) ([]*models.Inventory, error)
	GetOutOfStockItems(ctx context.Context) ([]*models.Inventory, error)
	GetItemsNeedingReorder(ctx context.Context) ([]*models.Inventory, error)
	UpdateStock(ctx context.Context, productID string, newStock int, updatedBy string) error
	ReserveStock(ctx context.Context, productID string, quantity int) error
	ReleaseStock(ctx context.Context, productID string, quantity int) error
	SellStock(ctx context.Context, productID string, quantity int, updatedBy string) error
	Restock(ctx context.Context, productID string, quantity int, updatedBy string) error
}

// InventoryRepository handles inventory database operations
type InventoryRepository struct {
	*base.BaseFilterableRepository[*models.Inventory]
	dbManager db.DBManager
}

// NewInventoryRepository creates a new inventory repository
func NewInventoryRepository(dbManager db.DBManager) InventoryRepositoryInterface {
	repo := &InventoryRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Inventory](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Create creates a new inventory record
func (r *InventoryRepository) Create(ctx context.Context, inventory *models.Inventory) error {
	return r.dbManager.Create(ctx, inventory)
}

// GetByID retrieves an inventory record by its ID
func (r *InventoryRepository) GetByID(ctx context.Context, id string, inventory *models.Inventory) (*models.Inventory, error) {
	err := r.dbManager.GetByID(ctx, id, inventory)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory: %w", err)
	}
	return inventory, nil
}

// GetByProductID retrieves inventory record by product ID
func (r *InventoryRepository) GetByProductID(ctx context.Context, productID string) (*models.Inventory, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "product_id", Operator: base.OpEqual, Value: productID},
			},
			Logic: base.LogicAnd,
		},
		Limit: 1,
	}

	var inventories []*models.Inventory
	err := r.dbManager.List(ctx, filter, &inventories)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory by product ID: %w", err)
	}

	if len(inventories) == 0 {
		return nil, fmt.Errorf("inventory not found for product ID: %s", productID)
	}

	return inventories[0], nil
}

// Update updates an existing inventory record
func (r *InventoryRepository) Update(ctx context.Context, inventory *models.Inventory) error {
	return r.dbManager.Update(ctx, inventory)
}

// Delete deletes an inventory record by ID
func (r *InventoryRepository) Delete(ctx context.Context, id string) error {
	return r.dbManager.Delete(ctx, id, &models.Inventory{})
}

// DeleteByProductID deletes inventory record by product ID
func (r *InventoryRepository) DeleteByProductID(ctx context.Context, productID string) error {
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get inventory for deletion: %w", err)
	}
	return r.Delete(ctx, inventory.ID)
}

// Find retrieves inventory records with filtering
func (r *InventoryRepository) Find(ctx context.Context, filter *base.Filter) ([]*models.Inventory, error) {
	var inventories []*models.Inventory
	if err := r.dbManager.List(ctx, filter, &inventories); err != nil {
		return nil, fmt.Errorf("failed to find inventory records: %w", err)
	}
	return inventories, nil
}

// CountWithFilter counts inventory records with filtering
func (r *InventoryRepository) CountWithFilter(ctx context.Context, filter *base.Filter) (int64, error) {
	count, err := r.dbManager.Count(ctx, filter, &models.Inventory{})
	if err != nil {
		return 0, fmt.Errorf("failed to count inventory records: %w", err)
	}
	return count, nil
}

// FindOne retrieves a single inventory record with filtering
func (r *InventoryRepository) FindOne(ctx context.Context, filter *base.Filter) (*models.Inventory, error) {
	inventories, err := r.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	if len(inventories) == 0 {
		return nil, fmt.Errorf("inventory record not found")
	}

	return inventories[0], nil
}

// GetLowStockItems retrieves all items with low stock
func (r *InventoryRepository) GetLowStockItems(ctx context.Context) ([]*models.Inventory, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "stock_status", Operator: base.OpEqual, Value: string(models.StockStatusLowStock)},
			},
			Logic: base.LogicAnd,
		},
	}
	return r.Find(ctx, filter)
}

// GetOutOfStockItems retrieves all out of stock items
func (r *InventoryRepository) GetOutOfStockItems(ctx context.Context) ([]*models.Inventory, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "stock_status", Operator: base.OpEqual, Value: string(models.StockStatusOutOfStock)},
			},
			Logic: base.LogicAnd,
		},
	}
	return r.Find(ctx, filter)
}

// GetItemsNeedingReorder retrieves items that need to be reordered
func (r *InventoryRepository) GetItemsNeedingReorder(ctx context.Context) ([]*models.Inventory, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "current_stock", Operator: base.OpLessEqual, Value: "reorder_point"},
			},
			Logic: base.LogicAnd,
		},
	}
	return r.Find(ctx, filter)
}

// UpdateStock updates the stock for a product
func (r *InventoryRepository) UpdateStock(ctx context.Context, productID string, newStock int, updatedBy string) error {
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	inventory.UpdateStock(newStock, updatedBy)
	return r.Update(ctx, inventory)
}

// ReserveStock reserves stock for a product
func (r *InventoryRepository) ReserveStock(ctx context.Context, productID string, quantity int) error {
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	if err := inventory.ReserveStock(quantity); err != nil {
		return err
	}

	return r.Update(ctx, inventory)
}

// ReleaseStock releases reserved stock for a product
func (r *InventoryRepository) ReleaseStock(ctx context.Context, productID string, quantity int) error {
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	if err := inventory.ReleaseStock(quantity); err != nil {
		return err
	}

	return r.Update(ctx, inventory)
}

// SellStock processes a sale and updates stock
func (r *InventoryRepository) SellStock(ctx context.Context, productID string, quantity int, updatedBy string) error {
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	if err := inventory.SellStock(quantity, updatedBy); err != nil {
		return err
	}

	return r.Update(ctx, inventory)
}

// Restock adds stock to inventory
func (r *InventoryRepository) Restock(ctx context.Context, productID string, quantity int, updatedBy string) error {
	inventory, err := r.GetByProductID(ctx, productID)
	if err != nil {
		return fmt.Errorf("failed to get inventory: %w", err)
	}

	if err := inventory.Restock(quantity, updatedBy); err != nil {
		return err
	}

	return r.Update(ctx, inventory)
}
