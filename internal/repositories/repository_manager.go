package repositories

import (
	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// RepositoryManager manages all repositories
type RepositoryManager struct {
	UserRepository    *UserRepository
	ProductRepository *ProductRepository
	OrderRepository   *OrderRepository
	CartRepository    *CartRepository
	AddressRepository *AddressRepository
}

// NewRepositoryManager creates a new repository manager with database manager
func NewRepositoryManager(dbManager interface{}) *RepositoryManager {
	return &RepositoryManager{
		UserRepository:    NewUserRepository(dbManager),
		ProductRepository: NewProductRepository(dbManager),
		OrderRepository:   NewOrderRepository(dbManager),
		CartRepository:    NewCartRepository(dbManager),
		AddressRepository: NewAddressRepository(dbManager),
	}
}

// Product Repository
type ProductRepository struct {
	*base.BaseFilterableRepository[*models.Product]
}

func NewProductRepository(dbManager interface{}) *ProductRepository {
	repo := &ProductRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Product](),
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Order Repository
type OrderRepository struct {
	*base.BaseFilterableRepository[*models.Order]
}

func NewOrderRepository(dbManager interface{}) *OrderRepository {
	repo := &OrderRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Order](),
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Cart Repository
type CartRepository struct {
	*base.BaseFilterableRepository[*models.Cart]
}

func NewCartRepository(dbManager interface{}) *CartRepository {
	repo := &CartRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Cart](),
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Address Repository
type AddressRepository struct {
	*base.BaseFilterableRepository[*models.Address]
}

func NewAddressRepository(dbManager interface{}) *AddressRepository {
	repo := &AddressRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Address](),
	}
	repo.SetDBManager(dbManager)
	return repo
}
