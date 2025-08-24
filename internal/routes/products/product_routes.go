package products

import (
	// "github.com/Kisanlink/ecommerce-backend/internal/handlers"
	// "github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterProductRoutes registers all product related routes
func RegisterProductRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager) {
	// TODO: Create NewProductHandler when product_handler.go is implemented
	// productHandler := handlers.NewProductHandler(repoManager.ProductRepository)

	// Placeholder routes until handlers are implemented
	products := r.Group("/products")
	{
		products.GET("", func(c *gin.Context) { c.JSON(501, gin.H{"error": "Product handler not implemented yet"}) })
		products.GET("/:id", func(c *gin.Context) { c.JSON(501, gin.H{"error": "Product handler not implemented yet"}) })
		products.GET("/category/:category", func(c *gin.Context) { c.JSON(501, gin.H{"error": "Product handler not implemented yet"}) })
		products.GET("/search", func(c *gin.Context) { c.JSON(501, gin.H{"error": "Product handler not implemented yet"}) })
	}

	// Protected product routes (commented until handler is implemented)
	/*
		protected := products.Group("/")
		protected.Use(middleware.FirebaseAuthMiddleware())
		{
			protected.POST("", productHandler.CreateProduct)           // Create new product
			protected.PUT("/:id", productHandler.UpdateProduct)        // Update product
			protected.DELETE("/:id", productHandler.DeleteProduct)     // Delete product
			protected.PUT("/:id/status", productHandler.UpdateProduct) // Update product status
		}
	*/

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
