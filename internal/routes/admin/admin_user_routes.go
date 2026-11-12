package admin

import (
	"context"
	"crypto/subtle"
	"net/http"
	"os"
	"time"

	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// RegisterAdminUserRoutes registers admin-only user management routes under /admin/users
func RegisterAdminUserRoutes(r *gin.RouterGroup, repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) {
	userHandler := handlers.NewUserHandler(serviceManager.UserService)

	admin := r.Group("/admin")
	// Require Firebase auth first, then admin guard against Postgres role
	admin.Use(middleware.FirebaseAuthMiddleware())
	admin.Use(middleware.AdminGuard(repoManager))

	users := admin.Group("/users")
	{
		users.GET("", userHandler.ListUsers)
		users.GET(":id", userHandler.GetUserByID)
		users.PUT(":id", userHandler.UpdateUser)
		users.DELETE(":id", userHandler.DeleteUser)
	}

	// One-time bootstrap endpoint to create the first admin (requires ADMIN_SECRET_KEY).
	// Rate-limited per IP, and promoteUserToAdmin refuses once any admin already exists.
	r.POST("/admin/setup/promote", middleware.RateLimitByIP(5, time.Minute), func(c *gin.Context) {
		promoteUserToAdmin(c, repoManager)
	})
}

// promoteUserToAdmin promotes a user to admin role
// Requires either: admin secret key OR existing admin authentication
func promoteUserToAdmin(c *gin.Context, repoManager *repositories.RepositoryManager) {
	var req struct {
		Email     string `json:"email" binding:"required"`
		SecretKey string `json:"secret_key"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email is required"})
		return
	}

	// Check secret key from environment (for initial setup)
	adminSecretKey := os.Getenv("ADMIN_SECRET_KEY")
	if adminSecretKey == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server misconfigured: ADMIN_SECRET_KEY not set"})
		return
	}

	// Constant-time comparison so the secret cannot be recovered via response timing.
	if subtle.ConstantTimeCompare([]byte(req.SecretKey), []byte(adminSecretKey)) != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid secret key"})
		return
	}

	ctx := context.Background()

	// First-admin gate: this bootstrap endpoint is only usable until an admin exists.
	// Once any admin is present, promotion must go through the authenticated admin
	// console (/admin/users), so the shared secret cannot be used to mint more admins.
	if adminCount, err := repoManager.UserRepository.CountUsersByRole(ctx, "admin"); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin state"})
		return
	} else if adminCount > 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "An admin already exists; use the admin console to manage roles"})
		return
	}

	// Find user by email
	user, err := repoManager.UserRepository.GetUserByEmail(ctx, req.Email)
	if err != nil || user == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found. Please signup first."})
		return
	}

	// Update role to admin
	user.Role = "admin"
	if err := repoManager.UserRepository.Update(ctx, user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "User promoted to admin successfully",
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"name":  user.Name,
			"role":  user.Role,
		},
	})
}
