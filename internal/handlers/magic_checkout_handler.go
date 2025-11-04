package handlers

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// MagicCheckoutHandler handles Magic Checkout API endpoints
type MagicCheckoutHandler struct {
	paymentService  services.PaymentServiceInterface
	couponService   services.CouponServiceInterface
	deliveryService services.DeliveryServiceInterface
	cartService     services.CartServiceInterface
	orderService    services.OrderServiceInterface
	logger          *zap.Logger
}

// NewMagicCheckoutHandler creates a new Magic Checkout handler
func NewMagicCheckoutHandler(
	paymentService services.PaymentServiceInterface,
	couponService services.CouponServiceInterface,
	deliveryService services.DeliveryServiceInterface,
	cartService services.CartServiceInterface,
	orderService services.OrderServiceInterface,
	logger *zap.Logger,
) *MagicCheckoutHandler {
	return &MagicCheckoutHandler{
		paymentService:  paymentService,
		couponService:   couponService,
		deliveryService: deliveryService,
		cartService:     cartService,
		orderService:    orderService,
		logger:          logger,
	}
}

// GetPromotions returns available coupons for Magic Checkout
// POST /api/v1/magic-checkout/get-promotions
func (h *MagicCheckoutHandler) GetPromotions(c *gin.Context) {
	var req struct {
		OrderID         string `json:"order_id"`
		RazorpayOrderID string `json:"razorpay_order_id"`
		Contact         string `json:"contact"`
		Email           string `json:"email"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid get promotions request", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"promotions": []interface{}{}})
		return
	}

	h.logger.Info("Magic Checkout requesting promotions",
		zap.String("order_id", req.OrderID),
		zap.String("razorpay_order_id", req.RazorpayOrderID),
		zap.String("contact", req.Contact))

	// Get active coupons from existing service (limit to 50 for display)
	activeCoupons, err := h.couponService.GetActiveCoupons(c.Request.Context(), 50, 0)
	if err != nil {
		h.logger.Error("Failed to get active coupons", zap.Error(err))
		c.JSON(http.StatusOK, gin.H{"promotions": []interface{}{}})
		return
	}

	// Format for Magic Checkout
	promotions := []map[string]interface{}{}

	// Extract coupons from response
	var couponsData []interface{}
	if responseData, ok := activeCoupons.Data.(map[string]interface{}); ok {
		if coupons, ok := responseData["coupons"].([]interface{}); ok {
			couponsData = coupons
		}
	}

	for _, coupon := range couponsData {
		if couponMap, ok := coupon.(map[string]interface{}); ok {
			// Get coupon ID (handle different possible field names)
			couponID := ""
			if id, ok := couponMap["id"].(string); ok {
				couponID = id
			} else if id, ok := couponMap["ID"].(string); ok {
				couponID = id
			}

			// Get rules (handle different possible structures)
			var rules []interface{}
			if rulesData, ok := couponMap["rules"].([]interface{}); ok {
				rules = rulesData
			} else if rulesData, ok := couponMap["Rules"].([]interface{}); ok {
				rules = rulesData
			}

			if len(rules) == 0 {
				continue
			}

			// Get the first rule for display
			firstRule := rules[0]
			var firstRuleMap map[string]interface{}
			if ruleMap, ok := firstRule.(map[string]interface{}); ok {
				firstRuleMap = ruleMap
			}

			if firstRuleMap == nil {
				continue
			}

			promotion := map[string]interface{}{
				"code":         couponMap["code"],
				"reference_id": couponID,
			}

			// Add description if available
			if desc, ok := couponMap["description"].(string); ok && desc != "" {
				promotion["description"] = desc
			} else {
				promotion["description"] = fmt.Sprintf("Use code %s", couponMap["code"])
			}

			// Determine type and value based on rule
			if ruleType, ok := firstRuleMap["type"].(string); ok {
				if ruleType == "percentage" {
					promotion["type"] = "percentage"
					if value, ok := firstRuleMap["discount_value"].(float64); ok {
						promotion["value"] = value
					} else if value, ok := firstRuleMap["DiscountValue"].(float64); ok {
						promotion["value"] = value
					}
				} else {
					promotion["type"] = "cart_value"
					// Convert to paise
					if value, ok := firstRuleMap["discount_value"].(float64); ok {
						promotion["value"] = int(value * 100)
					} else if value, ok := firstRuleMap["DiscountValue"].(float64); ok {
						promotion["value"] = int(value * 100)
					}
				}
			}

			promotions = append(promotions, promotion)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"promotions": promotions,
	})
}

// ApplyPromotion validates and applies a coupon
// POST /api/v1/magic-checkout/apply-promotion
func (h *MagicCheckoutHandler) ApplyPromotion(c *gin.Context) {
	var req struct {
		OrderID         string `json:"order_id"`
		RazorpayOrderID string `json:"razorpay_order_id"`
		Contact         string `json:"contact"`
		Email           string `json:"email"`
		Code            string `json:"code"`
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

	h.logger.Info("Magic Checkout applying promotion",
		zap.String("order_id", req.OrderID),
		zap.String("razorpay_order_id", req.RazorpayOrderID),
		zap.String("code", req.Code),
		zap.String("contact", req.Contact))

	// Get order amount from Razorpay order
	rzpOrder, err := h.paymentService.GetOrder(c.Request.Context(), req.RazorpayOrderID)
	if err != nil {
		h.logger.Error("Failed to get Razorpay order", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":        "INVALID_PROMOTION",
				"description": "Order not found",
			},
		})
		return
	}

	// Extract cart total from order notes
	cartTotal := float64(rzpOrder.Amount) / 100.0 // Convert from paise to rupees
	if rzpOrder.Notes != nil {
		if total, ok := rzpOrder.Notes["cart_total"].(float64); ok && total > 0 {
			cartTotal = total
		}
	}

	// For guest checkout, use phone as identifier
	userID := "guest_" + req.Contact
	if rzpOrder.Notes != nil {
		if uid, ok := rzpOrder.Notes["user_id"].(string); ok && uid != "" {
			userID = uid
		}
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

	// Extract discount from validation response
	validationData, ok := validation.Data.(map[string]interface{})
	if !ok {
		h.logger.Error("Invalid validation response format")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":        "INVALID_PROMOTION",
				"description": "Failed to process coupon",
			},
		})
		return
	}

	discount := 0.0
	if disc, ok := validationData["discount"].(float64); ok {
		discount = disc
	}

	coupon, ok := validationData["coupon"].(map[string]interface{})
	if !ok {
		h.logger.Error("Invalid coupon data in validation response")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":        "INVALID_PROMOTION",
				"description": "Failed to process coupon",
			},
		})
		return
	}

	bestRule, ok := validationData["best_rule"].(map[string]interface{})
	if !ok {
		h.logger.Error("Invalid best_rule data in validation response")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":        "INVALID_PROMOTION",
				"description": "Failed to process coupon",
			},
		})
		return
	}

	// Format response for Magic Checkout
	promotion := map[string]interface{}{
		"code": req.Code,
	}

	// Get coupon ID
	if couponID, ok := coupon["id"].(string); ok {
		promotion["reference_id"] = couponID
	} else if couponID, ok := coupon["ID"].(string); ok {
		promotion["reference_id"] = couponID
	}

	// Add description
	if desc, ok := coupon["description"].(string); ok && desc != "" {
		promotion["description"] = desc
	} else {
		promotion["description"] = fmt.Sprintf("Use code %s", req.Code)
	}

	// Set type and value based on rule type
	if ruleType, ok := bestRule["type"].(string); ok {
		if ruleType == "percentage" {
			promotion["type"] = "percentage"
			if value, ok := bestRule["discount_value"].(float64); ok {
				promotion["value"] = value
			} else if value, ok := bestRule["DiscountValue"].(float64); ok {
				promotion["value"] = value
			}
		} else {
			promotion["type"] = "cart_value"
			promotion["value"] = int(discount * 100) // Convert to paise
		}
	} else {
		// Fallback: assume cart_value if type not found
		promotion["type"] = "cart_value"
		promotion["value"] = int(discount * 100) // Convert to paise
	}

	c.JSON(http.StatusOK, gin.H{
		"promotion": promotion,
	})
}

// GetShippingInfo calculates shipping for given addresses
// POST /api/v1/magic-checkout/shipping-info
func (h *MagicCheckoutHandler) GetShippingInfo(c *gin.Context) {
	var req struct {
		OrderID         string `json:"order_id"`
		RazorpayOrderID string `json:"razorpay_order_id"`
		Email           string `json:"email"`
		Contact         string `json:"contact"`
		Addresses       []struct {
			ID           interface{} `json:"id"` // Can be string or number from Razorpay
			Zipcode      string      `json:"zipcode"`
			StateCode    string      `json:"state_code"`
			Country      string      `json:"country"`
			City         string      `json:"city,omitempty"`
			State        string      `json:"state,omitempty"`
			AddressLine1 string      `json:"address_line_1,omitempty"`
			AddressLine2 string      `json:"address_line_2,omitempty"`
		} `json:"addresses"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid shipping info request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"addresses": []interface{}{},
		})
		return
	}

	h.logger.Info("Magic Checkout requesting shipping info",
		zap.String("order_id", req.OrderID),
		zap.String("razorpay_order_id", req.RazorpayOrderID),
		zap.Int("address_count", len(req.Addresses)))

	// Get order details to determine weight and cart total
	var weight int = 500 // Default 500g
	var cartTotal float64 = 0

	if req.RazorpayOrderID != "" {
		rzpOrder, err := h.paymentService.GetOrder(c.Request.Context(), req.RazorpayOrderID)
		if err == nil && rzpOrder.Notes != nil {
			// Get item count from notes
			if itemCount, ok := rzpOrder.Notes["total_items"].(float64); ok && itemCount > 0 {
				weight = int(itemCount) * 100 // 100g per item
			}
			// Get cart total for free delivery check
			if total, ok := rzpOrder.Notes["cart_total"].(float64); ok && total > 0 {
				cartTotal = total
			}
		}
	}

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

		// Check serviceability via Delhivery with timeout
		// Use a context with timeout to prevent hanging (max 2 seconds)
		serviceabilityCtx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		serviceable := false

		// Check serviceability in a goroutine to avoid blocking
		serviceabilityChan := make(chan bool, 1)
		errorChan := make(chan error, 1)

		go func() {
			result, err := h.deliveryService.CheckServiceability(serviceabilityCtx, zipcode)
			if err != nil {
				errorChan <- err
				return
			}
			serviceabilityChan <- result
		}()

		// Wait for result with timeout
		select {
		case result := <-serviceabilityChan:
			// Serviceability check completed successfully
			serviceable = result
			h.logger.Info("Serviceability check completed",
				zap.String("zipcode", zipcode),
				zap.Bool("serviceable", serviceable))
		case err := <-errorChan:
			// Serviceability check failed - default to serviceable to allow checkout
			h.logger.Warn("Serviceability check failed, defaulting to serviceable",
				zap.String("zipcode", zipcode),
				zap.Error(err))
			serviceable = true
		case <-serviceabilityCtx.Done():
			// Timeout - default to serviceable to allow checkout
			h.logger.Warn("Serviceability check timeout, defaulting to serviceable",
				zap.String("zipcode", zipcode))
			serviceable = true
		}

		addressResult := map[string]interface{}{
			"id":           addressID,
			"serviceable":  serviceable,
			"cod":          serviceable, // COD available if serviceable
			"shipping_fee": 0,
			"cod_fee":      0,
		}

		if serviceable {
			// Calculate shipping charges
			charges, err := h.getShippingCharges(c.Request.Context(), zipcode, weight)
			if err != nil {
				h.logger.Error("Failed to calculate shipping", zap.Error(err))
				// Use fallback shipping rate
				charges = 40.0
			}

			// Check for free delivery (cart total >= 500)
			if cartTotal >= 500 {
				h.logger.Info("Free delivery applied",
					zap.Float64("cart_total", cartTotal))
				charges = 0
			}

			// Convert to paise
			addressResult["shipping_fee"] = int(charges * 100)

			// COD fee (₹20 if COD selected)
			addressResult["cod_fee"] = 2000 // ₹20 in paise
		}

		result = append(result, addressResult)
	}

	c.JSON(http.StatusOK, gin.H{
		"addresses": result,
	})
}

// getShippingCharges is a helper method to get shipping charges
// Uses a fallback calculation since getShippingCharges is private in DeliveryService
// TODO: Expose getShippingCharges as a public method in DeliveryService or use Estimate with user context
func (h *MagicCheckoutHandler) getShippingCharges(ctx context.Context, zipcode string, weight int) (float64, error) {
	// Use a simple fallback calculation based on weight and pincode
	baseShipping := 30.0
	weightBasedCharge := float64(weight) / 100.0 * 0.5 // 0.5 per 100g

	// Check if metro pincode (simplified check)
	isMetro := strings.HasPrefix(zipcode, "110") || // Delhi
		strings.HasPrefix(zipcode, "400") || // Mumbai
		strings.HasPrefix(zipcode, "560") || // Bangalore
		strings.HasPrefix(zipcode, "600") || // Chennai
		strings.HasPrefix(zipcode, "700") || // Kolkata
		strings.HasPrefix(zipcode, "500") || // Hyderabad
		strings.HasPrefix(zipcode, "411") || // Pune
		strings.HasPrefix(zipcode, "380") // Ahmedabad

	distanceMultiplier := 1.0
	if isMetro {
		distanceMultiplier = 0.9
	} else {
		distanceMultiplier = 1.1
	}

	calculatedShipping := (baseShipping + weightBasedCharge) * distanceMultiplier

	// Round to 2 decimal places and ensure minimum charge
	if calculatedShipping < 25.0 {
		calculatedShipping = 25.0
	}

	return calculatedShipping, nil
}
