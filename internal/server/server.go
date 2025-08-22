// server.go - Gin server setup for amrti-backend
// Author: Mani-KisanLink
package server

import (
	"github.com/Kisanlink/ecommerce-backend/internal/routes"
	"github.com/gin-gonic/gin"
)

// NewServer initializes and returns a Gin Engine
func NewServer() (*gin.Engine, error) {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	routes.RegisterRoutes(r)
	return r, nil
}
