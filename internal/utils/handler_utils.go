package utils

import "github.com/gin-gonic/gin"

// GetUserFromContext extracts user ID from gin context
func GetUserFromContext(c *gin.Context) string {
	if userID, exists := c.Get("user_id"); exists {
		return userID.(string)
	}
	return "system"
}
