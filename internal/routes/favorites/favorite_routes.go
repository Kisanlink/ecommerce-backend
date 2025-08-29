package favorites

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterFavoriteRoutes registers all favorite-related routes
func RegisterFavoriteRoutes(router *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	// Create favorite handler
	favoriteHandler := handlers.NewFavoriteHandler(serviceManager)

	// Favorite routes (all require authentication)
	favorites := router.Group("/favorites")
	favorites.Use(middleware.FirebaseAuthMiddleware())
	{
		favorites.POST("", favoriteHandler.AddFavorite)        // Add favorite
		favorites.GET("", favoriteHandler.GetFavorites)        // Get user's favorites
		favorites.DELETE("", favoriteHandler.RemoveFavorite)   // Remove favorite
		favorites.GET("/check", favoriteHandler.CheckFavorite) // Check if product is favorite
	}
}
