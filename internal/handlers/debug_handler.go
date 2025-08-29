package handlers

import (
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

type DebugHandler struct{}

func NewDebugHandler() *DebugHandler {
	return &DebugHandler{}
}

// CheckFirebaseStatus checks if Firebase is properly initialized
func (h *DebugHandler) CheckFirebaseStatus(c *gin.Context) {
	client := config.GetFirebaseAuth()

	status := map[string]interface{}{
		"firebase_initialized": client != nil,
		"message":              "Firebase authentication status",
	}

	if client != nil {
		status["status"] = "Firebase is properly initialized and ready"
	} else {
		status["status"] = "Firebase is NOT initialized - check service account file or environment variables"
		status["help"] = "Ensure firebase_service_account.json exists or proper environment variables are set"
	}

	utils.RespondWithSuccess(c, status)
}
