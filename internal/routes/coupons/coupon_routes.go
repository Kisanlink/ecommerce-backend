package coupons

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterCouponRoutes registers all coupon-related routes
func RegisterCouponRoutes(router *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager, logger *zap.Logger) {
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

	// Admin routes (admin access required)
	admin := router.Group("/coupons")
	admin.Use(middleware.FirebaseAuthMiddleware())
	admin.Use(middleware.AdminGuard(repoManager))
	{
		// Create coupon (admin only)
		admin.POST("", couponHandler.CreateCoupon)

		// Get all coupons (admin only)
		admin.GET("", couponHandler.GetAllCoupons)

		// Update coupon (admin only)
		admin.PUT("/:id", couponHandler.UpdateCoupon)

		// Delete coupon (admin only)
		admin.DELETE("/:id", couponHandler.DeleteCoupon)
	}

	// Admin coupon rule routes (separate group to avoid conflicts)
	adminRules := router.Group("/coupon-rules")
	adminRules.Use(middleware.FirebaseAuthMiddleware())
	adminRules.Use(middleware.AdminGuard(repoManager))
	{
		// Create coupon rule (admin only)
		adminRules.POST("/:couponId", couponHandler.CreateCouponRule)

		// Update coupon rule (admin only)
		adminRules.PUT("/:couponId/:ruleId", couponHandler.UpdateCouponRule)

		// Delete coupon rule (admin only)
		adminRules.DELETE("/:couponId/:ruleId", couponHandler.DeleteCouponRule)
	}
}
