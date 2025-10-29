package address

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterAddressRoutes registers all address related routes
func RegisterAddressRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	addressHandler := handlers.NewAddressHandler(serviceManager.AddressService)

	// Address management routes (protected)
	addresses := r.Group("/addresses")
	addresses.Use(middleware.FirebaseAuthMiddleware())
	{
		// Get all saved addresses
		addresses.GET("", addressHandler.GetUserAddresses)

		// Get last 4 addresses (for quick selection)
		addresses.GET("/history", addressHandler.GetLast4Addresses)

		// Get address history from orders
		addresses.GET("/order-history", addressHandler.GetAddressHistory)

		// Get both saved and history addresses
		addresses.GET("/all", addressHandler.GetAllAddresses)

		// Create new address
		addresses.POST("", addressHandler.CreateAddress)

		// Save address from order history
		addresses.POST("/save-from-order", addressHandler.SaveAddressFromOrder)

		// Get specific address by ID
		addresses.GET("/:id", addressHandler.GetAddressByID)

		// Update address
		addresses.PUT("/:id", addressHandler.UpdateAddress)

		// Set default address
		addresses.PUT("/:id/default", addressHandler.SetDefaultAddress)

		// Delete address
		addresses.DELETE("/:id", addressHandler.DeleteAddress)
	}
}
