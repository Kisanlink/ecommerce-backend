package services

import (
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// ServiceManager manages all services following the development rules
type ServiceManager struct {
	UserService     *UserService
	ProductService  *ProductService
	ProfileService  *ProfileService
	FavoriteService *FavoriteService
	CartService     CartServiceInterface
	S3Service       *S3Service
	DeliveryService DeliveryServiceInterface
	OrderService    OrderServiceInterface
}

// NewServiceManager creates a new service manager with repositories
func NewServiceManager(repoManager *repositories.RepositoryManager, s3Manager *db.S3Manager) *ServiceManager {
	// Initialize logger for services
	logger, _ := zap.NewProduction()

	var s3Service *S3Service
	if s3Manager != nil {
		s3Service = NewS3Service(s3Manager)
	} else {
		// Create an S3Service with nil manager - it will handle nil checks gracefully
		s3Service = NewS3Service(nil)
	}

	// Create services
	cartService := NewCartService(repoManager.CartRepository, repoManager.ProductRepository, repoManager.GetDBManager(), logger)
	deliveryService := NewDeliveryService(repoManager, logger)
	orderService := NewOrderService(repoManager.OrderRepository, cartService, repoManager.ProductRepository, repoManager.AddressRepository, s3Service, logger)

	return &ServiceManager{
		UserService:     NewUserService(repoManager.UserRepository),
		ProductService:  NewProductService(repoManager),
		ProfileService:  NewProfileService(repoManager),
		FavoriteService: NewFavoriteService(repoManager),
		CartService:     cartService,
		S3Service:       s3Service,
		DeliveryService: deliveryService,
		OrderService:    orderService,
	}
}
