package handlers

import (
	"context"
	"errors"
	"math"
	"net/http"
	"regexp"
	"strings"
	"time"

	"fmt"
	"html/template"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Compile regex once for better performance
var pincodeRegex = regexp.MustCompile(`^\d{6}$`)

// PaymentHandler handles payment-related HTTP requests
type PaymentHandler struct {
	paymentService  services.PaymentServiceInterface
	orderService    services.OrderServiceInterface
	deliveryService services.DeliveryServiceInterface
	whatsappService *services.WhatsAppService
	userService     *services.UserService
	productService  *services.ProductService
	addressService  *services.AddressService
	logger          *zap.Logger
	authContext     *auth.AuthContext
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentService services.PaymentServiceInterface, orderService services.OrderServiceInterface, deliveryService services.DeliveryServiceInterface, whatsappService *services.WhatsAppService, userService *services.UserService, productService *services.ProductService, addressService *services.AddressService, logger *zap.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentService:  paymentService,
		orderService:    orderService,
		deliveryService: deliveryService,
		whatsappService: whatsappService,
		userService:     userService,
		productService:  productService,
		addressService:  addressService,
		logger:          logger,
		authContext:     auth.NewAuthContext(),
	}
}

// CreateOrderRequest represents the request to create a payment order
type CreateOrderRequest struct {
	ShippingAddressID string  `json:"shipping_address_id,omitempty"` // Optional: if using saved addresses
	ShippingAddress   string  `json:"shipping_address,omitempty"`    // Optional: if providing new address
	Currency          string  `json:"currency,omitempty"`            // Ignored (hardcoded to INR)
	Subtotal          float64 `json:"subtotal,omitempty"`
	ShippingCost      float64 `json:"shipping_cost,omitempty"` // Optional: estimated shipping cost from estimate-shipping endpoint
	FirstName         string  `json:"first_name"`              // Required
	LastName          string  `json:"last_name,omitempty"`
	Mobile            string  `json:"mobile"` // Required (primary)
	MobileSecondary   string  `json:"mobile_secondary,omitempty"`
}

// CreateOrderResponse represents the response with order details
type CreateOrderResponse struct {
	OrderID     string                 `json:"order_id"`
	Amount      float64                `json:"amount"`
	Currency    string                 `json:"currency"`
	KeyID       string                 `json:"key_id"`
	Description string                 `json:"description"`
	Notes       map[string]interface{} `json:"notes"`
	CheckoutURL string                 `json:"checkout_url"`
}

// VerifyPaymentRequest represents payment verification request
type VerifyPaymentRequest struct {
	OrderID   string `json:"razorpay_order_id" binding:"required"`
	PaymentID string `json:"razorpay_payment_id" binding:"required"`
	Signature string `json:"razorpay_signature" binding:"required"`
}

// VerifyPaymentResponse represents payment verification response
type VerifyPaymentResponse struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	PaymentID       string `json:"payment_id,omitempty"`
	OrderID         string `json:"order_id,omitempty"`
	AWBNumber       string `json:"awb_number,omitempty"`
	TrackingURL     string `json:"tracking_url,omitempty"`
	InvoiceURL      string `json:"invoice_url,omitempty"`
	WhatsAppMessage string `json:"whatsapp_message,omitempty"`
	// FulfillmentStatus reflects the post-payment shipment state ("fulfilled",
	// "pending_shipment", "shipment_failed", "needs_manual_fulfillment"). ShipmentPending is
	// true whenever a shipment/tracking is not yet available, so the client can reassure the
	// customer ("payment received, arranging delivery") without implying the payment failed.
	FulfillmentStatus string `json:"fulfillment_status,omitempty"`
	ShipmentPending   bool   `json:"shipment_pending"`
	// PaymentPending is true when the signature is valid but Razorpay has not
	// reported the payment as "captured" yet (authorized-only, created, or the
	// status could not be fetched). No order is created in that case; the
	// order.paid/payment.captured webhook fulfills once capture lands.
	PaymentPending bool `json:"payment_pending"`
}

// CheckoutPage serves a minimal Razorpay Checkout HTML to initiate payment
// @Summary Hosted checkout page
// @Description Render a minimal Razorpay Checkout page for a given Razorpay order
// @Tags Payments
// @Produce html
// @Security FirebaseAuth
// @Param order_id query string true "Razorpay Order ID"
// @Success 200 {string} string "HTML checkout page"
// @Failure 400 {string} string "Missing order_id"
// @Failure 500 {string} string "Failed to fetch order"
// @Router /payments/checkout [get]
func (h *PaymentHandler) CheckoutPage(c *gin.Context) {
	orderID := c.Query("order_id")
	if orderID == "" {
		c.String(http.StatusBadRequest, "order_id query param is required")
		return
	}

	cfg := config.GetRazorpayConfig()

	// Fetch order details on the server to get amount and currency
	rzpOrder, err := h.paymentService.GetOrder(c.Request.Context(), orderID)
	if err != nil {
		h.logger.Error("Failed to fetch Razorpay order for checkout page", zap.Error(err), zap.String("order_id", orderID))
		c.String(http.StatusInternalServerError, fmt.Sprintf("failed to fetch order: %v", err))
		return
	}
	amountRupees := services.PaisaToRupees(rzpOrder.Amount)
	currency := rzpOrder.Currency
	if currency == "" {
		currency = "INR"
	}

	// Minimal HTML template with Razorpay Checkout
	const page = `<!doctype html>
<html>
<head>
	<meta charset="utf-8" />
	<meta name="viewport" content="width=device-width, initial-scale=1" />
	<title>Pay with Razorpay</title>
	<script src="https://checkout.razorpay.com/v1/checkout.js"></script>
	<style>body{font-family:system-ui,-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;padding:24px}button{padding:10px 16px;font-size:16px;cursor:pointer}</style>
	</head>
<body>
		<h2>Razorpay Payment</h2>
		<p>Order: <strong id="orderId">{{.OrderID}}</strong></p>
		<p>Amount: <strong id="amount">{{.Amount}}</strong> <span id="currency">{{.Currency}}</span></p>
	<button id="payBtn">Pay Now</button>
	<pre id="out" style="margin-top:16px;background:#f7f7f7;padding:12px;overflow:auto"></pre>
	<script>
		const out = document.getElementById('out');
		const options = {
			key: '{{.KeyID}}',
			order_id: '{{.OrderID}}',
				amount: {{.AmountPaisa}},
				currency: '{{.Currency}}',
			handler: function (response){
				out.textContent = JSON.stringify(response, null, 2);
				alert('Payment completed. Copy these IDs and call your /payments/verify API.');
			},
			modal: { ondismiss: function(){ out.textContent = 'Checkout dismissed'; } },
			theme: { color: '#3399cc' }
		};
		document.getElementById('payBtn').onclick = function(){
			const rzp = new Razorpay(options);
			rzp.open();
		};
	</script>
</body>
</html>`

	type data struct {
		KeyID, OrderID string
		Amount         float64
		AmountPaisa    int
		Currency       string
	}
	t, err := template.New("checkout").Parse(page)
	if err != nil {
		h.logger.Error("Failed to render checkout template", zap.Error(err))
		c.String(http.StatusInternalServerError, fmt.Sprintf("template error: %v", err))
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := t.Execute(c.Writer, data{KeyID: cfg.KeyID, OrderID: orderID, Amount: amountRupees, AmountPaisa: services.RupeesToPaisa(amountRupees), Currency: currency}); err != nil {
		h.logger.Error("Failed to execute checkout template", zap.Error(err))
	}
}

// CreateOrder creates ONLY a Razorpay payment intent - NO database order yet!
// Database order will be created only after payment verification
// @Summary Create Razorpay order (payment intent)
// @Description Create a Razorpay order for the current user's cart; no DB order yet
// @Tags Payments
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body handlers.CreateOrderRequest true "Payment intent request"
// @Success 200 {object} handlers.CreateOrderResponse "Payment order created"
// @Failure 400 {object} map[string]interface{} "Invalid request or empty cart"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to create payment order"
// @Router /payments/create-order [post]
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	requestID := c.GetString("request_id")
	h.logger.Info("=== CREATING PAYMENT ORDER (RAZORPAY INTENT) ===",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()),
		zap.String("flow_step", "create_order_start"))

	// Get user ID from context (works with both Firebase and JWT auth)
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User authentication failed for payment order creation",
			zap.String("request_id", requestID),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("error_type", "authentication_error"),
			zap.String("flow_step", "authentication_failed"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Map Firebase UID to internal user ID if needed. Before taking payment we must
	// resolve a real local user — refuse rather than attach the order to a raw Firebase
	// UID (#25), which would never match ownership checks and orphan the order.
	if !strings.HasPrefix(userID, "USER") && h.userService != nil {
		resolvedUserID, mapErr := h.mapFirebaseUIDToInternalUserID(c, userID)
		if mapErr != nil || !strings.HasPrefix(resolvedUserID, "USER") {
			h.logger.Error("Rejecting payment order: could not resolve caller to a local user",
				zap.String("request_id", requestID),
				zap.String("firebase_uid", userID),
				zap.Error(mapErr))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "We could not verify your account. Please sign in again and retry."})
			return
		}
		userID = resolvedUserID
	}

	h.logger.Info("User authenticated for payment order creation",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("flow_step", "authentication_success"))

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid payment order request data",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err),
			zap.String("error_type", "validation_error"),
			zap.String("flow_step", "request_binding_failed"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Validate required customer fields
	req.FirstName = strings.TrimSpace(req.FirstName)
	req.LastName = strings.TrimSpace(req.LastName)
	req.Mobile = strings.TrimSpace(req.Mobile)
	req.MobileSecondary = strings.TrimSpace(req.MobileSecondary)
	if req.FirstName == "" || req.Mobile == "" {
		h.logger.Error("Missing required customer fields",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("first_name", req.FirstName),
			zap.String("mobile", req.Mobile))
		c.JSON(http.StatusBadRequest, gin.H{"error": "first_name and mobile are required"})
		return
	}

	h.logger.Info("Customer details validated",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("first_name", req.FirstName),
		zap.String("mobile_length", fmt.Sprintf("%d", len(req.Mobile))))

	h.logger.Info("Customer details validated",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("first_name", req.FirstName),
		zap.String("mobile", req.Mobile))

	// Always use INR currency - require address_id to be provided
	useCurrency := "INR" // Always INR for all transactions
	shippingAddress := ""
	var shippingAddressID string = ""
	finalAmount := 0.0
	shippingCharges := 0.0

	// Require address_id to be provided - no caching
	if req.ShippingAddressID == "" {
		h.logger.Error("Missing shipping address ID in payment order request",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("flow_step", "missing_address_id"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "shipping_address_id is required"})
		return
	}

	h.logger.Info("Processing shipping address for payment order",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("address_id", req.ShippingAddressID),
		zap.String("flow_step", "resolving_address"))

	shippingAddressID = req.ShippingAddressID
	if addr, err := h.orderService.GetAddressByID(c.Request.Context(), userID, req.ShippingAddressID); err == nil && addr != nil {
		shippingAddress = h.orderService.FormatAddress(addr)

		// Gate payment on a real, serviceable address. Previously an empty/garbage address
		// (city/state "Unknown", pincode "000000") was accepted and the customer was charged,
		// after which shipment creation silently failed — leaving a paid order that could not be
		// delivered. Reject such orders BEFORE any Razorpay charge.
		pincode := strings.TrimSpace(addr.PostalCode)
		city := strings.TrimSpace(addr.City)
		state := strings.TrimSpace(addr.State)
		if !pincodeRegex.MatchString(pincode) || pincode == "000000" ||
			city == "" || strings.EqualFold(city, "Unknown") ||
			state == "" || strings.EqualFold(state, "Unknown") {
			h.logger.Warn("Rejecting payment order: incomplete/invalid shipping address",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("address_id", shippingAddressID),
				zap.String("city", city),
				zap.String("state", state),
				zap.String("postal_code", pincode))
			c.JSON(http.StatusBadRequest, gin.H{"error": "A valid shipping address (city, state and a 6-digit pincode) is required before payment."})
			return
		}
		// Confirm the pincode is serviceable BEFORE taking payment (#23: fail-closed).
		// A serviceability-API error is treated the same as "not serviceable" — we do not
		// charge a customer for an order we cannot confirm we can deliver. (The previous
		// fail-open let a Delhivery outage pass checkout, producing paid-but-unshippable
		// orders.) A transient error asks the customer to retry rather than silently proceed.
		if ok, svcErr := h.deliveryService.CheckServiceability(c.Request.Context(), pincode); svcErr != nil {
			h.logger.Warn("Rejecting payment order: serviceability check errored (fail-closed)",
				zap.String("request_id", requestID), zap.String("user_id", userID),
				zap.String("pincode", pincode), zap.Error(svcErr))
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "We could not confirm delivery to this pincode right now. Please try again in a few minutes."})
			return
		} else if !ok {
			h.logger.Warn("Rejecting payment order: pincode not serviceable",
				zap.String("request_id", requestID), zap.String("user_id", userID), zap.String("pincode", pincode))
			c.JSON(http.StatusBadRequest, gin.H{"error": "We currently cannot deliver to this pincode. Please use a different delivery address."})
			return
		}

		h.logger.Info("Shipping address resolved successfully",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("address_id", shippingAddressID),
			zap.String("address", shippingAddress),
			zap.String("city", addr.City),
			zap.String("state", addr.State),
			zap.String("postal_code", addr.PostalCode),
			zap.String("flow_step", "address_resolved"))
	} else {
		h.logger.Error("Failed to resolve shipping address for payment order",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("address_id", req.ShippingAddressID),
			zap.Error(err),
			zap.String("error_type", "address_resolution_error"),
			zap.String("flow_step", "address_resolution_failed"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shipping_address_id"})
		return
	}

	h.logger.Info("Starting cart retrieval for payment calculation",
		zap.String("request_id", requestID),
		zap.String("user_id", userID))

	// Step 1: Get user's cart to calculate total (NO order creation yet!)
	cart, err := h.orderService.GetUserCart(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to retrieve user cart for payment order",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("error_type", "cart_retrieval_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart"})
		return
	}

	if len(cart.Items) == 0 {
		h.logger.Error("User cart is empty - cannot create payment order",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("error_type", "empty_cart_error"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cart is empty"})
		return
	}

	h.logger.Info("Cart retrieved successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("cart_id", cart.ID),
		zap.Int("item_count", len(cart.Items)))

	// Step 2: Calculate totals dynamically from cart
	cartTotal := 0.0
	for _, item := range cart.Items {
		cartTotal += item.TotalPrice
	}
	cartTotal = utils.RoundToTwoDecimals(cartTotal)

	// Apply discount if coupon was applied
	discountedCartTotal := utils.ApplyDiscount(cartTotal, cart.DiscountAmount)

	h.logger.Info("Cart total calculated",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.Float64("cart_total", cartTotal),
		zap.Float64("discount_amount", cart.DiscountAmount),
		zap.Float64("discounted_cart_total", discountedCartTotal),
		zap.String("coupon_code", cart.CouponCode))

	// Force INR currency always
	req.Currency = useCurrency // Always "INR"

	// Track shipping source for notes
	var shippingSource string

	// Calculate totals from cart
	if finalAmount <= 0 || (req.Subtotal > 0) {
		// Log the shipping cost provided in the request
		h.logger.Info("Shipping cost analysis",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Float64("provided_shipping_cost", req.ShippingCost),
			zap.Float64("provided_subtotal", req.Subtotal),
			zap.Bool("has_shipping_cost", req.ShippingCost > 0),
			zap.Bool("has_subtotal", req.Subtotal > 0))

		// SECURITY (#1 amount integrity): the goods subtotal is ALWAYS the discounted
		// server cart total, and shipping is ALWAYS recomputed server-side. Client-supplied
		// req.Subtotal / req.ShippingCost are treated as display hints only and never
		// determine the charge, so a tampered subtotal or shipping value cannot reduce the
		// amount actually charged (previously `finalAmount = req.Subtotal` let a client pay
		// any amount they liked for the real cart).
		shippingCharges = h.calculateShippingChargesRobust(c.Request.Context(), userID, shippingAddress, discountedCartTotal, len(cart.Items))
		finalAmount = utils.AddShippingCharges(discountedCartTotal, shippingCharges)
		shippingSource = "server_calculation"

		if req.ShippingCost > 0 && math.Abs(req.ShippingCost-shippingCharges) > 0.01 {
			h.logger.Warn("Client shipping_cost differs from server calculation; using server value",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Float64("client_shipping_cost", req.ShippingCost),
				zap.Float64("server_shipping", shippingCharges))
		}
		if req.Subtotal > 0 && math.Abs(req.Subtotal-finalAmount) > 0.01 {
			h.logger.Warn("Client subtotal differs from server-computed amount; ignoring client value",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Float64("client_subtotal", req.Subtotal),
				zap.Float64("server_final_amount", finalAmount))
		}

		h.logger.Info("Shipping charges calculated",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Float64("original_cart_total", cartTotal),
			zap.Float64("discounted_cart_total", discountedCartTotal),
			zap.Float64("shipping_charges", shippingCharges),
			zap.Float64("final_amount", finalAmount))
	}

	// Apply business rule: Free delivery on cart total of 500 or more
	// Assumption: threshold is inclusive (>= 500)
	// Note: Free delivery is based on original cart total, not discounted total
	freeDelivery := false
	if cartTotal >= 500 {
		freeDelivery = true
		shippingCharges = 0
		finalAmount = discountedCartTotal // Use discounted total for final amount
		h.logger.Info("Free delivery threshold met",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Float64("original_cart_total", cartTotal),
			zap.Float64("discounted_cart_total", discountedCartTotal),
			zap.Float64("threshold", 500),
			zap.Bool("free_delivery", true),
			zap.Float64("adjusted_final_amount", finalAmount))
	}

	h.logger.Info("Final payment amounts calculated",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.Float64("original_cart_total", cartTotal),
		zap.Float64("discount_amount", cart.DiscountAmount),
		zap.Float64("discounted_cart_total", discountedCartTotal),
		zap.Float64("shipping_charges", shippingCharges),
		zap.Float64("final_amount", finalAmount),
		zap.String("coupon_code", cart.CouponCode),
		zap.Bool("free_delivery", freeDelivery))

	// Step 3: Create ONLY Razorpay payment order (NO database order!)
	// Store essential information in notes (max 15 fields for Razorpay)
	notes := map[string]interface{}{
		"user_id":          userID,
		"shipping_addr_id": shippingAddressID, // Use resolved address ID
		"cart_total":       cartTotal,
		"discount":         cart.DiscountAmount,
		"shipping":         shippingCharges,
		"final_amount":     finalAmount, // Total including shipping
		"items_count":      len(cart.Items),
		"currency":         "INR",                                                                             // Always INR
		"customer_info":    req.FirstName + "," + req.LastName + "," + req.Mobile + "," + req.MobileSecondary, // Combine customer fields
		"shipping_source":  shippingSource,                                                                    // Track shipping calculation method
	}

	// Add structured address data to notes for clean extraction during verification
	// Combine address fields to stay within 15-field limit
	if addr, err := h.orderService.GetAddressByID(c.Request.Context(), userID, shippingAddressID); err == nil && addr != nil {
		// Combine all address components into one field
		notes["address"] = addr.AddressLine1 + "|" + addr.City + "|" + addr.State + "|" + addr.PostalCode
	}

	// Add free delivery flag only if true (conditional field)
	if freeDelivery {
		notes["free_delivery"] = true
	}

	h.logger.Info("Creating Razorpay payment order",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.Float64("amount", finalAmount),
		zap.String("currency", req.Currency))

	razorpayOrder, err := h.paymentService.CreateOrder(c.Request.Context(), userID, finalAmount, req.Currency, "payment_intent_"+userID, notes)
	if err != nil {
		h.logger.Error("Failed to create Razorpay payment order",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("user_id", userID),
			zap.Float64("amount", finalAmount),
			zap.String("currency", req.Currency),
			zap.String("error_type", "payment_service_error"))
		// SECURITY (#28): error already logged above; do not leak internal/upstream
		// error text to the client.
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment order"})
		return
	}

	h.logger.Info("Razorpay payment order created successfully",
		zap.String("request_id", requestID),
		zap.String("razorpay_order_id", razorpayOrder.ID),
		zap.String("user_id", userID),
		zap.Int("amount_paisa", razorpayOrder.Amount))

	// Convert amount back to rupees for response
	amountInRupees := services.PaisaToRupees(razorpayOrder.Amount)

	cfg := config.GetRazorpayConfig()
	// Build absolute checkout URL for convenience (after order is created)
	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := c.Request.Host
	checkoutURL := fmt.Sprintf("%s://%s/api/v1/payments/checkout?order_id=%s", scheme, host, razorpayOrder.ID)

	description := "Kisanlink Order Payment"
	if freeDelivery {
		description = "Kisanlink Order Payment - Free Delivery"
	}

	response := CreateOrderResponse{
		OrderID:     razorpayOrder.ID,
		Amount:      amountInRupees,
		Currency:    razorpayOrder.Currency,
		KeyID:       cfg.KeyID,
		Description: description,
		Notes:       notes,
		CheckoutURL: checkoutURL,
	}

	h.logger.Info("Payment order response prepared - NO database order created yet",
		zap.String("request_id", requestID),
		zap.String("razorpay_order_id", razorpayOrder.ID),
		zap.String("user_id", userID),
		zap.Float64("amount_rupees", amountInRupees),
		zap.Float64("shipping_charges", shippingCharges),
		zap.String("checkout_url", checkoutURL),
		zap.Bool("free_delivery", freeDelivery))

	c.JSON(http.StatusOK, response)
}

// VerifyPayment verifies Razorpay payment signature
// @Summary Verify payment and create order
// @Description Verify Razorpay signature, create DB order, and create shipment
// @Tags Payments
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body handlers.VerifyPaymentRequest true "Verify payment payload"
// @Success 200 {object} handlers.VerifyPaymentResponse "Order created (and shipment if successful)"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Verification failed"
// @Router /payments/verify [post]
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	requestID := c.GetString("request_id")
	h.logger.Info("Payment verification requested",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()))

	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User authentication failed for payment verification",
			zap.String("request_id", requestID),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("error_type", "authentication_error"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Map Firebase UID to internal user ID. Normally the user already exists (it was
	// resolved/created at create-order), so this resolves on the first lookup. We must
	// NOT fall back to the raw Firebase UID as the order owner (#25) — that pollutes
	// ownership. If resolution genuinely fails (near-impossible after a successful
	// create-order), do not silently mis-own the order: report that the captured payment
	// will be reconciled (the order.paid/payment.captured webhook, bound to the
	// server-set notes.user_id, remains the safety net).
	internalUserID := userID
	if !strings.HasPrefix(userID, "USER") && h.userService != nil {
		resolved, mapErr := h.mapFirebaseUIDToInternalUserID(c, userID)
		if mapErr != nil || !strings.HasPrefix(resolved, "USER") {
			h.logger.Error("VerifyPayment: could not resolve caller to a local user; refusing to own the order with a Firebase UID",
				zap.String("request_id", requestID),
				zap.String("firebase_uid", userID),
				zap.Error(mapErr))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment received. We could not finalize your order automatically; it will be reconciled shortly."})
			return
		}
		internalUserID = resolved
		h.logger.Info("Mapped Firebase UID to internal user ID for payment verification",
			zap.String("firebase_uid", userID),
			zap.String("internal_id", internalUserID))
	}

	var req VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid payment verification request data",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err),
			zap.String("error_type", "validation_error"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Enhanced logging for debugging
	h.logger.Info("Payment verification details received",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("razorpay_order_id", req.OrderID),
		zap.String("payment_id", req.PaymentID),
		zap.Int("signature_length", len(req.Signature)))

	// Verify payment signature first with enhanced error handling
	h.logger.Info("Starting payment signature verification",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("order_id", req.OrderID),
		zap.String("payment_id", req.PaymentID))

	isValid, err := h.paymentService.VerifyPayment(c.Request.Context(), req.OrderID, req.PaymentID, req.Signature)
	if err != nil {
		h.logger.Error("Payment verification API error",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("order_id", req.OrderID),
			zap.String("payment_id", req.PaymentID),
			zap.String("error_type", "payment_api_error"))
		// SECURITY (#28): error already logged above; do not leak internal details.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Payment verification API failed",
			"code":  "PAYMENT_VERIFICATION_ERROR"})
		return
	}

	// Log verification result
	h.logger.Info("Payment signature verification completed",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("order_id", req.OrderID),
		zap.Bool("is_valid", isValid),
		zap.String("payment_id", req.PaymentID))

	var response VerifyPaymentResponse
	if isValid {
		h.logger.Info("Payment verified successfully - proceeding to create database order",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("payment_id", req.PaymentID),
			zap.String("razorpay_order_id", req.OrderID))

		// Fetch Razorpay payment to get customer details (contact, email, address)
		// Magic Checkout stores address and customer details in payment entity
		h.logger.Info("Fetching Razorpay payment for customer details",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("payment_id", req.PaymentID))

		rzpPayment, paymentErr := h.paymentService.GetPaymentStatus(c.Request.Context(), req.PaymentID)
		if paymentErr != nil {
			h.logger.Warn("Failed to fetch Razorpay payment - proceeding without payment metadata",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Error(paymentErr))
		}

		// #12 Captured-payment gate. A valid signature only proves the
		// (order_id, payment_id) pair came from Razorpay; it does NOT prove the
		// money was captured. Do not create an order or ship on an
		// authorized-only / created payment, nor when the status could not be
		// fetched (Razorpay outage). The order.paid / payment.captured webhook
		// is the safety net that fulfills once capture actually lands, so this
		// is a conservative 200, not a failure the customer should worry about.
		if rzpPayment == nil || rzpPayment.Status != "captured" {
			status := "unknown"
			if rzpPayment != nil {
				status = rzpPayment.Status
			}
			h.logger.Warn("Payment signature valid but not captured - deferring fulfillment to webhook",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("payment_id", req.PaymentID),
				zap.String("razorpay_order_id", req.OrderID),
				zap.String("payment_status", status))
			c.JSON(http.StatusOK, VerifyPaymentResponse{
				Success:         false,
				Message:         "Payment received and being confirmed. Your order will be arranged once the payment is captured.",
				PaymentID:       req.PaymentID,
				OrderID:         req.OrderID,
				PaymentPending:  true,
				ShipmentPending: true,
			})
			return
		}

		// Fetch Razorpay order to get notes (subtotal, cart_total, shipping_charges, shipping_address)
		h.logger.Info("Fetching Razorpay order metadata",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("razorpay_order_id", req.OrderID))

		// IMPORTANT: Fetch the complete order details from Razorpay API
		// After Magic Checkout completes, the order will contain customer_details with addresses
		rzpOrder, getErr := h.paymentService.GetOrder(c.Request.Context(), req.OrderID)
		if getErr != nil {
			h.logger.Warn("Failed to fetch Razorpay order metadata - proceeding without metadata",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Error(getErr))
		}

		// DEBUG: Log the FULL Razorpay order response to understand what's being returned
		if rzpOrder != nil {
			h.logger.Info("DEBUG: Full Razorpay Order Response",
				zap.String("request_id", requestID),
				zap.String("razorpay_order_id", rzpOrder.ID),
				zap.String("status", rzpOrder.Status),
				zap.Int("amount", rzpOrder.Amount),
				zap.String("receipt", rzpOrder.Receipt),
				zap.Bool("has_customer_details", rzpOrder.CustomerDetails != nil),
				zap.Bool("has_checkout", rzpOrder.Checkout != nil),
				zap.Any("checkout_data", rzpOrder.Checkout),
				zap.Any("notes", rzpOrder.Notes))

			if rzpOrder.CustomerDetails != nil {
				// SECURITY (#27): log presence only, never customer PII (email/contact/address).
				h.logger.Info("DEBUG: Razorpay CustomerDetails found",
					zap.String("request_id", requestID),
					zap.Bool("has_shipping_address", rzpOrder.CustomerDetails.ShippingAddress != nil),
					zap.Bool("has_billing_address", rzpOrder.CustomerDetails.BillingAddress != nil))

				if rzpOrder.CustomerDetails.ShippingAddress == nil {
					h.logger.Warn("DEBUG: CustomerDetails exists but ShippingAddress is NULL",
						zap.String("request_id", requestID))
				}
			} else {
				h.logger.Warn("DEBUG: Razorpay order has NO customer_details - Magic Checkout may not be enabled or address not selected",
					zap.String("request_id", requestID),
					zap.String("razorpay_order_id", rzpOrder.ID))
			}
		} else {
			h.logger.Error("DEBUG: Razorpay order is NULL",
				zap.String("request_id", requestID),
				zap.String("razorpay_order_id", req.OrderID))
		}

		// Derive shipping values from notes
		notes := map[string]interface{}{}
		var shippingAddress string = ""
		var shippingAddressID string = ""
		var subtotal float64 = 0
		var cartTotal float64 = 0
		var shippingCharges float64 = 0
		var etaMin, etaMax int = 0, 0
		var firstName, lastName, mobile, mobileSecondary, email string

		// Priority 1: Check if Razorpay order has customer_details from Magic Checkout
		var magicCheckoutAddress map[string]interface{}
		if rzpOrder != nil && rzpOrder.CustomerDetails != nil && rzpOrder.CustomerDetails.ShippingAddress != nil {
			// SECURITY (#27): log presence only, never customer PII.
			h.logger.Info("Found customer_details from Magic Checkout in Razorpay order",
				zap.String("request_id", requestID),
				zap.String("razorpay_order_id", req.OrderID))

			// Extract customer details from Magic Checkout
			email = rzpOrder.CustomerDetails.Email
			mobile = rzpOrder.CustomerDetails.Contact

			// Extract customer name from shipping address
			shippingAddr := rzpOrder.CustomerDetails.ShippingAddress
			if shippingAddr.Name != "" {
				nameParts := strings.Fields(strings.TrimSpace(shippingAddr.Name))
				if len(nameParts) > 0 {
					firstName = nameParts[0]
					if len(nameParts) > 1 {
						lastName = strings.Join(nameParts[1:], " ")
					}
				}
			}

			// Convert structured address to map for processing
			magicCheckoutAddress = map[string]interface{}{
				"line1":   shippingAddr.Line1,
				"line2":   shippingAddr.Line2,
				"city":    shippingAddr.City,
				"state":   shippingAddr.State,
				"zipcode": shippingAddr.Zipcode,
				"country": shippingAddr.Country,
				"name":    shippingAddr.Name,
				"contact": shippingAddr.Contact,
			}

			// SECURITY (#27): confirm extraction without logging the address PII.
			h.logger.Info("Extracted Magic Checkout address from Razorpay order",
				zap.String("request_id", requestID),
				zap.Bool("has_address_line1", shippingAddr.Line1 != ""),
				zap.Bool("has_zipcode", shippingAddr.Zipcode != ""))

			// Also update shipping fee from order if available
			if rzpOrder.ShippingFee > 0 {
				shippingCharges = float64(rzpOrder.ShippingFee) / 100.0 // Convert from paise to rupees
			}

			// Priority 2: Check if we have stored Magic Checkout address from shipping-info or callback
		} else {
			// Try multiple key variations since Razorpay may send order ID with or without prefix
			// IMPORTANT: Try all possible formats - the address might be stored under any of these
			keysToTry := []string{
				req.OrderID, // order_RmMkQm9FiEAFC1
				strings.TrimPrefix(req.OrderID, "order_"),            // RmMkQm9FiEAFC1
				"order_" + strings.TrimPrefix(req.OrderID, "order_"), // ensure order_ prefix
			}

			// Also try receipt ID if available (e.g., "mc_USER123_1234567890")
			if rzpOrder != nil && rzpOrder.Receipt != "" {
				keysToTry = append(keysToTry, rzpOrder.Receipt)
			}

			h.logger.Info("Attempting to retrieve stored Magic Checkout address",
				zap.String("request_id", requestID),
				zap.Strings("keys_to_try", keysToTry))

			var storedAddressData map[string]interface{}
			var foundKey string
			for _, key := range keysToTry {
				if data, exists := GetMagicCheckoutAddress(key); exists {
					storedAddressData = data
					foundKey = key
					break
				}
			}

			if storedAddressData != nil {
				h.logger.Info("SUCCESS: Found stored Magic Checkout address",
					zap.String("request_id", requestID),
					zap.String("found_key", foundKey),
					zap.String("razorpay_order_id", req.OrderID),
					zap.Any("address_data", storedAddressData))

				// Extract the shipping address from stored data
				if shippingAddr, ok := storedAddressData["shipping_address"].(map[string]interface{}); ok {
					magicCheckoutAddress = shippingAddr

					// IMPORTANT: Extract customer name from shipping address
					if name, ok := shippingAddr["name"].(string); ok && name != "" {
						nameParts := strings.Fields(strings.TrimSpace(name))
						if len(nameParts) > 0 {
							firstName = nameParts[0]
							if len(nameParts) > 1 {
								lastName = strings.Join(nameParts[1:], " ")
							}
						}
						h.logger.Info("Extracted customer name from shipping address",
							zap.String("request_id", requestID),
							zap.String("full_name", name),
							zap.String("firstName", firstName),
							zap.String("lastName", lastName))
					}
				}

				// Extract customer details from stored data
				if storedEmail, ok := storedAddressData["email"].(string); ok && storedEmail != "" {
					email = storedEmail
				}
				if storedContact, ok := storedAddressData["contact"].(string); ok && storedContact != "" {
					mobile = storedContact
				}

				// Extract customer details from customer_details if present
				if custDetails, ok := storedAddressData["customer_details"].(map[string]interface{}); ok {
					if name, ok := custDetails["name"].(string); ok && name != "" {
						nameParts := strings.Fields(strings.TrimSpace(name))
						if len(nameParts) > 0 {
							firstName = nameParts[0]
							if len(nameParts) > 1 {
								lastName = strings.Join(nameParts[1:], " ")
							}
						}
					}
					if fn, ok := custDetails["first_name"].(string); ok && fn != "" {
						firstName = fn
					}
					if ln, ok := custDetails["last_name"].(string); ok && ln != "" {
						lastName = ln
					}
				}
			} else {
				h.logger.Warn("No stored Magic Checkout address found",
					zap.String("request_id", requestID),
					zap.Strings("keys_tried", keysToTry))
			}
		}

		// Extract customer details from payment entity (Magic Checkout stores them here)
		if rzpPayment != nil {
			// SECURITY (#27): log presence/method only, never notes/contact/email PII.
			h.logger.Info("Razorpay Payment Full Details",
				zap.String("request_id", requestID),
				zap.Any("payment_method", rzpPayment.Method),
				zap.Bool("has_notes", len(rzpPayment.Notes) > 0),
				zap.Bool("has_contact", rzpPayment.Contact != ""),
				zap.Bool("has_email", rzpPayment.Email != ""),
				zap.Bool("has_customer_details", rzpPayment.CustomerDetails != nil))

			if rzpPayment.Contact != "" {
				mobile = rzpPayment.Contact
			}
			if rzpPayment.Email != "" {
				email = rzpPayment.Email
			}

			// PRIORITY: Check for customer_details in payment (Magic Checkout)
			// This is the most reliable source for shipping address after payment
			if rzpPayment.CustomerDetails != nil && rzpPayment.CustomerDetails.ShippingAddress != nil {
				addr := rzpPayment.CustomerDetails.ShippingAddress
				// SECURITY (#27): presence only, no address PII.
				h.logger.Info("Found customer_details.shipping_address in payment (Magic Checkout)",
					zap.String("request_id", requestID),
					zap.Bool("has_line1", addr.Line1 != ""),
					zap.Bool("has_zipcode", addr.Zipcode != ""))

				// Convert to map for consistent processing
				magicCheckoutAddress = map[string]interface{}{
					"name":    addr.Name,
					"line1":   addr.Line1,
					"line2":   addr.Line2,
					"city":    addr.City,
					"state":   addr.State,
					"zipcode": addr.Zipcode,
					"country": addr.Country,
					"contact": addr.Contact,
				}

				// Also extract customer name if not already set
				if firstName == "" && addr.Name != "" {
					nameParts := strings.Fields(strings.TrimSpace(addr.Name))
					if len(nameParts) > 0 {
						firstName = nameParts[0]
						if len(nameParts) > 1 {
							lastName = strings.Join(nameParts[1:], " ")
						}
					}
					// SECURITY (#27): confirm extraction without logging the name.
					h.logger.Info("Extracted customer name from payment.customer_details",
						zap.String("request_id", requestID),
						zap.Bool("has_name", firstName != ""))
				}
			}

			// Magic Checkout stores shipping address in payment metadata
			// Check for shipping_address in payment notes/metadata (fallback)
			if magicCheckoutAddress == nil && rzpPayment.Notes != nil {
				// SECURITY (#27): log only the note keys present, not their values (PII).
				noteKeys := make([]string, 0, len(rzpPayment.Notes))
				for key := range rzpPayment.Notes {
					noteKeys = append(noteKeys, key)
				}
				h.logger.Info("Payment notes present",
					zap.String("request_id", requestID),
					zap.Strings("note_keys", noteKeys))

				// Try to extract Magic Checkout shipping address
				if shippingAddr, ok := rzpPayment.Notes["shipping_address"].(map[string]interface{}); ok {
					magicCheckoutAddress = shippingAddr
					h.logger.Info("Found Magic Checkout shipping address in payment notes",
						zap.String("request_id", requestID))
				} else if addrNote, ok := rzpPayment.Notes["address"].(string); ok && addrNote != "" {
					// Fallback to string address
					shippingAddress = addrNote
					h.logger.Info("Using address string from payment notes",
						zap.String("request_id", requestID),
						zap.Bool("has_address", shippingAddress != ""))
				}
			}

			// Check if Magic Checkout sent address through Method field (for card payments)
			if rzpPayment.Method == "card" && magicCheckoutAddress == nil {
				// Check for billing address in card details
				if methodData, ok := rzpPayment.Notes["method"].(map[string]interface{}); ok {
					if cardData, ok := methodData["card"].(map[string]interface{}); ok {
						// SECURITY (#27): never log card data; presence only.
						h.logger.Info("Found card data in payment method",
							zap.String("request_id", requestID))
						// Extract billing address from card data
						magicCheckoutAddress = cardData
					}
				}
			}
		}
		if rzpOrder != nil && rzpOrder.Notes != nil {
			notes = rzpOrder.Notes

			// Log all order notes to understand structure
			h.logger.Info("Razorpay Order Full Notes",
				zap.String("request_id", requestID),
				zap.Any("order_notes", notes))
			// Note: shipping_address removed from notes to reduce field count
			if v, ok := notes["shipping_addr_id"].(string); ok {
				shippingAddressID = v
			}
			if v, ok := notes["final_amount"].(float64); ok {
				subtotal = v
			} else if v, ok := notes["final_amount"].(int); ok {
				subtotal = float64(v)
			}
			if v, ok := notes["cart_total"].(float64); ok {
				cartTotal = v
			} else if v, ok := notes["cart_total"].(int); ok {
				cartTotal = float64(v)
			}
			if v, ok := notes["shipping"].(float64); ok {
				shippingCharges = v
			} else if v, ok := notes["shipping"].(int); ok {
				shippingCharges = float64(v)
			}
			if shippingCharges == 0 && subtotal > 0 && cartTotal >= 0 {
				shippingCharges = subtotal - cartTotal
			}
			if v, ok := notes["eta_min_days"].(int); ok {
				etaMin = v
			} else if v, ok := notes["eta_min_days"].(float64); ok {
				etaMin = int(v)
			}
			if v, ok := notes["eta_max_days"].(int); ok {
				etaMax = v
			} else if v, ok := notes["eta_max_days"].(float64); ok {
				etaMax = int(v)
			}

			// Extract customer info from combined field: "firstName,lastName,mobile,mobileAlt"
			if customerInfo, ok := notes["customer_info"].(string); ok && customerInfo != "" {
				parts := strings.Split(customerInfo, ",")
				if len(parts) >= 4 {
					firstName = strings.TrimSpace(parts[0])
					lastName = strings.TrimSpace(parts[1])
					mobile = strings.TrimSpace(parts[2])
					mobileSecondary = strings.TrimSpace(parts[3])
				} else if len(parts) >= 3 {
					firstName = strings.TrimSpace(parts[0])
					lastName = strings.TrimSpace(parts[1])
					mobile = strings.TrimSpace(parts[2])
					mobileSecondary = ""
				} else if len(parts) >= 2 {
					firstName = strings.TrimSpace(parts[0])
					lastName = strings.TrimSpace(parts[1])
					mobile = ""
					mobileSecondary = ""
				} else if len(parts) >= 1 {
					firstName = strings.TrimSpace(parts[0])
					lastName = ""
					mobile = ""
					mobileSecondary = ""
				}
			}

			h.logger.Info("Extracted order metadata from Razorpay notes",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Float64("subtotal", subtotal),
				zap.Float64("cart_total", cartTotal),
				zap.Float64("shipping_charges", shippingCharges),
				zap.String("first_name", firstName),
				zap.String("mobile", mobile),
				zap.String("shipping_address_id", shippingAddressID))
		}

		// If address string missing but address ID present, resolve again (safety)
		if shippingAddress == "" && shippingAddressID != "" {
			h.logger.Info("Resolving shipping address from ID",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("address_id", shippingAddressID))

			if addr, err := h.orderService.GetAddressByID(c.Request.Context(), userID, shippingAddressID); err == nil && addr != nil {
				shippingAddress = h.orderService.FormatAddress(addr)
				h.logger.Info("Shipping address resolved from ID",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("address", shippingAddress))
			} else {
				h.logger.Warn("Failed to resolve shipping address from ID",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("address_id", shippingAddressID),
					zap.Error(err))
			}
		}
		if shippingCharges < 0 {
			shippingCharges = 0
		}

		// Create order - simplified approach, always create new order
		h.logger.Info("Creating database order",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("razorpay_order_id", req.OrderID),
			zap.Float64("shipping_charges", shippingCharges))

		// Create normalized customer and address data for the order
		// If we still don't have a name, try to extract from email
		if firstName == "" && email != "" {
			// Use part before @ as the name
			emailParts := strings.Split(email, "@")
			if len(emailParts) > 0 {
				namePart := strings.TrimSpace(emailParts[0])
				// Replace common separators with spaces and split
				namePart = strings.ReplaceAll(namePart, ".", " ")
				namePart = strings.ReplaceAll(namePart, "_", " ")
				namePart = strings.ReplaceAll(namePart, "-", " ")
				nameParts := strings.Fields(namePart)
				if len(nameParts) > 0 {
					firstName = strings.Title(nameParts[0])
					if len(nameParts) > 1 {
						lastName = strings.Title(strings.Join(nameParts[1:], " "))
					}
					// SECURITY (#27): confirm extraction without logging email/name PII.
					h.logger.Info("Extracted name from email",
						zap.String("request_id", requestID),
						zap.Bool("has_name", firstName != ""))
				}
			}
		}

		// Provide default values if still missing
		if firstName == "" {
			firstName = "Guest" // Default value for required field
		}
		if lastName == "" {
			lastName = "Customer" // Default value for required field
		}

		customerData := models.NewOrderCustomer(
			firstName,
			lastName,
			mobile,
			mobileSecondary, // phoneAlt from customer_info
			email,           // email from payment entity
		)

		// Create normalized address data - extract structured address from Razorpay notes
		var addressData *models.OrderAddress
		var userAddressID string // To store saved address ID for user

		// DEBUG: Log what address data we have from all sources
		// SECURITY (#27): log source presence only, never the address map (PII).
		h.logger.Info("ADDRESS DATA SUMMARY - All Sources",
			zap.String("request_id", requestID),
			zap.Bool("has_magic_checkout_address", len(magicCheckoutAddress) > 0),
			zap.Bool("has_rzp_order", rzpOrder != nil),
			zap.Bool("has_rzp_order_customer_details", rzpOrder != nil && rzpOrder.CustomerDetails != nil),
			zap.Bool("has_rzp_payment", rzpPayment != nil),
			zap.Bool("has_rzp_payment_customer_details", rzpPayment != nil && rzpPayment.CustomerDetails != nil),
			zap.String("razorpay_order_id", req.OrderID))

		// Priority 1: Check for Magic Checkout address data from payment
		if len(magicCheckoutAddress) > 0 {
			h.logger.Info("Processing Magic Checkout address",
				zap.String("request_id", requestID))

			// Extract address fields from Magic Checkout data
			addressLine1 := ""
			addressLine2 := ""
			city := ""
			state := ""
			postalCode := ""
			country := "India"

			// Try to extract customer name from Magic Checkout address (if available)
			if name, ok := magicCheckoutAddress["name"].(string); ok && name != "" {
				nameParts := strings.Fields(strings.TrimSpace(name))
				if len(nameParts) > 0 && firstName == "" {
					firstName = nameParts[0]
					if len(nameParts) > 1 {
						lastName = strings.Join(nameParts[1:], " ")
					}
					// SECURITY (#27): no name PII in logs.
					h.logger.Info("Extracted customer name from Magic Checkout address",
						zap.String("request_id", requestID),
						zap.Bool("has_name", firstName != ""))
				}
			}

			// Try to extract phone from Magic Checkout address (if available)
			if phone, ok := magicCheckoutAddress["phone"].(string); ok && phone != "" && mobile == "" {
				mobile = strings.TrimSpace(phone)
				// SECURITY (#27): no phone PII in logs.
				h.logger.Info("Extracted phone from Magic Checkout address",
					zap.String("request_id", requestID))
			} else if contact, ok := magicCheckoutAddress["contact"].(string); ok && contact != "" && mobile == "" {
				mobile = strings.TrimSpace(contact)
				// SECURITY (#27): no contact PII in logs.
				h.logger.Info("Extracted contact from Magic Checkout address",
					zap.String("request_id", requestID))
			}

			// Magic Checkout may send fields like: line1, line2, city, state, zipcode, country
			if line1, ok := magicCheckoutAddress["line1"].(string); ok {
				addressLine1 = strings.TrimSpace(line1)
			} else if line1, ok := magicCheckoutAddress["address_line_1"].(string); ok {
				addressLine1 = strings.TrimSpace(line1)
			} else if line1, ok := magicCheckoutAddress["address"].(string); ok {
				addressLine1 = strings.TrimSpace(line1)
			}

			if line2, ok := magicCheckoutAddress["line2"].(string); ok {
				addressLine2 = strings.TrimSpace(line2)
			} else if line2, ok := magicCheckoutAddress["address_line_2"].(string); ok {
				addressLine2 = strings.TrimSpace(line2)
			}

			if c, ok := magicCheckoutAddress["city"].(string); ok {
				city = strings.TrimSpace(c)
			}

			if s, ok := magicCheckoutAddress["state"].(string); ok {
				state = strings.TrimSpace(s)
			} else if s, ok := magicCheckoutAddress["state_code"].(string); ok {
				state = strings.TrimSpace(s)
			}

			if zip, ok := magicCheckoutAddress["zipcode"].(string); ok {
				postalCode = strings.TrimSpace(zip)
			} else if zip, ok := magicCheckoutAddress["postal_code"].(string); ok {
				postalCode = strings.TrimSpace(zip)
			} else if zip, ok := magicCheckoutAddress["pincode"].(string); ok {
				postalCode = strings.TrimSpace(zip)
			}

			if c, ok := magicCheckoutAddress["country"].(string); ok {
				country = strings.TrimSpace(c)
			}

			// Validate that we have meaningful address data before creating OrderAddress
			// At minimum we need: (addressLine1 OR city) AND postalCode
			hasValidAddress := (addressLine1 != "" || city != "") && postalCode != ""

			if hasValidAddress {
				// Create OrderAddress from Magic Checkout data
				addressData = models.NewOrderAddress(
					addressLine1,
					addressLine2,
					city,
					state,
					postalCode,
					country,
				)

				// Also save this address to user's address book for future use
				if h.addressService != nil && internalUserID != "" && addressLine1 != "" {
					// Use FindOrCreate to avoid duplicates and save to user's address book
					savedAddr, addrErr := h.addressService.FindOrCreateAddress(
						c.Request.Context(),
						internalUserID,
						addressLine1,
						city,
						state,
						postalCode,
						country,
					)

					if addrErr == nil && savedAddr != nil {
						userAddressID = savedAddr.ID
						h.logger.Info("Magic Checkout address saved to user's address book",
							zap.String("request_id", requestID),
							zap.String("user_id", internalUserID),
							zap.String("address_id", userAddressID))
					} else {
						h.logger.Warn("Failed to save Magic Checkout address to user's address book",
							zap.String("request_id", requestID),
							zap.Error(addrErr))
					}
				}

				h.logger.Info("Created OrderAddress from Magic Checkout data",
					zap.String("request_id", requestID),
					zap.String("address_line_1", addressLine1),
					zap.String("city", city),
					zap.String("state", state),
					zap.String("postal_code", postalCode))
			} else {
				h.logger.Warn("Magic Checkout address data is incomplete - skipping address creation",
					zap.String("request_id", requestID),
					zap.String("address_line_1", addressLine1),
					zap.String("city", city),
					zap.String("state", state),
					zap.String("postal_code", postalCode),
					zap.String("note", "Address must have (line1 OR city) AND postalCode"))
			}
		} else {
			// CRITICAL FALLBACK: Try to get address directly from Razorpay order/payment customer_details
			// This is a last resort if cache lookup failed
			h.logger.Warn("No address from cache - trying direct Razorpay API extraction",
				zap.String("request_id", requestID),
				zap.String("razorpay_order_id", req.OrderID))

			// Try order customer_details first
			if rzpOrder != nil && rzpOrder.CustomerDetails != nil && rzpOrder.CustomerDetails.ShippingAddress != nil {
				addr := rzpOrder.CustomerDetails.ShippingAddress
				magicCheckoutAddress = map[string]interface{}{
					"line1":   addr.Line1,
					"line2":   addr.Line2,
					"city":    addr.City,
					"state":   addr.State,
					"zipcode": addr.Zipcode,
					"country": addr.Country,
					"name":    addr.Name,
					"contact": addr.Contact,
				}
				// SECURITY (#27): presence only, no address PII.
				h.logger.Info("FALLBACK SUCCESS: Extracted address from Razorpay order customer_details",
					zap.String("request_id", requestID),
					zap.Bool("has_line1", addr.Line1 != ""),
					zap.Bool("has_zipcode", addr.Zipcode != ""))
			} else if rzpPayment != nil && rzpPayment.CustomerDetails != nil && rzpPayment.CustomerDetails.ShippingAddress != nil {
				// Try payment customer_details as final fallback
				addr := rzpPayment.CustomerDetails.ShippingAddress
				magicCheckoutAddress = map[string]interface{}{
					"line1":   addr.Line1,
					"line2":   addr.Line2,
					"city":    addr.City,
					"state":   addr.State,
					"zipcode": addr.Zipcode,
					"country": addr.Country,
					"name":    addr.Name,
					"contact": addr.Contact,
				}
				// SECURITY (#27): presence only, no address PII.
				h.logger.Info("FALLBACK SUCCESS: Extracted address from Razorpay payment customer_details",
					zap.String("request_id", requestID),
					zap.Bool("has_line1", addr.Line1 != ""),
					zap.Bool("has_zipcode", addr.Zipcode != ""))
			} else {
				// CRITICAL: Log why we don't have address data
				h.logger.Error("NO MAGIC CHECKOUT ADDRESS DATA FOUND - Address will NOT be saved!",
					zap.String("request_id", requestID),
					zap.String("razorpay_order_id", req.OrderID),
					zap.String("payment_id", req.PaymentID),
					zap.Bool("rzp_order_exists", rzpOrder != nil),
					zap.Bool("rzp_order_has_customer_details", rzpOrder != nil && rzpOrder.CustomerDetails != nil),
					zap.Bool("rzp_order_has_shipping_address", rzpOrder != nil && rzpOrder.CustomerDetails != nil && rzpOrder.CustomerDetails.ShippingAddress != nil),
					zap.Bool("rzp_payment_exists", rzpPayment != nil),
					zap.Bool("rzp_payment_has_customer_details", rzpPayment != nil && rzpPayment.CustomerDetails != nil),
					zap.Bool("rzp_payment_has_shipping_address", rzpPayment != nil && rzpPayment.CustomerDetails != nil && rzpPayment.CustomerDetails.ShippingAddress != nil),
					zap.String("debug_note", "Check if addresses-callback or shipping-info was called, or if Razorpay order/payment has customer_details"))
			}

			// If we found address in fallback, process it now
			if magicCheckoutAddress != nil && len(magicCheckoutAddress) > 0 {
				// Re-process the address extraction (duplicate code but necessary for fallback)
				addressLine1 := ""
				addressLine2 := ""
				city := ""
				state := ""
				postalCode := ""
				country := "India"

				if line1, ok := magicCheckoutAddress["line1"].(string); ok {
					addressLine1 = strings.TrimSpace(line1)
				}
				if line2, ok := magicCheckoutAddress["line2"].(string); ok {
					addressLine2 = strings.TrimSpace(line2)
				}
				if c, ok := magicCheckoutAddress["city"].(string); ok {
					city = strings.TrimSpace(c)
				}
				if s, ok := magicCheckoutAddress["state"].(string); ok {
					state = strings.TrimSpace(s)
				}
				if zip, ok := magicCheckoutAddress["zipcode"].(string); ok {
					postalCode = strings.TrimSpace(zip)
				}
				if c, ok := magicCheckoutAddress["country"].(string); ok {
					country = strings.TrimSpace(c)
				}

				hasValidAddress := (addressLine1 != "" || city != "") && postalCode != ""
				if hasValidAddress {
					addressData = models.NewOrderAddress(
						addressLine1,
						addressLine2,
						city,
						state,
						postalCode,
						country,
					)

					// Save to user's address book
					if h.addressService != nil && internalUserID != "" && addressLine1 != "" {
						savedAddr, addrErr := h.addressService.FindOrCreateAddress(
							c.Request.Context(),
							internalUserID,
							addressLine1,
							city,
							state,
							postalCode,
							country,
						)

						if addrErr == nil && savedAddr != nil {
							userAddressID = savedAddr.ID
							h.logger.Info("FALLBACK: Magic Checkout address saved to user's address book",
								zap.String("request_id", requestID),
								zap.String("user_id", internalUserID),
								zap.String("address_id", userAddressID))
						} else {
							h.logger.Warn("FALLBACK: Failed to save Magic Checkout address to user's address book",
								zap.String("request_id", requestID),
								zap.Error(addrErr))
						}
					}
				}
			}
		}

		// Priority 2: Try to extract structured address data from Razorpay notes
		if addressData == nil && rzpOrder != nil && rzpOrder.Notes != nil {
			notes := rzpOrder.Notes
			if addressStr, ok := notes["address"].(string); ok && addressStr != "" {
				// Parse combined address field: "street|city|state|postal"
				parts := strings.Split(addressStr, "|")
				street := ""
				city := ""
				state := ""
				postalCode := ""
				country := "India" // Default to India

				if len(parts) >= 4 {
					street = strings.TrimSpace(parts[0])
					city = strings.TrimSpace(parts[1])
					state = strings.TrimSpace(parts[2])
					postalCode = strings.TrimSpace(parts[3])
				} else if len(parts) >= 3 {
					street = strings.TrimSpace(parts[0])
					city = strings.TrimSpace(parts[1])
					state = strings.TrimSpace(parts[2])
					postalCode = "000000"
				} else if len(parts) >= 2 {
					street = strings.TrimSpace(parts[0])
					city = strings.TrimSpace(parts[1])
					state = "Unknown State"
					postalCode = "000000"
				} else if len(parts) >= 1 {
					street = strings.TrimSpace(parts[0])
					city = "Unknown City"
					state = "Unknown State"
					postalCode = "000000"
				}

				addressData = models.NewOrderAddress(
					street,
					"", // address line 2 not stored in notes
					city,
					state,
					postalCode,
					country,
				)

				h.logger.Info("Using structured address from Razorpay notes",
					zap.String("request_id", requestID),
					zap.String("street", street),
					zap.String("city", city),
					zap.String("state", state),
					zap.String("postal_code", postalCode),
					zap.String("country", country))
			}
		}

		// Fallback to database lookup if structured data not available in notes
		if addressData == nil && shippingAddressID != "" {
			if addr, err := h.orderService.GetAddressByID(c.Request.Context(), internalUserID, shippingAddressID); err == nil && addr != nil {
				addressData = models.NewOrderAddress(
					addr.AddressLine1,
					addr.AddressLine2,
					addr.City,
					addr.State,
					addr.PostalCode,
					addr.Country,
				)
				h.logger.Info("Using structured address from database",
					zap.String("request_id", requestID),
					zap.String("address_id", shippingAddressID),
					zap.String("city", addr.City),
					zap.String("state", addr.State),
					zap.String("postal_code", addr.PostalCode))
			}
		}

		// CRITICAL FALLBACK: If no valid address from Magic Checkout, try user's saved addresses
		// This handles the case where Razorpay doesn't call addresses-callback
		if addressData == nil && h.addressService != nil && internalUserID != "" {
			h.logger.Warn("No Magic Checkout address available, trying user's saved addresses",
				zap.String("request_id", requestID),
				zap.String("user_id", internalUserID))

			// First try default address
			if defaultAddr, err := h.addressService.GetDefaultAddress(c.Request.Context(), internalUserID); err == nil && defaultAddr != nil {
				addressData = models.NewOrderAddress(
					defaultAddr.AddressLine1,
					defaultAddr.AddressLine2,
					defaultAddr.City,
					defaultAddr.State,
					defaultAddr.PostalCode,
					defaultAddr.Country,
				)
				// Also set customer name from saved address if not already set
				if firstName == "" && defaultAddr.FirstName != "" {
					firstName = defaultAddr.FirstName
					lastName = defaultAddr.LastName
				}
				h.logger.Info("Using user's default saved address as fallback",
					zap.String("request_id", requestID),
					zap.String("address_id", defaultAddr.ID),
					zap.String("city", defaultAddr.City),
					zap.String("state", defaultAddr.State),
					zap.String("postal_code", defaultAddr.PostalCode),
					zap.String("first_name", defaultAddr.FirstName))
			} else {
				// No default, try to get any saved address
				h.logger.Warn("No default address, trying any saved address",
					zap.String("request_id", requestID),
					zap.String("user_id", internalUserID))

				if addresses, err := h.addressService.GetUserAddresses(c.Request.Context(), internalUserID); err == nil && len(addresses) > 0 {
					savedAddr := addresses[0] // Use first available address
					addressData = models.NewOrderAddress(
						savedAddr.AddressLine1,
						savedAddr.AddressLine2,
						savedAddr.City,
						savedAddr.State,
						savedAddr.PostalCode,
						savedAddr.Country,
					)
					if firstName == "" && savedAddr.FirstName != "" {
						firstName = savedAddr.FirstName
						lastName = savedAddr.LastName
					}
					h.logger.Info("Using user's first saved address as fallback",
						zap.String("request_id", requestID),
						zap.String("address_id", savedAddr.ID),
						zap.String("city", savedAddr.City),
						zap.String("state", savedAddr.State),
						zap.String("postal_code", savedAddr.PostalCode))
				} else {
					h.logger.Warn("No saved addresses found for user",
						zap.String("request_id", requestID),
						zap.String("user_id", internalUserID),
						zap.Error(err))
				}
			}
		}

		// Final fallback to string parsing
		if addressData == nil {
			h.logger.Warn("No structured address data available, falling back to string parsing",
				zap.String("request_id", requestID),
				zap.String("shipping_address", shippingAddress))
			addressData = h.parseShippingAddressToOrderAddress(shippingAddress)
		}

		// Note: Address is already saved in the order via addressData
		// The order will have the address stored in OrderAddress table
		// For now, we store address in the order. To save to user's address book,
		// we would need to add AddressService to PaymentHandler (can be done later)

		// SECURITY (#2 replay + user binding): a (razorpay_order_id, payment_id, signature)
		// triple must map to exactly ONE order, owned by exactly the authenticated user.
		//
		// (a) Bind the Razorpay order to the caller. The order's notes.user_id (set server-side
		//     at create-order) must match this user; otherwise a stolen/shared triple could be
		//     replayed by a different account.
		notesUserID := ""
		if rzpOrder != nil && rzpOrder.Notes != nil {
			if v, ok := rzpOrder.Notes["user_id"].(string); ok {
				notesUserID = strings.TrimSpace(v)
			}
		}
		if notesUserID != "" && notesUserID != internalUserID && notesUserID != userID {
			h.logger.Warn("Rejecting verify: Razorpay order belongs to a different user",
				zap.String("request_id", requestID),
				zap.String("caller_internal_id", internalUserID),
				zap.String("caller_uid", userID),
				zap.String("notes_user_id", notesUserID),
				zap.String("razorpay_order_id", req.OrderID))
			c.JSON(http.StatusForbidden, gin.H{"error": "This payment does not belong to the authenticated user"})
			return
		}

		// (b) Idempotency: if an order already exists for this razorpay_order_id, reuse it instead
		//     of creating a second order from the caller's CURRENT cart. This defeats the replay
		//     where a paid triple is re-submitted after swapping the cart contents.
		if existing, lookupErr := h.orderService.GetOrderByRazorpayOrderID(c.Request.Context(), req.OrderID); lookupErr == nil && existing != nil {
			if existing.UserID != internalUserID && existing.UserID != userID {
				h.logger.Warn("Rejecting verify: existing order for this razorpay_order_id belongs to another user",
					zap.String("request_id", requestID),
					zap.String("caller_internal_id", internalUserID),
					zap.String("order_user_id", existing.UserID),
					zap.String("razorpay_order_id", req.OrderID))
				c.JSON(http.StatusForbidden, gin.H{"error": "This payment does not belong to the authenticated user"})
				return
			}

			h.logger.Info("Idempotent verify: reusing existing order for razorpay_order_id",
				zap.String("request_id", requestID),
				zap.String("order_id", existing.ID),
				zap.String("razorpay_order_id", req.OrderID))

			if strings.TrimSpace(existing.RazorpayPaymentID) == "" {
				existing.RazorpayPaymentID = req.PaymentID
				if updErr := h.orderService.UpdateOrder(c.Request.Context(), existing); updErr != nil {
					// Non-fatal for the response, but the payment ID is how ownership
					// checks and reconciliation later find this order — surface it.
					h.logger.Error("Idempotent verify: failed to persist razorpay_payment_id on existing order",
						zap.Error(updErr),
						zap.String("order_id", existing.ID),
						zap.String("payment_id", req.PaymentID))
				}
			}

			// Re-run idempotent fulfillment (completes a shipment that failed the first time; a
			// no-op once shipped) and respond from the existing order.
			h.finalizeFulfillment(c.Request.Context(), existing.ID)
			if refreshed, rerr := h.orderService.GetOrderByID(c.Request.Context(), existing.ID); rerr == nil && refreshed != nil {
				existing = refreshed
			}

			shipmentPending := strings.TrimSpace(existing.DelhiveryWaybill) == ""
			c.JSON(http.StatusOK, VerifyPaymentResponse{
				Success:           true,
				Message:           "Payment already verified; your order is confirmed and being fulfilled.",
				PaymentID:         req.PaymentID,
				OrderID:           existing.ID,
				AWBNumber:         existing.DelhiveryWaybill,
				TrackingURL:       existing.TrackingURL,
				InvoiceURL:        existing.InvoiceURL,
				FulfillmentStatus: existing.FulfillmentStatus,
				ShipmentPending:   shipmentPending,
			})
			return
		}

		// Create the confirmed, paid order transactionally (#13/#14/#32): CreateConfirmedOrder
		// fetches the cart and REFUSES an empty one (no zero-item orders), binds the order to
		// razorpay_order_id AT INSERT (so a concurrent verify dedups via the unique index),
		// commits order+items atomically, and clears the cart ONLY after that commit. The
		// order is created with Status=confirmed / PaymentStatus=successful and the customer
		// shipment fields set up front — no separate swallowed UpdateOrder is needed.
		order, err := h.orderService.CreateConfirmedOrder(c.Request.Context(), services.ConfirmedOrderParams{
			UserID:            internalUserID,
			Customer:          customerData,
			Address:           addressData,
			PaymentMethod:     "razorpay",
			ShippingCharges:   shippingCharges,
			RazorpayOrderID:   req.OrderID,
			RazorpayPaymentID: req.PaymentID,
			CustomerFirstName: firstName,
			CustomerLastName:  lastName,
			CustomerPhone:     mobile,
			CustomerPhoneAlt:  mobileSecondary,
		})
		if errors.Is(err, services.ErrDuplicateOrder) {
			// A concurrent verify for the same razorpay_order_id created the order between the
			// idempotency lookup above and this insert. Reuse that order instead of failing;
			// the metadata + finalizeFulfillment steps below are idempotent and keyed on its ID.
			h.logger.Info("Concurrent verify already created the order; reusing it",
				zap.String("request_id", requestID),
				zap.String("razorpay_order_id", req.OrderID))
			existing, lookupErr := h.orderService.GetOrderByRazorpayOrderID(c.Request.Context(), req.OrderID)
			if lookupErr != nil || existing == nil {
				h.logger.Error("Duplicate order reported but lookup failed",
					zap.String("request_id", requestID),
					zap.Error(lookupErr),
					zap.String("razorpay_order_id", req.OrderID))
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment verified but failed to load order"})
				return
			}
			order = existing
		} else if err != nil {
			// Includes an empty cart (#14): a captured payment with no cart is a recoverable
			// state the webhook / manual reconciliation handles, not a reason to lose the order
			// silently — it is logged with the razorpay ids so it can be found.
			h.logger.Error("Failed to create database order after payment verification",
				zap.String("request_id", requestID),
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("payment_id", req.PaymentID),
				zap.String("razorpay_order_id", req.OrderID),
				zap.String("error_type", "order_creation_error"))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment verified but failed to create order"})
			return
		}

		h.logger.Info("Database order created successfully after payment verification",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("order_id", order.ID),
			zap.String("payment_id", req.PaymentID),
			zap.String("razorpay_order_id", req.OrderID),
			zap.Float64("total_amount", order.TotalAmount))

		// Persist full payment and shipping metadata on order and notes
		if rzpOrder != nil {
			h.logger.Info("Updating order with Razorpay metadata",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("razorpay_order_id", rzpOrder.ID),
				zap.Int("amount_paisa", rzpOrder.Amount))

			// Save rich metadata fields
			order.RazorpayOrderID = rzpOrder.ID
			order.RazorpayReceipt = rzpOrder.Receipt
			order.RazorpayAmountPaisa = rzpOrder.Amount
			// Reconcile the order total to what Razorpay actually captured, instead of trusting
			// the recomputed cart figure (which diverged from the charged amount, e.g. 360 vs 399).
			if rzpOrder.Amount > 0 {
				order.TotalAmount = services.PaisaToRupees(rzpOrder.Amount)
			}
			// Basic compact JSON string for notes
			order.RazorpayNotesJSON = "{}"
			// Delivery details
			order.ShippingCarrier = "Delhivery"
			order.ShippingMode = "Surface"
			order.ShippingCharge = shippingCharges
			order.EtaMinDays = etaMin
			order.EtaMaxDays = etaMax
			// Ensure shipping address stored
			if order.ShippingAddr == "" {
				order.ShippingAddr = shippingAddress
			}
			// Store shipping address ID for structured data access
			if shippingAddressID != "" {
				order.ShippingAddrID = shippingAddressID
			}
			// Persist customer checkout identity for shipment label
			if firstName != "" {
				order.CustomerFirstName = firstName
			}
			if lastName != "" {
				order.CustomerLastName = lastName
			}
			if mobile != "" {
				order.CustomerPhone = mobile
			}
			if mobileSecondary != "" {
				order.CustomerPhoneAlt = mobileSecondary
			}

			// Log payment status before updating order metadata
			h.logger.Info("Updating order metadata - preserving payment status",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("payment_status", order.PaymentStatus),
				zap.String("order_status", string(order.Status)))

			if updErr := h.orderService.UpdateOrder(c.Request.Context(), order); updErr != nil {
				h.logger.Warn("Failed to persist order metadata",
					zap.String("request_id", requestID),
					zap.Error(updErr),
					zap.String("order_id", order.ID))
			} else {
				h.logger.Info("Order metadata persisted successfully",
					zap.String("request_id", requestID),
					zap.String("order_id", order.ID),
					zap.String("payment_status", order.PaymentStatus),
					zap.String("order_status", string(order.Status)))
			}

			// Update notes compact string
			notesStr := fmt.Sprintf("rzp_order:%s, amount_paisa:%d, carrier:%s, mode:%s, charges:%.2f",
				rzpOrder.ID, rzpOrder.Amount, order.ShippingCarrier, order.ShippingMode, order.ShippingCharge)
			if notesErr := h.orderService.UpdateOrderNotes(c.Request.Context(), order.ID, notesStr); notesErr != nil {
				// Non-critical diagnostic metadata; log so it isn't silently lost.
				h.logger.Warn("Failed to persist order notes",
					zap.Error(notesErr), zap.String("order_id", order.ID))
			}
		}

		// Create the Delhivery shipment, decrement inventory, and advance the order.
		// finalizeFulfillment is idempotent and *records* its outcome on the order
		// (FulfillmentStatus / ShipmentError) instead of silently swallowing failures, so a
		// paid order that could not be shipped stays queryable and recoverable (via the
		// Razorpay webhook or the manual retry endpoint). A captured payment is never failed
		// because of a shipment problem.
		h.finalizeFulfillment(c.Request.Context(), order.ID)

		// Reload so the response and the WhatsApp step below reflect the shipment outcome.
		if refreshed, rerr := h.orderService.GetOrderByID(c.Request.Context(), order.ID); rerr == nil && refreshed != nil {
			order = refreshed
		}

		// Generate invoice
		h.logger.Info("Generating invoice for order",
			zap.String("request_id", requestID),
			zap.String("order_id", order.ID),
			zap.String("user_id", userID))

		if invoiceURL, invErr := h.orderService.GenerateAndAttachInvoice(c.Request.Context(), order.ID); invErr != nil {
			h.logger.Warn("Invoice generation failed",
				zap.String("request_id", requestID),
				zap.Error(invErr),
				zap.String("order_id", order.ID),
				zap.String("user_id", userID))
		} else if invoiceURL != "" {
			h.logger.Info("Invoice generated successfully",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("invoice_url", invoiceURL),
				zap.String("user_id", userID))
		}

		// Send WhatsApp notifications - get updated order with AWB number
		// Initialize response message (will be updated with WhatsApp status)
		responseMessage := "Payment verified and order created successfully."

		if h.whatsappService != nil && h.whatsappService.IsEnabled() {
			// Get the latest order data with AWB number and order items from database
			updatedOrder, _, getOrderErr := h.orderService.GetOrderDetails(c.Request.Context(), order.ID)
			if getOrderErr != nil {
				h.logger.Warn("Failed to get updated order for WhatsApp notification",
					zap.String("request_id", requestID),
					zap.String("order_id", order.ID),
					zap.Error(getOrderErr))
				updatedOrder = order // Use original order as fallback
			} else {
				// Log the AWB status and order items for debugging
				h.logger.Info("Retrieved updated order for WhatsApp notification",
					zap.String("request_id", requestID),
					zap.String("order_id", updatedOrder.ID),
					zap.String("awb_number", updatedOrder.DelhiveryWaybill),
					zap.String("tracking_url", updatedOrder.TrackingURL),
					zap.String("delhivery_status", updatedOrder.DelhiveryStatus),
					zap.Int("order_items_count", len(updatedOrder.OrderItems)))
			}

			// Extract customer phone number with detailed logging
			h.logger.Info("Extracting customer phone number from order for WhatsApp notification",
				zap.String("request_id", requestID),
				zap.String("order_id", updatedOrder.ID),
				zap.String("order_customer_phone", updatedOrder.CustomerPhone),
				zap.String("order_customer_phone_alt", updatedOrder.CustomerPhoneAlt),
				zap.Bool("has_customer_object", updatedOrder.Customer != nil),
				zap.String("customer_object_phone", func() string {
					if updatedOrder.Customer != nil {
						return updatedOrder.Customer.Phone
					}
					return ""
				}()),
				zap.String("customer_object_phone_alt", func() string {
					if updatedOrder.Customer != nil {
						return updatedOrder.Customer.PhoneAlt
					}
					return ""
				}()))

			customerPhone := h.whatsappService.GetCustomerPhoneFromOrder(updatedOrder)

			// Log the extracted phone number
			h.logger.Info("Customer phone number extracted for WhatsApp",
				zap.String("request_id", requestID),
				zap.String("order_id", updatedOrder.ID),
				zap.String("extracted_phone", customerPhone),
				zap.Bool("phone_found", customerPhone != ""))

			if customerPhone != "" {
				// Send payment confirmation notification using order_management_amrti template
				h.logger.Info("Attempting to send WhatsApp payment order confirmation notification",
					zap.String("request_id", requestID),
					zap.String("order_id", updatedOrder.ID),
					zap.String("customer_phone", customerPhone),
					zap.String("customer_phone_formatted", h.whatsappService.FormatPhoneNumber(customerPhone)),
					zap.String("awb_number", updatedOrder.DelhiveryWaybill),
					zap.String("order_number", updatedOrder.ID),
					zap.String("tracking_url", updatedOrder.TrackingURL))

				if err := h.whatsappService.SendPaymentOrderConfirmMessage(c.Request.Context(), updatedOrder, customerPhone); err != nil {
					h.logger.Error("Failed to send WhatsApp payment order confirmation message",
						zap.String("request_id", requestID),
						zap.String("order_id", updatedOrder.ID),
						zap.String("customer_phone", customerPhone),
						zap.String("customer_phone_formatted", h.whatsappService.FormatPhoneNumber(customerPhone)),
						zap.String("awb_number", updatedOrder.DelhiveryWaybill),
						zap.String("error_details", err.Error()),
						zap.Error(err))
					responseMessage += " WhatsApp message failed to send."
				} else {
					h.logger.Info("WhatsApp payment order confirmation message sent successfully",
						zap.String("request_id", requestID),
						zap.String("order_id", updatedOrder.ID),
						zap.String("customer_phone", customerPhone),
						zap.String("customer_phone_formatted", h.whatsappService.FormatPhoneNumber(customerPhone)),
						zap.String("awb_number", updatedOrder.DelhiveryWaybill),
						zap.String("tracking_url", updatedOrder.TrackingURL),
						zap.String("template_name", "mani_test1"),
						zap.Int("order_items_count", len(updatedOrder.OrderItems)))
					responseMessage += " WhatsApp message triggered successfully."
				}

				// Note: WhatsApp message is sent using mani_test1 template
				// This includes customer name, order number, tracking URL, and order items
				h.logger.Info("WhatsApp notification process completed",
					zap.String("request_id", requestID),
					zap.String("order_id", updatedOrder.ID),
					zap.String("customer_phone", customerPhone))
			} else {
				h.logger.Warn("No customer phone number found for WhatsApp notifications - checking order details",
					zap.String("request_id", requestID),
					zap.String("order_id", updatedOrder.ID),
					zap.String("order_customer_phone", updatedOrder.CustomerPhone),
					zap.String("order_customer_phone_alt", updatedOrder.CustomerPhoneAlt),
					zap.Bool("has_customer_object", updatedOrder.Customer != nil),
					zap.String("customer_id", func() string {
						if updatedOrder.Customer != nil {
							return updatedOrder.Customer.ID
						}
						return ""
					}()))
				responseMessage += " WhatsApp message skipped - no phone number found."
			}
		} else {
			h.logger.Info("WhatsApp service not available or disabled",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID))
			responseMessage = "Payment verified and order created successfully. WhatsApp service disabled."
		}

		// Clear user cart
		if err := h.orderService.ClearUserCart(c.Request.Context(), userID); err != nil {
			h.logger.Warn("Failed to clear user cart",
				zap.String("request_id", requestID),
				zap.Error(err),
				zap.String("user_id", userID))
		}

		// Get updated order to include tracking details and invoice URL if available
		updatedOrder, _ := h.orderService.GetOrderByID(c.Request.Context(), order.ID)
		if updatedOrder != nil {
			order = updatedOrder
		}

		// Extract WhatsApp message status from responseMessage
		whatsAppStatus := ""
		if strings.Contains(responseMessage, "WhatsApp message triggered successfully") {
			whatsAppStatus = "WhatsApp message triggered successfully"
		} else if strings.Contains(responseMessage, "WhatsApp message failed to send") {
			whatsAppStatus = "WhatsApp message failed to send"
		} else if strings.Contains(responseMessage, "WhatsApp message skipped") {
			whatsAppStatus = "WhatsApp message skipped - no phone number found"
		} else if h.whatsappService == nil || !h.whatsappService.IsEnabled() {
			whatsAppStatus = "WhatsApp service disabled"
		}

		shipmentPending := strings.TrimSpace(order.DelhiveryWaybill) == ""
		if shipmentPending {
			responseMessage += " Your order is confirmed; we're arranging delivery and tracking will appear shortly."
		}

		response = VerifyPaymentResponse{
			Success:           true,
			Message:           responseMessage,
			PaymentID:         req.PaymentID,
			OrderID:           order.ID,
			AWBNumber:         order.DelhiveryWaybill,
			TrackingURL:       order.TrackingURL,
			InvoiceURL:        order.InvoiceURL,
			WhatsAppMessage:   whatsAppStatus,
			FulfillmentStatus: order.FulfillmentStatus,
			ShipmentPending:   shipmentPending,
		}

		// Comprehensive success logging
		h.logger.Info("Payment verification completed successfully",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("order_id", order.ID),
			zap.String("payment_id", req.PaymentID),
			zap.String("razorpay_order_id", req.OrderID),
			zap.Float64("total_amount", order.TotalAmount),
			zap.Float64("shipping_charges", order.ShippingCharge),
			zap.String("awb_number", order.DelhiveryWaybill),
			zap.String("tracking_url", order.TrackingURL),
			zap.String("invoice_url", order.InvoiceURL))
	} else {
		// Enhanced failure response with more details
		response = VerifyPaymentResponse{
			Success: false,
			Message: "Payment verification failed - signature mismatch. Please contact support if payment was deducted.",
		}

		h.logger.Error("Payment verification failed - signature mismatch",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("payment_id", req.PaymentID),
			zap.String("razorpay_order_id", req.OrderID),
			zap.String("received_signature_length", fmt.Sprintf("%d", len(req.Signature))),
			zap.String("error_type", "signature_verification_failed"))

		// Return error status for failed verification instead of 200
		c.JSON(http.StatusBadRequest, response)
		return
	}

	c.JSON(http.StatusOK, response)
}

// finalizeFulfillment creates the Delhivery shipment for an already-paid order (if one does not
// exist yet), decrements inventory exactly once, advances the order to shipped on success, and
// records the outcome on the order (FulfillmentStatus / ShipmentError). It is idempotent and
// safe to call from payment verification, the Razorpay webhook, and the manual retry path.
//
// It deliberately returns nothing: a captured payment must never be failed because a shipment
// could not be created. Instead, failures are persisted so the order can be found and recovered.
// markFulfillmentOrLog records a fulfillment-recovery state and, crucially, does NOT
// swallow a failure to write it. A recovery marker (needs_manual_fulfillment /
// shipment_failed) is what keeps a problem order visible in the admin unshipped
// queue; if that write fails silently the order still looks healthy and is lost from
// the queue. So a failed marker write is surfaced at ERROR level (#31).
func (h *PaymentHandler) markFulfillmentOrLog(ctx context.Context, orderID, status, shipmentErr string) {
	if err := h.orderService.MarkFulfillmentStatus(ctx, orderID, status, shipmentErr); err != nil {
		h.logger.Error("CRITICAL: failed to persist fulfillment recovery marker - order may be hidden from the unshipped queue",
			zap.Error(err),
			zap.String("order_id", orderID),
			zap.String("intended_status", status),
			zap.String("shipment_error", shipmentErr))
	}
}

func (h *PaymentHandler) finalizeFulfillment(ctx context.Context, orderID string) {
	orderWithItems, orderItems, err := h.orderService.GetOrderWithItems(ctx, orderID)
	if err != nil || orderWithItems == nil {
		h.logger.Error("finalizeFulfillment: failed to load order",
			zap.Error(err), zap.String("order_id", orderID))
		h.markFulfillmentOrLog(ctx, orderID, models.FulfillmentNeedsManual, "could not load order/items")
		return
	}

	// Idempotent: if a shipment already exists, just ensure inventory + fulfilled flag are set.
	if strings.TrimSpace(orderWithItems.DelhiveryWaybill) != "" {
		if invErr := h.orderService.DeductInventoryForOrder(ctx, orderID); invErr != nil {
			h.logger.Warn("finalizeFulfillment: inventory deduction (already-shipped path) failed",
				zap.Error(invErr), zap.String("order_id", orderID))
		}
		if mErr := h.orderService.MarkFulfillmentStatus(ctx, orderID, models.FulfillmentFulfilled, ""); mErr != nil {
			h.logger.Warn("finalizeFulfillment: failed to mark fulfilled (already-shipped path)",
				zap.Error(mErr), zap.String("order_id", orderID))
		}
		return
	}

	if len(orderItems) == 0 {
		h.logger.Error("finalizeFulfillment: order has no items - cannot ship",
			zap.String("order_id", orderID))
		h.markFulfillmentOrLog(ctx, orderID, models.FulfillmentNeedsManual, "order has no items")
		return
	}

	// SECURITY (#8 validated-vs-shipped address): never attempt a shipment to a placeholder/invalid
	// address. The paid address is validated at create-order, but verify can fall back to Magic
	// Checkout / saved-address data, so guard again here. A bad address becomes an explicit
	// needs_manual_fulfillment (surfaced in the recovery queue) instead of a cryptic courier error
	// or a parcel sent to "Unknown / 000000".
	addrLower := strings.ToLower(strings.TrimSpace(orderWithItems.ShippingAddr))
	if addrLower == "" ||
		strings.Contains(addrLower, "address not provided") ||
		strings.Contains(addrLower, "unknown") ||
		strings.Contains(addrLower, "000000") {
		h.logger.Error("finalizeFulfillment: shipping address is placeholder/invalid - needs manual fulfillment",
			zap.String("order_id", orderID), zap.String("shipping_address", orderWithItems.ShippingAddr))
		h.markFulfillmentOrLog(ctx, orderID, models.FulfillmentNeedsManual, "invalid or placeholder shipping address")
		return
	}

	// SECURITY (#9/#10 shipment race): atomically claim the order before calling the courier so a
	// concurrent verify + webhook (or a double retry) cannot both create a shipment and leave two
	// waybills at the courier. Only the single winner proceeds past this point.
	claimed, claimErr := h.orderService.ClaimOrderForShipment(ctx, orderID)
	if claimErr != nil {
		h.logger.Error("finalizeFulfillment: failed to claim order for shipment",
			zap.Error(claimErr), zap.String("order_id", orderID))
		h.markFulfillmentOrLog(ctx, orderID, models.FulfillmentNeedsManual,
			"could not atomically claim order for shipment: "+claimErr.Error())
		return
	}
	if !claimed {
		// Another path already claimed/completed the shipment, or the order needs manual handling.
		h.logger.Info("finalizeFulfillment: order not claimable for shipment (already in progress/shipped or needs manual) - skipping",
			zap.String("order_id", orderID))
		return
	}

	shipmentResult, shipmentErr := h.deliveryService.CreateShipment(ctx, orderWithItems, orderItems)
	if shipmentErr != nil || shipmentResult == nil || !shipmentResult.Success || shipmentResult.Waybill == "" {
		reason := "shipment creation failed"
		if shipmentResult != nil && shipmentResult.Message != "" {
			reason = shipmentResult.Message
		}
		if shipmentErr != nil {
			reason = shipmentErr.Error()
		}
		h.logger.Error("finalizeFulfillment: shipment failed - order needs recovery",
			zap.String("order_id", orderID), zap.String("reason", reason))
		h.markFulfillmentOrLog(ctx, orderID, models.FulfillmentShipmentFailed, reason)
		return
	}

	awb := shipmentResult.Waybill
	trackingURL := shipmentResult.TrackingURL
	if trackingURL == "" {
		trackingURL = fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", awb)
	}

	if err := h.orderService.ApplyShipmentSuccess(ctx, orderID, awb, trackingURL, shipmentResult.Status); err != nil {
		// SECURITY (#10): the shipment EXISTS at the courier but we failed to persist the waybill.
		// Mark needs_manual_fulfillment (NOT shipment_failed) so the automatic claim will not
		// re-ship it — re-shipping would create a second parcel. The waybill is recorded in the
		// error text so an admin can reconcile it by hand.
		h.logger.Error("finalizeFulfillment: shipment created but DB update failed - needs manual reconciliation",
			zap.Error(err), zap.String("order_id", orderID), zap.String("waybill", awb))
		h.markFulfillmentOrLog(ctx, orderID, models.FulfillmentNeedsManual,
			fmt.Sprintf("shipment created at courier (waybill %s) but DB update failed: %s — reconcile manually, do NOT re-ship", awb, err.Error()))
		return
	}

	// Delivery is set; deduct inventory (non-fatal — the order is already fulfilled for the customer).
	if invErr := h.orderService.DeductInventoryForOrder(ctx, orderID); invErr != nil {
		h.logger.Warn("finalizeFulfillment: inventory deduction failed",
			zap.Error(invErr), zap.String("order_id", orderID))
	}

	h.logger.Info("finalizeFulfillment: order shipped",
		zap.String("order_id", orderID), zap.String("waybill", awb))
}

// ProcessWebhook handles Razorpay webhook events
// @Summary Razorpay webhook
// @Description Handle Razorpay webhook events (public endpoint)
// @Tags Payments
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Webhook processed"
// @Failure 400 {object} map[string]interface{} "Missing signature or invalid payload"
// @Failure 500 {object} map[string]interface{} "Processing failed"
// @Router /webhooks/razorpay [post]
func (h *PaymentHandler) ProcessWebhook(c *gin.Context) {
	// Get webhook signature from header
	signature := c.GetHeader("X-Razorpay-Signature")
	if signature == "" {
		h.logger.Error("Webhook signature missing")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing webhook signature"})
		return
	}

	// SECURITY (#30): cap the body size BEFORE reading it. The body is read prior to signature
	// verification, so without a cap an unauthenticated client could force large allocations.
	const maxWebhookBodyBytes = 1 << 20 // 1 MiB; Razorpay webhooks are only a few KB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookBodyBytes)

	// Event ID for observability / replay diagnostics. We intentionally do NOT reject on event
	// age: Razorpay legitimately retries webhook delivery for hours, so a timestamp cutoff would
	// drop valid late events. Replay safety instead comes from the idempotent fulfillment path
	// (finalizeFulfillment deducts inventory / creates a shipment at most once per order).
	eventID := c.GetHeader("X-Razorpay-Event-Id")

	// Read request body (bounded by MaxBytesReader above)
	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("Failed to read webhook body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Verify signature + parse. Nothing in the payload is trusted until this succeeds.
	webhook, err := h.paymentService.ParseWebhook(body, signature)
	if err != nil {
		h.logger.Error("Webhook signature/parse failed", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook"})
		return
	}

	h.logger.Info("Razorpay webhook received",
		zap.String("event", webhook.Event),
		zap.String("event_id", eventID))

	// Server-side fulfillment safety net: if the browser never completed /payments/verify, or the
	// shipment failed there, the webhook independently drives fulfillment. All paths are idempotent.
	switch webhook.Event {
	case "order.paid", "payment.captured":
		rzpOrderID := webhook.Payload.Order.Entity.ID
		if rzpOrderID == "" {
			rzpOrderID = webhook.Payload.Payment.Entity.OrderID
		}
		paymentID := webhook.Payload.Payment.Entity.ID
		if rzpOrderID == "" {
			h.logger.Warn("Webhook has no Razorpay order ID - cannot fulfill",
				zap.String("event", webhook.Event))
			break
		}
		h.fulfillFromWebhook(c.Request.Context(), rzpOrderID, paymentID)
	case "payment.failed":
		// SECURITY/CORRECTNESS (#35): record the failure on the order so it is queryable
		// and does not look perpetually "pending". Idempotent; no shipment side effects.
		h.markPaymentFailedFromWebhook(c.Request.Context(),
			webhook.Payload.Payment.Entity.OrderID,
			webhook.Payload.Payment.Entity.ID)
	default:
		h.logger.Info("Unhandled Razorpay webhook event", zap.String("event", webhook.Event))
	}

	// Always ack 2xx so Razorpay does not retry indefinitely; failures are recorded on the order.
	c.JSON(http.StatusOK, gin.H{"message": "ok"})
}

// markPaymentFailedFromWebhook records a Razorpay payment.failed event on the matching
// order. It is idempotent (re-marking an already-failed order is a no-op at the repo level)
// and has no shipment side effects. (#35)
func (h *PaymentHandler) markPaymentFailedFromWebhook(ctx context.Context, rzpOrderID, paymentID string) {
	if rzpOrderID == "" {
		h.logger.Warn("payment.failed webhook has no Razorpay order ID - cannot update order",
			zap.String("payment_id", paymentID))
		return
	}
	order, err := h.orderService.GetOrderByRazorpayOrderID(ctx, rzpOrderID)
	if err != nil || order == nil {
		h.logger.Warn("payment.failed webhook: no matching DB order",
			zap.String("razorpay_order_id", rzpOrderID), zap.String("payment_id", paymentID))
		return
	}
	// Do not overwrite a successful/paid order (a later failed event for a superseded
	// attempt must not undo a capture).
	if order.PaymentStatus == "paid" || order.PaymentStatus == "successful" {
		h.logger.Info("payment.failed webhook ignored - order already paid",
			zap.String("order_id", order.ID), zap.String("payment_status", order.PaymentStatus))
		return
	}
	if err := h.orderService.UpdatePaymentStatus(ctx, order.ID, "failed", paymentID); err != nil {
		h.logger.Error("payment.failed webhook: failed to update order payment status",
			zap.Error(err), zap.String("order_id", order.ID))
		return
	}
	h.logger.Info("payment.failed recorded on order",
		zap.String("order_id", order.ID), zap.String("razorpay_order_id", rzpOrderID))
}

// fulfillFromWebhook completes fulfillment for a paid Razorpay order from the webhook.
// It repairs orders that exist but were left unshipped (the common production failure), and
// loudly flags orders that were never created (which cannot be safely rebuilt from the webhook
// payload alone, since cart/line-items are not included) for manual reconciliation.
func (h *PaymentHandler) fulfillFromWebhook(ctx context.Context, rzpOrderID, paymentID string) {
	order, err := h.orderService.GetOrderByRazorpayOrderID(ctx, rzpOrderID)
	if err != nil || order == nil {
		h.logger.Error("Webhook: paid order has no matching DB order - needs manual reconciliation",
			zap.String("razorpay_order_id", rzpOrderID), zap.String("payment_id", paymentID))
		return
	}

	// Persist the captured payment ID for reconciliation if we don't have it yet. Defense-in-depth:
	// only accept a well-formed Razorpay payment id, even though the payload is signature-verified.
	if strings.HasPrefix(paymentID, "pay_") && order.RazorpayPaymentID == "" {
		order.RazorpayPaymentID = paymentID
		if uerr := h.orderService.UpdateOrder(ctx, order); uerr != nil {
			h.logger.Warn("Webhook: failed to persist payment ID",
				zap.Error(uerr), zap.String("order_id", order.ID))
		}
	}

	// Idempotent: completes the shipment if verify failed or never ran, advances status,
	// and deducts inventory exactly once.
	h.finalizeFulfillment(ctx, order.ID)
}

// GetPaymentStatus retrieves payment status from Razorpay
// @Summary Get payment status
// @Description Get Razorpay payment status by payment ID
// @Tags Payments
// @Produce json
// @Security FirebaseAuth
// @Param payment_id path string true "Razorpay Payment ID"
// @Success 200 {object} services.RazorpayPayment "Payment status"
// @Failure 400 {object} map[string]interface{} "Payment ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to get payment status"
// @Router /payments/{payment_id}/status [get]
func (h *PaymentHandler) GetPaymentStatus(c *gin.Context) {
	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	paymentID := c.Param("payment_id")
	if paymentID == "" {
		h.logger.Error("Payment ID missing in request")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Payment ID is required"})
		return
	}

	// Ownership check (#17/#38): a signature-less payment ID must not be a lookup
	// oracle. Only the owner of the order this payment belongs to — or an admin —
	// may read its status. Resolve the order by payment ID and compare the owner.
	// Return 404 (not 403) on mismatch/not-found so the endpoint can't be used to
	// enumerate which payment IDs exist.
	if !h.authContext.IsAdmin(c) {
		order, oErr := h.orderService.GetOrderByRazorpayPaymentID(c.Request.Context(), paymentID)
		if oErr != nil || order == nil || !h.callerOwnsOrder(c, userID, order.UserID) {
			h.logger.Warn("Payment status denied - caller does not own the order",
				zap.String("user_id", userID),
				zap.String("payment_id", paymentID))
			c.JSON(http.StatusNotFound, gin.H{"error": "Payment not found"})
			return
		}
	}

	// Get payment details from Razorpay
	payment, err := h.paymentService.GetPaymentStatus(c.Request.Context(), paymentID)
	if err != nil {
		h.logger.Error("Failed to get payment status",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("payment_id", paymentID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get payment status"})
		return
	}

	// Convert amount to rupees for response
	payment.Amount = services.RupeesToPaisa(services.PaisaToRupees(payment.Amount))

	h.logger.Info("Payment status retrieved successfully",
		zap.String("user_id", userID),
		zap.String("payment_id", paymentID),
		zap.String("status", payment.Status))

	c.JSON(http.StatusOK, payment)
}

// calculateShippingFromAddress calculates shipping charges using delivery service API
func (h *PaymentHandler) calculateShippingFromAddress(ctx context.Context, userID, shippingAddress string, cartTotal float64, itemCount int) (float64, error) {
	// Parse address to extract required components
	address, err := h.parseShippingAddress(shippingAddress)
	if err != nil {
		return 0, fmt.Errorf("failed to parse shipping address: %w", err)
	}

	// Create shipping estimate request
	estimateReq := requests.EstimateShippingRequest{
		Address: address,
	}

	// Get shipping estimate from delivery service
	estimate, err := h.deliveryService.Estimate(ctx, userID, estimateReq)
	if err != nil {
		return 0, fmt.Errorf("failed to get shipping estimate: %w", err)
	}

	// Extract shipping charges from the first available option
	if len(estimate.Options) > 0 {
		return estimate.Options[0].Price, nil
	}

	return 0, fmt.Errorf("no shipping options available")
}

// parseShippingAddress parses a formatted address string into components
func (h *PaymentHandler) parseShippingAddress(addressStr string) (struct {
	Street     string `json:"street" binding:"required" validate:"required,min=1,max=200"`
	City       string `json:"city" binding:"required" validate:"required,min=1,max=100"`
	State      string `json:"state" binding:"required" validate:"required,min=1,max=100"`
	PostalCode string `json:"postal_code" binding:"required" validate:"required,min=1,max=20"`
	Country    string `json:"country" binding:"required" validate:"required,min=1,max=100"`
}, error) {
	var address struct {
		Street     string `json:"street" binding:"required" validate:"required,min=1,max=200"`
		City       string `json:"city" binding:"required" validate:"required,min=1,max=100"`
		State      string `json:"state" binding:"required" validate:"required,min=1,max=100"`
		PostalCode string `json:"postal_code" binding:"required" validate:"required,min=1,max=20"`
		Country    string `json:"country" binding:"required" validate:"required,min=1,max=100"`
	}

	if addressStr == "" {
		return address, fmt.Errorf("shipping address cannot be empty")
	}

	// Split address by comma and extract components
	parts := strings.Split(addressStr, ",")
	if len(parts) < 3 {
		return address, fmt.Errorf("address format invalid, expected at least 3 parts separated by commas")
	}

	// Trim whitespace from all parts
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}

	// Extract components (assuming format: street, city, state, postal_code, country)
	address.Street = parts[0]

	if len(parts) >= 2 {
		address.City = parts[1]
	}

	if len(parts) >= 3 {
		address.State = parts[2]
	}

	if len(parts) >= 4 {
		address.PostalCode = parts[3]
	}

	if len(parts) >= 5 {
		address.Country = parts[4]
	} else {
		address.Country = "India" // Default to India
	}

	// Validate that we have the essential components
	if address.PostalCode == "" {
		// Try to find postal code in any part (6 digits)
		for _, part := range parts {
			if pincodeRegex.MatchString(part) {
				address.PostalCode = part
				break
			}
		}
		if address.PostalCode == "" {
			return address, fmt.Errorf("postal code not found in address")
		}
	}

	return address, nil
}

// calculateShippingChargesRobust provides a robust shipping calculation with intelligent fallback
func (h *PaymentHandler) calculateShippingChargesRobust(ctx context.Context, userID, shippingAddress string, cartTotal float64, itemCount int) float64 {
	// First, try to get real API charges from Delhivery
	if h.deliveryService != nil {
		if dynamicCharges, err := h.calculateShippingFromAddress(ctx, userID, shippingAddress, cartTotal, itemCount); err == nil {
			h.logger.Info("Successfully retrieved dynamic shipping charges from API",
				zap.String("user_id", userID),
				zap.Float64("api_shipping_cost", dynamicCharges))
			return dynamicCharges
		} else {
			h.logger.Warn("Delhivery API unavailable, using intelligent fallback calculation",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.Float64("cart_total", cartTotal),
				zap.Int("item_count", itemCount))
		}
	} else {
		h.logger.Warn("Delivery service not initialized, using intelligent fallback calculation",
			zap.String("user_id", userID))
	}

	// Intelligent fallback: Calculate based on cart value and item count
	// This provides more accurate estimates than hardcoded values
	baseShipping := 30.0 // Base shipping cost

	// Weight-based calculation: approximately 5 rupees per item
	weightBasedCharge := float64(itemCount) * 5.0

	// Distance estimation based on typical metro vs non-metro rates
	// Parse pincode to determine if it's likely metro (110xxx, 400xxx, etc.)
	distanceMultiplier := 1.0
	if address, err := h.parseShippingAddress(shippingAddress); err == nil {
		pincode := address.PostalCode
		// Metro area pincodes typically have lower shipping costs
		if isMetroPincode(pincode) {
			distanceMultiplier = 0.9 // 10% discount for metro areas
		} else {
			distanceMultiplier = 1.1 // 10% premium for non-metro areas
		}
	}

	// Calculate total shipping with all factors
	calculatedShipping := (baseShipping + weightBasedCharge) * distanceMultiplier

	// Round to 2 decimal places and ensure minimum charge
	calculatedShipping = math.Round(calculatedShipping*100) / 100
	if calculatedShipping < 25.0 {
		calculatedShipping = 25.0 // Minimum shipping charge
	}

	h.logger.Info("Applied intelligent shipping calculation",
		zap.String("user_id", userID),
		zap.Float64("base_shipping", baseShipping),
		zap.Float64("weight_charge", weightBasedCharge),
		zap.Float64("distance_multiplier", distanceMultiplier),
		zap.Float64("calculated_shipping", calculatedShipping),
		zap.String("fallback_reason", "API_unavailable"))

	return calculatedShipping
}

// isMetroPincode checks if a pincode belongs to a major metro area
func isMetroPincode(pincode string) bool {
	if len(pincode) != 6 {
		return false
	}

	// Major metro pincode prefixes
	metroPrefixes := []string{
		"110", "111", "112", // Delhi
		"400", "401", "402", "403", "421", "422", // Mumbai
		"560", "561", "562", // Bangalore
		"600", "601", "602", "603", // Chennai
		"700", "701", "702", // Kolkata
		"500", "501", "502", // Hyderabad
		"411", "412", "413", // Pune
		"380", "381", "382", // Ahmedabad
	}

	prefix := pincode[:3]
	for _, metroPrefix := range metroPrefixes {
		if prefix == metroPrefix {
			return true
		}
	}

	return false
}

// parseShippingAddressToOrderAddress parses a formatted shipping address string into OrderAddress struct
func (h *PaymentHandler) parseShippingAddressToOrderAddress(shippingAddress string) *models.OrderAddress {
	// Initialize with defaults
	addressLine1 := ""
	addressLine2 := ""
	city := ""
	state := ""
	postalCode := ""
	country := "India"

	// Clean the input address
	shippingAddress = strings.TrimSpace(shippingAddress)
	if shippingAddress == "" {
		return models.NewOrderAddress("Address not provided", "", "Unknown", "Unknown", "000000", "India")
	}

	// Try to parse structured address format: "Name, Address, City, State, Pincode, Country"
	if strings.Contains(shippingAddress, ",") {
		parts := strings.Split(shippingAddress, ",")

		// Trim all parts
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}

		// Handle different address formats
		if len(parts) >= 4 {
			// Format: "Name, Address, City, State, Pincode, Country"
			// Skip name (parts[0]) and use actual address
			if len(parts) > 1 {
				addressLine1 = parts[1]
			}
			if len(parts) > 2 {
				city = parts[2]
			}
			if len(parts) > 3 {
				state = parts[3]
			}
			if len(parts) > 4 {
				postalCode = parts[4]
			}
			if len(parts) > 5 {
				country = parts[5]
			}
		} else if len(parts) >= 3 {
			// Format: "Address, City, State" - try to extract pincode from address
			addressLine1 = parts[0]
			city = parts[1]
			state = parts[2]
		}
	} else {
		// Single line address - try to extract components
		addressLine1 = shippingAddress
	}

	// Extract postal code using regex if not found in parsing
	if postalCode == "" {
		re := regexp.MustCompile(`\b\d{6}\b`)
		if matches := re.FindString(shippingAddress); matches != "" {
			postalCode = matches
		}
	}

	// Clean up postal code - extract only 6 digits
	if postalCode != "" {
		re := regexp.MustCompile(`\d{6}`)
		if matches := re.FindString(postalCode); matches != "" {
			postalCode = matches
		} else {
			postalCode = ""
		}
	}

	// Validate and provide sensible defaults
	if addressLine1 == "" {
		addressLine1 = "Address not provided"
	}
	if city == "" {
		city = "Unknown City"
	}
	if state == "" {
		state = "Unknown State"
	}
	if postalCode == "" {
		postalCode = "000000"
	}
	if country == "" {
		country = "India"
	}

	// Log the parsed address for debugging
	h.logger.Info("Parsed shipping address",
		zap.String("original", shippingAddress),
		zap.String("address_line1", addressLine1),
		zap.String("city", city),
		zap.String("state", state),
		zap.String("postal_code", postalCode),
		zap.String("country", country))

	return models.NewOrderAddress(
		addressLine1,
		addressLine2,
		city,
		state,
		postalCode,
		country,
	)
}

// mapFirebaseUIDToInternalUserID maps a Firebase UID to an internal user ID
// If the user doesn't exist in the local database, it auto-creates them from Firebase
// callerOwnsOrder reports whether the authenticated caller owns an order whose
// stored UserID is ownerUserID. Orders are created with the internal user ID (or,
// as a fallback, the Firebase UID), so this resolves the caller's Firebase UID to
// its internal ID read-only (no auto-create) and also accepts a direct match to
// cover the Firebase-UID-fallback and JWT-internal-ID cases.
func (h *PaymentHandler) callerOwnsOrder(c *gin.Context, callerID, ownerUserID string) bool {
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

// mapFirebaseUIDToInternalUserID resolves a Firebase UID to the internal USER… id,
// auto-creating/linking a local user when needed. It returns an error rather than
// falling back to the Firebase UID (#25): attaching an order/payment to a raw Firebase
// UID pollutes ownership (it never matches the internal id ownership checks compare
// against). Callers must treat an error as "could not identify the account" and refuse
// to proceed, not silently continue with a non-user id.
func (h *PaymentHandler) mapFirebaseUIDToInternalUserID(c *gin.Context, firebaseUID string) (string, error) {
	// Try to get user by Firebase UID
	userResponse, err := h.userService.GetUserByFirebaseUID(c.Request.Context(), firebaseUID)
	if err == nil && userResponse != nil && userResponse.Data != nil {
		if userMap, ok := userResponse.Data.(map[string]interface{}); ok {
			if id, ok := userMap["id"].(string); ok {
				return id, nil
			}
		}
	}

	// User doesn't exist - try to auto-create from Firebase
	firebaseClient := config.GetFirebaseAuth()
	if firebaseClient == nil {
		h.logger.Error("Firebase client not initialized", zap.String("firebase_uid", firebaseUID))
		return "", fmt.Errorf("firebase client not initialized")
	}

	firebaseUser, firebaseErr := firebaseClient.GetUser(c.Request.Context(), firebaseUID)
	if firebaseErr != nil {
		h.logger.Error("Failed to get user from Firebase", zap.String("firebase_uid", firebaseUID), zap.Error(firebaseErr))
		return "", fmt.Errorf("failed to look up firebase user: %w", firebaseErr)
	}

	// Check for existing user by email or phone
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
		internalID := existingUser["id"].(string)
		h.userService.UpdateUserFirebaseUID(c.Request.Context(), internalID, firebaseUID)
		h.logger.Info("Linked Firebase UID to existing user", zap.String("firebase_uid", firebaseUID), zap.String("internal_id", internalID))
		return internalID, nil
	}

	// Create new user
	displayName := firebaseUser.DisplayName
	if displayName == "" {
		displayName = "User"
	}

	if firebaseUser.PhoneNumber != "" && firebaseUser.Email == "" {
		phoneReq := requests.PhoneSignupRequest{PhoneNumber: firebaseUser.PhoneNumber, Name: displayName}
		createResponse, _ := h.userService.CreatePhoneUser(c.Request.Context(), &phoneReq)
		if createResponse != nil && createResponse.Success {
			if userData, ok := createResponse.Data.(map[string]interface{}); ok {
				if id, ok := userData["id"].(string); ok {
					h.userService.UpdateUserFirebaseUID(c.Request.Context(), id, firebaseUID)
					h.logger.Info("Auto-created phone user", zap.String("firebase_uid", firebaseUID), zap.String("internal_id", id))
					return id, nil
				}
			}
		}
	} else {
		signupReq := requests.SignupRequest{Email: firebaseUser.Email, Name: displayName, Password: ""}
		createResponse, _ := h.userService.CreateGoogleUser(c.Request.Context(), &signupReq, firebaseUID)
		if createResponse != nil && createResponse.Success {
			if userData, ok := createResponse.Data.(map[string]interface{}); ok {
				if id, ok := userData["id"].(string); ok {
					h.logger.Info("Auto-created Google user", zap.String("firebase_uid", firebaseUID), zap.String("internal_id", id))
					return id, nil
				}
			}
		}
	}

	h.logger.Error("Failed to auto-create user", zap.String("firebase_uid", firebaseUID))
	return "", fmt.Errorf("failed to resolve or create a local user for firebase uid")
}

// CreateMagicCheckoutOrder creates a Razorpay order specifically for Magic Checkout
// @Summary Create Magic Checkout order
// @Description Create a Razorpay order with line items for Magic Checkout
// @Tags Payments
// @Accept json
// @Produce json
// @Security FirebaseAuth optional
// @Success 200 {object} map[string]interface{} "Magic Checkout order created"
// @Failure 400 {object} map[string]interface{} "Bad request"
// @Failure 401 {object} map[string]interface{} "Unauthorized"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /payments/magic-checkout/create-order [post]
func (h *PaymentHandler) CreateMagicCheckoutOrder(c *gin.Context) {
	requestID := c.GetString("request_id")

	// Support both authenticated and guest checkout
	// GetAuthenticatedUserID returns Firebase UID when available (from OptionalFirebaseAuth middleware)
	userID, authenticated := h.authContext.GetAuthenticatedUserID(c)

	if !authenticated {
		// For guest checkout, use session ID
		sessionID := c.GetHeader("X-Session-ID")
		if sessionID == "" {
			h.logger.Error("No authentication or session ID provided",
				zap.String("request_id", requestID))
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Please login or provide session ID for guest checkout",
			})
			return
		}
		userID = "session_" + sessionID
	}

	h.logger.Info("Creating Magic Checkout order",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.Bool("authenticated", authenticated))

	// Get cart - for now, only support authenticated users
	// TODO: Add session cart support for guest checkout
	if !authenticated {
		h.logger.Warn("Guest checkout not yet supported for Magic Checkout",
			zap.String("request_id", requestID))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Please login to use Magic Checkout. Guest checkout coming soon.",
		})
		return
	}

	// Map Firebase UID to internal user ID
	internalUserID := userID
	if authenticated && !strings.HasPrefix(userID, "USER") && h.userService != nil {
		// This is likely a Firebase UID, map it to internal user ID
		userResponse, err := h.userService.GetUserByFirebaseUID(c.Request.Context(), userID)
		if err == nil && userResponse != nil && userResponse.Data != nil {
			// Extract the internal user ID from the response
			if userMap, ok := userResponse.Data.(map[string]interface{}); ok {
				if id, ok := userMap["id"].(string); ok {
					internalUserID = id
					h.logger.Info("Mapped Firebase UID to internal user ID for Magic Checkout",
						zap.String("firebase_uid", userID),
						zap.String("internal_id", internalUserID))
				}
			}
		} else {
			// User doesn't exist in local database - auto-create from Firebase
			h.logger.Info("User not found in local database, attempting to auto-create from Firebase",
				zap.String("firebase_uid", userID))

			// Get user info from Firebase Admin SDK
			firebaseClient := config.GetFirebaseAuth()
			if firebaseClient == nil {
				h.logger.Error("Firebase client not initialized, cannot auto-create user",
					zap.String("firebase_uid", userID))
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to get user information",
				})
				return
			}

			// Get user record from Firebase
			firebaseUser, firebaseErr := firebaseClient.GetUser(c.Request.Context(), userID)
			if firebaseErr != nil {
				h.logger.Error("Failed to get user from Firebase",
					zap.String("firebase_uid", userID),
					zap.Error(firebaseErr))
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to get user information from Firebase",
				})
				return
			}

			// Try to find existing user by email or phone
			var existingUser map[string]interface{}

			// Check by email first (for Google/email users)
			if firebaseUser.Email != "" {
				emailResponse, _ := h.userService.GetUserByEmail(c.Request.Context(), firebaseUser.Email)
				if emailResponse != nil && emailResponse.Success && emailResponse.Data != nil {
					existingUser, _ = emailResponse.Data.(map[string]interface{})
				}
			}

			// If not found by email, check by phone number (for phone users)
			if existingUser == nil && firebaseUser.PhoneNumber != "" {
				phoneResponse, _ := h.userService.GetUserByPhoneNumber(c.Request.Context(), firebaseUser.PhoneNumber)
				if phoneResponse != nil && phoneResponse.Success && phoneResponse.Data != nil {
					existingUser, _ = phoneResponse.Data.(map[string]interface{})
				}
			}

			if existingUser != nil {
				// User exists, just link the Firebase UID
				internalUserID = existingUser["id"].(string)
				h.userService.UpdateUserFirebaseUID(c.Request.Context(), internalUserID, userID)
				// SECURITY (#27): no email/phone PII in logs.
				h.logger.Info("Linked Firebase UID to existing user",
					zap.String("firebase_uid", userID),
					zap.String("internal_id", internalUserID))
			} else {
				// Create new user in local database based on auth type
				displayName := firebaseUser.DisplayName
				if displayName == "" {
					displayName = "User"
				}

				var createResponse *base.BaseResponse
				var createErr error

				// Determine user type: phone user vs email/Google user
				if firebaseUser.PhoneNumber != "" && firebaseUser.Email == "" {
					// Phone-only user - use CreatePhoneUser
					phoneReq := requests.PhoneSignupRequest{
						PhoneNumber: firebaseUser.PhoneNumber,
						Name:        displayName,
					}
					createResponse, createErr = h.userService.CreatePhoneUser(c.Request.Context(), &phoneReq)
					if createErr == nil && createResponse != nil && createResponse.Success {
						// Link Firebase UID to the newly created phone user
						if userData, ok := createResponse.Data.(map[string]interface{}); ok {
							if id, ok := userData["id"].(string); ok {
								h.userService.UpdateUserFirebaseUID(c.Request.Context(), id, userID)
							}
						}
					}
					// SECURITY (#27): don't log the phone number.
					h.logger.Info("Creating phone user in local database",
						zap.String("firebase_uid", userID))
				} else {
					// Email/Google user - use CreateGoogleUser
					signupReq := requests.SignupRequest{
						Email:    firebaseUser.Email,
						Name:     displayName,
						Password: "", // Google/Firebase users don't have passwords
					}
					createResponse, createErr = h.userService.CreateGoogleUser(c.Request.Context(), &signupReq, userID)
					// SECURITY (#27): don't log the email.
					h.logger.Info("Creating Google/email user in local database",
						zap.String("firebase_uid", userID))
				}

				if createErr != nil || createResponse == nil || !createResponse.Success {
					// If user already exists (duplicate), try to fetch instead of failing.
					// We don't have concrete error types here, so use a best-effort heuristic:
					// - No low-level error AND
					// - createResponse present but Success == false
					//   => likely a validation/duplicate error.
					alreadyExists := (createErr == nil && createResponse != nil && !createResponse.Success)

					if alreadyExists {
						// SECURITY (#27): don't log email/phone PII.
						h.logger.Info("User already exists when auto-creating for Magic Checkout, fetching existing user instead",
							zap.String("firebase_uid", userID))

						// Try by email first
						if firebaseUser.Email != "" {
							if emailResp, err := h.userService.GetUserByEmail(c.Request.Context(), firebaseUser.Email); err == nil && emailResp != nil && emailResp.Success && emailResp.Data != nil {
								if data, ok := emailResp.Data.(map[string]interface{}); ok {
									if id, ok := data["id"].(string); ok && id != "" {
										internalUserID = id
										h.userService.UpdateUserFirebaseUID(c.Request.Context(), internalUserID, userID)
									}
								}
							}
						}
						// Then by phone if still empty
						if internalUserID == "" && firebaseUser.PhoneNumber != "" {
							if phoneResp, err := h.userService.GetUserByPhoneNumber(c.Request.Context(), firebaseUser.PhoneNumber); err == nil && phoneResp != nil && phoneResp.Success && phoneResp.Data != nil {
								if data, ok := phoneResp.Data.(map[string]interface{}); ok {
									if id, ok := data["id"].(string); ok && id != "" {
										internalUserID = id
										h.userService.UpdateUserFirebaseUID(c.Request.Context(), internalUserID, userID)
									}
								}
							}
						}

						if internalUserID == "" {
							// SECURITY (#27): no email/phone PII in logs.
							h.logger.Error("Failed to resolve existing user after duplicate error during auto-create",
								zap.String("firebase_uid", userID))
							c.JSON(http.StatusInternalServerError, gin.H{
								"error": "Failed to resolve user account",
							})
							return
						}

						// SECURITY (#27): no email/phone PII in logs.
						h.logger.Info("Resolved existing user after duplicate error during auto-create",
							zap.String("firebase_uid", userID),
							zap.String("internal_id", internalUserID))
					} else {
						// SECURITY (#27): no email/phone PII in logs.
						h.logger.Error("Failed to auto-create user in local database",
							zap.String("firebase_uid", userID),
							zap.Error(createErr))
						c.JSON(http.StatusInternalServerError, gin.H{
							"error": "Failed to create user account",
						})
						return
					}
				} else {
					if userData, ok := createResponse.Data.(map[string]interface{}); ok {
						if id, ok := userData["id"].(string); ok {
							internalUserID = id
						}
					}
					// SECURITY (#27): no email/phone PII in logs.
					h.logger.Info("Auto-created user in local database from Firebase",
						zap.String("firebase_uid", userID),
						zap.String("internal_id", internalUserID))
				}
			}
		}
	}

	// IMPORTANT: Check if user has a saved address (fallback for when Razorpay doesn't return address)
	// This is critical because Magic Checkout's addresses-callback may not be called
	var hasValidAddress bool
	var savedAddressInfo string
	var savedAddress *models.Address
	if h.addressService != nil && internalUserID != "" {
		// First check for default address
		if defaultAddr, err := h.addressService.GetDefaultAddress(c.Request.Context(), internalUserID); err == nil && defaultAddr != nil {
			hasValidAddress = true
			savedAddress = defaultAddr
			savedAddressInfo = fmt.Sprintf("%s, %s, %s %s", defaultAddr.AddressLine1, defaultAddr.City, defaultAddr.State, defaultAddr.PostalCode)
			h.logger.Info("User has default address for Magic Checkout",
				zap.String("request_id", requestID),
				zap.String("user_id", internalUserID),
				zap.String("address_id", defaultAddr.ID),
				zap.String("city", defaultAddr.City))
		} else {
			// Check for any saved address
			if addresses, err := h.addressService.GetUserAddresses(c.Request.Context(), internalUserID); err == nil && len(addresses) > 0 {
				hasValidAddress = true
				savedAddress = addresses[0]
				savedAddressInfo = fmt.Sprintf("%s, %s, %s %s", addresses[0].AddressLine1, addresses[0].City, addresses[0].State, addresses[0].PostalCode)
				h.logger.Info("User has saved address for Magic Checkout",
					zap.String("request_id", requestID),
					zap.String("user_id", internalUserID),
					zap.Int("address_count", len(addresses)),
					zap.String("city", addresses[0].City))
			}
		}
	}

	// Get user information (name, email, phone) for notes
	var userName, userEmail, userPhone string
	if h.userService != nil && internalUserID != "" {
		userResponse, err := h.userService.GetUserByID(c.Request.Context(), internalUserID)
		if err == nil && userResponse != nil && userResponse.Success && userResponse.Data != nil {
			if userMap, ok := userResponse.Data.(map[string]interface{}); ok {
				if name, ok := userMap["name"].(string); ok {
					userName = name
				}
				if email, ok := userMap["email"].(string); ok {
					userEmail = email
				}
				if phone, ok := userMap["phone_number"].(string); ok {
					userPhone = phone
				}
				// SECURITY (#27): no name/email/phone PII in logs.
				h.logger.Info("Retrieved user information for Magic Checkout notes",
					zap.String("request_id", requestID),
					zap.String("user_id", internalUserID),
					zap.Bool("has_name", userName != ""),
					zap.Bool("has_email", userEmail != ""),
					zap.Bool("has_phone", userPhone != ""))
			}
		} else {
			h.logger.Warn("Failed to get user information for notes",
				zap.String("request_id", requestID),
				zap.String("user_id", internalUserID),
				zap.Error(err))
		}
	}

	if !hasValidAddress {
		h.logger.Warn("User has no saved address - Magic Checkout may fail",
			zap.String("request_id", requestID),
			zap.String("user_id", internalUserID))
		// Return warning but allow checkout to proceed - Razorpay might provide address
		// Uncomment below to block checkout without address:
		// c.JSON(http.StatusBadRequest, gin.H{
		// 	"error": "Please save a delivery address before checkout",
		// 	"code":  "NO_ADDRESS",
		// })
		// return
	}

	// Get cart using internal user ID
	cart, err := h.orderService.GetUserCart(c.Request.Context(), internalUserID)
	if err != nil {
		h.logger.Error("Failed to get user cart",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to get cart",
		})
		return
	}

	if cart == nil || len(cart.Items) == 0 {
		h.logger.Warn("Cart is empty or not found",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("internal_user_id", internalUserID))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Cart is empty. Please add items to cart.",
		})
		return
	}

	// Calculate totals
	cartTotal := cart.GetTotalAmount()
	discountedTotal := utils.ApplyDiscount(cartTotal, cart.DiscountAmount)

	// Prepare line items for Magic Checkout
	lineItems := []map[string]interface{}{}
	lineItemsTotal := 0

	for _, item := range cart.Items {
		itemTotal := utils.RupeesToPaisa(item.UnitPrice * float64(item.Quantity)) // Convert to paise (rounded)
		lineItemsTotal += itemTotal

		// Fetch product details for proper display
		productName := item.ProductID
		productDescription := ""
		productImageURL := ""

		// Fetch product details for proper display (non-blocking - continue even if it fails)
		// Use a goroutine to fetch products in parallel, but don't block on errors
		func() {
			defer func() {
				if r := recover(); r != nil {
					// Recover from any panics in product fetching
					h.logger.Warn("Panic in product fetching, using ProductID",
						zap.String("request_id", requestID),
						zap.String("product_id", item.ProductID),
						zap.Any("panic", r))
				}
			}()

			if h.productService != nil {
				productReq := &requests.GetProductRequest{}
				productReq.ID = item.ProductID
				productResp, err := h.productService.GetProductByID(c.Request.Context(), productReq)
				if err == nil && productResp != nil && productResp.Success {
					// Extract product data from response (response.Data is typically a map after JSON unmarshaling)
					if productData, ok := productResp.Data.(map[string]interface{}); ok {
						// Handle map response
						if name, ok := productData["name"].(string); ok && name != "" {
							productName = name
						}
						if desc, ok := productData["description"].(string); ok && desc != "" {
							productDescription = desc
						}
						// Get first image if available - images is array of ProductImageResponse
						if images, ok := productData["images"].([]interface{}); ok && len(images) > 0 {
							if firstImage, ok := images[0].(map[string]interface{}); ok {
								// ProductImageResponse uses "image_url" field
								if url, ok := firstImage["image_url"].(string); ok && url != "" {
									productImageURL = url
								}
							}
						}
					}
				}
				// Silently continue if product fetch fails - use ProductID as fallback
			}
		}()

		lineItem := map[string]interface{}{
			"sku":         item.ProductID,
			"name":        productName,
			"price":       utils.RupeesToPaisa(item.UnitPrice), // in paise (rounded)
			"offer_price": utils.RupeesToPaisa(item.UnitPrice), // same as price for now
			"tax_amount":  0,
			"quantity":    item.Quantity,
			"description": productDescription,
			"image_url":   productImageURL,
			"product_url": fmt.Sprintf("https://amrti.com/products/%s", item.ProductID),
			"weight":      100, // 100g per item (default)
			"dimensions": map[string]int{
				"length": 10,
				"width":  10,
				"height": 10,
			},
			"variant_id": item.ProductID,
		}

		lineItems = append(lineItems, lineItem)
	}

	// Prepare notes for order tracking
	// Store essential information in notes (max 15 fields for Razorpay)
	notes := map[string]interface{}{
		"user_id":        userID,
		"cart_total":     cartTotal,
		"discount":       cart.DiscountAmount,
		"coupon_code":    cart.CouponCode,
		"total_items":    len(cart.Items),
		"magic_checkout": true,
		"authenticated":  authenticated,
	}

	// Add user information to notes
	if userName != "" {
		notes["customer_name"] = userName
	}
	if userEmail != "" {
		notes["customer_email"] = userEmail
	}
	if userPhone != "" {
		notes["customer_mobile"] = userPhone
	}

	// Add address information to notes (combine fields to stay within 15-field limit)
	if savedAddress != nil {
		// Combine address components into one field (similar to regular payment flow)
		notes["address"] = savedAddress.AddressLine1 + "|" + savedAddress.City + "|" + savedAddress.State + "|" + savedAddress.PostalCode
		// Store address ID for reference
		notes["shipping_addr_id"] = savedAddress.ID
		// Store address name (first name + last name from address)
		if savedAddress.FirstName != "" || savedAddress.LastName != "" {
			addressName := strings.TrimSpace(savedAddress.FirstName + " " + savedAddress.LastName)
			if addressName != "" {
				notes["address_name"] = addressName
			}
		}
		// Store address phone if available
		if savedAddress.Phone != "" {
			notes["address_phone"] = savedAddress.Phone
		}
	}

	// Create Razorpay order with line items
	// Generate short receipt (max 40 chars for Razorpay requirement)
	// Format: mc_<short_userID>_<timestamp>
	timestamp := time.Now().Unix()
	// Use first 6 chars of userID + timestamp to ensure < 40 chars
	// Format: "mc_" (3) + userID (6) + "_" (1) + timestamp (10) = 20 chars max
	userIDHash := userID
	if len(userIDHash) > 6 {
		userIDHash = userIDHash[:6]
	}
	receipt := fmt.Sprintf("mc_%s_%d", userIDHash, timestamp)
	// Ensure receipt is max 40 chars (should always be < 40 with this format)
	if len(receipt) > 40 {
		// Fallback: use just timestamp if still too long
		receipt = fmt.Sprintf("mc_%d", timestamp)
	}

	orderOptions := map[string]interface{}{
		"amount":           utils.RupeesToPaisa(discountedTotal), // Convert to paise (rounded)
		"currency":         "INR",
		"receipt":          receipt,
		"notes":            notes,
		"line_items_total": lineItemsTotal,
		"line_items":       lineItems,
	}

	razorpayOrder, err := h.paymentService.CreateOrderWithOptions(
		c.Request.Context(),
		orderOptions,
	)

	if err != nil {
		h.logger.Error("Failed to create Razorpay order",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("user_id", userID),
			zap.Float64("amount", discountedTotal),
			zap.Int("line_items_count", len(lineItems)),
			zap.Int("line_items_total", lineItemsTotal))
		// SECURITY (#28): error already logged above; do not leak internal details.
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create payment order. Please try again.",
		})
		return
	}

	// Prepare response
	cfg := config.GetRazorpayConfig()
	response := gin.H{
		"order_id":          razorpayOrder.ID,
		"amount":            services.PaisaToRupees(razorpayOrder.Amount),
		"currency":          "INR",
		"key_id":            cfg.KeyID,
		"receipt":           razorpayOrder.Receipt,
		"has_saved_address": hasValidAddress,
		"saved_address":     savedAddressInfo, // Fallback address if Magic Checkout doesn't provide one
	}

	// Add warning if no saved address (fallback won't work)
	if !hasValidAddress {
		response["address_warning"] = "No saved address found. Please ensure you enter a complete address during checkout."
	}

	// Store order data in cache for coupon apply-promotion to retrieve cart total
	// This is critical because Razorpay sends our receipt ID, not the Razorpay order ID
	StoreMagicCheckoutOrder(razorpayOrder.Receipt, razorpayOrder.ID, cartTotal, userID)

	h.logger.Info("Magic Checkout order created successfully",
		zap.String("request_id", requestID),
		zap.String("razorpay_order_id", razorpayOrder.ID),
		zap.String("receipt", razorpayOrder.Receipt),
		zap.String("user_id", userID),
		zap.Float64("cart_total", cartTotal),
		zap.Float64("discounted_total", discountedTotal),
		zap.Int("line_items", len(lineItems)))

	c.JSON(http.StatusOK, response)
}
