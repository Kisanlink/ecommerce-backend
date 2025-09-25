package inventory

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// SetupInventoryRoutes sets up inventory-related routes (Admin only)
// Users should only see stock information through product endpoints, not inventory endpoints
func SetupInventoryRoutes(router *gin.RouterGroup, inventoryHandler *handlers.InventoryHandler, repoManager *repositories.RepositoryManager) {
	// Admin-only inventory routes (require Firebase authentication and admin role)
	adminInventoryGroup := router.Group("/inventory")
	adminInventoryGroup.Use(middleware.FirebaseAuthMiddleware())
	adminInventoryGroup.Use(middleware.AdminGuard(repoManager))
	{
		// Core inventory management (Admin only)
		adminInventoryGroup.POST("", inventoryHandler.CreateInventory)
		adminInventoryGroup.GET("", inventoryHandler.ListInventory)
		adminInventoryGroup.GET("/product/:product_id", inventoryHandler.GetInventoryByProductID)
		adminInventoryGroup.PUT("/product/:product_id", inventoryHandler.UpdateInventory)
		adminInventoryGroup.DELETE("/product/:product_id", inventoryHandler.DeleteInventory)
		adminInventoryGroup.PUT("/stock/:product_id", inventoryHandler.UpdateStock)

		// Stock operations (Admin only)
		adminInventoryGroup.POST("/reserve/:product_id", inventoryHandler.ReserveStock)
		adminInventoryGroup.POST("/release/:product_id", inventoryHandler.ReleaseStock)
		adminInventoryGroup.POST("/sell/:product_id", inventoryHandler.SellStock)
		adminInventoryGroup.POST("/restock/:product_id", inventoryHandler.Restock)

		// Admin monitoring and alerts
		adminInventoryGroup.GET("/low-stock", inventoryHandler.GetLowStockItems)
		adminInventoryGroup.GET("/out-of-stock", inventoryHandler.GetOutOfStockItems)
		adminInventoryGroup.GET("/needs-reorder", inventoryHandler.GetItemsNeedingReorder)
	}
}
