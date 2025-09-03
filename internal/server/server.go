// server.go - Gin server setup for amrti-backend
// Author: Mani-KisanLink
package server

import (
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/routes"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// NewServer initializes and returns a Gin Engine with dependency injection
func NewServer(repoManager *repositories.RepositoryManager, serviceManager *services.ServiceManager) (*gin.Engine, error) {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})

	// Create logger for routes (we can reuse the main logger)
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	routes.RegisterRoutes(r, repoManager, serviceManager, logger)
	return r, nil
}
