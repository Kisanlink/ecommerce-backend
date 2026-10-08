package admin

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// RegisterAdminInventoryRoutes registers admin-only inventory management routes under /admin/inventory
func RegisterAdminInventoryRoutes(r *gin.RouterGroup, inventoryHandler *handlers.InventoryHandler, repoManager *repositories.RepositoryManager) {
	admin := r.Group("/admin")
	// Require Firebase auth first, then admin guard against Postgres role
	admin.Use(middleware.FirebaseAuthMiddleware())
	admin.Use(middleware.AdminGuard(repoManager))

	inventory := admin.Group("/inventory")
	{
		// Core inventory management
		inventory.POST("", inventoryHandler.CreateInventory)
		inventory.GET("", inventoryHandler.ListInventory)
		inventory.GET("/product/:product_id", inventoryHandler.GetInventoryByProductID)
		inventory.PUT("/product/:product_id", inventoryHandler.UpdateInventory)
		inventory.DELETE("/product/:product_id", inventoryHandler.DeleteInventory)
		inventory.PUT("/stock/:product_id", inventoryHandler.UpdateStock)

		// Stock operations
		inventory.POST("/reserve/:product_id", inventoryHandler.ReserveStock)
		inventory.POST("/release/:product_id", inventoryHandler.ReleaseStock)
		inventory.POST("/sell/:product_id", inventoryHandler.SellStock)
		inventory.POST("/restock/:product_id", inventoryHandler.Restock)

		// Monitoring and alerts
		inventory.GET("/low-stock", inventoryHandler.GetLowStockItems)
		inventory.GET("/out-of-stock", inventoryHandler.GetOutOfStockItems)
		inventory.GET("/needs-reorder", inventoryHandler.GetItemsNeedingReorder)
	}
}
