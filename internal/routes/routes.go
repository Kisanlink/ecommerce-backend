package routes

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/admin"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/cart"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/delivery"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/favorites"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/orders"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/payment"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/products"
	profiles "github.com/Kisanlink/ecommerce-backend/internal/routes/profile"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/recipes"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/reviews"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/users"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RegisterRoutes registers all application routes by delegating to specific route modules
func RegisterRoutes(r *gin.Engine, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager, logger *zap.Logger) {
	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "ecommerce-backend",
			"version": "1.0.0",
		})
	})

	// Debug endpoint to check Firebase status
	debugHandler := handlers.NewDebugHandler()
	r.GET("/debug/firebase", debugHandler.CheckFirebaseStatus)

	// API versioning
	v1 := r.Group("/api/v1")
	{
		// Register all route modules with both repositories and services
		auth.RegisterAuthRoutes(v1, repoManager, serviceManager)
		users.RegisterUserRoutes(v1, repoManager, serviceManager)
		admin.RegisterAdminUserRoutes(v1, repoManager, serviceManager)
		products.RegisterProductRoutes(v1, serviceManager, repoManager)
		profiles.RegisterProfileRoutes(v1, repoManager, serviceManager)
		favorites.RegisterFavoriteRoutes(v1, repoManager, serviceManager)
		orders.RegisterOrderRoutes(v1, repoManager, serviceManager, logger) // Updated with logger
		cart.RegisterCartRoutes(v1, repoManager, serviceManager)
		recipes.RegisterRecipeRoutes(v1, repoManager)
		payment.RegisterPaymentRoutes(v1, serviceManager)
		delivery.RegisterDeliveryRoutes(v1, repoManager, serviceManager, logger)
		reviews.RegisterReviewRoutes(v1, serviceManager, logger) // Add review routes
	}

	// Legacy routes (without versioning) for backward compatibility
	legacy := r.Group("/")
	{
		auth.RegisterAuthRoutes(legacy, repoManager, serviceManager)
		recipes.RegisterRecipeRoutes(legacy, repoManager)
		// Add other legacy routes as needed
	}
}
