package delivery

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterDeliveryRoutes registers all delivery related routes
func RegisterDeliveryRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager, logger *zap.Logger) {
	// Validate inputs
	if r == nil || repoManager == nil || serviceManager == nil {
		if logger != nil {
			logger.Error("Invalid parameters for delivery routes registration")
		}
		return
	}

	// Initialize logger if not provided
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	// Validate required services
	if serviceManager.OrderService == nil {
		logger.Error("OrderService is required for delivery routes but not available")
		return
	}

	// Initialize services and handler
	deliveryService := services.NewDeliveryService(repoManager, logger)
	deliveryHandler := handlers.NewDeliveryHandler(deliveryService, serviceManager.OrderService, logger)

	// All delivery routes require authentication
	delivery := r.Group("/delivery")
	delivery.Use(middleware.FirebaseAuthMiddleware())
	{
		// Serviceability check (authenticated)
		delivery.GET("/serviceability/:pincode", deliveryHandler.CheckServiceability)

		// Shipping estimation
		delivery.POST("/estimate", deliveryHandler.Estimate)
		delivery.GET("/addresses", deliveryHandler.GetUserAddresses)

		// Tracking by waybill (authenticated)
		delivery.GET("/track/:waybill", deliveryHandler.GetTrackingInfo)

		// Order shipment operations
		orders := delivery.Group("/orders/:id")
		{
			orders.POST("/shipment", deliveryHandler.CreateOrderShipment)
			orders.GET("/shipment", deliveryHandler.GetOrderShipment)
		}
	}

	logger.Info("Delivery routes registered successfully")
}
