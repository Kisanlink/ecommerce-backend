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

	// Create session cart handler
	var sessionCartHandler *handlers.SessionCartHandler
	if serviceManager != nil && serviceManager.CartService != nil && serviceManager.SessionCartService != nil {
		sessionCartHandler = handlers.NewSessionCartHandler(serviceManager.CartService, serviceManager.SessionCartService, logger)
	}

	// Create original cart handler for authenticated-only operations
	var cartHandler *handlers.CartHandler
	if serviceManager != nil && serviceManager.CartService != nil {
		cartHandler = handlers.NewCartHandler(serviceManager.CartService, logger)
		if serviceManager.DeliveryService != nil {
			cartHandler.SetDeliveryService(serviceManager.DeliveryService)
		}
		if serviceManager.AddressService != nil {
			cartHandler.SetAddressService(serviceManager.AddressService)
		}
	} else {
		logger.Error("Cart service not available in service manager")
	}

	// Public cart routes (no authentication required, but can detect auth if present)
	cart := r.Group("/cart")

	// Apply optional authentication middleware to detect auth when present
	cart.Use(middleware.OptionalFirebaseAuth())

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

	// Check if session cart handler is available
	if sessionCartHandler != nil {
		// Core cart operations (work for both authenticated and anonymous users)
		cart.GET("", sessionCartHandler.GetCart)                             // Get user's cart
		cart.GET("/summary", sessionCartHandler.GetCartSummary)              // Get cart summary
		cart.POST("/items", sessionCartHandler.AddToCart)                    // Add item to cart
		cart.PUT("/items/:product_id", sessionCartHandler.UpdateCartItem)    // Update cart item
		cart.DELETE("/items/:product_id", sessionCartHandler.RemoveFromCart) // Remove item from cart
		cart.DELETE("/clear", sessionCartHandler.ClearCart)                  // Clear entire cart

		// Coupon operations (work for both authenticated and anonymous users)
		cart.POST("/apply-coupon", sessionCartHandler.ApplyCoupon)      // Apply coupon to cart
		cart.DELETE("/remove-coupon", sessionCartHandler.RemoveCoupon)  // Remove coupon from cart
		cart.GET("/validate-coupon", sessionCartHandler.ValidateCoupon) // Validate coupon for cart

		// Cart migration endpoints (require authentication)
		authenticated := cart.Group("")
		authenticated.Use(middleware.FirebaseAuthMiddleware())
		{
			authenticated.POST("/migrate", sessionCartHandler.MigrateCart) // Migrate session cart to user cart
			authenticated.POST("/merge", sessionCartHandler.MergeCart)     // Merge session cart with user cart
		}

		logger.Info("Session cart routes registered successfully")
	} else {
		// Fallback when session cart service is not available
		cart.Any("/*path", func(c *gin.Context) {
			logger.Warn("Session cart service unavailable for request", zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "Cart service temporarily unavailable",
			})
		})

		logger.Warn("Session cart routes registered with fallback handlers - service unavailable")
	}

	// Authenticated-only cart routes (require authentication)
	if cartHandler != nil {
		authenticatedCart := r.Group("/cart")
		authenticatedCart.Use(middleware.FirebaseAuthMiddleware())

		// Advanced cart operations (require authentication)
		authenticatedCart.GET("/count", cartHandler.GetCartItemCount) // Get item count
		authenticatedCart.GET("/validate", cartHandler.ValidateCart)  // Validate cart

		// Increment/Decrement operations (require authentication)
		authenticatedCart.POST("/items/:product_id/increment", cartHandler.IncrementCartItem) // Increment cart item quantity
		authenticatedCart.POST("/items/:product_id/decrement", cartHandler.DecrementCartItem) // Decrement cart item quantity

		// Health check for cart service
		authenticatedCart.GET("/health", cartHandler.HealthCheck) // Service health

		// Admin operations (should be protected by admin middleware in production)
		admin := authenticatedCart.Group("/admin")
		{
			admin.POST("/cleanup-duplicates", cartHandler.CleanupDuplicateCarts) // Clean up duplicate carts
		}

		// Checkout routes (require authentication)
		checkout := authenticatedCart.Group("/checkout")
		{
			checkout.GET("/prepare", cartHandler.PrepareCheckout)
			checkout.POST("/estimate-shipping", cartHandler.EstimateShipping)
			checkout.POST("/apply-coupon", cartHandler.ApplyCoupon)
			checkout.DELETE("/remove-coupon", cartHandler.RemoveCoupon)
		}

		logger.Info("Authenticated cart routes registered successfully")
	}
}
