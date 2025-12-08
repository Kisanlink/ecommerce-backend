package services

import (
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"go.uber.org/zap"
)

// ServiceManager manages all services following the development rules
type ServiceManager struct {
	UserService            *UserService
	ProductService         *ProductService
	InventoryService       *InventoryService
	ProfileService         *ProfileService
	FavoriteService        *FavoriteService
	SessionFavoriteService SessionFavoriteServiceInterface
	AddressService         AddressServiceInterface
	CartService            CartServiceInterface
	SessionCartService     SessionCartServiceInterface
	R2Service              *R2Service // Cloudflare R2 storage service
	DeliveryService        DeliveryServiceInterface
	OrderService           OrderServiceInterface
	ReviewService          *ReviewService
	ImageService           ImageServiceInterface
	WhatsAppService        *WhatsAppService
}

// NewServiceManager creates a new service manager with repositories
func NewServiceManager(repoManager *repositories.RepositoryManager) *ServiceManager {
	// Initialize logger for services
	logger, _ := zap.NewProduction()

	// Initialize R2 service for Cloudflare R2 storage
	var r2Service *R2Service
	r2Svc, err := NewR2Service(logger)
	if err != nil {
		logger.Warn("R2 service initialization failed - R2 storage will not be available",
			zap.Error(err))
	} else {
		r2Service = r2Svc
		logger.Info("R2 service initialized successfully",
			zap.String("bucket", r2Service.GetBucketName()))
	}

	// Create services
	addressService := NewAddressService(repoManager, logger)
	cartService := NewCartService(repoManager.CartRepository, repoManager.ProductRepository, repoManager.CouponRepository, repoManager.GetDBManager(), logger)
	sessionCartService := NewSessionCartService(repoManager.CartRepository, repoManager.ProductRepository, repoManager.CouponRepository, logger)
	sessionFavoriteService := NewSessionFavoriteService(repoManager.FavoriteRepository, repoManager.ProductRepository, repoManager.ImageRepository, repoManager.InventoryRepository, logger)
	deliveryService := NewDeliveryService(repoManager, logger)
	orderService := NewOrderService(repoManager.OrderRepository, cartService, repoManager.ProductRepository, repoManager.AddressRepository, repoManager.CouponRepository, r2Service, logger)

	// Create review service with PostgreSQL support
	userService := NewUserService(repoManager.UserRepository)
	profileService := NewProfileService(repoManager)
	reviewService := NewReviewService(repoManager.ReviewRepository, userService, profileService, logger)

	// Create image service
	imageService := NewImageService(repoManager.ImageRepository, repoManager.ProductRepository, logger)

	// Create WhatsApp service
	whatsappService := NewWhatsAppService(logger)

	return &ServiceManager{
		UserService:            userService,
		ProductService:         NewProductService(repoManager),
		InventoryService:       NewInventoryService(repoManager),
		ProfileService:         NewProfileService(repoManager),
		FavoriteService:        NewFavoriteService(repoManager),
		SessionFavoriteService: sessionFavoriteService,
		AddressService:         addressService,
		CartService:            cartService,
		SessionCartService:     sessionCartService,
		R2Service:              r2Service,
		DeliveryService:        deliveryService,
		OrderService:           orderService,
		ReviewService:          reviewService,
		ImageService:           imageService,
		WhatsAppService:        whatsappService,
	}
}
