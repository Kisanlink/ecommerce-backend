package orders

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterOrderRoutes registers user order routes (authenticated users)
// Admin order routes are registered separately via admin.RegisterAdminOrderRoutes
func RegisterOrderRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager, logger *zap.Logger) {
	orderHandler := handlers.NewOrderHandler(serviceManager.OrderService, serviceManager.UserService, logger)

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
	// SECURITY (#6): the unauthenticated /debug/order/:id endpoint, which returned order PII
	// (user_id, shipping address, waybill, total) for any guessable order ID, has been removed.
}
