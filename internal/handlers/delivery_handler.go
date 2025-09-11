package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	authctx "github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ShipmentHandler handles shipment-related APIs
type ShipmentHandler struct {
	DelhiveryBaseURL string
	DelhiveryToken   string
}

// NewShipmentHandler creates new shipment handler
func NewShipmentHandler(baseURL, token string) *ShipmentHandler {
	return &ShipmentHandler{
		DelhiveryBaseURL: baseURL,
		DelhiveryToken:   token,
	}
}

// CreateShipment godoc
// @Summary Create shipment in Delhivery
// @Description Creates an order shipment and returns waybill + tracking URL
// @Tags Shipment
// @Accept json
// @Produce json
// @Param shipment body requests.Shipment true "Shipment Request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /shipment [post]
func (h *ShipmentHandler) CreateShipment(c *gin.Context) {
	var shipmentReq requests.DelhiveryShipmentRequest
	if err := c.ShouldBindJSON(&shipmentReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid shipment payload"})
		return
	}

	// Prepare payload: Delhivery expects format=json&data=<json> (x-www-form-urlencoded)
	payload := requests.NewCreateShipmentPayload(shipmentReq)
	dataJSON, _ := json.Marshal(payload.Data)

	form := url.Values{}
	form.Set("format", "json")
	form.Set("data", string(dataJSON))

	// Send request to Delhivery
	apiURL := fmt.Sprintf("%s/api/cmu/create.json", h.DelhiveryBaseURL)
	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Authorization", "Token "+h.DelhiveryToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to call Delhivery API"})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// Parse response
	shipmentResp, err := responses.ParseShipmentResponse(body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to parse Delhivery response"})
		return
	}

	// Validate success
	if !shipmentResp.IsSuccessful() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "shipment creation failed",
			"message": shipmentResp.GetErrorMessage(),
		})
		return
	}

	// Success → return waybill + tracking URL
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"waybill":      shipmentResp.GetWaybill(),
		"status":       shipmentResp.GetStatus(),
		"tracking_url": shipmentResp.GetTrackingURL(),
	})
}

// DeliveryHandler wires HTTP routes to DeliveryService and OrderService
type DeliveryHandler struct {
	deliveryService services.DeliveryServiceInterface
	orderService    services.OrderServiceInterface
	logger          *zap.Logger
	auth            *authctx.AuthContext
}

// NewDeliveryHandler constructs a DeliveryHandler
func NewDeliveryHandler(deliveryService services.DeliveryServiceInterface, orderService services.OrderServiceInterface, logger *zap.Logger) *DeliveryHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &DeliveryHandler{
		deliveryService: deliveryService,
		orderService:    orderService,
		logger:          logger,
		auth:            authctx.NewAuthContext(),
	}
}

// CheckServiceability handles GET /serviceability/:pincode
// @Summary Check serviceability by pincode
// @Description Check if delivery is serviceable for a given pincode
// @Tags Delivery
// @Produce json
// @Security FirebaseAuth
// @Param pincode path string true "Pincode"
// @Success 200 {object} map[string]interface{} "Serviceability status"
// @Failure 400 {object} map[string]interface{} "Invalid pincode or error"
// @Router /delivery/serviceability/{pincode} [get]
func (h *DeliveryHandler) CheckServiceability(c *gin.Context) {
	pincode := c.Param("pincode")
	ok, err := h.deliveryService.CheckServiceability(c.Request.Context(), pincode)
	if err != nil {
		h.logger.Warn("Serviceability check failed", zap.Error(err), zap.String("pincode", pincode))
		c.JSON(http.StatusBadRequest, gin.H{"serviceable": false, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"serviceable": ok, "pincode": pincode})
}

// Estimate handles POST /estimate
// @Summary Estimate shipping
// @Description Get shipping options and charges for an address
// @Tags Delivery
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.EstimateShippingRequest true "Address for estimation"
// @Success 200 {object} map[string]interface{} "Estimate computed"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Router /delivery/estimate [post]
func (h *DeliveryHandler) Estimate(c *gin.Context) {
	userID, ok := h.auth.GetAuthenticatedUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}
	var req requests.EstimateShippingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Phone number is optional for shipping estimation

	est, err := h.deliveryService.Estimate(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Warn("Estimate failed", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, est)
}

// GetTrackingInfo handles GET /track/:waybill
// @Summary Get tracking info
// @Description Get tracking URL and info for a waybill
// @Tags Delivery
// @Produce json
// @Security FirebaseAuth
// @Param waybill path string true "Delhivery Waybill"
// @Success 200 {object} map[string]interface{} "Tracking info"
// @Failure 400 {object} map[string]interface{} "Invalid waybill"
// @Router /delivery/track/{waybill} [get]
func (h *DeliveryHandler) GetTrackingInfo(c *gin.Context) {
	waybill := c.Param("waybill")
	info, err := h.deliveryService.GetTrackingInfo(c.Request.Context(), waybill)
	if err != nil {
		h.logger.Warn("Tracking fetch failed", zap.Error(err), zap.String("waybill", waybill))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// CreateOrderShipment handles POST /orders/:id/shipment
// @Summary Create shipment for order
// @Description Create a Delhivery shipment for the given order ID
// @Tags Delivery
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Order ID"
// @Success 200 {object} map[string]interface{} "Shipment created"
// @Failure 400 {object} map[string]interface{} "Invalid order id"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Router /delivery/orders/{id}/shipment [post]
func (h *DeliveryHandler) CreateOrderShipment(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order id is required"})
		return
	}
	order, items, err := h.orderService.GetOrderWithItems(c.Request.Context(), orderID)
	if err != nil || order == nil {
		h.logger.Warn("Order not found for shipment", zap.Error(err), zap.String("order_id", orderID))
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	res, err := h.deliveryService.CreateShipment(c.Request.Context(), order, items)
	if err != nil || res == nil || !res.Success {
		msg := "shipment creation failed"
		if res != nil && res.Message != "" {
			msg = res.Message
		}
		if err != nil {
			h.logger.Error("Create shipment error", zap.Error(err), zap.String("order_id", orderID))
		}
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": msg})
		return
	}
	// Update order with shipment details
	order.DelhiveryWaybill = res.Waybill
	order.DelhiveryStatus = res.Status
	order.TrackingURL = res.TrackingURL
	if uerr := h.orderService.UpdateOrder(c.Request.Context(), order); uerr != nil {
		h.logger.Warn("Failed to persist order shipment fields", zap.Error(uerr), zap.String("order_id", orderID))
	}
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"order_id":     order.ID,
		"waybill":      res.Waybill,
		"status":       res.Status,
		"tracking_url": res.TrackingURL,
	})
}

// GetOrderShipment handles GET /orders/:id/shipment
// @Summary Get order shipment
// @Description Get shipment details (waybill, status, tracking URL) for an order
// @Tags Delivery
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Order ID"
// @Success 200 {object} map[string]interface{} "Shipment details"
// @Failure 400 {object} map[string]interface{} "Invalid order id"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Router /delivery/orders/{id}/shipment [get]
func (h *DeliveryHandler) GetOrderShipment(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order id is required"})
		return
	}
	order, err := h.orderService.GetOrderByID(c.Request.Context(), orderID)
	if err != nil || order == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	resp := gin.H{
		"order_id":     order.ID,
		"waybill":      order.DelhiveryWaybill,
		"status":       order.DelhiveryStatus,
		"tracking_url": order.TrackingURL,
	}
	// If no waybill, hint to create
	if strings.TrimSpace(order.DelhiveryWaybill) == "" {
		resp["hint"] = "No shipment yet. POST /api/v1/delivery/orders/:id/shipment to create."
	}
	c.JSON(http.StatusOK, resp)
}
