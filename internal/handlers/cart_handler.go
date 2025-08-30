package handlers

import (
	"fmt"
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CartHandler handles cart operations using the cart service
type CartHandler struct {
	cartService services.CartServiceInterface
	logger      *zap.Logger
}

// NewCartHandler creates a new cart handler with cart service
func NewCartHandler(cartService services.CartServiceInterface, logger *zap.Logger) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		logger:      logger,
	}
}

// getUserID extracts user ID from the Firebase authentication context
func (h *CartHandler) getUserID(c *gin.Context) (string, error) {
	// Try to get Firebase UID first (primary authentication method)
	firebaseUID, exists := c.Get("firebase_uid")
	if exists {
		if uidStr, ok := firebaseUID.(string); ok && uidStr != "" {
			return uidStr, nil
		}
	}

	// Fallback to regular user_id (for backward compatibility)
	userID, exists := c.Get("user_id")
	if exists {
		if userIDStr, ok := userID.(string); ok && userIDStr != "" {
			return userIDStr, nil
		}
	}

	h.logger.Error("User ID not found in context - no firebase_uid or user_id present")
	return "", fmt.Errorf("user ID not found in context")
}

// GetCart retrieves the user's cart
func (h *CartHandler) GetCart(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	cart, err := h.cartService.GetUserCart(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user cart", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load cart"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cart,
		"message": "Cart retrieved successfully",
	})
}

// AddToCart adds an item to the user's cart
func (h *CartHandler) AddToCart(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	var req requests.AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate quantity
	if req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity must be positive"})
		return
	}

	cart, err := h.cartService.AddItemToCart(c.Request.Context(), userID, req.ProductID, req.Quantity)
	if err != nil {
		h.logger.Error("Failed to add item to cart",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", req.ProductID),
			zap.Int("quantity", req.Quantity))

		// Check for specific error types
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		if err.Error()[:len("insufficient stock")] == "insufficient stock" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item to cart"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    cart,
		"message": "Item added to cart successfully",
	})
}

// UpdateCartItem updates the quantity of an item in the cart
func (h *CartHandler) UpdateCartItem(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	var req requests.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate quantity
	if req.Quantity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity cannot be negative"})
		return
	}

	cart, err := h.cartService.UpdateCartItem(c.Request.Context(), userID, productID, req.Quantity)
	if err != nil {
		h.logger.Error("Failed to update cart item",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Int("quantity", req.Quantity))

		// Check for specific error types
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		if err.Error()[:len("insufficient stock")] == "insufficient stock" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cart item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cart,
		"message": "Cart item updated successfully",
	})
}

// IncrementCartItem increments the quantity of an item in the cart by a specified amount (default: 1)
func (h *CartHandler) IncrementCartItem(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	var req requests.IncrementCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body provided, default to increment by 1
		req.Amount = 1
	}

	// Default to 1 if amount is 0 or negative
	if req.Amount <= 0 {
		req.Amount = 1
	}

	cart, err := h.cartService.IncrementCartItem(c.Request.Context(), userID, productID, req.Amount)
	if err != nil {
		h.logger.Error("Failed to increment cart item",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Int("amount", req.Amount))

		// Check for specific error types
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		if err.Error() == "item not found in cart" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item not found in cart"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to increment cart item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cart,
		"message": "Cart item incremented successfully",
	})
}

// DecrementCartItem decrements the quantity of an item in the cart by a specified amount (default: 1)
func (h *CartHandler) DecrementCartItem(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	var req requests.DecrementCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// If no body provided, default to decrement by 1
		req.Amount = 1
	}

	// Default to 1 if amount is 0 or negative
	if req.Amount <= 0 {
		req.Amount = 1
	}

	cart, err := h.cartService.DecrementCartItem(c.Request.Context(), userID, productID, req.Amount)
	if err != nil {
		h.logger.Error("Failed to decrement cart item",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Int("amount", req.Amount))

		// Check for specific error types
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
			return
		}
		if err.Error() == "item not found in cart" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Item not found in cart"})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decrement cart item"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cart,
		"message": "Cart item decremented successfully",
	})
}

// RemoveFromCart removes an item from the cart
func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Product ID is required"})
		return
	}

	cart, err := h.cartService.RemoveCartItem(c.Request.Context(), userID, productID)
	if err != nil {
		h.logger.Error("Failed to remove cart item",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID))

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove item from cart"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cart,
		"message": "Item removed from cart successfully",
	})
}

// ClearCart removes all items from the user's cart
func (h *CartHandler) ClearCart(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if err := h.cartService.ClearCart(c.Request.Context(), userID); err != nil {
		h.logger.Error("Failed to clear cart", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear cart"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cart cleared successfully",
	})
}

// GetCartSummary retrieves cart summary (total items, total price)
func (h *CartHandler) GetCartSummary(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	summary, err := h.cartService.GetCartSummary(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get cart summary", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart summary"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
		"message": "Cart summary retrieved successfully",
	})
}

// GetCartItemCount retrieves the total number of items in the cart
func (h *CartHandler) GetCartItemCount(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	cart, err := h.cartService.GetUserCart(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get cart for item count", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart item count"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"item_count": cart.TotalItems},
		"message": "Cart item count retrieved successfully",
	})
}

// ValidateCart validates all items in the cart (stock availability, price changes)
func (h *CartHandler) ValidateCart(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	result, err := h.cartService.ValidateCart(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to validate cart", zap.Error(err), zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to validate cart"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"message": "Cart validated successfully",
	})
}

// HealthCheck checks cart service health
func (h *CartHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"service":   "cart_handler",
			"status":    "healthy",
			"timestamp": gin.H{"unix": gin.H{}},
		},
		"message": "Cart service is running",
	})
}
