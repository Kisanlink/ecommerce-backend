package handlers

import (
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DebugHandler struct {
	logger *zap.Logger
}

func NewDebugHandler(logger *zap.Logger) *DebugHandler {
	return &DebugHandler{
		logger: logger,
	}
}

// CheckFirebaseStatus checks if Firebase is properly initialized
// @Summary Firebase status
// @Description Check if Firebase Admin SDK is initialized on server
// @Tags Debug
// @Produce json
// @Success 200 {object} map[string]interface{} "Firebase status"
// @Router /debug/firebase [get]
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

// CheckDelhiveryConfig returns Delhivery configuration status (for debugging)
// @Summary Check Delhivery Configuration
// @Description Returns the status of Delhivery configuration for debugging purposes
// @Tags Debug
// @Produce json
// @Success 200 {object} map[string]interface{} "Configuration status"
// @Router /debug/delhivery-config [get]
func (h *DebugHandler) CheckDelhiveryConfig(c *gin.Context) {
	delhiveryConfig := config.GetDelhiveryConfig()
	issues := config.ValidateDelhiveryConfig(delhiveryConfig)

	response := map[string]interface{}{
		"base_url":         delhiveryConfig.BaseURL,
		"is_production":    delhiveryConfig.IsProduction,
		"warehouse_name":   delhiveryConfig.WarehouseName,
		"origin_pincode":   delhiveryConfig.OriginPincode,
		"origin_city":      delhiveryConfig.OriginCity,
		"origin_state":     delhiveryConfig.OriginState,
		"origin_phone":     delhiveryConfig.OriginPhone,
		"default_weight":   delhiveryConfig.DefaultWeight,
		"api_token_set":    delhiveryConfig.APIToken != "",
		"api_token_length": len(delhiveryConfig.APIToken),
		"api_token_preview": func() string {
			if len(delhiveryConfig.APIToken) >= 8 {
				return delhiveryConfig.APIToken[:8] + "..."
			}
			return "TOO_SHORT"
		}(),
		"config_valid":  len(issues) == 0,
		"config_issues": issues,
		"timestamp":     c.GetHeader("X-Request-ID"),
	}

	h.logger.Info("Delhivery configuration debug check requested",
		zap.Bool("config_valid", len(issues) == 0),
		zap.Int("issues_count", len(issues)),
		zap.Strings("issues", issues))

	utils.RespondWithSuccess(c, response)
}
