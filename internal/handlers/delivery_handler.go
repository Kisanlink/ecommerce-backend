package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
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
// @Param shipment body requests.DelhiveryShipmentRequest true "Shipment Request"
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
	userService     *services.UserService
	logger          *zap.Logger
	auth            *authctx.AuthContext
}

// NewDeliveryHandler constructs a DeliveryHandler
func NewDeliveryHandler(deliveryService services.DeliveryServiceInterface, orderService services.OrderServiceInterface, userService *services.UserService, logger *zap.Logger) *DeliveryHandler {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &DeliveryHandler{
		deliveryService: deliveryService,
		orderService:    orderService,
		userService:     userService,
		logger:          logger,
		auth:            authctx.NewAuthContext(),
	}
}

// callerOwnsOrder reports whether the authenticated caller owns an order whose
// stored UserID is ownerUserID. Mirrors the resolution used when orders are
// created: the caller's Firebase UID is resolved to its internal ID read-only (no
// auto-create), with a direct match accepted to cover the Firebase-UID-fallback
// and JWT-internal-ID cases.
func (h *DeliveryHandler) callerOwnsOrder(c *gin.Context, callerID, ownerUserID string) bool {
	if ownerUserID == "" {
		return false
	}
	if ownerUserID == callerID {
		return true
	}
	if !strings.HasPrefix(callerID, "USER") && h.userService != nil {
		if resp, err := h.userService.GetUserByFirebaseUID(c.Request.Context(), callerID); err == nil && resp != nil && resp.Data != nil {
			if userMap, ok := resp.Data.(map[string]interface{}); ok {
				if id, ok := userMap["id"].(string); ok && id == ownerUserID {
					return true
				}
			}
		}
	}
	return false
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

	// Ownership check (#18): a waybill must not be a tracking oracle for any
	// logged-in user. Only the owner of the order carrying this waybill — or an
	// admin — may track it. Return 404 on mismatch/not-found to avoid confirming
	// which waybills exist.
	if !h.auth.IsAdmin(c) {
		callerID, ok := h.auth.GetAuthenticatedUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}
		order, oErr := h.orderService.GetOrderByWaybill(c.Request.Context(), waybill)
		if oErr != nil || order == nil || !h.callerOwnsOrder(c, callerID, order.UserID) {
			h.logger.Warn("Tracking denied - caller does not own the waybill",
				zap.String("user_id", callerID),
				zap.String("waybill", waybill))
			c.JSON(http.StatusNotFound, gin.H{"error": "Shipment not found"})
			return
		}
	}

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

	// #11: never overwrite an existing shipment. If the order already has a waybill, refuse rather
	// than creating a second parcel at the courier and losing the first AWB from the record.
	if strings.TrimSpace(order.DelhiveryWaybill) != "" {
		c.JSON(http.StatusConflict, gin.H{
			"success": false,
			"message": "order already has a shipment",
			"waybill": order.DelhiveryWaybill,
		})
		return
	}

	// #9: atomically claim the order so an admin double-click, or a concurrent webhook/verify,
	// cannot both create a shipment. The admin claim permits retrying shipment_failed /
	// needs_manual orders (e.g. after fixing the address) but still refuses a fulfilled order or
	// one with an attempt in progress.
	claimed, claimErr := h.orderService.ClaimOrderForShipmentAdmin(c.Request.Context(), orderID)
	if claimErr != nil {
		h.logger.Error("Failed to claim order for admin shipment", zap.Error(claimErr), zap.String("order_id", orderID))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "could not lock order for shipment"})
		return
	}
	if !claimed {
		c.JSON(http.StatusConflict, gin.H{"success": false, "message": "a shipment for this order is already in progress or completed"})
		return
	}

	res, err := h.deliveryService.CreateShipment(c.Request.Context(), order, items)
	if err != nil || res == nil || !res.Success || res.Waybill == "" {
		msg := "shipment creation failed"
		if res != nil && res.Message != "" {
			msg = res.Message
		}
		if err != nil {
			h.logger.Error("Create shipment error", zap.Error(err), zap.String("order_id", orderID))
		}
		// Release the claim (shipment_failed is retryable) so it can be re-attempted after a fix.
		_ = h.orderService.MarkFulfillmentStatus(c.Request.Context(), orderID, models.FulfillmentShipmentFailed, msg)
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": msg})
		return
	}

	trackingURL := res.TrackingURL
	if trackingURL == "" {
		trackingURL = fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", res.Waybill)
	}

	// Persist the waybill, advance the order to shipped, and mark fulfilled — one write.
	if aerr := h.orderService.ApplyShipmentSuccess(c.Request.Context(), orderID, res.Waybill, trackingURL, res.Status); aerr != nil {
		// #10: the parcel exists at the courier but we could not persist it — flag for manual
		// reconciliation (NOT shipment_failed) so it is never automatically re-shipped.
		h.logger.Error("Shipment created but DB update failed (admin path)",
			zap.Error(aerr), zap.String("order_id", orderID), zap.String("waybill", res.Waybill))
		_ = h.orderService.MarkFulfillmentStatus(c.Request.Context(), orderID, models.FulfillmentNeedsManual,
			fmt.Sprintf("shipment created at courier (waybill %s) but DB update failed: %s — reconcile manually, do NOT re-ship", res.Waybill, aerr.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "shipment created but could not be saved; recorded for manual reconciliation",
			"waybill": res.Waybill,
		})
		return
	}

	// Deduct inventory exactly once (idempotent; non-fatal).
	if invErr := h.orderService.DeductInventoryForOrder(c.Request.Context(), orderID); invErr != nil {
		h.logger.Warn("Inventory deduction failed (admin path)", zap.Error(invErr), zap.String("order_id", orderID))
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"order_id":     order.ID,
		"waybill":      res.Waybill,
		"status":       res.Status,
		"tracking_url": trackingURL,
	})
}

// SyncOrderDeliveryStatus handles POST /delivery/orders/:id/sync
// @Summary Sync delivery status from courier
// @Description Pull the latest Delhivery tracking status for an order and advance it to delivered when reported
// @Tags Delivery
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Order ID"
// @Success 200 {object} map[string]interface{} "Delivery status synced"
// @Failure 400 {object} map[string]interface{} "Invalid order id or no shipment"
// @Router /delivery/orders/{id}/sync [post]
func (h *DeliveryHandler) SyncOrderDeliveryStatus(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order id is required"})
		return
	}
	info, err := h.deliveryService.SyncDeliveryStatus(c.Request.Context(), orderID)
	if err != nil || info == nil {
		msg := "failed to sync delivery status"
		if err != nil {
			msg = err.Error()
		}
		h.logger.Warn("Failed to sync delivery status", zap.Error(err), zap.String("order_id", orderID))
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": msg})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"order_id":     orderID,
		"status":       info.Status,
		"tracking_url": info.TrackingURL,
		"last_update":  info.LastUpdate,
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

	// Ownership check (#17): order IDs must not be an enumeration oracle. Only the
	// order's owner — or an admin — may read its shipment. Return 404 on mismatch
	// so the response is indistinguishable from a non-existent order.
	if !h.auth.IsAdmin(c) {
		callerID, ok := h.auth.GetAuthenticatedUserID(c)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
			return
		}
		if !h.callerOwnsOrder(c, callerID, order.UserID) {
			h.logger.Warn("Order shipment denied - caller does not own the order",
				zap.String("user_id", callerID),
				zap.String("order_id", orderID))
			c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
			return
		}
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

// GetConfigurationStatus handles GET /delivery/debug/config
// @Summary Get Delhivery configuration status
// @Description Returns current Delhivery configuration for debugging
// @Tags Delivery Debug
// @Produce json
// @Success 200 {object} map[string]interface{} "Configuration status"
// @Router /delivery/debug/config [get]
func (h *DeliveryHandler) GetConfigurationStatus(c *gin.Context) {
	// Type assert to get the concrete service with the method
	if svc, ok := h.deliveryService.(*services.DeliveryService); ok {
		config := svc.GetConfigurationStatus()
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"config":  config,
			"hint":    "Check 'config_issues' for any problems. DELHIVERY_WAREHOUSE_NAME must EXACTLY match your pickup location name in Delhivery dashboard (case-sensitive).",
		})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to get configuration"})
}

// ValidateWarehouse handles GET /delivery/debug/validate-warehouse
// @Summary Validate Delhivery warehouse configuration
// @Description Validates that the warehouse name matches a registered pickup location in Delhivery
// @Tags Delivery Debug
// @Produce json
// @Success 200 {object} map[string]interface{} "Warehouse validation result"
// @Router /delivery/debug/validate-warehouse [get]
func (h *DeliveryHandler) ValidateWarehouse(c *gin.Context) {
	// Type assert to get the concrete service with the method
	if svc, ok := h.deliveryService.(*services.DeliveryService); ok {
		result, err := svc.ValidateWarehouse(c.Request.Context())
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		response := gin.H{
			"success":              result.IsValid,
			"warehouse_name":       result.WarehouseName,
			"issues":               result.Issues,
			"available_warehouses": result.AvailableWarehouses,
		}
		if !result.IsValid {
			response["fix_instructions"] = []string{
				"1. Log in to your Delhivery dashboard",
				"2. Go to Settings → Pickup Locations",
				"3. Find your pickup location and copy the EXACT name (case-sensitive)",
				"4. Set DELHIVERY_WAREHOUSE_NAME environment variable to this exact name",
				"5. Also set DELHIVERY_ORIGIN_ADDRESS to your actual pickup address",
				"6. Restart the server",
			}
		}
		c.JSON(http.StatusOK, response)
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": "unable to validate warehouse"})
}
