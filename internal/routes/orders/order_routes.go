package orders

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterOrderRoutes registers all order related routes
func RegisterOrderRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager, logger *zap.Logger) {
	// Create order handler with dependencies
	orderHandler := handlers.NewOrderHandler(serviceManager.OrderService, logger)

	// Protected order routes (requires authentication)
	orders := r.Group("/orders")
	orders.Use(middleware.FirebaseAuthMiddleware())
	{
		orders.POST("", orderHandler.CreateOrder)                  // Create new order manually
		orders.GET("", orderHandler.GetUserOrders)                 // Get all user orders
		orders.GET("/:id", orderHandler.GetOrderDetails)           // Get specific order details
		orders.GET("/:id/status", orderHandler.GetOrderStatus)     // Get order status
		orders.GET("/:id/tracking", orderHandler.GetOrderTracking) // Get order tracking
	}

	// Debug endpoint without authentication
	debug := r.Group("/debug")
	{
		debug.GET("/order/:id", orderHandler.GetOrderDetailsDebug) // Debug order details
	}

	// Future route groups (commented until handlers are implemented)
	// Admin order routes
	adminOrders := r.Group("/admin/orders")
	adminOrders.Use(middleware.FirebaseAuthMiddleware())
	{
		adminOrders.GET("", orderHandler.GetAllOrders)                 // GET /api/v1/admin/orders
		adminOrders.GET("/:id", orderHandler.GetOrderDetailsAdmin)     // GET /api/v1/admin/orders/:id
		adminOrders.PUT("/:id/status", orderHandler.UpdateOrderStatus) // PUT /api/v1/admin/orders/:id/status
		adminOrders.PUT("/:id/payment", orderHandler.UpdatePayment)    // PUT /api/v1/admin/orders/:id/payment
		adminOrders.PUT("/:id/shipment", orderHandler.UpdateShipment)  // PUT /api/v1/admin/orders/:id/shipment
		adminOrders.GET("/stats", orderHandler.GetOrderStats)          // GET /api/v1/admin/orders/stats
		adminOrders.GET("/export", orderHandler.ExportOrders)          // GET /api/v1/admin/orders/export
	}
}
