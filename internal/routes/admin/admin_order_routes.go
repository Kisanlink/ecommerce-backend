package admin

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterAdminOrderRoutes registers admin-only order management routes under /admin/orders
func RegisterAdminOrderRoutes(r *gin.RouterGroup, orderHandler *handlers.OrderHandler, repoManager *repositories.RepositoryManager) {
	admin := r.Group("/admin")
	// Require Firebase auth first, then admin guard against Postgres role
	admin.Use(middleware.FirebaseAuthMiddleware())
	admin.Use(middleware.AdminGuard(repoManager))

	orders := admin.Group("/orders")
	{
		orders.GET("", orderHandler.GetAllOrders)                 // Get all orders
		orders.GET("/:id", orderHandler.GetOrderDetailsAdmin)     // Get specific order details
		orders.PUT("/:id/status", orderHandler.UpdateOrderStatus) // Update order status
		orders.PUT("/:id/payment", orderHandler.UpdatePayment)    // Update payment details
		orders.PUT("/:id/shipment", orderHandler.UpdateShipment)  // Update shipment details
		orders.GET("/stats", orderHandler.GetOrderStats)          // Get order statistics
		orders.GET("/export", orderHandler.ExportOrders)          // Export orders
	}
}
