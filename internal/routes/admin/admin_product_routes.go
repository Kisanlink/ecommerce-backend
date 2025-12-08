package admin

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterAdminProductRoutes registers admin-only product management routes under /admin/products
func RegisterAdminProductRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	productHandler := handlers.NewProductHandler(serviceManager.ProductService, serviceManager.R2Service)

	admin := r.Group("/admin")
	// Require Firebase auth first, then admin guard against Postgres role
	admin.Use(middleware.FirebaseAuthMiddleware())
	admin.Use(middleware.AdminGuard(repoManager))

	products := admin.Group("/products")
	{
		// Product CRUD operations
		products.POST("", productHandler.CreateProduct)       // Create new product
		products.PUT("/:id", productHandler.UpdateProduct)    // Update product
		products.DELETE("/:id", productHandler.DeleteProduct) // Delete product

		// Product image management
		products.POST("/:id/images", productHandler.UploadProductImage)              // Upload product image
		products.GET("/:id/images", productHandler.GetProductImages)                 // Get product images
		products.DELETE("/:id/images/:imageId", productHandler.DeleteProductImage)   // Delete product image
		products.PUT("/:id/images/:imageId/primary", productHandler.SetPrimaryImage) // Set primary image
	}
}
