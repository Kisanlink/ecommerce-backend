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

		uid, ok := uidVal.(string)
		if !ok || uid == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid auth context"})
			return
		}
		// SECURITY (#5 account takeover): resolve admin strictly by the verified, linked
		// firebase_uid. The previous email fallback granted admin to any token whose email matched
		// a local admin record even without a linked UID, which (combined with unverified-email
		// account linking) was an admin-takeover path. There is deliberately no email fallback:
		// an admin must be linked by firebase_uid.
		ctx := context.Background()
		userRepo := repoManager.UserRepository
		user, err := userRepo.GetByFirebaseUID(ctx, uid)
		if err != nil || user == nil || !strings.EqualFold(strings.TrimSpace(user.Role), "admin") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}

		// Set user_id in context for handlers to use
		c.Set("user_id", user.ID)
		c.Set("role", user.Role)
		c.Next()
	}
}
