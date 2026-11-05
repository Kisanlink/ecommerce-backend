package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// FirebaseAuthMiddleware verifies Firebase ID tokens
func FirebaseAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip OPTIONS preflight requests - CORS middleware handles these
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

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

		// SECURITY (#19): a token is trusted ONLY when its signature verifies. The
		// previous code, on an "issued at"/"future timestamp" (clock-skew) error,
		// base64-decoded the JWT payload and trusted user_id/sub WITHOUT verifying
		// the signature — a forged-token auth bypass (any uid + a future iat logs in
		// as anyone). There is no safe way to accept an unverified token, so reject
		// on ANY verification error. If genuine clock skew is ever seen, the fix is
		// to sync the server clock (NTP) — never to trust an unverified payload.
		token, err := client.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			utils.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token — if this persists, ensure the server clock is synced", err)
			return
		}

		// Set user info in context
		c.Set("firebase_uid", token.UID)
		if email, ok := token.Claims["email"].(string); ok && email != "" {
			c.Set("firebase_email", email)
		}
		// Also capture phone number if available (for phone-auth users)
		if phone, ok := token.Claims["phone_number"].(string); ok && phone != "" {
			c.Set("firebase_phone", phone)
		}
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
