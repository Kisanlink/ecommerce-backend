package utils

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// APIResponse represents a standard API response
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// RespondWithSuccess sends a success response
func RespondWithSuccess(c *gin.Context, data interface{}) {
	response := APIResponse{
		Success:   true,
		Message:   "Success",
		Data:      data,
		Timestamp: time.Now(),
	}
	c.JSON(http.StatusOK, response)
}

// RespondWithError sends an error response
func RespondWithError(c *gin.Context, statusCode int, message string, err error) {
	response := APIResponse{
		Success:   false,
		Message:   message,
		Timestamp: time.Now(),
	}

	if err != nil {
		response.Error = err.Error()
	}

	c.JSON(statusCode, response)
}
