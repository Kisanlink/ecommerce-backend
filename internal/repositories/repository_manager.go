package repositories

import (
	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// RepositoryManager manages all repositories
type RepositoryManager struct {
	UserRepository    *UserRepository
	ProductRepository *ProductRepository
	OrderRepository   *OrderRepository
	CartRepository    *CartRepository
	AddressRepository *AddressRepository
	RecipesRepository *RecipesRepository
	ProfileRepository *ProfileRepository
}

// NewRepositoryManager creates a new repository manager with database manager
func NewRepositoryManager(dbManager db.DBManager, logger *zap.Logger) *RepositoryManager {
	// Create recipes repository with logger
	recipesRepo, err := NewRecipesRepository(logger)
	if err != nil {
		logger.Error("Failed to create recipes repository", zap.Error(err))
		// Continue without recipes repository rather than failing completely
	}

	return &RepositoryManager{
		UserRepository:    NewUserRepository(dbManager),
		ProductRepository: NewProductRepository(dbManager),
		OrderRepository:   NewOrderRepository(dbManager),
		CartRepository:    NewCartRepository(dbManager),
		AddressRepository: NewAddressRepository(dbManager),
		RecipesRepository: recipesRepo,
		ProfileRepository: NewProfileRepository(dbManager),
	}
}

// Order Repository
type OrderRepository struct {
	*base.BaseFilterableRepository[*models.Order]
	dbManager db.DBManager
}

func NewOrderRepository(dbManager db.DBManager) *OrderRepository {
	repo := &OrderRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Order](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Cart Repository
type CartRepository struct {
	*base.BaseFilterableRepository[*models.Cart]
	dbManager db.DBManager
}

func NewCartRepository(dbManager db.DBManager) *CartRepository {
	repo := &CartRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Cart](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Address Repository
type AddressRepository struct {
	*base.BaseFilterableRepository[*models.Address]
	dbManager db.DBManager
}

func NewAddressRepository(dbManager db.DBManager) *AddressRepository {
	repo := &AddressRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Address](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}
