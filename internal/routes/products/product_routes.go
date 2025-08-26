package products

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterProductRoutes registers all product related routes
func RegisterProductRoutes(r *gin.RouterGroup, serviceManager *services.ServiceManager, repoManager *repositories.RepositoryManager) {
	// Create product handler with service
	productHandler := handlers.NewProductHandler(serviceManager.ProductService)

	// Public product routes (no authentication required)
	products := r.Group("/products")
	{
		products.GET("", productHandler.GetProducts)                              // Get all products with filters
		products.GET("/:id", productHandler.GetProduct)                           // Get product by ID
		products.GET("/category/:category", productHandler.GetProductsByCategory) // Get products by category
		products.GET("/search", productHandler.SearchProducts)                    // Search products
	}

	// Admin-only product routes (requires authentication + admin role)
	adminProducts := products.Group("/")
	adminProducts.Use(middleware.FirebaseAuthMiddleware())
	adminProducts.Use(middleware.AdminGuard(repoManager))
	{
		adminProducts.POST("", productHandler.CreateProduct)       // Create new product (admin only)
		adminProducts.PUT("/:id", productHandler.UpdateProduct)    // Update product (admin only)
		adminProducts.DELETE("/:id", productHandler.DeleteProduct) // Delete product (admin only)
	}

	// Future route groups (commented until handlers are implemented)
	/*
		// Product categories
		categories := r.Group("/categories")
		{
			categories.GET("", categoryHandler.GetCategories)              // Get all categories
			categories.GET("/:id", categoryHandler.GetCategory)           // Get category by ID
			categories.GET("/:id/products", productHandler.GetProducts)   // Get products in category
		}

		// Product reviews
		reviews := products.Group("/:productId/reviews")
		reviews.Use(middleware.OptionalFirebaseAuth())
		{
			reviews.GET("", reviewHandler.GetProductReviews)              // Get product reviews
			reviews.POST("", reviewHandler.CreateReview)                 // Create review (requires auth)
			reviews.GET("/:reviewId", reviewHandler.GetReview)           // Get specific review
			reviews.PUT("/:reviewId", reviewHandler.UpdateReview)        // Update review (owner only)
			reviews.DELETE("/:reviewId", reviewHandler.DeleteReview)     // Delete review (owner/admin only)
		}

		// Product inventory (admin only)
		inventory := products.Group("/:productId/inventory")
		inventory.Use(middleware.FirebaseAuthMiddleware()) // Add admin middleware later
		{
			inventory.GET("", inventoryHandler.GetProductInventory)
			inventory.PUT("", inventoryHandler.UpdateProductInventory)
			inventory.POST("/restock", inventoryHandler.RestockProduct)
		}
	*/
}
