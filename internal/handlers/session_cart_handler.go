package handlers

import (
	"math"
	"net/http"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SessionCartHandler handles cart operations for both authenticated and anonymous users
type SessionCartHandler struct {
	cartService        services.CartServiceInterface
	sessionCartService services.SessionCartServiceInterface
	logger             *zap.Logger
	authContext        *auth.AuthContext
}

// NewSessionCartHandler creates a new session cart handler
func NewSessionCartHandler(cartService services.CartServiceInterface, sessionCartService services.SessionCartServiceInterface, logger *zap.Logger) *SessionCartHandler {
	return &SessionCartHandler{
		cartService:        cartService,
		sessionCartService: sessionCartService,
		logger:             logger,
		authContext:        auth.NewAuthContext(),
	}
}

// getCartIdentifier returns either userID (if authenticated) or sessionID (if anonymous)
func (h *SessionCartHandler) getCartIdentifier(c *gin.Context) (string, bool, error) {
	// Try to get authenticated user ID first
	if userID, authenticated := h.authContext.GetAuthenticatedUserID(c); authenticated {
		h.logger.Info("User is authenticated", zap.String("user_id", userID))
		return userID, true, nil
	}

	h.logger.Info("User is not authenticated, using session-based cart")

	// If not authenticated, try to get session ID from header or query param
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = c.Query("session_id")
	}

	// If no session ID provided, generate a new one
	if sessionID == "" {
		sessionID = h.sessionCartService.GenerateSessionID()
		h.logger.Info("Generated new session ID for anonymous user", zap.String("session_id", sessionID))
	}

	// Validate session ID
	if !h.sessionCartService.ValidateSessionID(sessionID) {
		h.logger.Error("Invalid session ID", zap.String("session_id", sessionID))
		return "", false, nil
	}

	return sessionID, false, nil
}

// GetCart retrieves the user's cart (authenticated or session-based)
// @Summary Get cart
// @Description Get the user's shopping cart (works for both authenticated and anonymous users)
// @Tags Cart
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param session_id query string false "Session ID for anonymous users (alternative to header)"
// @Success 200 {object} map[string]interface{} "Cart retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid session ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart [get]
func (h *SessionCartHandler) GetCart(c *gin.Context) {
	cartIdentifier, isAuthenticated, err := h.getCartIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	var cart interface{}
	var cartErr error

	if isAuthenticated {
		// Use regular cart service for authenticated users
		cart, cartErr = h.cartService.GetUserCart(c.Request.Context(), cartIdentifier)
	} else {
		// Use session cart service for anonymous users
		cart, cartErr = h.sessionCartService.GetSessionCart(c.Request.Context(), cartIdentifier)
	}

	if cartErr != nil {
		h.logger.Error("Failed to get cart", zap.Error(cartErr), zap.String("identifier", cartIdentifier))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve cart",
		})
		return
	}

	// Convert cart to response structure
	cartModel, ok := cart.(*models.Cart)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Invalid cart data",
		})
		return
	}
	cartResponse := responses.NewCartResponse(cartModel)

	// Calculate discounted total
	discounted := round2Session(cartResponse.TotalPrice - cartResponse.DiscountAmount)
	if discounted < 0 {
		discounted = 0
	}

	response := gin.H{
		"success": true,
		"data": gin.H{
			"cart":             cartResponse,
			"discount_amount":  round2Session(cartResponse.DiscountAmount),
			"discounted_total": discounted,
			"is_authenticated": isAuthenticated,
		},
		"message": "Cart retrieved successfully",
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["data"].(gin.H)["session_id"] = cartIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// AddToCart adds an item to the user's cart (authenticated or session-based)
// @Summary Add item to cart
// @Description Add a product to the user's shopping cart (works for both authenticated and anonymous users)
// @Tags Cart
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param request body requests.AddToCartRequest true "Cart item information"
// @Success 200 {object} map[string]interface{} "Item added to cart successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/items [post]
func (h *SessionCartHandler) AddToCart(c *gin.Context) {
	cartIdentifier, isAuthenticated, err := h.getCartIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
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

	var cart interface{}
	var cartErr error

	if isAuthenticated {
		// Use regular cart service for authenticated users
		cart, cartErr = h.cartService.AddItemToCart(c.Request.Context(), cartIdentifier, req.ProductID, req.Quantity)
	} else {
		// Use session cart service for anonymous users
		cart, cartErr = h.sessionCartService.AddItemToSessionCart(c.Request.Context(), cartIdentifier, req.ProductID, req.Quantity)
	}

	if cartErr != nil {
		h.logger.Error("Failed to add item to cart",
			zap.Error(cartErr),
			zap.String("identifier", cartIdentifier),
			zap.String("product_id", req.ProductID),
			zap.Int("quantity", req.Quantity))

		// Check if it's a stock error
		if strings.Contains(cartErr.Error(), "insufficient stock") {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   cartErr.Error(),
			})
			return
		}

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add item to cart",
		})
		return
	}

	// Convert cart to response structure
	cartModel, ok := cart.(*models.Cart)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Invalid cart data",
		})
		return
	}
	cartResponse := responses.NewCartResponse(cartModel)

	response := gin.H{
		"success": true,
		"data": gin.H{
			"cart": cartResponse,
		},
		"message": "Item added to cart successfully",
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["data"].(gin.H)["session_id"] = cartIdentifier
	}

	c.JSON(http.StatusCreated, response)
}

// UpdateCartItem updates the quantity of an item in the cart
// @Summary Update cart item
// @Description Update quantity for a specific product in the cart (works for both authenticated and anonymous users)
// @Tags Cart
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param product_id path string true "Product ID"
// @Param request body requests.UpdateCartItemRequest true "Update payload"
// @Success 200 {object} map[string]interface{} "Cart item updated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/items/{product_id} [put]
func (h *SessionCartHandler) UpdateCartItem(c *gin.Context) {
	cartIdentifier, isAuthenticated, err := h.getCartIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
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

	var cart interface{}
	var cartErr error

	if isAuthenticated {
		// Use regular cart service for authenticated users
		cart, cartErr = h.cartService.UpdateCartItem(c.Request.Context(), cartIdentifier, productID, req.Quantity)
	} else {
		// Use session cart service for anonymous users
		cart, cartErr = h.sessionCartService.UpdateSessionCartItem(c.Request.Context(), cartIdentifier, productID, req.Quantity)
	}

	if cartErr != nil {
		h.logger.Error("Failed to update cart item",
			zap.Error(cartErr),
			zap.String("identifier", cartIdentifier),
			zap.String("product_id", productID),
			zap.Int("quantity", req.Quantity))

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to update cart item",
		})
		return
	}

	// Convert cart to response structure
	cartModel, ok := cart.(*models.Cart)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Invalid cart data",
		})
		return
	}
	cartResponse := responses.NewCartResponse(cartModel)

	response := gin.H{
		"success": true,
		"data": gin.H{
			"cart": cartResponse,
		},
		"message": "Cart item updated successfully",
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["data"].(gin.H)["session_id"] = cartIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// RemoveFromCart removes an item from the cart
// @Summary Remove cart item
// @Description Remove a specific product from the cart (works for both authenticated and anonymous users)
// @Tags Cart
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param product_id path string true "Product ID"
// @Success 200 {object} map[string]interface{} "Item removed from cart"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/items/{product_id} [delete]
func (h *SessionCartHandler) RemoveFromCart(c *gin.Context) {
	cartIdentifier, isAuthenticated, err := h.getCartIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
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

	var cart interface{}
	var cartErr error

	if isAuthenticated {
		// Use regular cart service for authenticated users
		cart, cartErr = h.cartService.RemoveCartItem(c.Request.Context(), cartIdentifier, productID)
	} else {
		// Use session cart service for anonymous users
		cart, cartErr = h.sessionCartService.RemoveSessionCartItem(c.Request.Context(), cartIdentifier, productID)
	}

	if cartErr != nil {
		h.logger.Error("Failed to remove cart item",
			zap.Error(cartErr),
			zap.String("identifier", cartIdentifier),
			zap.String("product_id", productID))

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to remove cart item",
		})
		return
	}

	// Convert cart to response structure
	cartModel, ok := cart.(*models.Cart)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Invalid cart data",
		})
		return
	}
	cartResponse := responses.NewCartResponse(cartModel)

	response := gin.H{
		"success": true,
		"data": gin.H{
			"cart": cartResponse,
		},
		"message": "Item removed from cart successfully",
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["data"].(gin.H)["session_id"] = cartIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// ClearCart removes all items from the cart
// @Summary Clear cart
// @Description Remove all items from the cart (works for both authenticated and anonymous users)
// @Tags Cart
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Success 200 {object} map[string]interface{} "Cart cleared"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/clear [delete]
func (h *SessionCartHandler) ClearCart(c *gin.Context) {
	cartIdentifier, isAuthenticated, err := h.getCartIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	var cartErr error

	if isAuthenticated {
		// Use regular cart service for authenticated users
		cartErr = h.cartService.ClearCart(c.Request.Context(), cartIdentifier)
	} else {
		// Use session cart service for anonymous users
		cartErr = h.sessionCartService.ClearSessionCart(c.Request.Context(), cartIdentifier)
	}

	if cartErr != nil {
		h.logger.Error("Failed to clear cart",
			zap.Error(cartErr),
			zap.String("identifier", cartIdentifier))

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to clear cart",
		})
		return
	}

	response := gin.H{
		"success": true,
		"message": "Cart cleared successfully",
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["session_id"] = cartIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// GetCartSummary retrieves the cart summary
// @Summary Get cart summary
// @Description Get a summary of the cart (works for both authenticated and anonymous users)
// @Tags Cart
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Success 200 {object} map[string]interface{} "Cart summary retrieved"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/summary [get]
func (h *SessionCartHandler) GetCartSummary(c *gin.Context) {
	cartIdentifier, isAuthenticated, err := h.getCartIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	var summary *models.CartSummary
	var summaryErr error

	if isAuthenticated {
		// Use regular cart service for authenticated users
		summary, summaryErr = h.cartService.GetCartSummary(c.Request.Context(), cartIdentifier)
	} else {
		// Use session cart service for anonymous users
		summary, summaryErr = h.sessionCartService.GetSessionCartSummary(c.Request.Context(), cartIdentifier)
	}

	if summaryErr != nil {
		h.logger.Error("Failed to get cart summary",
			zap.Error(summaryErr),
			zap.String("identifier", cartIdentifier))

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to get cart summary",
		})
		return
	}

	response := gin.H{
		"success": true,
		"data":    summary,
		"message": "Cart summary retrieved successfully",
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["session_id"] = cartIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// MigrateCart migrates a session cart to a user cart (requires authentication)
// @Summary Migrate session cart to user cart
// @Description Migrate anonymous session cart to authenticated user cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param X-Session-ID header string true "Session ID to migrate"
// @Success 200 {object} map[string]interface{} "Cart migrated successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 400 {object} map[string]interface{} "Invalid session ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/migrate [post]
func (h *SessionCartHandler) MigrateCart(c *gin.Context) {
	// This endpoint requires authentication
	userID, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required for cart migration",
		})
		return
	}

	// Get session ID from header
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID is required",
		})
		return
	}

	// Validate session ID
	if !h.sessionCartService.ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	// Migrate session cart to user cart
	cart, err := h.sessionCartService.MigrateSessionCartToUser(c.Request.Context(), sessionID, userID)
	if err != nil {
		h.logger.Error("Failed to migrate session cart",
			zap.Error(err),
			zap.String("session_id", sessionID),
			zap.String("user_id", userID))

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to migrate cart",
		})
		return
	}

	// Convert cart to response structure
	cartResponse := responses.NewCartResponse(cart)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cartResponse,
		"message": "Cart migrated successfully",
	})
}

// MergeCart merges a session cart with user's existing cart (requires authentication)
// @Summary Merge session cart with user cart
// @Description Merge anonymous session cart with authenticated user's existing cart
// @Tags Cart
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param X-Session-ID header string true "Session ID to merge"
// @Success 200 {object} map[string]interface{} "Cart merged successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 400 {object} map[string]interface{} "Invalid session ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /cart/merge [post]
func (h *SessionCartHandler) MergeCart(c *gin.Context) {
	// This endpoint requires authentication
	userID, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "Authentication required for cart merge",
		})
		return
	}

	// Get session ID from header
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID is required",
		})
		return
	}

	// Validate session ID
	if !h.sessionCartService.ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	// Merge session cart with user cart
	cart, err := h.sessionCartService.MergeSessionCartWithUserCart(c.Request.Context(), sessionID, userID)
	if err != nil {
		h.logger.Error("Failed to merge session cart",
			zap.Error(err),
			zap.String("session_id", sessionID),
			zap.String("user_id", userID))

		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to merge cart",
		})
		return
	}

	// Convert cart to response structure
	cartResponse := responses.NewCartResponse(cart)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    cartResponse,
		"message": "Cart merged successfully",
	})
}

// round2Session rounds a float64 to 2 decimal places
func round2Session(f float64) float64 {
	return math.Round(f*100) / 100
}
