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

	// Debug endpoints - now admin-gated (previously public, which leaked Delhivery config).
	debug := r.Group("/delivery/debug")
	debug.Use(middleware.FirebaseAuthMiddleware())
	debug.Use(middleware.AdminGuard(repoManager))
	{
		debug.GET("/config", deliveryHandler.GetConfigurationStatus)
		debug.GET("/validate-warehouse", deliveryHandler.ValidateWarehouse)
	}

	// Authenticated delivery routes (any signed-in user)
	delivery := r.Group("/delivery")
	delivery.Use(middleware.FirebaseAuthMiddleware())
	{
		// Serviceability check (authenticated)
		delivery.GET("/serviceability/:pincode", deliveryHandler.CheckServiceability)

		// Shipping estimation
		delivery.POST("/estimate", deliveryHandler.Estimate)

		// Tracking by waybill (authenticated)
		delivery.GET("/track/:waybill", deliveryHandler.GetTrackingInfo)

		// Read shipment details for an order (authenticated)
		delivery.GET("/orders/:id/shipment", deliveryHandler.GetOrderShipment)
	}

	// Admin-only shipment mutations: creating/retrying a shipment and syncing courier status
	// are recovery/operations tooling and must not be callable by ordinary users.
	adminDelivery := r.Group("/delivery/orders/:id")
	adminDelivery.Use(middleware.FirebaseAuthMiddleware())
	adminDelivery.Use(middleware.AdminGuard(repoManager))
	{
		adminDelivery.POST("/shipment", deliveryHandler.CreateOrderShipment)
		adminDelivery.POST("/sync", deliveryHandler.SyncOrderDeliveryStatus)
	}

	logger.Info("Delivery routes registered successfully")
}
