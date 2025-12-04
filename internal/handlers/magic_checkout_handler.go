package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Global cache for Magic Checkout data
var (
	magicCheckoutAddressCache = make(map[string]map[string]interface{})
	magicCheckoutOrderCache   = make(map[string]MagicCheckoutOrderData) // receipt/order_id -> order data
	magicCheckoutCacheMutex   sync.RWMutex
)

// MagicCheckoutOrderData stores order information for coupon calculation
type MagicCheckoutOrderData struct {
	RazorpayOrderID string  // The actual Razorpay order ID (order_xxx)
	ReceiptID       string  // Our receipt ID (mc_xxx_timestamp)
	CartTotal       float64 // Cart total in rupees
	UserID          string  // User ID
}

// StoreMagicCheckoutOrder stores order data for later retrieval during apply-promotion
func StoreMagicCheckoutOrder(receiptID string, razorpayOrderID string, cartTotal float64, userID string) {
	magicCheckoutCacheMutex.Lock()
	defer magicCheckoutCacheMutex.Unlock()
	data := MagicCheckoutOrderData{
		RazorpayOrderID: razorpayOrderID,
		ReceiptID:       receiptID,
		CartTotal:       cartTotal,
		UserID:          userID,
	}
	// Store by both receipt ID and Razorpay order ID for flexible lookup
	magicCheckoutOrderCache[receiptID] = data
	magicCheckoutOrderCache[razorpayOrderID] = data
}

// GetMagicCheckoutOrder retrieves stored order data by receipt ID or Razorpay order ID
func GetMagicCheckoutOrder(orderID string) (MagicCheckoutOrderData, bool) {
	magicCheckoutCacheMutex.RLock()
	defer magicCheckoutCacheMutex.RUnlock()
	data, exists := magicCheckoutOrderCache[orderID]
	return data, exists
}

// StoreMagicCheckoutAddress stores address data temporarily
func StoreMagicCheckoutAddress(razorpayOrderID string, addressData map[string]interface{}) {
	magicCheckoutCacheMutex.Lock()
	defer magicCheckoutCacheMutex.Unlock()
	magicCheckoutAddressCache[razorpayOrderID] = addressData
}

// GetMagicCheckoutAddress retrieves stored address data
func GetMagicCheckoutAddress(razorpayOrderID string) (map[string]interface{}, bool) {
	magicCheckoutCacheMutex.RLock()
	defer magicCheckoutCacheMutex.RUnlock()
	data, exists := magicCheckoutAddressCache[razorpayOrderID]
	return data, exists
}

// MagicCheckoutHandler handles Magic Checkout API endpoints
type MagicCheckoutHandler struct {
	paymentService  services.PaymentServiceInterface
	couponService   services.CouponServiceInterface
	deliveryService services.DeliveryServiceInterface
	cartService     services.CartServiceInterface
	orderService    services.OrderServiceInterface
	addressService  services.AddressServiceInterface
	logger          *zap.Logger
}

// NewMagicCheckoutHandler creates a new Magic Checkout handler
func NewMagicCheckoutHandler(
	paymentService services.PaymentServiceInterface,
	couponService services.CouponServiceInterface,
	deliveryService services.DeliveryServiceInterface,
	cartService services.CartServiceInterface,
	orderService services.OrderServiceInterface,
	addressService services.AddressServiceInterface,
	logger *zap.Logger,
) *MagicCheckoutHandler {
	return &MagicCheckoutHandler{
		paymentService:  paymentService,
		couponService:   couponService,
		deliveryService: deliveryService,
		cartService:     cartService,
		orderService:    orderService,
		addressService:  addressService,
		logger:          logger,
	}
}

// GetPromotions returns available coupons for Magic Checkout
// POST /api/v1/magic-checkout/get-promotions
func (h *MagicCheckoutHandler) GetPromotions(c *gin.Context) {
	var req struct {
		OrderID         string  `json:"order_id"`
		RazorpayOrderID string  `json:"razorpay_order_id"`
		Contact         string  `json:"contact"`
		Email           string  `json:"email"`
		Amount          float64 `json:"amount"` // Cart amount in paise
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid get promotions request", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"promotions": []interface{}{}})
		return
	}

	h.logger.Info("Magic Checkout requesting promotions",
		zap.String("order_id", req.OrderID),
		zap.String("razorpay_order_id", req.RazorpayOrderID),
		zap.String("contact", req.Contact),
		zap.Float64("amount", req.Amount))

	// Get active coupons with rules
	activeCouponsResp, err := h.couponService.GetActiveCoupons(c.Request.Context(), 50, 0)
	if err != nil {
		h.logger.Error("Failed to get active coupons", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"promotions": []interface{}{}})
		return
	}

	// Razorpay get-promotions only needs: code, summary, description
	promotions := []map[string]interface{}{}

	// Extract coupons from response
	var couponsData []interface{}
	if responseData, ok := activeCouponsResp.Data.(map[string]interface{}); ok {
		if coupons, ok := responseData["coupons"].([]interface{}); ok {
			couponsData = coupons
		}
	}

	for _, couponData := range couponsData {
		couponMap, ok := couponData.(map[string]interface{})
		if !ok {
			continue
		}

		code, _ := couponMap["code"].(string)
		if code == "" {
			continue
		}

		description, _ := couponMap["description"].(string)
		if description == "" {
			description = fmt.Sprintf("Apply code %s for discount", code)
		}

		// Create simple promotion for get-promotions (Razorpay only needs these 3 fields)
		promotion := map[string]interface{}{
			"code":        code,
			"summary":     description,
			"description": description,
		}

		promotions = append(promotions, promotion)
	}

	h.logger.Info("Returning promotions list",
		zap.Int("count", len(promotions)))

	c.JSON(http.StatusOK, gin.H{
		"promotions": promotions,
	})
}

// ApplyPromotion validates and applies a coupon
// POST /api/v1/magic-checkout/apply-promotion
func (h *MagicCheckoutHandler) ApplyPromotion(c *gin.Context) {
	var req struct {
		OrderID         string  `json:"order_id"`
		RazorpayOrderID string  `json:"razorpay_order_id"`
		Contact         string  `json:"contact"`
		Email           string  `json:"email"`
		Code            string  `json:"code"`
		Amount          float64 `json:"amount"`          // Cart amount in paise (sent by Razorpay)
		CartAmount      float64 `json:"cart_amount"`     // Alternative field name
		SubTotal        float64 `json:"sub_total"`       // Another alternative
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid apply promotion request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":        "INVALID_REQUEST",
				"description": "Invalid request format",
			},
		})
		return
	}

	// Determine which order ID to use
	orderID := req.RazorpayOrderID
	if orderID == "" {
		orderID = req.OrderID
	}

	h.logger.Info("Magic Checkout applying promotion - RAW REQUEST",
		zap.String("order_id", req.OrderID),
		zap.String("razorpay_order_id", req.RazorpayOrderID),
		zap.String("resolved_order_id", orderID),
		zap.String("code", req.Code),
		zap.String("contact", req.Contact),
		zap.Float64("amount", req.Amount),
		zap.Float64("cart_amount", req.CartAmount),
		zap.Float64("sub_total", req.SubTotal))

	// Get cart total from various sources (priority order)
	var cartTotal float64 = 0
	var userID string = "guest_" + req.Contact

	// Priority 1: Get from our order cache (stored when order was created)
	// This is the most reliable source as we store the actual cart total
	if orderData, found := GetMagicCheckoutOrder(orderID); found {
		cartTotal = orderData.CartTotal
		userID = orderData.UserID
		h.logger.Info("Retrieved cart total from order cache",
			zap.String("order_id", orderID),
			zap.String("receipt", orderData.ReceiptID),
			zap.String("razorpay_order_id", orderData.RazorpayOrderID),
			zap.Float64("cart_total", cartTotal),
			zap.String("user_id", userID))
	}

	// Priority 2: Check if Razorpay sent amount directly in request
	if cartTotal == 0 && req.Amount > 0 {
		cartTotal = req.Amount / 100.0 // Convert from paise to rupees
		h.logger.Info("Using amount from request",
			zap.Float64("amount_paise", req.Amount),
			zap.Float64("cart_total_rupees", cartTotal))
	} else if cartTotal == 0 && req.CartAmount > 0 {
		cartTotal = req.CartAmount / 100.0
		h.logger.Info("Using cart_amount from request",
			zap.Float64("cart_amount_paise", req.CartAmount),
			zap.Float64("cart_total_rupees", cartTotal))
	} else if cartTotal == 0 && req.SubTotal > 0 {
		cartTotal = req.SubTotal / 100.0
		h.logger.Info("Using sub_total from request",
			zap.Float64("sub_total_paise", req.SubTotal),
			zap.Float64("cart_total_rupees", cartTotal))
	}

	// Priority 3: Try to get from Razorpay order API
	if cartTotal == 0 && strings.HasPrefix(orderID, "order_") {
		rzpOrder, err := h.paymentService.GetOrder(c.Request.Context(), orderID)
		if err == nil {
			cartTotal = float64(rzpOrder.Amount) / 100.0
			if rzpOrder.Notes != nil {
				if total, ok := rzpOrder.Notes["cart_total"].(float64); ok && total > 0 {
					cartTotal = total
				}
				if uid, ok := rzpOrder.Notes["user_id"].(string); ok && uid != "" {
					userID = uid
				}
			}
			h.logger.Info("Retrieved cart total from Razorpay API",
				zap.String("order_id", orderID),
				zap.Float64("cart_total", cartTotal),
				zap.String("user_id", userID))
		} else {
			h.logger.Warn("Could not fetch from Razorpay API",
				zap.String("order_id", orderID),
				zap.Error(err))
		}
	}

	// Priority 4: Last resort - use a minimum default but warn
	if cartTotal == 0 {
		cartTotal = 500.0 // ₹500 minimum default
		h.logger.Error("CRITICAL: No cart total found! Using minimum default",
			zap.String("order_id", orderID),
			zap.Float64("default_cart_total", cartTotal),
			zap.String("note", "Order cache may have expired or order was not created via Magic Checkout"))
	}

	// Validate coupon using existing service
	validation, err := h.couponService.ValidateCoupon(
		c.Request.Context(),
		req.Code,
		cartTotal,
		userID,
	)

	if err != nil || !validation.Success {
		h.logger.Warn("Coupon validation failed",
			zap.String("code", req.Code),
			zap.Error(err))

		errorMsg := "Invalid or expired coupon"
		if validation != nil && validation.Message != "" {
			errorMsg = validation.Message
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":        "INVALID_PROMOTION",
				"description": errorMsg,
			},
		})
		return
	}

	// Extract data from validation response
	validationData, ok := validation.Data.(map[string]interface{})
	if !ok {
		h.logger.Error("Invalid validation response format",
			zap.String("type", fmt.Sprintf("%T", validation.Data)))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":        "INVALID_PROMOTION",
				"description": "Failed to process coupon",
			},
		})
		return
	}

	// Extract coupon and rule from validation response
	var coupon *models.Coupon
	var bestRule *models.CouponRule
	var calculatedDiscount float64

	// Get coupon
	if couponData, ok := validationData["coupon"].(*models.Coupon); ok {
		coupon = couponData
	}

	// Get best rule
	if ruleData, ok := validationData["best_rule"].(*models.CouponRule); ok {
		bestRule = ruleData
	}

	// Get calculated discount
	if disc, ok := validationData["discount"].(float64); ok {
		calculatedDiscount = disc
	}

	// Validate we have the required data
	if coupon == nil || bestRule == nil {
		h.logger.Error("Missing coupon or rule data in validation response")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":        "INVALID_PROMOTION",
				"description": "Failed to process coupon",
			},
		})
		return
	}

	// Build Razorpay promotion response based on coupon rule type
	// Reference: https://razorpay.com/docs/payments/magic-checkout/rto-intelligence/coupons/
	promotion := map[string]interface{}{
		"type":         "offer",
		"code":         req.Code,
		"reference_id": coupon.ID,
		"description":  coupon.Description,
	}

	if promotion["description"] == "" {
		promotion["description"] = fmt.Sprintf("Discount with code %s", req.Code)
	}

	// Always send as fixed_amount with the calculated discount amount
	// The coupon service already calculated the correct discount based on cart total and rules
	// This is more reliable than depending on Razorpay's percentage calculation

	// Use the discount already calculated by coupon service (it respects max_discount, min_amount, etc.)
	discountAmount := calculatedDiscount

	// Ensure discount doesn't exceed cart total
	if discountAmount > cartTotal && cartTotal > 0 {
		discountAmount = cartTotal
	}

	// Always send as fixed_amount in paise - Razorpay will deduct this from total
	promotion["value_type"] = "fixed_amount"
	promotion["value"] = int(discountAmount * 100) // Convert rupees to paise

	h.logger.Info("Coupon discount calculated",
		zap.String("code", req.Code),
		zap.String("rule_type", string(bestRule.RuleType)),
		zap.Float64("cart_total_rupees", cartTotal),
		zap.Float64("discount_value_in_rule", bestRule.DiscountValue),
		zap.Float64("calculated_discount_rupees", calculatedDiscount),
		zap.Float64("final_discount_rupees", discountAmount),
		zap.Int("final_discount_paise", int(discountAmount*100)))

	h.logger.Info("Coupon promotion response prepared",
		zap.String("code", req.Code),
		zap.String("rule_type", string(bestRule.RuleType)),
		zap.Any("promotion", promotion))

	c.JSON(http.StatusOK, gin.H{
		"promotion": promotion,
	})
}

// UpdateAddresses handles address updates from Magic Checkout
// POST /api/v1/magic-checkout/addresses-callback
func (h *MagicCheckoutHandler) UpdateAddresses(c *gin.Context) {
	// Read raw body for debugging
	rawBody, _ := c.GetRawData()
	h.logger.Info("RAW addresses-callback request body from Razorpay",
		zap.String("raw_body", string(rawBody)),
		zap.Int("body_length", len(rawBody)))

	// Parse into struct from raw body
	var req struct {
		OrderID         string                 `json:"order_id"`
		RazorpayOrderID string                 `json:"razorpay_order_id"`
		Email           string                 `json:"email"`
		Contact         string                 `json:"contact"`
		ShippingAddress map[string]interface{} `json:"shipping_address"`
		BillingAddress  map[string]interface{} `json:"billing_address"`
		CustomerDetails map[string]interface{} `json:"customer_details"`
	}

	// Unmarshal from raw body
	if err := json.Unmarshal(rawBody, &req); err != nil {
		h.logger.Error("Invalid addresses callback request - failed to parse JSON",
			zap.Error(err),
			zap.String("raw_body", string(rawBody)))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request",
		})
		return
	}

	h.logger.Info("Magic Checkout addresses callback PARSED successfully",
		zap.String("order_id", req.OrderID),
		zap.String("razorpay_order_id", req.RazorpayOrderID),
		zap.String("contact", req.Contact),
		zap.String("email", req.Email),
		zap.Any("shipping_address", req.ShippingAddress),
		zap.Any("billing_address", req.BillingAddress),
		zap.Any("customer_details", req.CustomerDetails))

	// Store the address information in cache/memory for later retrieval during payment verification
	if req.RazorpayOrderID != "" && req.ShippingAddress != nil {
		// Store the address data in a temporary cache keyed by Razorpay order ID
		// This will be retrieved during payment verification
		addressData := map[string]interface{}{
			"shipping_address": req.ShippingAddress,
			"billing_address":  req.BillingAddress,
			"email":            req.Email,
			"contact":          req.Contact,
			"customer_details": req.CustomerDetails,
		}

		// Store in global cache for retrieval during payment verification
		// This will be retrieved during VerifyPayment
		StoreMagicCheckoutAddress(req.RazorpayOrderID, addressData)
		h.logger.Info("Successfully stored Magic Checkout address for order",
			zap.String("razorpay_order_id", req.RazorpayOrderID),
			zap.Any("address_data", addressData))

		// Also extract and log formatted address for debugging
		addressLine1 := ""
		addressLine2 := ""
		city := ""
		state := ""
		postalCode := ""
		country := "India"

		// Extract fields from shipping address
		if line1, ok := req.ShippingAddress["line1"].(string); ok {
			addressLine1 = strings.TrimSpace(line1)
		} else if line1, ok := req.ShippingAddress["address_line_1"].(string); ok {
			addressLine1 = strings.TrimSpace(line1)
		} else if line1, ok := req.ShippingAddress["address"].(string); ok {
			addressLine1 = strings.TrimSpace(line1)
		}

		if line2, ok := req.ShippingAddress["line2"].(string); ok {
			addressLine2 = strings.TrimSpace(line2)
		} else if line2, ok := req.ShippingAddress["address_line_2"].(string); ok {
			addressLine2 = strings.TrimSpace(line2)
		}

		if c, ok := req.ShippingAddress["city"].(string); ok {
			city = strings.TrimSpace(c)
		}

		if s, ok := req.ShippingAddress["state"].(string); ok {
			state = strings.TrimSpace(s)
		} else if s, ok := req.ShippingAddress["state_code"].(string); ok {
			state = strings.TrimSpace(s)
		}

		if zip, ok := req.ShippingAddress["zipcode"].(string); ok {
			postalCode = strings.TrimSpace(zip)
		} else if zip, ok := req.ShippingAddress["postal_code"].(string); ok {
			postalCode = strings.TrimSpace(zip)
		} else if zip, ok := req.ShippingAddress["pincode"].(string); ok {
			postalCode = strings.TrimSpace(zip)
		}

		if c, ok := req.ShippingAddress["country"].(string); ok {
			country = strings.TrimSpace(c)
		}

		h.logger.Info("Parsed Magic Checkout address",
			zap.String("razorpay_order_id", req.RazorpayOrderID),
			zap.String("line1", addressLine1),
			zap.String("line2", addressLine2),
			zap.String("city", city),
			zap.String("state", state),
			zap.String("postal_code", postalCode),
			zap.String("country", country))

		// Try to save address to database if we have valid data and user ID
		// Get user ID from the order cache
		orderID := req.RazorpayOrderID
		if orderID == "" {
			orderID = req.OrderID
		}

		if orderData, found := GetMagicCheckoutOrder(orderID); found && orderData.UserID != "" {
			// We have a valid user ID from the order, save the address to database
			if h.addressService != nil && addressLine1 != "" && postalCode != "" {
				savedAddr, err := h.addressService.FindOrCreateAddress(
					c.Request.Context(),
					orderData.UserID,
					addressLine1,
					city,
					state,
					postalCode,
					country,
				)
				if err == nil && savedAddr != nil {
					h.logger.Info("Successfully saved Magic Checkout address to database",
						zap.String("razorpay_order_id", req.RazorpayOrderID),
						zap.String("user_id", orderData.UserID),
						zap.String("address_id", savedAddr.ID))
				} else {
					h.logger.Warn("Failed to save Magic Checkout address to database",
						zap.String("razorpay_order_id", req.RazorpayOrderID),
						zap.String("user_id", orderData.UserID),
						zap.Error(err))
				}
			}
		} else {
			h.logger.Info("No user ID found in order cache, address will be saved during payment verification",
				zap.String("order_id", orderID))
		}
	}

	// Return success response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Address updated successfully",
	})
}

// GetShippingInfo calculates shipping for given addresses
// POST /api/v1/magic-checkout/shipping-info
func (h *MagicCheckoutHandler) GetShippingInfo(c *gin.Context) {
	// Read raw body first for debugging
	rawBody, _ := c.GetRawData()
	h.logger.Info("RAW shipping-info request body from Razorpay",
		zap.String("raw_body", string(rawBody)),
		zap.Int("body_length", len(rawBody)))

	var req struct {
		OrderID         string  `json:"order_id"`
		RazorpayOrderID string  `json:"razorpay_order_id"`
		Email           string  `json:"email"`
		Contact         string  `json:"contact"`
		Amount          float64 `json:"amount"`      // Amount in paise from Razorpay
		CartAmount      float64 `json:"cart_amount"` // Alternative field
		Addresses       []struct {
			ID           interface{} `json:"id"` // Can be string or number from Razorpay
			Zipcode      string      `json:"zipcode"`
			StateCode    string      `json:"state_code"`
			Country      string      `json:"country"`
			City         string      `json:"city,omitempty"`
			State        string      `json:"state,omitempty"`
			AddressLine1 string      `json:"address_line_1,omitempty"`
			AddressLine2 string      `json:"address_line_2,omitempty"`
			Line1        string      `json:"line1,omitempty"`
			Line2        string      `json:"line2,omitempty"`
			Name         string      `json:"name,omitempty"`
			Tag          string      `json:"tag,omitempty"`
		} `json:"addresses"`
	}

	// Parse from raw body
	if err := json.Unmarshal(rawBody, &req); err != nil {
		h.logger.Error("Invalid shipping info request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"addresses": []interface{}{},
		})
		return
	}

	// Determine which order ID to use (Razorpay sends our receipt in order_id)
	orderID := req.RazorpayOrderID
	if orderID == "" {
		orderID = req.OrderID
	}

	// Also try to get the actual Razorpay order ID (starts with order_)
	razorpayOrderID := req.RazorpayOrderID
	if !strings.HasPrefix(razorpayOrderID, "order_") {
		razorpayOrderID = "order_" + razorpayOrderID
	}

	h.logger.Info("SHIPPING INFO REQUEST - RAW DATA",
		zap.String("order_id", req.OrderID),
		zap.String("razorpay_order_id", req.RazorpayOrderID),
		zap.String("resolved_order_id", orderID),
		zap.String("contact", req.Contact),
		zap.String("email", req.Email),
		zap.Float64("amount", req.Amount),
		zap.Float64("cart_amount", req.CartAmount),
		zap.Int("address_count", len(req.Addresses)))

	// CRITICAL FIX: Store address and customer info from shipping-info call
	// Razorpay may send full address here and NOT call addresses-callback
	// We must ALWAYS store whatever address data we receive to avoid losing it
	if len(req.Addresses) > 0 {
		addr := req.Addresses[0] // Use first address

		// Build address line - prefer line1 over address_line_1
		addressLine1 := addr.Line1
		if addressLine1 == "" {
			addressLine1 = addr.AddressLine1
		}
		addressLine2 := addr.Line2
		if addressLine2 == "" {
			addressLine2 = addr.AddressLine2
		}

		// Build state - prefer state over state_code
		state := addr.State
		if state == "" {
			state = addr.StateCode
		}

		// Check if this is a "full" address (has line1 OR city) vs just serviceability data
		hasFullAddress := addressLine1 != "" || addr.City != ""

		h.logger.Info("Shipping-info address data received",
			zap.String("order_id", req.OrderID),
			zap.String("razorpay_order_id", req.RazorpayOrderID),
			zap.String("name", addr.Name),
			zap.String("line1", addressLine1),
			zap.String("city", addr.City),
			zap.String("state", state),
			zap.String("zipcode", addr.Zipcode),
			zap.String("contact", req.Contact),
			zap.String("email", req.Email),
			zap.Bool("has_full_address", hasFullAddress))

		// ALWAYS store address data - don't discard it!
		// Build address data with whatever we have
		addressData := map[string]interface{}{
			"shipping_address": map[string]interface{}{
				"name":    addr.Name,
				"line1":   addressLine1,
				"line2":   addressLine2,
				"city":    addr.City,
				"state":   state,
				"zipcode": addr.Zipcode,
				"country": addr.Country,
				"tag":     addr.Tag,
			},
			"contact": req.Contact,
			"email":   req.Email,
		}

		// Store by ALL possible key formats for reliable retrieval later
		keysStored := []string{}

		// Key 1: Raw order_id from request (could be receipt ID like "mc_xxx_timestamp")
		if req.OrderID != "" {
			StoreMagicCheckoutAddress(req.OrderID, addressData)
			keysStored = append(keysStored, req.OrderID)
		}

		// Key 2: Razorpay order ID without prefix (like "RmMkQm9FiEAFC1")
		if req.RazorpayOrderID != "" {
			StoreMagicCheckoutAddress(req.RazorpayOrderID, addressData)
			keysStored = append(keysStored, req.RazorpayOrderID)
		}

		// Key 3: Razorpay order ID with order_ prefix (like "order_RmMkQm9FiEAFC1")
		if razorpayOrderID != "" && razorpayOrderID != req.RazorpayOrderID {
			StoreMagicCheckoutAddress(razorpayOrderID, addressData)
			keysStored = append(keysStored, razorpayOrderID)
		}

		// Key 4: Also without order_ prefix if it was added
		if strings.HasPrefix(req.RazorpayOrderID, "order_") {
			withoutPrefix := strings.TrimPrefix(req.RazorpayOrderID, "order_")
			StoreMagicCheckoutAddress(withoutPrefix, addressData)
			keysStored = append(keysStored, withoutPrefix)
		}

		h.logger.Info("STORED Magic Checkout address with multiple keys",
			zap.Strings("keys", keysStored),
			zap.Bool("has_full_address", hasFullAddress),
			zap.String("name", addr.Name),
			zap.String("zipcode", addr.Zipcode),
			zap.String("contact", req.Contact))

		// Also save full address to database if we have valid data and user ID
		if hasFullAddress && h.addressService != nil && addressLine1 != "" && addr.Zipcode != "" {
			// Try to get user ID from order cache
			lookupOrderID := orderID
			if lookupOrderID == "" {
				lookupOrderID = req.RazorpayOrderID
			}
			if lookupOrderID == "" {
				lookupOrderID = req.OrderID
			}

			if orderData, found := GetMagicCheckoutOrder(lookupOrderID); found && orderData.UserID != "" {
				savedAddr, err := h.addressService.FindOrCreateAddress(
					c.Request.Context(),
					orderData.UserID,
					addressLine1,
					addr.City,
					state,
					addr.Zipcode,
					addr.Country,
				)
				if err == nil && savedAddr != nil {
					h.logger.Info("Successfully saved shipping-info address to database",
						zap.String("order_id", lookupOrderID),
						zap.String("user_id", orderData.UserID),
						zap.String("address_id", savedAddr.ID),
						zap.String("city", addr.City),
						zap.String("zipcode", addr.Zipcode))
				} else {
					h.logger.Warn("Failed to save shipping-info address to database",
						zap.String("order_id", lookupOrderID),
						zap.String("user_id", orderData.UserID),
						zap.Error(err))
				}
			}
		}
	}

	// Get order details to determine weight and cart total
	var weight int = 500 // Default 500g
	var cartTotal float64 = 0

	// Priority 1: Get from our order cache (most reliable)
	if orderData, found := GetMagicCheckoutOrder(orderID); found {
		cartTotal = orderData.CartTotal
		h.logger.Info("SUCCESS: Retrieved cart total from order cache",
			zap.String("order_id", orderID),
			zap.String("receipt", orderData.ReceiptID),
			zap.String("razorpay_order", orderData.RazorpayOrderID),
			zap.Float64("cart_total", cartTotal))
	} else {
		h.logger.Warn("CACHE MISS: Order not found in cache",
			zap.String("order_id", orderID),
			zap.String("note", "Order may not have been created via Magic Checkout or server was restarted"))
	}

	// Priority 2: Check if Razorpay sent amount in request
	if cartTotal == 0 && req.Amount > 0 {
		cartTotal = req.Amount / 100.0 // Convert paise to rupees
		h.logger.Info("Using amount from request",
			zap.Float64("amount_paise", req.Amount),
			zap.Float64("cart_total_rupees", cartTotal))
	} else if cartTotal == 0 && req.CartAmount > 0 {
		cartTotal = req.CartAmount / 100.0
		h.logger.Info("Using cart_amount from request",
			zap.Float64("cart_amount_paise", req.CartAmount),
			zap.Float64("cart_total_rupees", cartTotal))
	}

	// Priority 3: Try Razorpay API if we have a valid order ID
	if cartTotal == 0 && strings.HasPrefix(orderID, "order_") {
		rzpOrder, err := h.paymentService.GetOrder(c.Request.Context(), orderID)
		if err == nil && rzpOrder.Notes != nil {
			// Get item count from notes
			if itemCount, ok := rzpOrder.Notes["total_items"].(float64); ok && itemCount > 0 {
				weight = int(itemCount) * 100 // 100g per item
			}
			// Get cart total for free delivery check
			if total, ok := rzpOrder.Notes["cart_total"].(float64); ok && total > 0 {
				cartTotal = total
			}
			h.logger.Info("Retrieved cart total from Razorpay API",
				zap.String("order_id", orderID),
				zap.Float64("cart_total", cartTotal))
		}
	}

	// Log final cart total decision
	h.logger.Info("SHIPPING CALCULATION - CART TOTAL DECISION",
		zap.Float64("final_cart_total", cartTotal),
		zap.Bool("qualifies_for_free_delivery", cartTotal >= 500))

	// Process each address
	result := []map[string]interface{}{}

	for _, addr := range req.Addresses {
		// Convert ID to string (can be number or string from Razorpay)
		addressID := ""
		switch v := addr.ID.(type) {
		case string:
			addressID = v
		case float64:
			addressID = fmt.Sprintf("%.0f", v)
		case int:
			addressID = fmt.Sprintf("%d", v)
		default:
			addressID = fmt.Sprintf("%v", v)
		}

		h.logger.Info("Processing address",
			zap.String("id", addressID),
			zap.String("zipcode", addr.Zipcode),
			zap.String("state", addr.StateCode))

		// Clean zipcode (remove spaces, ensure 6 digits)
		zipcode := strings.TrimSpace(addr.Zipcode)
		zipcode = strings.ReplaceAll(zipcode, " ", "")
		if len(zipcode) != 6 {
			h.logger.Warn("Invalid zipcode format",
				zap.String("zipcode", addr.Zipcode),
				zap.String("cleaned", zipcode))
			result = append(result, map[string]interface{}{
				"id":           addressID,
				"serviceable":  false,
				"cod":          false,
				"shipping_fee": 0,
				"cod_fee":      0,
			})
			continue
		}

		// For Magic Checkout, we default to serviceable for all valid Indian pincodes
		// This ensures a smooth checkout experience - actual delivery constraints
		// can be handled during order fulfillment if needed
		serviceable := true

		h.logger.Info("Magic Checkout serviceability - defaulting to serviceable for all pincodes",
			zap.String("zipcode", zipcode),
			zap.Bool("serviceable", serviceable))

		addressResult := map[string]interface{}{
			"id":           addressID,
			"serviceable":  serviceable,
			"cod":          serviceable, // COD available if serviceable
			"shipping_fee": 0,
			"cod_fee":      0,
		}

		if serviceable {
			// Check for FREE delivery first (cart total >= 500)
			// This applies regardless of coupon - based on original cart value
			if cartTotal >= 500 {
				// FREE DELIVERY - cart is above ₹500
				addressResult["shipping_fee"] = 0
				h.logger.Info("FREE DELIVERY applied - cart total >= ₹500",
					zap.String("zipcode", zipcode),
					zap.Float64("cart_total", cartTotal),
					zap.String("reason", "Order value above ₹500"))
			} else {
				// Calculate shipping charges using real Delhivery API
				charges, err := h.deliveryService.GetShippingCharges(c.Request.Context(), zipcode, weight, 0)
				if err != nil {
					h.logger.Error("Failed to get shipping charges from Delhivery API, using fallback",
						zap.Error(err),
						zap.String("zipcode", zipcode),
						zap.Int("weight", weight))
					// Use fallback shipping rate based on weight
					charges = 40.0 + (float64(weight)/500.0)*10.0 // Base ₹40 + ₹10 per 500g
				}

				// Convert to paise
				addressResult["shipping_fee"] = int(charges * 100)

				h.logger.Info("Shipping charges calculated",
					zap.String("zipcode", zipcode),
					zap.Float64("cart_total", cartTotal),
					zap.Float64("charges_rupees", charges),
					zap.Int("charges_paise", int(charges*100)),
					zap.String("note", "Add ₹500+ to cart for free delivery"))
			}

			// COD fee (₹20 if COD selected)
			addressResult["cod_fee"] = 2000 // ₹20 in paise
		}

		result = append(result, addressResult)
	}

	c.JSON(http.StatusOK, gin.H{
		"addresses": result,
	})
}

// AbandonedCartWebhook handles abandoned checkout notifications from Razorpay Magic Checkout
// POST /api/v1/magic-checkout/abandoned-cart
func (h *MagicCheckoutHandler) AbandonedCartWebhook(c *gin.Context) {
	var req struct {
		Event   string `json:"event"`
		Payload struct {
			Order struct {
				ID       string                 `json:"id"`
				Receipt  string                 `json:"receipt"`
				Amount   int                    `json:"amount"`
				Currency string                 `json:"currency"`
				Notes    map[string]interface{} `json:"notes"`
			} `json:"order"`
			Checkout struct {
				Contact         string `json:"contact"`
				Email           string `json:"email"`
				ShippingAddress struct {
					Name    string `json:"name"`
					Line1   string `json:"line1"`
					Line2   string `json:"line2"`
					City    string `json:"city"`
					State   string `json:"state"`
					Zipcode string `json:"zipcode"`
					Country string `json:"country"`
				} `json:"shipping_address"`
				Promotions []struct {
					Code  string `json:"code"`
					Value int    `json:"value"`
				} `json:"promotions"`
			} `json:"checkout"`
			AbandonedAt int64  `json:"abandoned_at"`
			Reason      string `json:"reason"` // "timeout", "user_exit", "payment_failed"
		} `json:"payload"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid abandoned cart webhook request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	h.logger.Info("Abandoned cart webhook received",
		zap.String("event", req.Event),
		zap.String("order_id", req.Payload.Order.ID),
		zap.String("contact", req.Payload.Checkout.Contact),
		zap.String("email", req.Payload.Checkout.Email),
		zap.Int("amount", req.Payload.Order.Amount),
		zap.String("reason", req.Payload.Reason),
		zap.Int64("abandoned_at", req.Payload.AbandonedAt))

	// Extract user info from order notes
	userID := ""
	if notes := req.Payload.Order.Notes; notes != nil {
		if uid, ok := notes["user_id"].(string); ok {
			userID = uid
		}
	}

	// Log detailed abandoned cart info for analytics/recovery
	h.logger.Info("Abandoned cart details",
		zap.String("user_id", userID),
		zap.String("receipt", req.Payload.Order.Receipt),
		zap.Float64("cart_value_rupees", float64(req.Payload.Order.Amount)/100),
		zap.String("customer_phone", req.Payload.Checkout.Contact),
		zap.String("customer_email", req.Payload.Checkout.Email),
		zap.String("shipping_city", req.Payload.Checkout.ShippingAddress.City),
		zap.String("shipping_pincode", req.Payload.Checkout.ShippingAddress.Zipcode))

	// TODO: Implement cart recovery actions:
	// 1. Send WhatsApp reminder after 1 hour
	// 2. Send email with cart recovery link
	// 3. Store in abandoned_carts table for analytics
	// 4. Trigger retargeting campaign

	// Example: Send WhatsApp notification (uncomment when ready)
	/*
	if req.Payload.Checkout.Contact != "" && h.whatsappService != nil {
		go func() {
			ctx := context.Background()
			message := fmt.Sprintf(
				"Hi! You left items worth ₹%.0f in your cart. Complete your purchase now!",
				float64(req.Payload.Order.Amount)/100,
			)
			h.whatsappService.SendMessage(ctx, req.Payload.Checkout.Contact, message)
		}()
	}
	*/

	// Always return 200 to acknowledge receipt
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Abandoned cart webhook received",
	})
}

