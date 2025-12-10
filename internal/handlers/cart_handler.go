package handlers

import (
	"fmt"
	"math"
	"net/http"
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
) // CartHandler handles cart operations using the cart service
type CartHandler struct {
	cartService services.CartServiceInterface
	userService *services.UserService
	logger      *zap.Logger
	authContext *auth.AuthContext
	deliverySvc services.DeliveryServiceInterface
}

// Coupon represents a simple coupon configuration for runtime validation
func round2(f float64) float64 {
	return math.Round(f*100) / 100
}

// NewCartHandler creates a new cart handler with cart service
func NewCartHandler(cartService services.CartServiceInterface, userService *services.UserService, logger *zap.Logger) *CartHandler {
	return &CartHandler{
		cartService: cartService,
		userService: userService,
		logger:      logger,
		authContext: auth.NewAuthContext(),
	}
}

// SetDeliveryService allows wiring the delivery service for checkout flows
func (h *CartHandler) SetDeliveryService(svc services.DeliveryServiceInterface) {
	h.deliverySvc = svc
}

// authenticateUser is a helper method to get authenticated user ID
// It maps Firebase UIDs to internal user IDs, auto-creating users if needed
func (h *CartHandler) authenticateUser(c *gin.Context) (string, bool) {
	userID, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		return "", false
	}

	// Check if this is a Firebase UID (they typically don't start with "USER")
	// Internal user IDs start with "USER" prefix
	if !strings.HasPrefix(userID, "USER") {
		// This is likely a Firebase UID, map it to internal user ID
		if h.userService != nil {
			userResponse, err := h.userService.GetUserByFirebaseUID(c.Request.Context(), userID)
			if err == nil && userResponse != nil && userResponse.Data != nil {
				// Extract the internal user ID from the response
				if userMap, ok := userResponse.Data.(map[string]interface{}); ok {
					if internalID, ok := userMap["id"].(string); ok {
						return internalID, true
					}
				}
			}

			// User doesn't exist in local database - auto-create from Firebase
			h.logger.Info("User not found in local database, attempting to auto-create from Firebase",
				zap.String("firebase_uid", userID))

			firebaseClient := config.GetFirebaseAuth()
			if firebaseClient == nil {
				h.logger.Error("Firebase client not initialized, cannot auto-create user",
					zap.String("firebase_uid", userID))
				return userID, true // Return Firebase UID as fallback
			}

			firebaseUser, firebaseErr := firebaseClient.GetUser(c.Request.Context(), userID)
			if firebaseErr != nil {
				h.logger.Error("Failed to get user from Firebase",
					zap.String("firebase_uid", userID),
					zap.Error(firebaseErr))
				return userID, true // Return Firebase UID as fallback
			}

			// Try to find existing user by email or phone
			var existingUser map[string]interface{}
			if firebaseUser.Email != "" {
				emailResponse, _ := h.userService.GetUserByEmail(c.Request.Context(), firebaseUser.Email)
				if emailResponse != nil && emailResponse.Success && emailResponse.Data != nil {
					existingUser, _ = emailResponse.Data.(map[string]interface{})
				}
			}
			if existingUser == nil && firebaseUser.PhoneNumber != "" {
				phoneResponse, _ := h.userService.GetUserByPhoneNumber(c.Request.Context(), firebaseUser.PhoneNumber)
				if phoneResponse != nil && phoneResponse.Success && phoneResponse.Data != nil {
					existingUser, _ = phoneResponse.Data.(map[string]interface{})
				}
			}

			if existingUser != nil {
				// User exists, link Firebase UID
				internalID := existingUser["id"].(string)
				h.userService.UpdateUserFirebaseUID(c.Request.Context(), internalID, userID)
				h.logger.Info("Linked Firebase UID to existing user",
					zap.String("firebase_uid", userID),
					zap.String("internal_id", internalID))
				return internalID, true
			}

			// Create new user
			displayName := firebaseUser.DisplayName
			if displayName == "" {
				displayName = "User"
			}

			if firebaseUser.PhoneNumber != "" && firebaseUser.Email == "" {
				// Phone user
				phoneReq := requests.PhoneSignupRequest{
					PhoneNumber: firebaseUser.PhoneNumber,
					Name:        displayName,
				}
				createResponse, createErr := h.userService.CreatePhoneUser(c.Request.Context(), &phoneReq)
				if createErr == nil && createResponse != nil && createResponse.Success {
					if userData, ok := createResponse.Data.(map[string]interface{}); ok {
						if id, ok := userData["id"].(string); ok {
							h.userService.UpdateUserFirebaseUID(c.Request.Context(), id, userID)
							h.logger.Info("Auto-created phone user",
								zap.String("firebase_uid", userID),
								zap.String("internal_id", id))
							return id, true
						}
					}
				}
			} else {
				// Email/Google user
				signupReq := requests.SignupRequest{
					Email:    firebaseUser.Email,
					Name:     displayName,
					Password: "",
				}
				createResponse, createErr := h.userService.CreateGoogleUser(c.Request.Context(), &signupReq, userID)
				if createErr == nil && createResponse != nil && createResponse.Success {
					if userData, ok := createResponse.Data.(map[string]interface{}); ok {
						if id, ok := userData["id"].(string); ok {
							h.logger.Info("Auto-created Google user",
								zap.String("firebase_uid", userID),
								zap.String("internal_id", id))
							return id, true
						}
					}
				}
			}

			h.logger.Error("Failed to auto-create user in local database",
				zap.String("firebase_uid", userID))
		}
	}

	return userID, true
}

// handleCartServiceError handles common cart service errors
func (h *CartHandler) handleCartServiceError(c *gin.Context, err error, operation string, userID string, additionalFields ...zap.Field) {
	// Prepare log fields
	logFields := []zap.Field{
		zap.Error(err),
		zap.String("user_id", userID),
		zap.String("operation", operation),
	}
	logFields = append(logFields, additionalFields...)

	h.logger.Error("Cart operation failed", logFields...)

	errMsg := err.Error()

	// Handle specific error types
	switch {
	case errMsg == "product not found":
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Product not found",
		})
	case errMsg == "item not found in cart":
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "Item not found in cart",
		})
	case strings.HasPrefix(errMsg, "insufficient stock"):
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   errMsg,
		})
	case errMsg == "invalid coupon code":
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid coupon code",
		})
	case errMsg == "cart is empty":
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Cart is empty. Add items to cart before applying coupon",
		})
	case errMsg == "coupon cannot be used for this cart":
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Coupon cannot be used for this cart. Check minimum order amount or usage limits",
		})
	case strings.HasPrefix(errMsg, "failed to update cart with coupon"):
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to apply coupon to cart",
		})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to " + operation + ": " + errMsg,
		})
	}
}

// GetCart retrieves the user's cart
// GetCart handles GET /api/v1/cart
// @Summary Get user's cart
// @Description Get the current user's shopping cart with all items
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Cart retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart [get]
func (h *CartHandler) GetCart(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	cart, err := h.cartService.GetUserCart(c.Request.Context(), userID)
	if err != nil {
		h.handleCartServiceError(c, err, "retrieve cart", userID)
		return
	}

	// Convert cart model to response structure (excludes coupon_code for security)
	cartResponse := responses.NewCartResponse(cart)

	discounted := round2(cart.TotalPrice - cart.DiscountAmount)
	if discounted < 0 {
		discounted = 0
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cart":             cartResponse,
			"discount_amount":  round2(cart.DiscountAmount),
			"discounted_total": discounted,
		},
		"message": "Cart retrieved successfully",
	})
}

// AddToCart adds an item to the user's cart
// AddToCart handles POST /api/v1/cart/items
// @Summary Add item to cart
// @Description Add a product to the user's shopping cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.AddToCartRequest true "Cart item information"
// @Success 200 {object} map[string]interface{} "Item added to cart successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/items [post]
func (h *CartHandler) AddToCart(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	var req requests.AddToCartRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate quantity
	if req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Quantity must be positive",
		})
		return
	}

	cart, err := h.cartService.AddItemToCart(c.Request.Context(), userID, req.ProductID, req.Quantity)
	if err != nil {
		h.handleCartServiceError(c, err, "add item to cart", userID,
			zap.String("product_id", req.ProductID),
			zap.Int("quantity", req.Quantity))
		return
	}

	// Convert cart model to response structure (excludes coupon_code for security)
	cartResponse := responses.NewCartResponse(cart)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    cartResponse,
		"message": "Item added to cart successfully",
	})
}

// UpdateCartItem updates the quantity of an item in the cart
// @Summary Update cart item
// @Description Update quantity for a specific product in the cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param product_id path string true "Product ID"
// @Param request body requests.UpdateCartItemRequest true "Update payload"
// @Success 200 {object} map[string]interface{} "Cart item updated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/items/{product_id} [put]
func (h *CartHandler) UpdateCartItem(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Product ID is required",
		})
		return
	}

	var req requests.UpdateCartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format: " + err.Error(),
		})
		return
	}

	// Validate quantity
	if req.Quantity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Quantity cannot be negative",
		})
		return
	}

	cart, err := h.cartService.UpdateCartItem(c.Request.Context(), userID, productID, req.Quantity)
	if err != nil {
		h.handleCartServiceError(c, err, "update cart item", userID,
			zap.String("product_id", productID),
			zap.Int("quantity", req.Quantity))
		return
	}

	// Convert cart model to response structure (excludes coupon_code for security)
	cartResponse := responses.NewCartResponse(cart)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cartResponse,
		"message": "Cart item updated successfully",
	})
}

// IncrementCartItem increments the quantity of an item in the cart by a specified amount (default: 1)
// @Summary Increment cart item
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param product_id path string true "Product ID"
// @Param request body requests.IncrementCartItemRequest true "Increment payload"
// @Success 200 {object} map[string]interface{} "Item incremented"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/items/{product_id}/increment [post]
func (h *CartHandler) IncrementCartItem(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Product ID is required",
		})
		return
	}

	var req requests.IncrementCartItemRequest
	// Default to increment by 1 if no body provided or on error
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		req.Amount = 1
	}

	cart, err := h.cartService.IncrementCartItem(c.Request.Context(), userID, productID, req.Amount)
	if err != nil {
		h.handleCartServiceError(c, err, "increment cart item", userID,
			zap.String("product_id", productID),
			zap.Int("amount", req.Amount))
		return
	}

	// Convert cart model to response structure (excludes coupon_code for security)
	cartResponse := responses.NewCartResponse(cart)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cartResponse,
		"message": "Cart item incremented successfully",
	})
}

// DecrementCartItem decrements the quantity of an item in the cart by a specified amount (default: 1)
// @Summary Decrement cart item
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param product_id path string true "Product ID"
// @Param request body requests.DecrementCartItemRequest true "Decrement payload"
// @Success 200 {object} map[string]interface{} "Item decremented"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/items/{product_id}/decrement [post]
func (h *CartHandler) DecrementCartItem(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Product ID is required",
		})
		return
	}

	var req requests.DecrementCartItemRequest
	// Default to decrement by 1 if no body provided or on error
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount <= 0 {
		req.Amount = 1
	}

	cart, err := h.cartService.DecrementCartItem(c.Request.Context(), userID, productID, req.Amount)
	if err != nil {
		h.handleCartServiceError(c, err, "decrement cart item", userID,
			zap.String("product_id", productID),
			zap.Int("amount", req.Amount))
		return
	}

	// Convert cart model to response structure (excludes coupon_code for security)
	cartResponse := responses.NewCartResponse(cart)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cartResponse,
		"message": "Cart item decremented successfully",
	})
}

// RemoveFromCart removes an item from the cart
// @Summary Remove item from cart
// @Tags Cart
// @Produce json
// @Security FirebaseAuth
// @Param product_id path string true "Product ID"
// @Success 200 {object} map[string]interface{} "Item removed"
// @Failure 400 {object} map[string]interface{} "Product ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/items/{product_id} [delete]
func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	productID := c.Param("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Product ID is required",
		})
		return
	}

	cart, err := h.cartService.RemoveCartItem(c.Request.Context(), userID, productID)
	if err != nil {
		h.handleCartServiceError(c, err, "remove cart item", userID,
			zap.String("product_id", productID))
		return
	}

	// Convert cart model to response structure (excludes coupon_code for security)
	cartResponse := responses.NewCartResponse(cart)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cartResponse,
		"message": "Item removed from cart successfully",
	})
}

// ClearCart removes all items from the user's cart
// @Summary Clear cart
// @Tags Cart
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Cart cleared"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/clear [delete]
func (h *CartHandler) ClearCart(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	if err := h.cartService.ClearCart(c.Request.Context(), userID); err != nil {
		h.handleCartServiceError(c, err, "clear cart", userID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Cart cleared successfully",
	})
}

// GetCartSummary retrieves cart summary (total items, total price)
// @Summary Cart summary
// @Tags Cart
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Cart summary"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/summary [get]
func (h *CartHandler) GetCartSummary(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	summary, err := h.cartService.GetCartSummary(c.Request.Context(), userID)
	if err != nil {
		h.handleCartServiceError(c, err, "get cart summary", userID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    summary,
		"message": "Cart summary retrieved successfully",
	})
}

// GetCartItemCount retrieves the total number of items in the cart
// @Summary Cart item count
// @Tags Cart
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Item count"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/count [get]
func (h *CartHandler) GetCartItemCount(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	cart, err := h.cartService.GetUserCart(c.Request.Context(), userID)
	if err != nil {
		h.handleCartServiceError(c, err, "get cart item count", userID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"item_count": cart.TotalItems},
		"message": "Cart item count retrieved successfully",
	})
}

// ValidateCart validates all items in the cart (stock availability, price changes)
// @Summary Validate cart
// @Tags Cart
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Cart validated"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/validate [get]
func (h *CartHandler) ValidateCart(c *gin.Context) {
	userID, authenticated := h.authenticateUser(c)
	if !authenticated {
		return
	}

	result, err := h.cartService.ValidateCart(c.Request.Context(), userID)
	if err != nil {
		h.handleCartServiceError(c, err, "validate cart", userID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"message": "Cart validated successfully",
	})
}

// HealthCheck checks cart service health
// @Summary Cart service health
// @Tags Cart
// @Produce json
// @Success 200 {object} map[string]interface{} "Service health"
// @Router /cart/health [get]
func (h *CartHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"service":   "cart_handler",
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		},
		"message": "Cart service is running",
	})
}

// CleanupDuplicateCarts is an admin endpoint to clean up duplicate carts
// @Summary Admin: cleanup duplicate carts
// @Tags Admin - Cart
// @Produce json
// @Security FirebaseAuth
// @Router /cart/admin/cleanup-duplicates [post]
func (h *CartHandler) CleanupDuplicateCarts(c *gin.Context) {
	h.logger.Info("Admin initiated cart cleanup request")

	ctx := c.Request.Context()

	// Call the service to cleanup duplicates
	if err := h.cartService.CleanupDuplicateCarts(ctx); err != nil {
		h.logger.Error("Cart cleanup failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to cleanup duplicate carts",
			"details": err.Error(),
		})
		return
	}

	h.logger.Info("Cart cleanup completed successfully")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Duplicate carts cleaned up successfully",
		"data": gin.H{
			"operation": "cleanup_duplicate_carts",
			"timestamp": time.Now().Unix(),
		},
	})
}

// PrepareCheckout validates the cart and returns data ready for checkout
// @Summary Prepare checkout
// @Tags Cart
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Checkout prepared"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/checkout/prepare [get]
func (h *CartHandler) PrepareCheckout(c *gin.Context) {
	userID, ok := h.authenticateUser(c)
	if !ok {
		return
	}

	result, err := h.cartService.ValidateCart(c.Request.Context(), userID)
	if err != nil {
		h.handleCartServiceError(c, err, "prepare checkout", userID)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
		"message": "Checkout data prepared",
	})
}

// EstimateShipping returns shipping options via delivery service
// @Summary Estimate shipping (cart)
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.EstimateShippingRequest true "Address"
// @Success 200 {object} map[string]interface{} "Shipping estimated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/checkout/estimate-shipping [post]
func (h *CartHandler) EstimateShipping(c *gin.Context) {
	userID, ok := h.authenticateUser(c)
	if !ok {
		return
	}

	h.logger.Info("=== SHIPPING ESTIMATION REQUEST ===",
		zap.String("user_id", userID),
		zap.String("flow_step", "estimate_shipping_start"))

	var req requests.EstimateShippingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid shipping estimation request",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("flow_step", "request_binding_failed"))
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}

	h.logger.Info("Shipping estimation request details",
		zap.String("user_id", userID),
		zap.String("street", req.Address.Street),
		zap.String("city", req.Address.City),
		zap.String("state", req.Address.State),
		zap.String("postal_code", req.Address.PostalCode),
		zap.String("country", req.Address.Country),
		zap.String("shipping_method", req.ShippingMethod),
		zap.String("flow_step", "request_details"))

	if h.deliverySvc == nil {
		h.logger.Error("Delivery service not configured",
			zap.String("user_id", userID),
			zap.String("flow_step", "delivery_service_missing"))
		c.JSON(http.StatusNotImplemented, gin.H{"success": false, "error": "Delivery service not configured"})
		return
	}

	est, err := h.deliverySvc.Estimate(c.Request.Context(), userID, req)
	if err != nil {
		h.logger.Error("Estimate shipping failed",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("postal_code", req.Address.PostalCode),
			zap.String("flow_step", "delivery_estimation_failed"))
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Failed to estimate shipping", "details": err.Error()})
		return
	}

	h.logger.Info("Shipping estimation successful",
		zap.String("user_id", userID),
		zap.String("address_id", est.AddressID),
		zap.Float64("subtotal", est.Subtotal),
		zap.Int("options_count", len(est.Options)),
		zap.String("flow_step", "estimation_success"))

	// Transform the DeliveryEstimate response to match expected format
	response := gin.H{
		"success": true,
		"data": gin.H{
			"shipping_options": transformDeliveryOptions(est.Options),
			"subtotal":         est.Subtotal,
			"user_id":          est.UserID,
			"address_id":       est.AddressID, // Include the address_id in response
		},
		"message": "Shipping estimated successfully",
	}

	c.JSON(http.StatusOK, response)
}

// transformDeliveryOptions converts DeliveryOption slice to the expected format
func transformDeliveryOptions(options []models.DeliveryOption) []gin.H {
	result := make([]gin.H, len(options))
	for i, opt := range options {
		result[i] = gin.H{
			"method":            string(opt.Method),
			"carrier":           opt.Carrier,
			"service_level":     opt.ServiceLevel,
			"cost":              opt.Price,
			"currency":          opt.Currency,
			"delivery_days_min": opt.EtaDaysMin,
			"delivery_days_max": opt.EtaDaysMax,
			"description":       fmt.Sprintf("%s %s (%d-%d business days)", opt.Carrier, opt.ServiceLevel, opt.EtaDaysMin, opt.EtaDaysMax),
		}
	}
	return result
}

// ApplyCoupon validates and applies a coupon logically (no persistence)
// @Summary Apply coupon
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.ApplyCouponRequest true "Coupon request"
// @Success 200 {object} map[string]interface{} "Coupon applied"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/checkout/apply-coupon [post]
func (h *CartHandler) ApplyCoupon(c *gin.Context) {
	userID, ok := h.authenticateUser(c)
	if !ok {
		return
	}

	var req requests.ApplyCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Invalid request: " + err.Error()})
		return
	}
	req.Sanitize()

	cart, discount, err := h.cartService.ApplyCoupon(c.Request.Context(), userID, req.CouponCode)
	if err != nil {
		h.handleCartServiceError(c, err, "apply coupon", userID)
		return
	}
	total := round2(cart.TotalPrice)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cart_total":       total,
			"discount_amount":  discount,
			"discounted_total": round2(total - discount),
		},
		"message": "Coupon applied",
	})
}

// RemoveCoupon returns the original cart totals (since coupons aren't persisted)
// @Summary Remove coupon
// @Tags Cart
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Coupon removed"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/checkout/remove-coupon [delete]
func (h *CartHandler) RemoveCoupon(c *gin.Context) {
	userID, ok := h.authenticateUser(c)
	if !ok {
		return
	}

	cart, err := h.cartService.RemoveCoupon(c.Request.Context(), userID)
	if err != nil {
		h.handleCartServiceError(c, err, "remove coupon", userID)
		return
	}

	// Get the original total price (without any discount)
	originalTotal := round2(cart.TotalPrice)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"cart_total":       originalTotal,
			"discount_amount":  0,
			"discounted_total": originalTotal,
			"coupon":           nil,
		},
		"message": "Coupon removed successfully",
	})
}
