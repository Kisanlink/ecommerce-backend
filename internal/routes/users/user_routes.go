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
		// User profile operations (any authenticated user, self only)
		users.GET("/profile", userHandler.GetCurrentUserProfile) // Get current user profile
		users.PUT("/profile", userHandler.UpdateCurrentUser)     // Update current user profile

		// User management — admin only. These expose every user's PII and allow
		// updating/deleting arbitrary accounts, so they MUST sit behind AdminGuard
		// (previously only FirebaseAuthMiddleware, so any authenticated customer
		// could dump the user table or delete accounts via full IDOR).
		adminUsers := users.Group("")
		adminUsers.Use(middleware.AdminGuard(repoManager))
		{
			adminUsers.GET("/:id", userHandler.GetUserByID)   // Get user by ID
			adminUsers.PUT("/:id", userHandler.UpdateUser)    // Update user by ID
			adminUsers.DELETE("/:id", userHandler.DeleteUser) // Delete user by ID
			adminUsers.GET("", userHandler.ListUsers)         // Get all users
		}
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
