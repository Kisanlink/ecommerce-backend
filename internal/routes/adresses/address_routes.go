package routes

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// AddressRoutes sets up routes for address management
func AddressRoutes(router *gin.Engine, addressHandler *handlers.AddressHandler) {
	addressGroup := router.Group("/api/v1/addresses")
	addressGroup.Use(middleware.FirebaseAuthMiddleware())

	// Address management endpoints
	addressGroup.GET("", addressHandler.GetAddresses)           // Get all user addresses
	addressGroup.POST("", addressHandler.CreateAddress)        // Create new address
	addressGroup.GET("/:id", addressHandler.GetAddress)        // Get single address by ID
	addressGroup.PUT("/:id", addressHandler.UpdateAddress)     // Update existing address
	addressGroup.DELETE("/:id", addressHandler.DeleteAddress)  // Delete address
	addressGroup.PUT("/:id/default", addressHandler.SetDefaultAddress) // Set as default
}
