package products

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterProductRoutes registers public product routes (no authentication required)
// Admin product routes are registered separately via admin.RegisterAdminProductRoutes
func RegisterProductRoutes(r *gin.RouterGroup, serviceManager *services.ServiceManager) {
	productHandler := handlers.NewProductHandler(serviceManager.ProductService, serviceManager.R2Service)

	// Public product routes (no authentication required)
	products := r.Group("/products")
	{
		products.GET("", productHandler.GetProducts)                              // Get all products with filters
		products.GET("/:id", productHandler.GetProduct)                           // Get product by ID
		products.GET("/category/:category", productHandler.GetProductsByCategory) // Get products by category
		products.GET("/search", productHandler.SearchProducts)                    // Search products
	}
}
