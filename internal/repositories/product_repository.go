package repositories

import (
	"context"
	"log"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
)

// ProductRepositoryInterface defines the contract for product repository operations
type ProductRepositoryInterface interface {
	Create(ctx context.Context, product *models.Product) error
	GetByID(ctx context.Context, id string, product *models.Product) (*models.Product, error)
	Update(ctx context.Context, product *models.Product) error
	Delete(ctx context.Context, id string, product *models.Product) error
	Find(ctx context.Context, filter *base.Filter) ([]*models.Product, error)
	CountWithFilter(ctx context.Context, filter *base.Filter) (int64, error)
}

// ProductRepository handles product data operations
type ProductRepository struct {
	*base.BaseFilterableRepository[*models.Product]
	dbManager db.DBManager
}

// NewProductRepository creates a new product repository
func NewProductRepository(dbManager db.DBManager) *ProductRepository {
	repo := &ProductRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Product](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
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
