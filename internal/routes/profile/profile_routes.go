package profiles

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterProfileRoutes registers all profile related routes
func RegisterProfileRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	// Create profile handler
	profileHandler := handlers.NewProfileHandler(serviceManager)

	// Profile routes - all require authentication
	profiles := r.Group("/profiles")
	profiles.Use(middleware.FirebaseAuthMiddleware())
	{
		// Core functionality routes
		// Note: /auto endpoint removed - profiles are now automatically created during signup
		// profiles.POST("/auto", profileHandler.CreateAutoProfile)          // DEPRECATED: Profiles auto-created during signup
		profiles.GET("", profileHandler.GetProfile)                       // Get user's profile
		profiles.PUT("", profileHandler.UpdateProfile)                    // Update profile
		profiles.POST("/upload-image", profileHandler.UploadProfileImage) // Upload profile image to S3
		profiles.GET("/image-url", profileHandler.GetProfileImageURL)     // Get profile image presigned URL
	}
}
