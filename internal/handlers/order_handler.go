package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	orderService services.OrderServiceInterface
	userService  *services.UserService
	authContext  *auth.AuthContext
	logger       *zap.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService services.OrderServiceInterface, userService *services.UserService, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		userService:  userService,
		authContext:  auth.NewAuthContext(),
		logger:       logger,
	}
}

// mapFirebaseUIDToInternalUserID ensures we always use the internal USER... ID for orders.
// It mirrors the logic used in payment/cart/profile flows:
//   1) Try GetUserByFirebaseUID
//   2) If not found, fetch from Firebase Admin and try by email/phone
//   3) Auto-create a local user when needed
func (h *OrderHandler) mapFirebaseUIDToInternalUserID(c *gin.Context, firebaseUID string) string {
	if h.userService == nil || firebaseUID == "" {
		return firebaseUID
	}

	ctx := c.Request.Context()

	// 1) Direct lookup by Firebase UID
	if userResp, err := h.userService.GetUserByFirebaseUID(ctx, firebaseUID); err == nil && userResp != nil && userResp.Success && userResp.Data != nil {
		if userMap, ok := userResp.Data.(map[string]interface{}); ok {
			if internalID, ok := userMap["id"].(string); ok && internalID != "" {
				h.logger.Info("OrderHandler: mapped Firebase UID to internal user ID",
					zap.String("firebase_uid", firebaseUID),
					zap.String("internal_id", internalID))
				return internalID
			}
		}
	}

	// 2) Enrich email/phone from Firebase Admin
	var firebaseEmail, firebasePhone string
	if emailVal, exists := c.Get("firebase_email"); exists {
		if email, ok := emailVal.(string); ok {
			firebaseEmail = email
		}
	}
	if phoneVal, exists := c.Get("firebase_phone"); exists {
		if phone, ok := phoneVal.(string); ok {
			firebasePhone = phone
		}
	}

	if firebaseEmail == "" || firebasePhone == "" {
		if firebaseClient := config.GetFirebaseAuth(); firebaseClient != nil {
			if firebaseUser, err := firebaseClient.GetUser(ctx, firebaseUID); err == nil && firebaseUser != nil {
				if firebaseEmail == "" && firebaseUser.Email != "" {
					firebaseEmail = firebaseUser.Email
				}
				if firebasePhone == "" && firebaseUser.PhoneNumber != "" {
					firebasePhone = firebaseUser.PhoneNumber
				}
			} else if err != nil {
				h.logger.Error("OrderHandler: failed to fetch user from Firebase",
					zap.String("firebase_uid", firebaseUID),
					zap.Error(err))
			}
		} else {
			h.logger.Error("OrderHandler: Firebase client not initialized",
				zap.String("firebase_uid", firebaseUID))
		}
	}

	// 3) Resolve or create by phone
	if firebasePhone != "" {
		if byPhone, _ := h.userService.GetUserByPhoneNumber(ctx, firebasePhone); byPhone != nil && byPhone.Success && byPhone.Data != nil {
			if userMap, ok := byPhone.Data.(map[string]interface{}); ok {
				if internalID, ok := userMap["id"].(string); ok && internalID != "" {
					_ = h.userService.UpdateUserFirebaseUID(ctx, internalID, firebaseUID)
					h.logger.Info("OrderHandler: linked Firebase UID to existing phone user",
						zap.String("firebase_uid", firebaseUID),
						zap.String("internal_id", internalID))
					return internalID
				}
			}
		}
	}

	// 4) Resolve or create by email
	if firebaseEmail != "" && strings.Contains(firebaseEmail, "@") {
		if byEmail, _ := h.userService.GetUserByEmail(ctx, firebaseEmail); byEmail != nil && byEmail.Success && byEmail.Data != nil {
			if userMap, ok := byEmail.Data.(map[string]interface{}); ok {
				if internalID, ok := userMap["id"].(string); ok && internalID != "" {
					_ = h.userService.UpdateUserFirebaseUID(ctx, internalID, firebaseUID)
					h.logger.Info("OrderHandler: linked Firebase UID to existing email user",
						zap.String("firebase_uid", firebaseUID),
						zap.String("internal_id", internalID))
					return internalID
				}
			}
		}
	}

	// 5) Auto-create minimal user if we have at least an email or phone
	if firebasePhone != "" && firebaseEmail == "" {
		// Phone-only user
		phoneReq := requests.PhoneSignupRequest{
			PhoneNumber: firebasePhone,
			Name:        "User",
		}
		if createResp, err := h.userService.CreatePhoneUser(ctx, &phoneReq); err == nil && createResp != nil && createResp.Success {
			if data, ok := createResp.Data.(map[string]interface{}); ok {
				if internalID, ok := data["id"].(string); ok && internalID != "" {
					_ = h.userService.UpdateUserFirebaseUID(ctx, internalID, firebaseUID)
					h.logger.Info("OrderHandler: auto-created phone user from Firebase UID",
						zap.String("firebase_uid", firebaseUID),
						zap.String("internal_id", internalID))
					return internalID
				}
			}
		}
	}

	if firebaseEmail != "" {
		// Email/Google user
		name := firebaseEmail
		if at := strings.Index(firebaseEmail, "@"); at > 0 {
			name = firebaseEmail[:at]
		}
		signupReq := requests.SignupRequest{
			Email:    firebaseEmail,
			Name:     name,
			Password: "",
		}
		if createResp, err := h.userService.CreateGoogleUser(ctx, &signupReq, firebaseUID); err == nil && createResp != nil && createResp.Success {
			if data, ok := createResp.Data.(map[string]interface{}); ok {
				if internalID, ok := data["id"].(string); ok && internalID != "" {
					h.logger.Info("OrderHandler: auto-created email/Google user from Firebase UID",
						zap.String("firebase_uid", firebaseUID),
						zap.String("internal_id", internalID))
					return internalID
				}
			}
		}
	}

	// Fallback: return the original UID (order service will likely not find a cart for it)
	h.logger.Warn("OrderHandler: failed to resolve Firebase UID to internal user ID; using raw UID as fallback",
		zap.String("firebase_uid", firebaseUID))
	return firebaseUID
}

// CreateOrder creates a new order manually (alternative to payment flow)
// POST /api/v1/orders
// @Summary Create order from cart (manual)
// @Description Create an order directly from the user's cart without Razorpay flow
// @Tags Orders
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body object{cart_id=string,customer_first_name=string,customer_last_name=string,customer_phone=string,customer_phone_alt=string,customer_email=string,address_line_1=string,address_line_2=string,city=string,state=string,postal_code=string,country=string,payment_method=string,notes=string} true "Order creation payload"
// @Success 201 {object} map[string]interface{} "Order created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to create order"
// @Router /orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	requestID := c.GetString("request_id")
	h.logger.Info("Creating order from cart - direct API",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path))

	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User authentication failed - no user ID found",
			zap.String("request_id", requestID),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Ensure we are working with internal USER... ID, not raw Firebase UID,
	// so that we hit the same cart/profile rows used everywhere else.
	if !strings.HasPrefix(userID, "USER") {
		userID = h.mapFirebaseUIDToInternalUserID(c, userID)
	}

	h.logger.Info("User authenticated for order creation",
		zap.String("request_id", requestID),
		zap.String("user_id", userID))

	var req struct {
		CartID        string `json:"cart_id" binding:"required"`
		PaymentMethod string `json:"payment_method" binding:"required"`
		Notes         string `json:"notes"`
		// Customer data for normalized schema
		CustomerFirstName string `json:"customer_first_name" binding:"required"`
		CustomerLastName  string `json:"customer_last_name" binding:"required"`
		CustomerPhone     string `json:"customer_phone" binding:"required"`
		CustomerPhoneAlt  string `json:"customer_phone_alt"`
		CustomerEmail     string `json:"customer_email"`
		// Address data for normalized schema
		AddressLine1 string `json:"address_line_1" binding:"required"`
		AddressLine2 string `json:"address_line_2"`
		City         string `json:"city" binding:"required"`
		State        string `json:"state" binding:"required"`
		PostalCode   string `json:"postal_code" binding:"required"`
		Country      string `json:"country"`
		// Legacy support
		ShippingAddress string `json:"shipping_address"` // Optional for backward compatibility
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request data for order creation",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err),
			zap.String("error_type", "validation_error"))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set default country if not provided
	if req.Country == "" {
		req.Country = "India"
	}

	// If legacy shipping_address is provided but normalized fields are missing, extract them
	if req.ShippingAddress != "" && (req.AddressLine1 == "" || req.CustomerFirstName == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Please provide customer and address details in the new format"})
		return
	}

	// Create normalized customer data
	customerData := models.NewOrderCustomer(
		req.CustomerFirstName,
		req.CustomerLastName,
		req.CustomerPhone,
		req.CustomerPhoneAlt,
		req.CustomerEmail,
	)

	// Create normalized address data
	addressData := models.NewOrderAddress(
		req.AddressLine1,
		req.AddressLine2,
		req.City,
		req.State,
		req.PostalCode,
		req.Country,
	)

	h.logger.Info("Order creation request validated",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("cart_id", req.CartID),
		zap.String("payment_method", req.PaymentMethod))

	order, err := h.orderService.CreateOrder(
		c.Request.Context(),
		userID,
		customerData,
		addressData,
		req.PaymentMethod,
		0, // Default shipping charges
	)

	if err != nil {
		h.logger.Error("Failed to create order from cart",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("cart_id", req.CartID),
			zap.String("payment_method", req.PaymentMethod),
			zap.Error(err),
			zap.String("error_type", "service_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	h.logger.Info("Order created successfully via direct API",
		zap.String("request_id", requestID),
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.String("cart_id", req.CartID),
		zap.Float64("total_amount", order.TotalAmount),
		zap.String("status", string(order.Status)),
		zap.String("payment_method", req.PaymentMethod))

	response := responses.NewCreateOrderResponse(order)
	c.JSON(http.StatusCreated, response)
}

// GetUserOrders retrieves all orders for the authenticated user
// GET /api/v1/orders
// @Summary List user's orders
// @Description Get a paginated list of orders for the authenticated user
// @Tags Orders
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(10)
// @Success 200 {object} map[string]interface{} "Orders retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /orders [get]
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	requestID := c.GetString("request_id")
	h.logger.Info("Retrieving user orders",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path))

	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User authentication failed for orders retrieval",
			zap.String("request_id", requestID),
			zap.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Map Firebase UID to internal user ID if needed
	// Orders are stored with database user ID (USER...), not Firebase UID
	internalUserID := userID
	if !strings.HasPrefix(userID, "USER") {
		// This is likely a Firebase UID, map it to internal user ID
		if h.userService != nil {
			userResponse, err := h.userService.GetUserByFirebaseUID(c.Request.Context(), userID)
			if err == nil && userResponse != nil && userResponse.Success {
				if userData, ok := userResponse.Data.(map[string]interface{}); ok {
					if id, ok := userData["id"].(string); ok {
						internalUserID = id
						h.logger.Info("Mapped Firebase UID to internal user ID for orders retrieval",
							zap.String("request_id", requestID),
							zap.String("firebase_uid", userID),
							zap.String("internal_id", internalUserID))
					}
				}
			} else {
				h.logger.Error("Failed to map Firebase UID to internal user ID in GetUserOrders",
					zap.String("request_id", requestID),
					zap.String("firebase_uid", userID),
					zap.Error(err))
				// Continue with Firebase UID as fallback (will likely return no results)
			}
		}
	}

	// Parse query parameters
	page := 1
	limit := 10
	if p := c.Query("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	h.logger.Info("Parsed pagination parameters",
		zap.String("request_id", requestID),
		zap.String("user_id", internalUserID),
		zap.Int("page", page),
		zap.Int("limit", limit))

	orders, err := h.orderService.GetUserOrders(c.Request.Context(), internalUserID)
	if err != nil {
		h.logger.Error("Failed to retrieve user orders from service",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err),
			zap.String("error_type", "service_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders"})
		return
	}

	h.logger.Info("Orders retrieved from database",
		zap.String("request_id", requestID),
		zap.String("user_id", internalUserID),
		zap.Int("total_orders", len(orders)))

	// Simple pagination (in production, you'd want to do this at the database level)
	start := (page - 1) * limit
	end := start + limit
	if start >= len(orders) {
		orders = []models.Order{}
	} else if end > len(orders) {
		orders = orders[start:]
	} else {
		orders = orders[start:end]
	}

	h.logger.Info("Orders pagination applied",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.Int("returned_count", len(orders)),
		zap.Bool("has_more", end < len(orders)))

	response := responses.NewGetUserOrdersResponse(orders, page, limit)
	c.JSON(http.StatusOK, response)
}

// GetOrderDetails retrieves details of a specific order
// GET /api/v1/orders/:id
// @Summary Get order details
// @Description Get details and items of a specific order
// @Tags Orders
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Order ID"
// @Success 200 {object} map[string]interface{} "Order details retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Order ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied"
// @Failure 404 {object} map[string]interface{} "Order not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /orders/{id} [get]
func (h *OrderHandler) GetOrderDetails(c *gin.Context) {
	requestID := c.GetString("request_id")
	orderID := c.Param("id")

	h.logger.Info("Retrieving order details",
		zap.String("request_id", requestID),
		zap.String("order_id", orderID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path))

	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User authentication failed for order details",
			zap.String("request_id", requestID),
			zap.String("order_id", orderID),
			zap.String("client_ip", c.ClientIP()))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Map Firebase UID to internal user ID if needed
	// Orders are stored with database user ID (USER...), not Firebase UID
	internalUserID := userID
	if !strings.HasPrefix(userID, "USER") {
		// This is likely a Firebase UID, map it to internal user ID
		if h.userService != nil {
			userResponse, err := h.userService.GetUserByFirebaseUID(c.Request.Context(), userID)
			if err == nil && userResponse != nil && userResponse.Success {
				if userData, ok := userResponse.Data.(map[string]interface{}); ok {
					if id, ok := userData["id"].(string); ok {
						internalUserID = id
						h.logger.Info("Mapped Firebase UID to internal user ID for order details",
							zap.String("request_id", requestID),
							zap.String("firebase_uid", userID),
							zap.String("internal_id", internalUserID))
					}
				}
			} else {
				h.logger.Error("Failed to map Firebase UID to internal user ID in GetOrderDetails",
					zap.String("request_id", requestID),
					zap.String("firebase_uid", userID),
					zap.Error(err))
				// Continue with Firebase UID as fallback (will likely fail ownership check)
			}
		}
	}

	if orderID == "" {
		h.logger.Error("Order ID missing in request",
			zap.String("request_id", requestID),
			zap.String("user_id", internalUserID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	h.logger.Info("Fetching order details from service",
		zap.String("request_id", requestID),
		zap.String("user_id", internalUserID),
		zap.String("order_id", orderID))

	// Get order with details and enriched items (for product images)
	order, enrichedItems, err := h.orderService.GetOrderDetails(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order details from service",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("user_id", internalUserID),
			zap.String("error_type", "service_error"))

		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			h.logger.Warn("Order not found in database",
				zap.String("request_id", requestID),
				zap.String("order_id", orderID),
				zap.String("user_id", internalUserID))
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Order not found. Please check the order ID and try again.",
				"code":  "ORDER_NOT_FOUND"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to retrieve order details. Please try again later.",
				"code":  "INTERNAL_ERROR"})
		}
		return
	}

	// Verify the order belongs to the authenticated user
	if order.UserID != internalUserID {
		h.logger.Warn("User attempted to access order that doesn't belong to them",
			zap.String("request_id", requestID),
			zap.String("user_id", internalUserID),
			zap.String("order_id", orderID),
			zap.String("order_owner_id", order.UserID),
			zap.String("security_event", "unauthorized_access_attempt"))
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.logger.Info("Order details retrieved successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", internalUserID),
		zap.String("order_id", orderID),
		zap.String("order_status", string(order.Status)),
		zap.Float64("total_amount", order.TotalAmount),
		zap.Int("item_count", len(order.OrderItems)))

	// Build enhanced response with all product images and details
	response := responses.NewGetOrderDetailsResponseEnhanced(order, enrichedItems)
	c.JSON(http.StatusOK, response)
}

// GetOrderStatus retrieves the current status of an order
// GET /api/v1/orders/:id/status
// @Summary Get order status
// @Description Get current status and payment status of an order
// @Tags Orders
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Order ID"
// @Success 200 {object} map[string]interface{} "Order status retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Order ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /orders/{id}/status [get]
func (h *OrderHandler) GetOrderStatus(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	order, err := h.orderService.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		return
	}

	// Verify the order belongs to the authenticated user
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	response := responses.NewGetOrderStatusResponse(order)
	c.JSON(http.StatusOK, response)
}

// GetOrderTracking retrieves tracking information for an order
// GET /api/v1/orders/:id/tracking
// @Summary Get order tracking
// @Description Get tracking details and shipping status of an order
// @Tags Orders
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Order ID"
// @Success 200 {object} map[string]interface{} "Order tracking retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Order ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 403 {object} map[string]interface{} "Access denied"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /orders/{id}/tracking [get]
func (h *OrderHandler) GetOrderTracking(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	order, err := h.orderService.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		return
	}

	// Verify the order belongs to the authenticated user
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	response := responses.NewGetOrderTrackingResponse(order)
	c.JSON(http.StatusOK, response)
}

// GetOrderDetailsDebug (removed, #6): the unauthenticated order-details debug endpoint leaked
// order PII (user_id, shipping address, waybill, total) for any guessable order ID. Authenticated,
// ownership-checked order reads are available via GetOrderDetails.

// ==========================
// Admin Order Management
// ==========================

// GetAllOrders lists orders for admin with optional filters and pagination
// GET /api/v1/admin/orders
func (h *OrderHandler) GetAllOrders(c *gin.Context) {
	// Parse filters
	status := c.Query("status")
	paymentStatus := c.Query("payment_status")
	q := c.Query("q")
	fromStr := c.Query("from")
	toStr := c.Query("to")
	page := 1
	limit := 20
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	var from, to *time.Time
	if fromStr != "" {
		if t, err := time.Parse(time.RFC3339, fromStr); err == nil {
			from = &t
		}
	}
	if toStr != "" {
		if t, err := time.Parse(time.RFC3339, toStr); err == nil {
			to = &t
		}
	}

	orders, total, err := h.orderService.ListOrders(c.Request.Context(), status, paymentStatus, q, from, to, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":     page,
			"limit":    limit,
			"total":    total,
			"has_more": page*limit < total,
		},
	})
}

// GetUnshippedOrders returns paid orders that have no shipment yet (recovery view).
// GET /api/v1/admin/orders/unshipped
func (h *OrderHandler) GetUnshippedOrders(c *gin.Context) {
	page := 1
	limit := 50
	if v := c.Query("page"); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := c.Query("limit"); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 200 {
			limit = l
		}
	}

	orders, total, err := h.orderService.ListUnfulfilledOrders(c.Request.Context(), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list unshipped orders"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":     page,
			"limit":    limit,
			"total":    total,
			"has_more": page*limit < total,
		},
	})
}

// GetOrderDetailsAdmin returns order details for admin without ownership checks
// GET /api/v1/admin/orders/:id
func (h *OrderHandler) GetOrderDetailsAdmin(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}
	order, enrichedItems, err := h.orderService.GetOrderDetails(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		return
	}
	c.JSON(http.StatusOK, responses.NewGetOrderDetailsResponseEnhanced(order, enrichedItems))
}

// UpdateOrderStatus updates order status by admin
// PUT /api/v1/admin/orders/:id/status
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}
	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Status == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}
	if err := h.orderService.UpdateOrderStatus(c.Request.Context(), orderID, models.OrderStatus(req.Status)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Order status updated"})
}

// UpdatePayment updates payment status and optional payment_id
// PUT /api/v1/admin/orders/:id/payment
func (h *OrderHandler) UpdatePayment(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}
	var req struct {
		PaymentStatus string `json:"payment_status" binding:"required"`
		PaymentID     string `json:"payment_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.PaymentStatus == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_status is required"})
		return
	}
	if err := h.orderService.UpdatePaymentStatus(c.Request.Context(), orderID, req.PaymentStatus, req.PaymentID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update payment status"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Payment status updated"})
}

// UpdateShipment sets shipment details
// PUT /api/v1/admin/orders/:id/shipment
func (h *OrderHandler) UpdateShipment(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}
	var req struct {
		ShippingCarrier string `json:"shipping_carrier"`
		ShippingMode    string `json:"shipping_mode"`
		Waybill         string `json:"delhivery_waybill"`
		TrackingURL     string `json:"tracking_url"`
		EtaMinDays      *int   `json:"eta_min_days"`
		EtaMaxDays      *int   `json:"eta_max_days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid payload"})
		return
	}
	order, err := h.orderService.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load order"})
		return
	}
	if req.ShippingCarrier != "" {
		order.ShippingCarrier = req.ShippingCarrier
	}
	if req.ShippingMode != "" {
		order.ShippingMode = req.ShippingMode
	}
	if req.Waybill != "" {
		order.DelhiveryWaybill = req.Waybill
	}
	if req.TrackingURL != "" {
		order.TrackingURL = req.TrackingURL
	}
	if req.EtaMinDays != nil {
		order.EtaMinDays = *req.EtaMinDays
	}
	if req.EtaMaxDays != nil {
		order.EtaMaxDays = *req.EtaMaxDays
	}
	if err := h.orderService.UpdateOrder(c.Request.Context(), order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update shipment"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Shipment updated"})
}

// GetOrderStats returns aggregate stats for admin
// GET /api/v1/admin/orders/stats
func (h *OrderHandler) GetOrderStats(c *gin.Context) {
	stats, err := h.orderService.GetOrderStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get stats"})
		return
	}
	c.JSON(http.StatusOK, stats)
}

// ExportOrders exports orders as CSV
// GET /api/v1/admin/orders/export
func (h *OrderHandler) ExportOrders(c *gin.Context) {
	data, filename, err := h.orderService.ExportOrdersCSV(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export orders"})
		return
	}
	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "text/csv", data)
}
