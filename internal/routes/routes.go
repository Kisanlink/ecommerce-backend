package routes

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/admin"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/cart"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/coupons"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/delivery"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/favorites"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/image"
	"github.com/Kisanlink/ecommerce-backend/internal/routes/inventory"
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

	// Debug endpoints
	debugHandler := handlers.NewDebugHandler(logger)
	debugHandlerWithServices := handlers.NewDebugHandlerWithServices(logger, serviceManager.OrderService)
	r.GET("/debug/firebase", debugHandler.CheckFirebaseStatus)
	r.GET("/debug/delhivery-config", debugHandler.CheckDelhiveryConfig)
	r.POST("/debug/invoice/:id", debugHandlerWithServices.GenerateInvoiceDebug)

	// API versioning
	v1 := r.Group("/api/v1")
	{
		// Register all route modules with both repositories and services
		auth.RegisterAuthRoutes(v1, repoManager, serviceManager)
		users.RegisterUserRoutes(v1, repoManager, serviceManager)
		admin.RegisterAdminUserRoutes(v1, repoManager, serviceManager)
		admin.RegisterAdminRecipeRoutes(v1, repoManager) // Add admin recipe routes
		products.RegisterProductRoutes(v1, serviceManager, repoManager)
		profiles.RegisterProfileRoutes(v1, repoManager, serviceManager)
		favorites.RegisterFavoriteRoutes(v1, repoManager, serviceManager)
		orders.RegisterOrderRoutes(v1, repoManager, serviceManager, logger) // Updated with logger
		cart.RegisterCartRoutes(v1, repoManager, serviceManager)
		recipes.RegisterRecipeRoutes(v1, repoManager, serviceManager)
		payment.RegisterPaymentRoutes(v1, repoManager, serviceManager)
		delivery.RegisterDeliveryRoutes(v1, repoManager, serviceManager, logger)
		reviews.RegisterReviewRoutes(v1, serviceManager, repoManager, logger) // Add review routes
		coupons.RegisterCouponRoutes(v1, repoManager, serviceManager, logger) // Add coupon routes

		// Register image routes
		imageHandler := handlers.NewImageHandler(serviceManager.ImageService, nil, logger) // AuthContext will be set in the handler
		image.SetupImageRoutes(v1, imageHandler, middleware.FirebaseAuthMiddleware())

		// Register inventory routes
		inventoryHandler := handlers.NewInventoryHandler(serviceManager.InventoryService)
		inventory.SetupInventoryRoutes(v1, inventoryHandler, repoManager)
	}

	// Legacy routes (without versioning) for backward compatibility
	legacy := r.Group("/")
	{
		auth.RegisterAuthRoutes(legacy, repoManager, serviceManager)
		recipes.RegisterRecipeRoutes(legacy, repoManager, serviceManager)
		// Add other legacy routes as needed
	}
}
