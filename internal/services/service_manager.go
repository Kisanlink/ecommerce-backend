package services

import (
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
)

// ServiceManager manages all services following the development rules
type ServiceManager struct {
	UserService *UserService
	// TODO: Add other services as they are implemented
	// ProductService *ProductService
	// OrderService   *OrderService
	// CartService    *CartService
}

// NewServiceManager creates a new service manager with repositories
func NewServiceManager(repoManager *repositories.RepositoryManager) *ServiceManager {
	return &ServiceManager{
		UserService: NewUserService(repoManager.UserRepository),
		// TODO: Initialize other services as they are implemented
		// ProductService: NewProductService(repoManager.ProductRepository),
		// OrderService:   NewOrderService(repoManager.OrderRepository),
		// CartService:    NewCartService(repoManager.CartRepository),
	}
}
