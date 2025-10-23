package admin

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterAdminRecipeRoutes registers admin-only recipe management routes under /admin/recipes
func RegisterAdminRecipeRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager) {
	adminRecipeHandler := handlers.NewAdminRecipeHandler(repoManager)

	admin := r.Group("/admin")
	// Require Firebase auth first, then admin guard against Postgres role
	admin.Use(middleware.FirebaseAuthMiddleware())
	admin.Use(middleware.AdminGuard(repoManager))

	recipes := admin.Group("/recipes")
	{
		// Pending recipe review management
		pending := recipes.Group("/pending")
		{
			pending.GET("", adminRecipeHandler.GetPendingRecipeReviews)          // Get all pending reviews
			pending.GET("/:id", adminRecipeHandler.GetPendingRecipeReviewByID)   // Get specific pending review
			pending.POST("/:id/approve", adminRecipeHandler.ApproveRecipeReview) // Approve a recipe
			pending.POST("/:id/reject", adminRecipeHandler.RejectRecipeReview)   // Reject a recipe
			pending.DELETE("/:id", adminRecipeHandler.DeletePendingRecipeReview) // Delete a pending review
		}
	}
}
