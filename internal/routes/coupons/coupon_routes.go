package coupons

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterCouponRoutes registers public and user coupon routes
// Admin coupon routes are registered separately via admin.RegisterAdminCouponRoutes
// Returns the coupon handler for use in admin routes
func RegisterCouponRoutes(router *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager, logger *zap.Logger) *handlers.CouponHandler {
	// Initialize coupon service and handler using the repository manager's database connection
	couponRepo := repositories.NewCouponRepository(repoManager.GetDBManager(), logger)
	couponRuleRepo := repositories.NewCouponRuleRepository(repoManager.GetDBManager(), logger)
	couponService := services.NewCouponService(couponRepo, couponRuleRepo, logger)
	couponHandler := handlers.NewCouponHandler(couponService, logger)

	// Public routes (no authentication required)
	coupons := router.Group("/coupons")
	{
		// Get coupon by code (public)
		coupons.GET("/:code", couponHandler.GetCouponByCode)

		// Get active coupons (public)
		coupons.GET("/active", couponHandler.GetActiveCoupons)

		// Validate coupon (public)
		coupons.POST("/validate", couponHandler.ValidateCoupon)
	}

	// Protected routes (authentication required)
	protected := router.Group("/coupons")
	protected.Use(middleware.FirebaseAuthMiddleware())
	{
		// Apply coupon (authenticated users)
		protected.POST("/apply", couponHandler.ApplyCouponToUser)

		// Get coupon usage stats (authenticated users)
		protected.GET("/:code/stats", couponHandler.GetCouponUsageStats)
	}

	return couponHandler
}
