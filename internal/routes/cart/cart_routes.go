package cart

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterCartRoutes registers all cart related routes
func RegisterCartRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	// Initialize logger
	logger, _ := zap.NewProduction()

	// Create cart handler with cart service
	var cartHandler *handlers.CartHandler
	if serviceManager.CartService != nil {
		cartHandler = handlers.NewCartHandler(serviceManager.CartService, logger)
	}

	// Cart routes (requires authentication)
	cart := r.Group("/cart")
	cart.Use(middleware.FirebaseAuthMiddleware())
	{
		if cartHandler != nil {
			// Core cart operations
			cart.GET("", cartHandler.GetCart)                // Get user's cart
			cart.GET("/summary", cartHandler.GetCartSummary) // Get cart summary
			cart.GET("/count", cartHandler.GetCartItemCount) // Get item count
			cart.GET("/validate", cartHandler.ValidateCart)  // Validate cart

			// Cart item management
			cart.POST("/items", cartHandler.AddToCart)                               // Add item to cart
			cart.PUT("/items/:product_id", cartHandler.UpdateCartItem)               // Update cart item
			cart.POST("/items/:product_id/increment", cartHandler.IncrementCartItem) // Increment cart item quantity
			cart.POST("/items/:product_id/decrement", cartHandler.DecrementCartItem) // Decrement cart item quantity
			cart.DELETE("/items/:product_id", cartHandler.RemoveFromCart)            // Remove item from cart

			// Clear cart operation
			cart.DELETE("/clear", cartHandler.ClearCart) // Clear entire cart

			// Health check for cart service
			cart.GET("/health", cartHandler.HealthCheck) // Service health
		} else {
			// Fallback when cart service is not available
			cart.Any("/*path", func(c *gin.Context) {
				c.JSON(503, gin.H{
					"status":  "error",
					"message": "Cart service temporarily unavailable",
				})
			})
		}
	}

	// Future route groups (commented until handlers are implemented)
	/*
		// Cart checkout preparation
		checkout := cart.Group("/checkout")
		{
			checkout.GET("/prepare", cartHandler.PrepareCheckout)          // Prepare cart for checkout
			checkout.POST("/estimate-shipping", cartHandler.EstimateShipping) // Estimate shipping costs
			checkout.POST("/apply-coupon", cartHandler.ApplyCoupon)        // Apply discount coupon
			checkout.DELETE("/remove-coupon", cartHandler.RemoveCoupon)    // Remove discount coupon
		}

		// Saved for later / Wishlist
		saved := cart.Group("/saved")
		{
			saved.GET("", cartHandler.GetSavedItems)                       // Get saved items
			saved.POST("/items/:itemId", cartHandler.SaveForLater)         // Save cart item for later
			saved.PUT("/items/:itemId/move-to-cart", cartHandler.MoveToCart) // Move saved item back to cart
			saved.DELETE("/items/:itemId", cartHandler.RemoveSavedItem)    // Remove saved item
		}
	*/
}
