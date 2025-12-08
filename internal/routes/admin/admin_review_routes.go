package admin

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterAdminReviewRoutes registers admin-only review management routes under /admin/reviews
func RegisterAdminReviewRoutes(r *gin.RouterGroup, reviewHandler *handlers.ReviewHandler, repoManager *repositories.RepositoryManager) {
	admin := r.Group("/admin")
	// Require Firebase auth first, then admin guard against Postgres role
	admin.Use(middleware.FirebaseAuthMiddleware())
	admin.Use(middleware.AdminGuard(repoManager))

	reviews := admin.Group("/reviews")
	{
		reviews.GET("", reviewHandler.AdminListReviews)                         // Get all reviews with admin filtering
		reviews.GET("/flagged", reviewHandler.AdminGetFlaggedReviews)           // Get flagged reviews
		reviews.PUT("/:reviewId", reviewHandler.AdminManageReview)              // Manage review (flag, unflag, approve, reject, delete)
		reviews.DELETE("/flagged", reviewHandler.AdminBulkDeleteFlaggedReviews) // Bulk delete all flagged reviews
	}
}
