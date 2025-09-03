package handlers

import (
	"net/http"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// OrderHandler handles order-related HTTP requests
type OrderHandler struct {
	orderService services.OrderServiceInterface
	logger       *zap.Logger
}

// NewOrderHandler creates a new order handler
func NewOrderHandler(orderService services.OrderServiceInterface, logger *zap.Logger) *OrderHandler {
	return &OrderHandler{
		orderService: orderService,
		logger:       logger,
	}
}

// CreateOrder creates a new order manually (alternative to payment flow)
// POST /api/v1/orders
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		CartID          string `json:"cart_id" binding:"required"`
		ShippingAddress string `json:"shipping_address" binding:"required"`
		PaymentMethod   string `json:"payment_method" binding:"required"`
		Notes           string `json:"notes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order, err := h.orderService.CreateOrderFromCart(
		c.Request.Context(),
		userID.(string),
		req.CartID,
		req.ShippingAddress,
		req.PaymentMethod,
	)

	if err != nil {
		h.logger.Error("Failed to create order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	h.logger.Info("Order created via direct API",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID.(string)))

	c.JSON(http.StatusCreated, gin.H{
		"message": "Order created successfully",
		"order":   order,
	})
}

// GetUserOrders retrieves all orders for the authenticated user
// GET /api/v1/orders
func (h *OrderHandler) GetUserOrders(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
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

	orders, err := h.orderService.GetUserOrders(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.Error("Failed to get user orders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve orders"})
		return
	}

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
func (h *OrderHandler) GetOrderDetails(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	orderID := c.Param("id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order ID is required"})
		return
	}

	// Get order with items
	order, orderItems, err := h.orderService.GetOrderWithItems(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to get order details", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve order details"})
		return
	}

	// Verify the order belongs to the authenticated user
	if order.UserID != userID.(string) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"order": order,
		"items": orderItems,
	})
}

// GetOrderStatus retrieves the current status of an order
// GET /api/v1/orders/:id/status
func (h *OrderHandler) GetOrderStatus(c *gin.Context) {
	userID, exists := c.Get("user_id")
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
	if order.UserID != userID.(string) {
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
func (h *OrderHandler) GetOrderTracking(c *gin.Context) {
	userID, exists := c.Get("user_id")
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
	if order.UserID != userID.(string) {
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
func (h *OrderHandler) GetAllOrders(c *gin.Context) {
	// TODO: Implement admin-only order retrieval
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Admin functionality not yet implemented"})
}

// UpdateOrderStatus updates order status (admin only)
func (h *OrderHandler) UpdateOrderStatus(c *gin.Context) {
	// TODO: Implement admin order status updates
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Admin functionality not yet implemented"})
}

// GetOrderStats retrieves order statistics (admin only)
func (h *OrderHandler) GetOrderStats(c *gin.Context) {
	// TODO: Implement order statistics
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Admin functionality not yet implemented"})
}
