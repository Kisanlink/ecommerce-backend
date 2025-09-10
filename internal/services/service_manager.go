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
	AddressService  AddressServiceInterface
	CartService     CartServiceInterface
	S3Service       *S3Service
	DeliveryService DeliveryServiceInterface
	OrderService    OrderServiceInterface
	ReviewService   *ReviewService
	ImageService    ImageServiceInterface
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
	addressService := NewAddressService(repoManager, logger)
	cartService := NewCartService(repoManager.CartRepository, repoManager.ProductRepository, repoManager.CouponRepository, repoManager.GetDBManager(), logger)
	deliveryService := NewDeliveryService(repoManager, logger)
	orderService := NewOrderService(repoManager.OrderRepository, cartService, repoManager.ProductRepository, repoManager.AddressRepository, repoManager.CouponRepository, s3Service, logger)

	// Create review service with DynamoDB support
	dbManager := repoManager.GetDBManager()
	userService := NewUserService(repoManager.UserRepository)
	profileService := NewProfileService(repoManager)
	reviewService := NewReviewService(dbManager, userService, profileService, logger)

	// Create image service
	imageService := NewImageService(repoManager.ImageRepository, repoManager.ProductRepository, logger)

	return &ServiceManager{
		UserService:     userService,
		ProductService:  NewProductService(repoManager),
		ProfileService:  NewProfileService(repoManager),
		FavoriteService: NewFavoriteService(repoManager),
		AddressService:  addressService,
		CartService:     cartService,
		S3Service:       s3Service,
		DeliveryService: deliveryService,
		OrderService:    orderService,
		ReviewService:   reviewService,
		ImageService:    imageService,
	}
}
