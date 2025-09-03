package payment

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterPaymentRoutes registers all payment related routes
func RegisterPaymentRoutes(r *gin.RouterGroup, serviceManager *services.ServiceManager) {
	// Initialize logger
	logger, _ := zap.NewProduction()

	// Create payment service and handler
	paymentService := services.NewPaymentService(logger)
	paymentHandler := handlers.NewPaymentHandler(paymentService, serviceManager.OrderService, serviceManager.DeliveryService, logger) // Payment routes (requires authentication)
	payments := r.Group("/payments")
	payments.Use(middleware.FirebaseAuthMiddleware())
	{
		// Core payment functionality
		payments.POST("/create-order", paymentHandler.CreateOrder)           // Create Razorpay order
		payments.POST("/verify", paymentHandler.VerifyPayment)               // Verify payment signature
		payments.GET("/:payment_id/status", paymentHandler.GetPaymentStatus) // Get payment status
		// Simple hosted checkout helper (requires auth to prevent abuse)
		payments.GET("/checkout", paymentHandler.CheckoutPage) // ?order_id=order_xxx
	}

	// Webhooks (public endpoints for payment providers)
	webhooks := r.Group("/webhooks")
	{
		webhooks.POST("/razorpay", paymentHandler.ProcessWebhook) // Razorpay webhook
	}

	// Legacy routes for compatibility (deprecated)
	payments.POST("/process", func(c *gin.Context) {
		c.JSON(501, gin.H{
			"error": "This endpoint is deprecated. Use POST /payments/create-order instead",
		})
	})
	payments.POST("/intent", func(c *gin.Context) {
		c.JSON(501, gin.H{
			"error": "This endpoint is deprecated. Use POST /payments/create-order instead",
		})
	})
	payments.POST("/confirm", func(c *gin.Context) {
		c.JSON(501, gin.H{
			"error": "This endpoint is deprecated. Use POST /payments/verify instead",
		})
	})
	payments.POST("/cancel", func(c *gin.Context) {
		c.JSON(501, gin.H{
			"error": "Payment cancellation not implemented yet",
		})
	})
	payments.GET("/history", func(c *gin.Context) {
		c.JSON(501, gin.H{
			"error": "Payment history not implemented yet",
		})
	})

	// Future route groups (commented until fully implemented)
	/*
		// Payment methods management
		methods := payments.Group("/methods")
		{
			methods.GET("", paymentHandler.GetPaymentMethods)                     // Get user's saved payment methods
			methods.POST("", paymentHandler.AddPaymentMethod)                     // Add new payment method
			methods.PUT("/:methodId", paymentHandler.UpdatePaymentMethod)         // Update payment method
			methods.DELETE("/:methodId", paymentHandler.DeletePaymentMethod)      // Delete payment method
			methods.PUT("/:methodId/default", paymentHandler.SetDefaultPaymentMethod) // Set default payment method
		}

		// Payment refunds (admin functionality)
		refunds := payments.Group("/refunds")
		{
			refunds.POST("/:paymentId", paymentHandler.ProcessRefund)             // Process refund
			refunds.GET("/:refundId/status", paymentHandler.GetRefundStatus)      // Get refund status
			refunds.GET("/history", paymentHandler.GetRefundHistory)              // Get refund history
		}

		// Payment analytics (admin only)
		analytics := r.Group("/admin/payments")
		analytics.Use(middleware.FirebaseAuthMiddleware()) // Add admin middleware later
		{
			analytics.GET("/stats", paymentHandler.GetPaymentStats)              // Get payment statistics
			analytics.GET("/transactions", paymentHandler.GetAllTransactions)    // Get all transactions
			analytics.GET("/export", paymentHandler.ExportTransactions)          // Export transactions
		}
	*/
}
