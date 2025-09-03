package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type DeliveryHandler struct {
	service services.DeliveryServiceInterface
	logger  *zap.Logger
	authCtx *auth.AuthContext
}

func NewDeliveryHandler(service services.DeliveryServiceInterface, logger *zap.Logger) *DeliveryHandler {
	return &DeliveryHandler{service: service, logger: logger, authCtx: auth.NewAuthContext()}
}

func (h *DeliveryHandler) Estimate(c *gin.Context) {
	userID, ok := h.authCtx.GetAuthenticatedUserID(c)
	if !ok {
		return
	}

	var req requests.EstimateShippingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	est, err := h.service.Estimate(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Error("delivery estimate failed", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "failed to estimate delivery"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": est})
}

// TestShipmentCreation manually creates a Delhivery shipment for testing
func (h *DeliveryHandler) TestShipmentCreation(c *gin.Context) {
	userID, ok := h.authCtx.GetAuthenticatedUserID(c)
	if !ok {
		return
	}

	var req struct {
		OrderID string `json:"order_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "invalid request: " + err.Error()})
		return
	}

	h.logger.Info("Manual Delhivery shipment test requested",
		zap.String("user_id", userID),
		zap.String("order_id", req.OrderID))

	c.JSON(http.StatusOK, gin.H{
		"success":  true,
		"message":  "Manual shipment creation is not yet implemented. Check server logs for Delhivery integration details.",
		"order_id": req.OrderID,
		"note":     "This endpoint is for debugging. The actual shipment creation happens during payment verification."})
}
