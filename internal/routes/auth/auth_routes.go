package auth

import (
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterAuthRoutes registers all authentication related routes
func RegisterAuthRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	// Firebase Auth Handler with UserService dependency injection
	firebaseAuth := handlers.NewFirebaseAuthHandler(serviceManager.UserService)

	// Firebase Authentication routes
	auth := r.Group("/auth")
	{
		// Public auth endpoints
		auth.POST("/signup", firebaseAuth.Signup)
		auth.POST("/signin", firebaseAuth.Login) // Fixed: now uses Login method
		auth.POST("/login", firebaseAuth.Login)  // Alternative login endpoint
		auth.POST("/verify-token", firebaseAuth.VerifyToken)
		auth.POST("/refresh-token", firebaseAuth.RefreshToken) // Now uses proper RefreshToken method

		// Google OAuth endpoints
		auth.GET("/google", firebaseAuth.GetGoogleAuthURL) // Get Google OAuth info
		auth.POST("/google", firebaseAuth.GoogleSignIn)    // Google Sign-In with ID token

		// Protected auth endpoints
		protected := auth.Group("/")
		protected.Use(middleware.FirebaseAuthMiddleware())
		{
			protected.GET("/profile", firebaseAuth.GetProfile)
			protected.PUT("/profile", firebaseAuth.UpdateProfile)
			protected.DELETE("/account", firebaseAuth.DeleteAccount)
			protected.POST("/logout", firebaseAuth.Logout) // Proper logout: revoke refresh tokens, do not delete account
		}
	}

	// OAuth routes (Google Sign-In is now active)
	// Additional OAuth providers can be added here:
	// oauth := r.Group("/oauth")
	// {
	// 	oauth.GET("/facebook", firebaseAuth.FacebookAuth)
	// 	oauth.POST("/facebook", firebaseAuth.FacebookSignIn)
	// 	oauth.GET("/callback/:provider", firebaseAuth.OAuthCallback)
	// }
}
