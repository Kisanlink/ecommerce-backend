package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	orderService services.OrderServiceInterface
	authContext  *auth.AuthContext
	logger       *zap.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService services.OrderServiceInterface, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		authContext:  auth.NewAuthContext(),
		logger:       logger,
	}
}

// CreateOrder creates a new order manually (alternative to payment flow)
// POST /api/v1/orders
// @Summary Create order from cart (manual)
// @Description Create an order directly from the user's cart without Razorpay flow
// @Tags Orders
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body object{cart_id=string,shipping_address=string,payment_method=string,notes=string} true "Order creation payload"
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

	h.logger.Info("User authenticated for order creation",
		zap.String("request_id", requestID),
		zap.String("user_id", userID))

	var req struct {
		CartID          string `json:"cart_id" binding:"required"`
		ShippingAddress string `json:"shipping_address" binding:"required"`
		PaymentMethod   string `json:"payment_method" binding:"required"`
		Notes           string `json:"notes"`
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

	h.logger.Info("Order creation request validated",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("cart_id", req.CartID),
		zap.String("payment_method", req.PaymentMethod))

	order, err := h.orderService.CreateOrderFromCart(
		c.Request.Context(),
		userID,
		req.CartID,
		req.ShippingAddress,
		req.PaymentMethod,
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

	c.JSON(http.StatusCreated, gin.H{
		"message": "Order created successfully",
		"order":   order,
	})
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
		zap.String("user_id", userID),
		zap.Int("page", page),
		zap.Int("limit", limit))

	orders, err := h.orderService.GetUserOrders(c.Request.Context(), userID)
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
		zap.String("user_id", userID),
		zap.Int("total_orders", len(orders)))

	// Simple pagination (in production, you'd want to do this at the database level)
	start := (page - 1) * limit
	end := start + limit
	if start >= len(orders) {
		orders = []*models.Order{}
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

	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"pagination": gin.H{
			"page":     page,
			"limit":    limit,
			"total":    len(orders),
			"has_more": end < len(orders),
		},
	})
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

	if orderID == "" {
		h.logger.Error("Order ID missing in request",
			zap.String("request_id", requestID),
			zap.String("user_id", userID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	h.logger.Info("Fetching order details from service",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("order_id", orderID))

	// Get order with items and product details
	order, orderItemsWithProducts, err := h.orderService.GetOrderWithItemsAndProducts(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order details from service",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("user_id", userID),
			zap.String("error_type", "service_error"))

		// Check if it's a "not found" error
		if strings.Contains(err.Error(), "not found") {
			h.logger.Warn("Order not found in database",
				zap.String("request_id", requestID),
				zap.String("order_id", orderID),
				zap.String("user_id", userID))
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
	if order.UserID != userID {
		h.logger.Warn("User attempted to access order that doesn't belong to them",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("order_id", orderID),
			zap.String("order_owner_id", order.UserID),
			zap.String("security_event", "unauthorized_access_attempt"))
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	h.logger.Info("Order details retrieved successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("order_id", orderID),
		zap.String("order_status", string(order.Status)),
		zap.Float64("total_amount", order.TotalAmount),
		zap.Int("item_count", len(orderItemsWithProducts)))

	c.JSON(http.StatusOK, gin.H{
		"order": order,
		"items": orderItemsWithProducts,
	})
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
		h.logger.Error("Failed to get order status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order status"})
		return
	}

	// Verify the order belongs to the authenticated user
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order_id":       order.ID,
		"status":         order.Status,
		"payment_status": order.PaymentStatus,
		"created_at":     order.CreatedAt,
		"updated_at":     order.UpdatedAt,
	})
}

// GetOrderTracking retrieves tracking information for an order
// GET /api/v1/orders/:id/tracking
// @Summary Get order tracking info
// @Description Get tracking timeline and basic shipment info for an order
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
		h.logger.Error("Failed to get order for tracking", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order"})
		return
	}

	// Verify the order belongs to the authenticated user
	if order.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Basic tracking information
	trackingInfo := gin.H{
		"order_id":    order.ID,
		"status":      order.Status,
		"created_at":  order.CreatedAt,
		"updated_at":  order.UpdatedAt,
		"tracking_id": "", // Will be populated when Delhivery integration is added
		"carrier":     "", // Will be populated when Delhivery integration is added
	}

	// Status timeline (basic implementation)
	timeline := []gin.H{
		{
			"status":      "pending",
			"description": "Order created and waiting for payment",
			"timestamp":   order.CreatedAt,
			"completed":   order.Status != models.OrderStatusPending,
		},
	}

	if order.Status != models.OrderStatusPending {
		timeline = append(timeline, gin.H{
			"status":      "confirmed",
			"description": "Payment confirmed, order is being processed",
			"timestamp":   order.UpdatedAt, // In production, you'd track status change timestamps
			"completed":   order.Status != models.OrderStatusConfirmed,
		})
	}

	if order.Status == models.OrderStatusShipped || order.Status == models.OrderStatusDelivered {
		timeline = append(timeline, gin.H{
			"status":      "shipped",
			"description": "Order has been shipped",
			"timestamp":   order.UpdatedAt,
			"completed":   order.Status == models.OrderStatusDelivered,
		})
	}

	if order.Status == models.OrderStatusDelivered {
		timeline = append(timeline, gin.H{
			"status":      "delivered",
			"description": "Order has been delivered",
			"timestamp":   order.UpdatedAt,
			"completed":   true,
		})
	}

	trackingInfo["timeline"] = timeline

	// TODO: Integrate with Delhivery API for real tracking
	// This would involve:
	// 1. Storing Delhivery waybill/tracking ID when order is shipped
	// 2. Calling Delhivery API to get real-time tracking updates
	// 3. Returning actual carrier information and delivery estimates

	c.JSON(http.StatusOK, trackingInfo)
}

// Future methods for admin functionality
// These can be added later when admin middleware is implemented

// GetAllOrders retrieves all orders (admin only)
// @Summary Admin: list all orders
// @Tags Orders
// @Produce json
// @Router /admin/orders [get]
func (h *OrderHandler) GetAllOrders(c *gin.Context) {
	// TODO: Implement admin-only order retrieval
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Admin functionality not yet implemented"})
}

// UpdateOrderStatus updates order status (admin only)
// @Summary Admin: update order status
// @Tags Orders
// @Accept json
// @Produce json
// @Router /admin/orders/{id}/status [put]
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	// TODO: Implement admin order status updates
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Admin functionality not yet implemented"})
}

// GetOrderStats retrieves order statistics (admin only)
// @Summary Admin: get order statistics
// @Tags Orders
// @Produce json
// @Router /admin/orders/stats [get]
func (h *OrderHandler) GetOrderStats(c *gin.Context) {
	// TODO: Implement order statistics
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Admin functionality not yet implemented"})
}

// GetOrderDetailsDebug retrieves order details without authentication (debug only)
// @Summary Debug: get order details (no auth)
// @Tags Debug
// @Produce json
// @Param id path string true "Order ID"
// @Router /debug/order/{id} [get]
func (h *OrderHandler) GetOrderDetailsDebug(c *gin.Context) {
	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	order, err := h.orderService.GetOrderByID(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order for debug", zap.Error(err), zap.String("order_id", orderID))
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	// Return detailed order info including Delhivery fields
	response := gin.H{
		"order_id":          order.ID,
		"user_id":           order.UserID,
		"total_amount":      order.TotalAmount,
		"status":            order.Status,
		"payment_status":    order.PaymentStatus,
		"shipping_address":  order.ShippingAddr,
		"razorpay_order_id": order.RazorpayOrderID,
		"delhivery_waybill": order.DelhiveryWaybill, // Check this field
		"delhivery_status":  order.DelhiveryStatus,  // Check this field
		"tracking_url":      order.TrackingURL,      // Check this field
		"created_at":        order.CreatedAt,
		"updated_at":        order.UpdatedAt,
		"debug_note":        "This is a debug endpoint without authentication",
	}

	c.JSON(http.StatusOK, response)
}
