package routes

import (
	"os"
	"strings"

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
	// Health check endpoint (CORS is handled by global middleware)
	// HEAD request - returns status only (no body) - preferred method
	r.HEAD("/health", func(c *gin.Context) {
		logger.Info("Health check: Backend is healthy and live", zap.String("method", "HEAD"))
		c.Status(200)
	})
	// GET request - returns JSON response - kept for backward compatibility
	r.GET("/health", func(c *gin.Context) {
		logger.Info("Health check: Backend is healthy and live", zap.String("method", "GET"))
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "ecommerce-backend",
			"version": "1.0.0",
		})
	})

	// Debug endpoints — DISABLED by default. These expose Firebase/Delhivery config
	// status and can generate any order's invoice, so they are registered only when
	// ENABLE_DEBUG_ROUTES=true and, even then, require an authenticated admin
	// (Firebase auth + AdminGuard). In production leave ENABLE_DEBUG_ROUTES unset.
	if strings.EqualFold(strings.TrimSpace(os.Getenv("ENABLE_DEBUG_ROUTES")), "true") {
		debugHandler := handlers.NewDebugHandler(logger)
		debugHandlerWithServices := handlers.NewDebugHandlerWithServices(logger, serviceManager.OrderService)
		debugGroup := r.Group("/debug")
		debugGroup.Use(middleware.FirebaseAuthMiddleware())
		debugGroup.Use(middleware.AdminGuard(repoManager))
		debugGroup.GET("/firebase", debugHandler.CheckFirebaseStatus)
		debugGroup.GET("/delhivery-config", debugHandler.CheckDelhiveryConfig)
		debugGroup.POST("/invoice/:id", debugHandlerWithServices.GenerateInvoiceDebug)
		logger.Warn("Debug routes are ENABLED via ENABLE_DEBUG_ROUTES=true (admin-authenticated); do not enable in production")
	}

	// API versioning
	v1 := r.Group("/api/v1")
	{
		// ============================================
		// PUBLIC ROUTES (no authentication required)
		// ============================================
		auth.RegisterAuthRoutes(v1, repoManager, serviceManager)
		users.RegisterUserRoutes(v1, repoManager, serviceManager)
		products.RegisterProductRoutes(v1, serviceManager)
		recipes.RegisterRecipeRoutes(v1, repoManager, serviceManager)
		reviews.RegisterReviewRoutes(v1, serviceManager, logger)

		// ============================================
		// AUTHENTICATED USER ROUTES
		// ============================================
		profiles.RegisterProfileRoutes(v1, repoManager, serviceManager)
		favorites.RegisterFavoriteRoutes(v1, repoManager, serviceManager)
		orders.RegisterOrderRoutes(v1, repoManager, serviceManager, logger)
		cart.RegisterCartRoutes(v1, repoManager, serviceManager)
		payment.RegisterPaymentRoutes(v1, repoManager, serviceManager)
		delivery.RegisterDeliveryRoutes(v1, repoManager, serviceManager, logger)
		couponHandler := coupons.RegisterCouponRoutes(v1, repoManager, serviceManager, logger)

		// Register image routes
		imageHandler := handlers.NewImageHandler(serviceManager.ImageService, nil, logger)
		image.SetupImageRoutes(v1, imageHandler, middleware.FirebaseAuthMiddleware())

		// ============================================
		// ADMIN ROUTES (under /api/v1/admin/*)
		// ============================================
		admin.RegisterAdminUserRoutes(v1, repoManager, serviceManager)
		admin.RegisterAdminRecipeRoutes(v1, repoManager)
		admin.RegisterAdminProductRoutes(v1, repoManager, serviceManager)

		// Create handlers for admin routes
		inventoryHandler := handlers.NewInventoryHandler(serviceManager.InventoryService)
		admin.RegisterAdminInventoryRoutes(v1, inventoryHandler, repoManager)

		orderHandler := handlers.NewOrderHandler(serviceManager.OrderService, services.NewPaymentService(logger), serviceManager.UserService, logger)
		admin.RegisterAdminOrderRoutes(v1, orderHandler, repoManager)

		reviewHandler := handlers.NewReviewHandler(serviceManager.ReviewService, logger)
		admin.RegisterAdminReviewRoutes(v1, reviewHandler, repoManager)

		admin.RegisterAdminCouponRoutes(v1, couponHandler, repoManager)
	}

	// Legacy routes (without versioning) for backward compatibility
	legacy := r.Group("/")
	{
		auth.RegisterAuthRoutes(legacy, repoManager, serviceManager)
		recipes.RegisterRecipeRoutes(legacy, repoManager, serviceManager)
	}
}
