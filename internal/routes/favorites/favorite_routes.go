package favorites

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterFavoriteRoutes registers all favorite-related routes
func RegisterFavoriteRoutes(router *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		// Fallback to development logger if production fails
		logger, _ = zap.NewDevelopment()
	}
	defer logger.Sync()

	// Create session favorite handler
	var sessionFavoriteHandler *handlers.SessionFavoriteHandler
	if serviceManager != nil && serviceManager.FavoriteService != nil && serviceManager.SessionFavoriteService != nil {
		sessionFavoriteHandler = handlers.NewSessionFavoriteHandler(serviceManager.FavoriteService, serviceManager.SessionFavoriteService, logger)
	}

	// Create original favorite handler for authenticated-only operations
	var favoriteHandler *handlers.FavoriteHandler
	if serviceManager != nil && serviceManager.FavoriteService != nil {
		favoriteHandler = handlers.NewFavoriteHandler(serviceManager)
	} else {
		logger.Error("Favorite service not available in service manager")
	}

	// Public favorite routes (no authentication required, but can detect auth if present)
	favorites := router.Group("/favorites")

	// Apply optional authentication middleware to detect auth when present
	favorites.Use(middleware.OptionalFirebaseAuth())

	// Add error recovery middleware for favorite routes
	favorites.Use(func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("Favorite route panic recovered", zap.Any("panic", r))
				c.JSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error":   "Internal server error",
				})
				c.Abort()
			}
		}()
		c.Next()
	})

	// Check if session favorite handler is available
	if sessionFavoriteHandler != nil {
		// Core favorite operations (work for both authenticated and anonymous users)
		favorites.GET("", sessionFavoriteHandler.GetFavorites)            // Get user's favorites
		favorites.POST("", sessionFavoriteHandler.AddFavorite)            // Add favorite
		favorites.DELETE("", sessionFavoriteHandler.RemoveFavorite)       // Remove favorite
		favorites.GET("/check", sessionFavoriteHandler.CheckFavorite)     // Check if product is favorite
		favorites.DELETE("/clear", sessionFavoriteHandler.ClearFavorites) // Clear all favorites

		// Favorite migration endpoints (require authentication)
		authenticated := favorites.Group("")
		authenticated.Use(middleware.FirebaseAuthMiddleware())
		{
			authenticated.POST("/migrate", sessionFavoriteHandler.MigrateFavorites) // Migrate session favorites to user favorites
			authenticated.POST("/merge", sessionFavoriteHandler.MergeFavorites)     // Merge session favorites with user favorites
		}

		logger.Info("Session favorite routes registered successfully")
	} else {
		// Fallback when session favorite service is not available
		favorites.Any("/*path", func(c *gin.Context) {
			logger.Warn("Session favorite service unavailable for request", zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"success": false,
				"error":   "Favorite service temporarily unavailable",
			})
		})

		logger.Warn("Session favorite routes registered with fallback handlers - service unavailable")
	}

	// Authenticated-only favorite routes (require authentication) - kept for backward compatibility
	if favoriteHandler != nil {
		authenticatedFavorites := router.Group("/favorites")
		authenticatedFavorites.Use(middleware.FirebaseAuthMiddleware())
		{
			// These routes are kept for backward compatibility but are now redundant
			// since the session handler above handles both authenticated and anonymous users
			// authenticatedFavorites.POST("", favoriteHandler.AddFavorite)        // Add favorite
			// authenticatedFavorites.GET("", favoriteHandler.GetFavorites)        // Get user's favorites
			// authenticatedFavorites.DELETE("", favoriteHandler.RemoveFavorite)   // Remove favorite
			// authenticatedFavorites.GET("/check", favoriteHandler.CheckFavorite) // Check if product is favorite
		}

		logger.Info("Authenticated favorite routes registered (backward compatibility)")
	}
}
