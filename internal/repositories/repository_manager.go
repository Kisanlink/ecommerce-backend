package repositories

import (
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/database"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// RepositoryManager manages all repositories
type RepositoryManager struct {
	UserRepository                *UserRepository
	ProductRepository             ProductRepositoryInterface
	OrderRepository               *OrderRepository
	CartRepository                CartRepositoryInterface // Changed to interface
	AddressRepository             *AddressRepository
	RecipesRepository             *RecipesRepository
	PendingRecipeReviewRepository *PendingRecipeReviewRepository // Pending recipe reviews
	ReviewRepository              *ReviewRepository              // Product reviews
	ProfileRepository             *ProfileRepository
	FavoriteRepository            *FavoriteRepository
	ImageRepository               ImageRepositoryInterface     // Image repository
	InventoryRepository           InventoryRepositoryInterface // Inventory repository
	CouponRepository              CouponRepositoryInterface    // Coupon repository
	RedisManager                  *database.RedisManager       // Redis manager for cart operations

	// Database manager for accessing GORM DB instance
	dbManager db.DBManager
}

// GetDBManager returns the database manager
func (rm *RepositoryManager) GetDBManager() db.DBManager {
	return rm.dbManager
}

// NewRepositoryManager creates a new repository manager with database manager
// pgManager is passed separately for repositories that need direct GORM access
func NewRepositoryManager(dbManager db.DBManager, pgManager *db.PostgresManager, logger *zap.Logger) *RepositoryManager {
	// Create recipes repository with pgManager and logger (needs direct GORM access)
	recipesRepo := NewRecipesRepository(pgManager, logger)

	// Create pending recipe review repository with pgManager and logger (needs direct GORM access)
	pendingRecipeReviewRepo := NewPendingRecipeReviewRepository(pgManager, logger)

	// Create review repository with pgManager and logger (needs direct GORM access)
	reviewRepo := NewReviewRepository(pgManager, logger)

	// Initialize Redis manager
	redisManager, err := database.NewRedisManager(logger)
	if err != nil {
		logger.Error("Failed to create Redis manager", zap.Error(err))
		// Continue without Redis - fall back to PostgreSQL-only cart
	}

	return &RepositoryManager{
		UserRepository:                NewUserRepository(dbManager),
		ProductRepository:             NewProductRepository(dbManager),
		OrderRepository:               NewOrderRepository(dbManager),
		CartRepository:                NewCartRepository(dbManager, redisManager, logger), // Use new cart repository
		AddressRepository:             NewAddressRepository(dbManager),
		RecipesRepository:             recipesRepo,
		PendingRecipeReviewRepository: pendingRecipeReviewRepo,
		ReviewRepository:              reviewRepo,
		ProfileRepository:             NewProfileRepository(dbManager),
		FavoriteRepository:            NewFavoriteRepository(dbManager),
		ImageRepository:               NewImageRepository(dbManager),          // Image repository
		InventoryRepository:           NewInventoryRepository(dbManager),      // Inventory repository
		CouponRepository:              NewCouponRepository(dbManager, logger), // Coupon repository
		RedisManager:                  redisManager,
		dbManager:                     dbManager, // Store dbManager for access
	}
}

// Close closes all repository connections properly
func (rm *RepositoryManager) Close() error {
	var errs []error

	// Close Redis connection if available
	if rm.RedisManager != nil {
		if err := rm.RedisManager.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close Redis: %w", err))
		}
	}

	// Note: Database manager is closed by the main application
	// as it may be shared across multiple repository managers

	if len(errs) > 0 {
		return fmt.Errorf("errors closing repository connections: %v", errs)
	}

	return nil
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
