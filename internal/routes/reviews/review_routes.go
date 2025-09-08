package reviews

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterReviewRoutes registers all review related routes
func RegisterReviewRoutes(r *gin.RouterGroup, serviceManager *services.ServiceManager, logger *zap.Logger) {
	// Initialize logger if not provided
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	// Validate required services
	if serviceManager == nil || serviceManager.ReviewService == nil {
		logger.Error("ReviewService is required but not available in ServiceManager")
		return
	}

	// Create review handler
	reviewHandler := handlers.NewReviewHandler(serviceManager.ReviewService, logger)

	// Product review routes
	products := r.Group("/products")
	{
		// Review routes for specific products
		productReviews := products.Group("/:id/reviews")
		{
			// Public routes (no authentication required)
			productReviews.GET("", reviewHandler.GetProductReviews)    // Get all reviews for a product
			productReviews.GET("/stats", reviewHandler.GetReviewStats) // Get review statistics for a product

			// Authenticated routes (require login)
			authenticated := productReviews.Group("/")
			authenticated.Use(middleware.FirebaseAuthMiddleware())
			{
				authenticated.POST("", reviewHandler.CreateReview)              // Create a new review
				authenticated.PUT("/:reviewId", reviewHandler.UpdateReview)     // Update review (owner only)
				authenticated.DELETE("/:reviewId", reviewHandler.DeleteReview)  // Delete review (owner/admin only)
				authenticated.POST("/:reviewId/like", reviewHandler.LikeReview) // Like/unlike a review
			}
		}
	}

	// User review routes
	users := r.Group("/users")
	{
		// Public user review routes (can view any user's reviews)
		users.GET("/:id/reviews", reviewHandler.GetUserReviews) // Get all reviews by a user
	}

	// Admin review routes (future implementation)
	// admin := r.Group("/admin/reviews")
	// admin.Use(middleware.FirebaseAuthMiddleware())
	// admin.Use(middleware.AdminGuard(repoManager)) // TODO: Add admin guard
	// {
	// 	admin.GET("", reviewHandler.GetAllReviews)                    // Get all reviews (admin)
	// 	admin.GET("/flagged", reviewHandler.GetFlaggedReviews)        // Get flagged reviews
	// 	admin.PUT("/:reviewId/flag", reviewHandler.FlagReview)        // Flag/unflag a review
	// 	admin.DELETE("/:reviewId/force", reviewHandler.ForceDeleteReview) // Force delete review
	// }

	logger.Info("Review routes registered successfully")
}
