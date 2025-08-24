package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
)

// AdminGuard returns a middleware that checks if the authenticated Firebase user has role=admin in Postgres.
// It requires FirebaseAuthMiddleware to have run before (to set firebase_uid).
func AdminGuard(repoManager *repositories.RepositoryManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		uidVal, ok := c.Get("firebase_uid")
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Missing auth"})
			return
		}

		uid := uidVal.(string)
		// Lookup user by firebase_uid and verify role; fallback to email if not linked
		ctx := context.Background()
		userRepo := repoManager.UserRepository
		user, err := userRepo.GetByFirebaseUID(ctx, uid)
		if err != nil || user == nil {
			// Try by email from token claims (set by FirebaseAuthMiddleware)
			if emailVal, ok := c.Get("firebase_email"); ok {
				if emailStr, ok2 := emailVal.(string); ok2 && emailStr != "" {
					user, err = userRepo.GetUserByEmail(ctx, emailStr)
				}
			}
		}
		if err != nil || user == nil || !strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		c.Next()
	}
}
