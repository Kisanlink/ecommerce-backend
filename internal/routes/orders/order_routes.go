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

	// Future route groups (commented until handlers are implemented)
	/*
		// Admin order routes (add admin middleware later)
		adminOrders := r.Group("/admin/orders")
		adminOrders.Use(middleware.FirebaseAuthMiddleware()) // Add admin middleware later
		{
			adminOrders.GET("", orderHandler.GetAllOrders)                 // Get all orders
			adminOrders.PUT("/:id/status", orderHandler.UpdateOrderStatus) // Update order status
			adminOrders.GET("/stats", orderHandler.GetOrderStats)          // Get order statistics
			adminOrders.GET("/export", orderHandler.ExportOrders)          // Export orders
		}

		// Order items
		items := orders.Group("/:orderId/items")
		{
			items.GET("", orderHandler.GetOrderItems)                      // Get order items
			items.PUT("/:itemId", orderHandler.UpdateOrderItem)           // Update order item (before fulfillment)
		}

		// Order history and receipts
		history := orders.Group("/:orderId")
		{
			history.GET("/receipt", orderHandler.GetOrderReceipt)          // Get order receipt
			history.GET("/invoice", orderHandler.GetOrderInvoice)          // Get order invoice
			history.POST("/reorder", orderHandler.ReorderItems)            // Reorder same items
		}
	*/
}
