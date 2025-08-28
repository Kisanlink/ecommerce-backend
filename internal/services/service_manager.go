package services

import (
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
)

// ServiceManager manages all services following the development rules
type ServiceManager struct {
	UserService    *UserService
	ProductService *ProductService
	ProfileService *ProfileService
	S3Service      *S3Service
	// TODO: Add other services as they are implemented
	// OrderService   *OrderService
	// CartService    *CartService
}

// NewServiceManager creates a new service manager with repositories
func NewServiceManager(repoManager *repositories.RepositoryManager) *ServiceManager {
	return &ServiceManager{
		UserService:    NewUserService(repoManager.UserRepository),
		ProductService: NewProductService(repoManager),
		ProfileService: NewProfileService(repoManager),
		S3Service:      nil, // TODO: Initialize with S3Manager when available
		// TODO: Initialize other services as they are implemented
		// OrderService:   NewOrderService(repoManager.OrderRepository),
		// CartService:    NewCartService(repoManager.CartRepository),
	}
}
