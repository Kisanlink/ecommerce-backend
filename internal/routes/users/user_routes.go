package users

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterUserRoutes registers all user related routes
func RegisterUserRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	userHandler := handlers.NewUserHandler(serviceManager.UserService)

	// User management routes (protected)
	users := r.Group("/users")
	users.Use(middleware.FirebaseAuthMiddleware())
	{
		// User profile operations
		users.GET("/profile", userHandler.GetCurrentUserProfile) // Get current user profile
		users.PUT("/profile", userHandler.UpdateCurrentUser)     // Update current user profile
		users.DELETE("/profile", userHandler.DeleteCurrentUser)  // Delete current user

		// User management (admin only - add admin middleware later)
		users.GET("/:id", userHandler.GetUserByID)   // Get user by ID
		users.PUT("/:id", userHandler.UpdateUser)    // Update user by ID
		users.DELETE("/:id", userHandler.DeleteUser) // Delete user by ID
		users.GET("", userHandler.ListUsers)         // Get all users (admin)
	}

	// Future route groups (commented until handlers are implemented)
	/*
		// User addresses
		addresses := users.Group("/:userId/addresses")
		{
			addresses.GET("", addressHandler.GetUserAddresses)
			addresses.POST("", addressHandler.CreateAddress)
			addresses.GET("/:addressId", addressHandler.GetAddress)
			addresses.PUT("/:addressId", addressHandler.UpdateAddress)
			addresses.DELETE("/:addressId", addressHandler.DeleteAddress)
			addresses.PUT("/:addressId/default", addressHandler.SetDefaultAddress)
		}

		// User preferences
		preferences := users.Group("/:userId/preferences")
		{
			preferences.GET("", userHandler.GetUserPreferences)
			preferences.PUT("", userHandler.UpdateUserPreferences)
		}

		// User activity/history
		activity := users.Group("/:userId/activity")
		{
			activity.GET("/orders", userHandler.GetUserOrderHistory)
			activity.GET("/reviews", userHandler.GetUserReviews)
			activity.GET("/wishlist", userHandler.GetUserWishlist)
		}
	*/
}
