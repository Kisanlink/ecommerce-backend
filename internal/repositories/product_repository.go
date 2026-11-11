package repositories

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"gorm.io/gorm"
)

// ProductRepositoryInterface defines the contract for product repository operations
type ProductRepositoryInterface interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id string, product *models.Product) (*models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id string, product *models.Product) error
	Find(ctx context.Context, filter *base.Filter) ([]*models.Product, error)
	CountWithFilter(ctx context.Context, filter *base.Filter) (int64, error)
	FindOne(ctx context.Context, filter *base.Filter) (*models.Product, error)
	Exists(ctx context.Context, id string) (bool, error)
	SoftDelete(ctx context.Context, id string, deletedBy string) error
	// DecrementStockAtomic decrements a product's stock by qty in a single atomic
	// UPDATE. Pass a non-nil tx to enlist it in a larger transaction, or nil to run
	// on the repository's own connection. Returns oversold=true when pre-update
	// stock was below qty.
	DecrementStockAtomic(ctx context.Context, tx *gorm.DB, productID string, qty int) (bool, error)
}

// ProductRepository handles product data operations
type ProductRepository struct {
	*base.BaseFilterableRepository[*models.Product]
	dbManager db.DBManager
	pgManager *db.PostgresManager // direct GORM access for atomic stock updates
}

// NewProductRepository creates a new product repository
func NewProductRepository(dbManager db.DBManager, pgManager *db.PostgresManager) *ProductRepository {
	repo := &ProductRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Product](),
		dbManager:                dbManager,
		pgManager:                pgManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// DecrementStockAtomic decrements a product's stock by qty in a single atomic UPDATE
// (GREATEST(stock - qty, 0)), so concurrent deductions cannot lose updates and stock
// never goes negative. stock_status is flipped to "Out of Stock" when the result
// reaches zero. It reports oversold=true when the pre-update stock was below qty, so
// the caller can surface the oversell instead of silently flooring it. When tx is
// non-nil the update runs inside that transaction (so it can be atomic with other
// writes, e.g. the order's inventory_deducted guard); when nil it runs on the
// repository's own connection.
func (r *ProductRepository) DecrementStockAtomic(ctx context.Context, tx *gorm.DB, productID string, qty int) (bool, error) {
	gdb := tx
	if gdb == nil {
		if r.pgManager == nil {
			return false, fmt.Errorf("postgres manager unavailable for atomic stock decrement")
		}
		var err error
		gdb, err = r.pgManager.GetDB(ctx, false)
		if err != nil {
			return false, fmt.Errorf("failed to get db for stock decrement: %w", err)
		}
	}

	// Read current stock first (on the same connection/tx) so we can report an oversell.
	var current int
	if err := gdb.WithContext(ctx).
		Model(&models.Product{}).
		Where("id = ? AND deleted_at IS NULL", productID).
		Select("stock").
		Scan(&current).Error; err != nil {
		return false, fmt.Errorf("failed to read stock for %s: %w", productID, err)
	}
	oversold := current < qty

	res := gdb.WithContext(ctx).
		Model(&models.Product{}).
		Where("id = ? AND deleted_at IS NULL", productID).
		Updates(map[string]interface{}{
			"stock":        gorm.Expr("GREATEST(stock - ?, 0)", qty),
			"stock_status": gorm.Expr("CASE WHEN stock - ? <= 0 THEN ? ELSE stock_status END", qty, "Out of Stock"),
		})
	if res.Error != nil {
		return oversold, fmt.Errorf("failed to decrement stock for %s: %w", productID, res.Error)
	}
	return oversold, nil
}

// Create creates a new product
func (r *ProductRepository) Create(ctx context.Context, product *models.Product) error {
	log.Printf("Creating product: %+v", product)

	if err := r.dbManager.Create(ctx, product); err != nil {
		log.Printf("Error creating product: %v", err)
		return err
	}

	log.Printf("Successfully created product with ID: %s", product.ID)
	return nil
}

// GetByID retrieves a product by ID
func (r *ProductRepository) GetByID(ctx context.Context, id string, product *models.Product) (*models.Product, error) {
	log.Printf("Getting product by ID: %s", id)

	if err := r.dbManager.GetByID(ctx, id, product); err != nil {
		log.Printf("Error getting product by ID %s: %v", id, err)
		return nil, err
	}

	log.Printf("Successfully retrieved product: %s", product.Name)
	return product, nil
}

// Update updates a product
func (r *ProductRepository) Update(ctx context.Context, product *models.Product) error {
	log.Printf("Updating product: %s", product.ID)

	if err := r.dbManager.Update(ctx, product); err != nil {
		log.Printf("Error updating product: %v", err)
		return err
	}

	log.Printf("Successfully updated product: %s", product.ID)
	return nil
}

// Delete soft deletes a product
func (r *ProductRepository) Delete(ctx context.Context, id string, product *models.Product) error {
	log.Printf("Soft deleting product: %s", id)

	if err := r.dbManager.SoftDelete(ctx, id, product, "system"); err != nil {
		log.Printf("Error deleting product: %v", err)
		return err
	}

	log.Printf("Successfully deleted product: %s", id)
	return nil
}

// Find retrieves products with filtering
func (r *ProductRepository) Find(ctx context.Context, filter *base.Filter) ([]*models.Product, error) {
	log.Printf("Finding products with filter: %+v", filter)

	var products []*models.Product
	if err := r.dbManager.List(ctx, filter, &products); err != nil {
		log.Printf("Error finding products: %v", err)
		return nil, err
	}

	log.Printf("Successfully found %d products", len(products))
	return products, nil
}

// CountWithFilter counts products with filtering
func (r *ProductRepository) CountWithFilter(ctx context.Context, filter *base.Filter) (int64, error) {
	log.Printf("Counting products with filter: %+v", filter)

	count, err := r.dbManager.Count(ctx, filter, &models.Product{})
	if err != nil {
		log.Printf("Error counting products: %v", err)
		return 0, err
	}

	log.Printf("Successfully counted %d products", count)
	return count, nil
}

// FindOne retrieves a single product with filtering
func (r *ProductRepository) FindOne(ctx context.Context, filter *base.Filter) (*models.Product, error) {
	log.Printf("Finding one product with filter: %+v", filter)

	products, err := r.Find(ctx, filter)
	if err != nil {
		log.Printf("Error finding product: %v", err)
		return nil, err
	}

	if len(products) == 0 {
		return nil, fmt.Errorf("product not found")
	}

	log.Printf("Successfully found product: %s", products[0].Name)
	return products[0], nil
}

// Exists checks if a product exists by ID
func (r *ProductRepository) Exists(ctx context.Context, id string) (bool, error) {
	log.Printf("Checking if product exists: %s", id)

	var product models.Product
	_, err := r.GetByID(ctx, id, &product)
	if err != nil {
		// If error is "not found", return false
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "record not found") {
			return false, nil
		}
		log.Printf("Error checking product existence: %v", err)
		return false, err
	}

	log.Printf("Product exists: %s", id)
	return true, nil
}

// SoftDelete soft deletes a product
func (r *ProductRepository) SoftDelete(ctx context.Context, id string, deletedBy string) error {
	log.Printf("Soft deleting product: %s by %s", id, deletedBy)

	var product models.Product
	if err := r.dbManager.SoftDelete(ctx, id, &product, deletedBy); err != nil {
		log.Printf("Error soft deleting product: %v", err)
		return err
	}

	log.Printf("Successfully soft deleted product: %s", id)
	return nil
}
