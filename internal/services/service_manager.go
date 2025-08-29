package services

import (
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
)

// ServiceManager manages all services following the development rules
type ServiceManager struct {
	UserService     *UserService
	ProductService  *ProductService
	ProfileService  *ProfileService
	FavoriteService *FavoriteService
	S3Service       *S3Service
	// TODO: Add other services as they are implemented
	// OrderService   *OrderService
	// CartService    *CartService
}

// NewServiceManager creates a new service manager with repositories
func NewServiceManager(repoManager *repositories.RepositoryManager, s3Manager *db.S3Manager) *ServiceManager {
	var s3Service *S3Service
	if s3Manager != nil {
		s3Service = NewS3Service(s3Manager)
	} else {
		// Create an S3Service with nil manager - it will handle nil checks gracefully
		s3Service = NewS3Service(nil)
	}

	return &ServiceManager{
		UserService:     NewUserService(repoManager.UserRepository),
		ProductService:  NewProductService(repoManager),
		ProfileService:  NewProfileService(repoManager),
		FavoriteService: NewFavoriteService(repoManager),
		S3Service:       s3Service,
		// TODO: Initialize other services as they are implemented
		// OrderService:   NewOrderService(repoManager.OrderRepository),
		// CartService:    NewCartService(repoManager.CartRepository),
	}
}
