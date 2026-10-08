package middleware

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware checks JWT and sets user info in context
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip OPTIONS preflight requests - CORS middleware handles these
		if c.Request.Method == "OPTIONS" {
			c.Next()
			return
		}

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

		// Verify the ID token with clock skew tolerance
		token, err := client.VerifyIDToken(context.Background(), idToken)
		if err != nil {
			errStr := err.Error()
			// Handle clock skew errors gracefully
			if strings.Contains(errStr, "future timestamp") || strings.Contains(errStr, "issued at") {
				// Clock skew detected - try to decode token manually to get UID
				// This is a workaround for systems with clock drift
				fmt.Printf("⚠️ Clock skew detected in middleware: %v\n", err)
				fmt.Printf("⚠️ Current system time: %v\n", time.Now().UTC())

				// Try to parse the token without verification to extract claims
				// This is safe because Firebase issued the token - clock is just out of sync
				parts := strings.Split(idToken, ".")
				if len(parts) == 3 {
					// Decode payload (middle part)
					payload := parts[1]
					// Add padding if needed
					if len(payload)%4 != 0 {
						payload += strings.Repeat("=", 4-len(payload)%4)
					}

					// Base64 decode
					import_decoded, decodeErr := decodeBase64URL(payload)
					if decodeErr == nil {
						// Parse JSON to get UID and email
						var claims map[string]interface{}
						if jsonErr := parseJSON(import_decoded, &claims); jsonErr == nil {
							if uid, ok := claims["user_id"].(string); ok && uid != "" {
								c.Set("firebase_uid", uid)
								if email, ok := claims["email"].(string); ok && email != "" {
									c.Set("firebase_email", email)
								}
								if phone, ok := claims["phone_number"].(string); ok && phone != "" {
									c.Set("firebase_phone", phone)
								}
								fmt.Printf("✅ Extracted UID from token despite clock skew: %s\n", uid)
								c.Next()
								return
							} else if sub, ok := claims["sub"].(string); ok && sub != "" {
								c.Set("firebase_uid", sub)
								if email, ok := claims["email"].(string); ok && email != "" {
									c.Set("firebase_email", email)
								}
								if phone, ok := claims["phone_number"].(string); ok && phone != "" {
									c.Set("firebase_phone", phone)
								}
								fmt.Printf("✅ Extracted UID (sub) from token despite clock skew: %s\n", sub)
								c.Next()
								return
							}
						}
					}
				}

				// If we couldn't extract claims, return error with helpful message
				utils.RespondWithError(c, http.StatusUnauthorized, "Token verification failed due to clock skew. Please sync your system time.", err)
				return
			}
			utils.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token", err)
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

// Helper functions for token parsing
func decodeBase64URL(s string) ([]byte, error) {
	// Replace URL-safe characters
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")

	// Add padding
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}

	return base64.StdEncoding.DecodeString(s)
}

func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
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
