// server.go - Gin server setup for amrti-backend
// Author: Mani-KisanLink
package server

import (
	"github.com/Kisanlink/ecommerce-backend/internal/middleware"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/routes"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/ecommerce-backend/internal/swagger"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewServer initializes and returns a Gin Engine with dependency injection
func NewServer(repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) (*gin.Engine, error) {
	r := gin.Default()

	// Add global CORS middleware for all routes
	r.Use(middleware.CORSMiddlewareWithConfig(middleware.APICORSConfig()))

	r.GET("/ping", func(c *gin.Context) {
		c.String(200, "Backend alive")
	})

	// Create logger for routes (we can reuse the main logger)
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	routes.RegisterRoutes(r, repoManager, serviceManager, logger)

	// Register Swagger documentation routes
	swagger.RegisterSwaggerRoutes(r)

	return r, nil
}
