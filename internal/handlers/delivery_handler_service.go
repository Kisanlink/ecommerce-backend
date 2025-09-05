package handlers

import (
    "net/http"

    "github.com/Kisanlink/ecommerce-backend/entities/requests"
    "github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
    "github.com/Kisanlink/ecommerce-backend/internal/services"
    "github.com/gin-gonic/gin"
    "go.uber.org/zap"
)

// DeliveryHandler wires HTTP to DeliveryService
type DeliveryHandler struct {
    deliveryService services.DeliveryServiceInterface
    orderService    services.OrderServiceInterface
    logger          *zap.Logger
    authContext     *auth.AuthContext
}

func NewDeliveryHandler(dSvc services.DeliveryServiceInterface, oSvc services.OrderServiceInterface, logger *zap.Logger) *DeliveryHandler {
    return &DeliveryHandler{deliveryService: dSvc, orderService: oSvc, logger: logger, authContext: auth.NewAuthContext()}
}

// CheckServiceability checks if a pincode is serviceable
// GET /api/v1/delivery/serviceability/:pincode
func (h *DeliveryHandler) CheckServiceability(c *gin.Context) {
    pincode := c.Param("pincode")
    if pincode == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "pincode is required"})
        return
    }
    ok, err := h.deliveryService.CheckServiceability(c.Request.Context(), pincode)
    if err != nil {
        if h.logger != nil { h.logger.Error("Serviceability check failed", zap.Error(err), zap.String("pincode", pincode)) }
        c.JSON(http.StatusBadRequest, gin.H{"serviceable": false, "error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, gin.H{"serviceable": ok})
}

// Estimate calculates shipping estimate for the authenticated user
// POST /api/v1/delivery/estimate
func (h *DeliveryHandler) Estimate(c *gin.Context) {
    userID, ok := h.authContext.GetAuthenticatedUserID(c)
    if !ok || userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    var req requests.EstimateShippingRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        if h.logger != nil { h.logger.Warn("Invalid estimate request", zap.Error(err)) }
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
        return
    }
    est, err := h.deliveryService.Estimate(c.Request.Context(), userID, req)
    if err != nil {
        if h.logger != nil { h.logger.Error("Estimate failed", zap.Error(err)) }
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, est)
}

// GetLastEstimate returns the cached estimate for the authenticated user
// GET /api/v1/delivery/estimate/last
func (h *DeliveryHandler) GetLastEstimate(c *gin.Context) {
    userID, ok := h.authContext.GetAuthenticatedUserID(c)
    if !ok || userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    est, err := h.deliveryService.GetLastEstimate(c.Request.Context(), userID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, est)
}

// GetTrackingInfo returns a tracking link for a waybill
// GET /api/v1/delivery/track/:waybill
func (h *DeliveryHandler) GetTrackingInfo(c *gin.Context) {
    waybill := c.Param("waybill")
    if waybill == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "waybill is required"})
        return
    }
    info, err := h.deliveryService.GetTrackingInfo(c.Request.Context(), waybill)
    if err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, info)
}

// CreateOrderShipment generates a shipment (AWB/waybill/tracking) for an existing order
// POST /api/v1/delivery/orders/:id/shipment
func (h *DeliveryHandler) CreateOrderShipment(c *gin.Context) {
    userID, ok := h.authContext.GetAuthenticatedUserID(c)
    if !ok || userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    orderID := c.Param("id")
    if orderID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
        return
    }
    if h.orderService == nil || h.deliveryService == nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Services unavailable"})
        return
    }
    order, items, err := h.orderService.GetOrderWithItems(c.Request.Context(), orderID)
    if err != nil {
        if h.logger != nil { h.logger.Error("Failed to load order for shipment", zap.Error(err), zap.String("order_id", orderID)) }
        c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
        return
    }
    if order.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return
    }
    if order.DelhiveryWaybill != "" && order.TrackingURL != "" {
        c.JSON(http.StatusOK, gin.H{
            "success":      true,
            "message":      "Shipment already created",
            "waybill":      order.DelhiveryWaybill,
            "status":       order.DelhiveryStatus,
            "tracking_url": order.TrackingURL,
        })
        return
    }
    result, shipErr := h.deliveryService.CreateShipment(c.Request.Context(), order, items)
    if shipErr != nil || result == nil || !result.Success || result.Waybill == "" {
        msg := "Failed to create shipment"
        if shipErr != nil {
            msg = shipErr.Error()
        } else if result != nil && result.Message != "" {
            msg = result.Message
        }
        if h.logger != nil {
            if shipErr != nil { h.logger.Error("Shipment creation failed", zap.Error(shipErr), zap.String("order_id", orderID))
            } else { h.logger.Error("Shipment creation failed", zap.String("reason", msg), zap.String("order_id", orderID)) }
        }
        c.JSON(http.StatusBadRequest, gin.H{
            "error":   msg,
            "code":    "SHIPMENT_CREATE_FAILED",
            "order_id": orderID,
            "hints": []string{
                "Ensure pickup_location matches dashboard Facility Name exactly (case-sensitive)",
                "Verify API token and Authorization header",
                "Confirm address has a valid 6-digit pincode and phone",
                "Payment mode should be Prepaid when cod_amount is 0",
            },
        })
        return
    }
    order.DelhiveryWaybill = result.Waybill
    order.DelhiveryStatus = result.Status
    order.TrackingURL = result.TrackingURL
    if updErr := h.orderService.UpdateOrder(c.Request.Context(), order); updErr != nil {
        if h.logger != nil { h.logger.Warn("Failed to persist shipment fields on order", zap.Error(updErr), zap.String("order_id", orderID)) }
    }
    c.JSON(http.StatusOK, gin.H{
        "success":      true,
        "message":      result.Message,
        "waybill":      result.Waybill,
        "status":       result.Status,
        "tracking_url": result.TrackingURL,
    })
}

// GetOrderShipment returns current shipment info and tracking link
// GET /api/v1/delivery/orders/:id/shipment
func (h *DeliveryHandler) GetOrderShipment(c *gin.Context) {
    userID, ok := h.authContext.GetAuthenticatedUserID(c)
    if !ok || userID == "" {
        c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
        return
    }
    orderID := c.Param("id")
    if orderID == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
        return
    }
    order, _, err := h.orderService.GetOrderWithItems(c.Request.Context(), orderID)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
        return
    }
    if order.UserID != userID {
        c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
        return
    }
    waybill := order.DelhiveryWaybill
    trackingURL := order.TrackingURL
    status := order.DelhiveryStatus
    if waybill != "" && trackingURL == "" {
        if info, _ := h.deliveryService.GetTrackingInfo(c.Request.Context(), waybill); info != nil {
            trackingURL = info.TrackingURL
            if status == "" { status = info.Status }
        }
    }
    c.JSON(http.StatusOK, gin.H{
        "order_id":     order.ID,
        "waybill":      waybill,
        "status":       status,
        "tracking_url": trackingURL,
    })
}
