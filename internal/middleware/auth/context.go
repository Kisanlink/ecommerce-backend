package auth

import (
	"github.com/gin-gonic/gin"
)

// AuthContext provides helper methods for accessing authenticated user data
type AuthContext struct{}

// GetAuthenticatedUserID returns the authenticated user ID from context
// Works with both Firebase and JWT authentication
func (ac *AuthContext) GetAuthenticatedUserID(c *gin.Context) (string, bool) {
	// Try Firebase UID first
	if firebaseUID, exists := c.Get("firebase_uid"); exists {
		if uid, ok := firebaseUID.(string); ok && uid != "" {
			return uid, true
		}
	}

	// Try JWT user_id as fallback
	if userID, exists := c.Get("user_id"); exists {
		if uid, ok := userID.(string); ok && uid != "" {
			return uid, true
		}
	}

	return "", false
}

// GetAuthenticatedUserEmail returns the authenticated user email from context
func (ac *AuthContext) GetAuthenticatedUserEmail(c *gin.Context) (string, bool) {
	// Try Firebase email first
	if firebaseEmail, exists := c.Get("firebase_email"); exists {
		if email, ok := firebaseEmail.(string); ok && email != "" {
			return email, true
		}
	}

	// Try JWT email as fallback
	if jwtEmail, exists := c.Get("email"); exists {
		if email, ok := jwtEmail.(string); ok && email != "" {
			return email, true
		}
	}

	return "", false
}

// GetAuthenticatedUserRole returns the authenticated user role from context
func (ac *AuthContext) GetAuthenticatedUserRole(c *gin.Context) (string, bool) {
	if role, exists := c.Get("role"); exists {
		if r, ok := role.(string); ok && r != "" {
			return r, true
		}
	}

	return "", false
}

// IsAuthenticated checks if the request has valid authentication
func (ac *AuthContext) IsAuthenticated(c *gin.Context) bool {
	_, exists := ac.GetAuthenticatedUserID(c)
	return exists
}

// IsAdmin checks if the authenticated user has admin role
func (ac *AuthContext) IsAdmin(c *gin.Context) bool {
	role, exists := ac.GetAuthenticatedUserRole(c)
	return exists && role == "admin"
}

// NewAuthContext creates a new AuthContext instance
func NewAuthContext() *AuthContext {
	return &AuthContext{}
}
