package repositories

import (
	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/database"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// RepositoryManager manages all repositories
type RepositoryManager struct {
	UserRepository     *UserRepository
	ProductRepository  *ProductRepository
	OrderRepository    *OrderRepository
	CartRepository     CartRepositoryInterface // Changed to interface
	AddressRepository  *AddressRepository
	RecipesRepository  *RecipesRepository
	ProfileRepository  *ProfileRepository
	FavoriteRepository *FavoriteRepository
	RedisManager       *database.RedisManager // Redis manager for cart operations

	// Database manager for accessing GORM DB instance
	dbManager db.DBManager
}

// GetDBManager returns the database manager
func (rm *RepositoryManager) GetDBManager() db.DBManager {
	return rm.dbManager
}

// NewRepositoryManager creates a new repository manager with database manager
func NewRepositoryManager(dbManager db.DBManager, logger *zap.Logger) *RepositoryManager {
	// Create recipes repository with logger
	recipesRepo, err := NewRecipesRepository(logger)
	if err != nil {
		logger.Error("Failed to create recipes repository", zap.Error(err))
		// Continue without recipes repository rather than failing completely
	}

	// Initialize Redis manager
	redisManager, err := database.NewRedisManager(logger)
	if err != nil {
		logger.Error("Failed to create Redis manager", zap.Error(err))
		// Continue without Redis - fall back to PostgreSQL-only cart
	}

	return &RepositoryManager{
		UserRepository:     NewUserRepository(dbManager),
		ProductRepository:  NewProductRepository(dbManager),
		OrderRepository:    NewOrderRepository(dbManager),
		CartRepository:     NewCartRepository(dbManager, redisManager, logger), // Use new cart repository
		AddressRepository:  NewAddressRepository(dbManager),
		RecipesRepository:  recipesRepo,
		ProfileRepository:  NewProfileRepository(dbManager),
		FavoriteRepository: NewFavoriteRepository(dbManager),
		RedisManager:       redisManager,
		dbManager:          dbManager, // Store dbManager for access
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
