package admin

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterAdminCouponRoutes registers admin-only coupon management routes under /admin/coupons
func RegisterAdminCouponRoutes(r *gin.RouterGroup, couponHandler *handlers.CouponHandler, repoManager *repositories.RepositoryManager) {
	admin := r.Group("/admin")
	// Require Firebase auth first, then admin guard against Postgres role
	admin.Use(middleware.FirebaseAuthMiddleware())
	admin.Use(middleware.AdminGuard(repoManager))

	// Coupon management routes
	coupons := admin.Group("/coupons")
	{
		coupons.POST("", couponHandler.CreateCoupon)       // Create new coupon
		coupons.GET("", couponHandler.GetAllCoupons)       // Get all coupons with pagination
		coupons.PUT("/:id", couponHandler.UpdateCoupon)    // Update coupon
		coupons.DELETE("/:id", couponHandler.DeleteCoupon) // Delete coupon
	}

	// Coupon rule management routes
	couponRules := admin.Group("/coupon-rules")
	{
		couponRules.POST("/:couponId", couponHandler.CreateCouponRule)             // Create coupon rule
		couponRules.PUT("/:couponId/:ruleId", couponHandler.UpdateCouponRule)      // Update coupon rule
		couponRules.DELETE("/:couponId/:ruleId", couponHandler.DeleteCouponRule)   // Delete coupon rule
	}
}
