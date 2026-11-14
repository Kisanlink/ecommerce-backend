package admin

import (
	"crypto/subtle"
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/handlers"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// adminBootstrapLockKey is a fixed Postgres advisory-lock key used to serialize first-admin
// bootstrap attempts across backend instances (pg_advisory_xact_lock). The value is arbitrary but
// must stay stable so every instance contends on the same lock.
const adminBootstrapLockKey int64 = 7295018

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

	ctx := c.Request.Context()

	// The first-admin gate and the promotion must be ONE atomic unit, otherwise two concurrent
	// requests with the correct secret can both read adminCount==0 and promote different users
	// (a process-local mutex would not protect multiple backend instances). Run both inside a
	// single transaction guarded by a Postgres transaction-scoped advisory lock so bootstrap
	// attempts serialize across every instance. The shared pgManager is reused via the order
	// repository to avoid separate user-repo transaction plumbing.
	pgManager := repoManager.OrderRepository.GetPostgresManager()
	if pgManager == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify admin state"})
		return
	}

	emailNorm := strings.ToLower(strings.TrimSpace(req.Email))
	errAdminExists := errors.New("admin already exists")
	errUserNotFound := errors.New("user not found")

	txErr := pgManager.WithTransaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", adminBootstrapLockKey).Error; err != nil {
			return err
		}
		// Re-check inside the lock. Role is matched case/space-insensitively (as AdminGuard does) so a
		// legacy admin stored as "Admin" is still counted, and deleted_at is filtered explicitly
		// because the base model's DeletedAt is a plain *time.Time (no gorm soft-delete auto-filter).
		var adminCount int64
		if err := tx.Model(&models.User{}).
			Where("deleted_at IS NULL AND LOWER(TRIM(role)) = ?", "admin").
			Count(&adminCount).Error; err != nil {
			return err
		}
		if adminCount > 0 {
			return errAdminExists
		}
		res := tx.Model(&models.User{}).
			Where("deleted_at IS NULL AND LOWER(TRIM(email)) = ?", emailNorm).
			Update("role", "admin")
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return errUserNotFound
		}
		return nil
	})

	switch {
	case errors.Is(txErr, errAdminExists):
		c.JSON(http.StatusForbidden, gin.H{"error": "An admin already exists; use the admin console to manage roles"})
		return
	case errors.Is(txErr, errUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found. Please signup first."})
		return
	case txErr != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to promote user to admin"})
		return
	}

	// Reload for the response (best-effort — the promotion already committed above).
	if user, err := repoManager.UserRepository.GetUserByEmail(ctx, req.Email); err == nil && user != nil {
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
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "User promoted to admin successfully"})
}
