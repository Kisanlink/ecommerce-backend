package admin

import (
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
}
