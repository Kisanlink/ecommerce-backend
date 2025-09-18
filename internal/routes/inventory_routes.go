package routes

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

// SetupInventoryRoutes sets up inventory-related routes
func SetupInventoryRoutes(router *gin.RouterGroup, inventoryHandler *handlers.InventoryHandler) {
	// Create inventory routes group
	inventoryGroup := router.Group("/inventory")
	{
		// Public routes (no authentication required)
		inventoryGroup.GET("/product/:product_id", inventoryHandler.GetInventoryByProductID)
		inventoryGroup.GET("", inventoryHandler.ListInventory)
		inventoryGroup.GET("/low-stock", inventoryHandler.GetLowStockItems)
		inventoryGroup.GET("/out-of-stock", inventoryHandler.GetOutOfStockItems)
		inventoryGroup.GET("/needs-reorder", inventoryHandler.GetItemsNeedingReorder)

		// Stock operations (no authentication required for basic operations)
		inventoryGroup.POST("/reserve/:product_id", inventoryHandler.ReserveStock)
		inventoryGroup.POST("/release/:product_id", inventoryHandler.ReleaseStock)
		inventoryGroup.POST("/sell/:product_id", inventoryHandler.SellStock)
		inventoryGroup.POST("/restock/:product_id", inventoryHandler.Restock)

		// Admin routes (require authentication and admin role)
		adminInventoryGroup := inventoryGroup.Group("")
		adminInventoryGroup.Use(middleware.AuthMiddleware())
		// Note: AdminGuard requires repository manager - we'll need to pass it or use a different approach
		// adminInventoryGroup.Use(middleware.AdminGuard())
		{
			// Inventory management
			adminInventoryGroup.POST("", inventoryHandler.CreateInventory)
			adminInventoryGroup.PUT("/product/:product_id", inventoryHandler.UpdateInventory)
			adminInventoryGroup.DELETE("/product/:product_id", inventoryHandler.DeleteInventory)
			adminInventoryGroup.PUT("/stock/:product_id", inventoryHandler.UpdateStock)
		}
	}
}
