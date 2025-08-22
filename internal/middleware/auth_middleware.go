package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware checks JWT and sets user info in context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing or invalid token"})
			return
		}
		tokenStr := strings.TrimPrefix(header, "Bearer ")
		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// FirebaseAuthMiddleware verifies Firebase ID tokens
func FirebaseAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" || !strings.HasPrefix(header, "Bearer ") {
			utils.RespondWithError(c, http.StatusUnauthorized, "Missing or invalid Authorization header", nil)
			return
		}

		idToken := strings.TrimPrefix(header, "Bearer ")

		client := config.GetFirebaseAuth()
		if client == nil {
			utils.RespondWithError(c, http.StatusInternalServerError, "Firebase not initialized", nil)
			return
		}

		// Verify the ID token
		token, err := client.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			utils.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token", err)
			return
		}

		// Set user info in context
		c.Set("firebase_uid", token.UID)
		c.Set("firebase_email", token.Claims["email"])
		c.Next()
	}
}

// OptionalFirebaseAuth allows requests with or without auth
func OptionalFirebaseAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header != "" && strings.HasPrefix(header, "Bearer ") {
			idToken := strings.TrimPrefix(header, "Bearer ")

			client := config.GetFirebaseAuth()
			if client != nil {
				token, err := client.VerifyIDToken(context.Background(), idToken)
				if err == nil {
					c.Set("firebase_uid", token.UID)
					c.Set("firebase_email", token.Claims["email"])
				}
			}
		}
		c.Next()
	}
}

// AdminMiddleware allows only admin role
func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		c.Next()
	}
}
