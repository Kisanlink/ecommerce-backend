package image

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/gin-gonic/gin"
)

// SetupImageRoutes configures all image-related routes
func SetupImageRoutes(router *gin.RouterGroup, imageHandler *handlers.ImageHandler, authMiddleware gin.HandlerFunc) {
	// Image routes with authentication
	imageRoutes := router.Group("/images")
	imageRoutes.Use(authMiddleware)
	{
		// CRUD operations for individual images
		imageRoutes.POST("", imageHandler.CreateImage)       // POST /api/v1/images
		imageRoutes.GET("/:id", imageHandler.GetImage)       // GET /api/v1/images/:id
		imageRoutes.PUT("/:id", imageHandler.UpdateImage)    // PUT /api/v1/images/:id
		imageRoutes.DELETE("/:id", imageHandler.DeleteImage) // DELETE /api/v1/images/:id
	}

	// Product-specific image routes (using different path to avoid conflict)
	productImageRoutes := router.Group("/product-images")
	productImageRoutes.Use(authMiddleware)
	{
		// Get all images for a product
		productImageRoutes.GET("/:productId", imageHandler.GetImagesByProductID) // GET /api/v1/product-images/:productId

		// Get primary image for a product
		productImageRoutes.GET("/:productId/primary", imageHandler.GetPrimaryImageByProductID) // GET /api/v1/product-images/:productId/primary

		// Set primary image
		productImageRoutes.PUT("/:productId/:imageId/primary", imageHandler.SetPrimaryImage) // PUT /api/v1/product-images/:productId/:imageId/primary

		// Bulk operations
		productImageRoutes.POST("/:productId/bulk", imageHandler.AddMultipleImages) // POST /api/v1/product-images/:productId/bulk
		productImageRoutes.PUT("/:productId/reorder", imageHandler.ReorderImages)   // PUT /api/v1/product-images/:productId/reorder
	}
}
