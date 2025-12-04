package payment

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterPaymentRoutes registers all payment related routes
func RegisterPaymentRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	// Initialize logger
	logger, _ := zap.NewProduction()

	// Create payment service and handler
	paymentService := services.NewPaymentService(logger)

	// Create address service for payment handler
	addressService := services.NewAddressService(repoManager, logger)

	paymentHandler := handlers.NewPaymentHandler(paymentService, serviceManager.OrderService, serviceManager.DeliveryService, serviceManager.WhatsAppService, serviceManager.UserService, serviceManager.ProductService, addressService, logger) // Payment routes (requires authentication)
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

	// Magic Checkout routes (public endpoints called by Razorpay)
	// CORS is handled by global middleware, but we ensure it's explicitly allowed
	magicCheckout := r.Group("/magic-checkout")
	magicCheckout.Use(middleware.CORSMiddlewareWithConfig(middleware.APICORSConfig())) // Ensure CORS for Magic Checkout callbacks
	{
		// Initialize services for Magic Checkout handler
		paymentServiceForMC := services.NewPaymentService(logger)
		couponRepo := repositories.NewCouponRepository(repoManager.GetDBManager(), logger)
		couponRuleRepo := repositories.NewCouponRuleRepository(repoManager.GetDBManager(), logger)
		couponService := services.NewCouponService(couponRepo, couponRuleRepo, logger)
		addressServiceForMC := services.NewAddressService(repoManager, logger)

		magicCheckoutHandler := handlers.NewMagicCheckoutHandler(
			paymentServiceForMC,
			couponService,
			serviceManager.DeliveryService,
			serviceManager.CartService,
			serviceManager.OrderService,
			addressServiceForMC,
			logger,
		)

		// These endpoints are called by Razorpay Magic Checkout
		// They must be publicly accessible (no auth middleware)
		magicCheckout.POST("/get-promotions", magicCheckoutHandler.GetPromotions)
		magicCheckout.POST("/apply-promotion", magicCheckoutHandler.ApplyPromotion)
		magicCheckout.POST("/shipping-info", magicCheckoutHandler.GetShippingInfo)
		magicCheckout.POST("/addresses-callback", magicCheckoutHandler.UpdateAddresses)  // Address updates from Magic Checkout
		magicCheckout.POST("/abandoned-cart", magicCheckoutHandler.AbandonedCartWebhook) // Abandoned cart webhook from Razorpay
	}

	// Magic Checkout order creation - available at /api/v1/magic-checkout/create-order
	// Uses optional auth so it works with or without token
	magicCheckoutAuth := r.Group("/magic-checkout")
	magicCheckoutAuth.Use(middleware.CORSMiddlewareWithConfig(middleware.APICORSConfig()))
	magicCheckoutAuth.Use(middleware.OptionalFirebaseAuth())
	magicCheckoutAuth.POST("/create-order", paymentHandler.CreateMagicCheckoutOrder)

	// Magic Checkout order creation endpoint (supports both authenticated and guest)
	// Note: This endpoint is in /payments group but doesn't require auth (optional auth)
	// CORS is handled by global middleware, but we ensure it's explicitly allowed
	magicPayments := r.Group("/payments")
	magicPayments.Use(middleware.CORSMiddlewareWithConfig(middleware.APICORSConfig())) // Ensure CORS for Magic Checkout
	magicPayments.Use(middleware.OptionalFirebaseAuth())                               // Optional auth - verifies token if provided
	magicPayments.POST("/magic-checkout/create-order", paymentHandler.CreateMagicCheckoutOrder)

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
