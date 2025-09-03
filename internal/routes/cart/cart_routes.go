package cart

import (
	"net/http"

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
	logger, err := zap.NewProduction()
	if err != nil {
		// Fallback to development logger if production fails
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	// Create cart handler with cart service
	var cartHandler *handlers.CartHandler
	if serviceManager != nil && serviceManager.CartService != nil {
		cartHandler = handlers.NewCartHandler(serviceManager.CartService, logger)
		if serviceManager.DeliveryService != nil {
			cartHandler.SetDeliveryService(serviceManager.DeliveryService)
		}
	} else {
		logger.Error("Cart service not available in service manager")
	}

	// Cart routes (requires authentication)
	cart := r.Group("/cart")

	// Apply authentication middleware with error recovery
	cart.Use(middleware.FirebaseAuthMiddleware())

	// Add error recovery middleware for cart routes
	cart.Use(func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Cart route panic recovered", zap.Any("panic", r))
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	})

	// Check if cart handler is available
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

		// Admin operations (should be protected by admin middleware in production)
		admin := cart.Group("/admin")
		{
			admin.POST("/cleanup-duplicates", cartHandler.CleanupDuplicateCarts) // Clean up duplicate carts
		}

		logger.Info("Cart routes registered successfully")
	} else {
		// Fallback when cart service is not available
		cart.Any("/*path", func(c *gin.Context) {
			logger.Warn("Cart service unavailable for request", zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "Cart service temporarily unavailable",
			})
		})

		logger.Warn("Cart routes registered with fallback handlers - service unavailable")
	}

	// Checkout routes
	if cartHandler != nil {
		checkout := cart.Group("/checkout")
		{
			checkout.GET("/prepare", cartHandler.PrepareCheckout)
			checkout.POST("/estimate-shipping", cartHandler.EstimateShipping)
			checkout.POST("/apply-coupon", cartHandler.ApplyCoupon)
			checkout.DELETE("/remove-coupon", cartHandler.RemoveCoupon)
		}
	}
}
