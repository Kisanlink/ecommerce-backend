package handlers

import (
	"context"
	"math"
	"net/http"
	"regexp"
	"strings"

	"fmt"
	"html/template"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
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
	logger          *zap.Logger
	authContext     *auth.AuthContext
}

// NewPaymentHandler creates a new payment handler
func NewPaymentHandler(paymentService services.PaymentServiceInterface, orderService services.OrderServiceInterface, deliveryService services.DeliveryServiceInterface, logger *zap.Logger) *PaymentHandler {
	return &PaymentHandler{
		paymentService:  paymentService,
		orderService:    orderService,
		deliveryService: deliveryService,
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
	FirstName         string  `json:"first_name"` // Required
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
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	PaymentID   string `json:"payment_id,omitempty"`
	OrderID     string `json:"order_id,omitempty"`
	AWBNumber   string `json:"awb_number,omitempty"`
	TrackingURL string `json:"tracking_url,omitempty"`
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

	// Calculate totals from cart
	if finalAmount <= 0 || (req.Subtotal > 0) {
		// Calculate from provided subtotal or get dynamic shipping charges
		subtotal := req.Subtotal
		if subtotal > 0 {
			shippingCharges = subtotal - discountedCartTotal
			if shippingCharges < 0 {
				shippingCharges = 0
			}
			finalAmount = subtotal
			h.logger.Info("Using provided subtotal",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Float64("provided_subtotal", subtotal),
				zap.Float64("discounted_cart_total", discountedCartTotal),
				zap.Float64("calculated_shipping", shippingCharges))
		} else {
			// Get shipping charges dynamically from delivery service API with robust fallback
			h.logger.Info("Calculating dynamic shipping charges",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Float64("discounted_cart_total", discountedCartTotal),
				zap.Int("item_count", len(cart.Items)))

			shippingCharges = h.calculateShippingChargesRobust(c.Request.Context(), userID, shippingAddress, discountedCartTotal, len(cart.Items))
			finalAmount = utils.AddShippingCharges(discountedCartTotal, shippingCharges)
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment order", "details": err.Error()})
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
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Payment verification API failed",
			"code":    "PAYMENT_VERIFICATION_ERROR",
			"details": err.Error()})
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

		// Check if this payment has already been processed to avoid duplicate orders
		h.logger.Info("Checking for existing orders with this Razorpay order ID",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("razorpay_order_id", req.OrderID))

		existingOrders, getOrdersErr := h.orderService.GetUserOrders(c.Request.Context(), userID)
		if getOrdersErr == nil {
			for _, existingOrder := range existingOrders {
				if existingOrder.RazorpayOrderID == req.OrderID {
					h.logger.Warn("Duplicate order creation attempt - order already exists",
						zap.String("request_id", requestID),
						zap.String("user_id", userID),
						zap.String("existing_order_id", existingOrder.ID),
						zap.String("razorpay_order_id", req.OrderID),
						zap.String("event_type", "duplicate_order_prevention"))

					// Return the existing order details instead of creating new one
					response = VerifyPaymentResponse{
						Success:     true,
						Message:     "Payment already processed - returning existing order",
						PaymentID:   req.PaymentID,
						OrderID:     existingOrder.ID,
						AWBNumber:   existingOrder.DelhiveryWaybill,
						TrackingURL: existingOrder.TrackingURL,
					}
					c.JSON(http.StatusOK, response)
					return
				}
			}
		} else {
			h.logger.Warn("Failed to check for existing orders",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Error(getOrdersErr))
		}

		// Fetch Razorpay order to get notes (subtotal, cart_total, shipping_charges, shipping_address)
		h.logger.Info("Fetching Razorpay order metadata",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("razorpay_order_id", req.OrderID))

		rzpOrder, getErr := h.paymentService.GetOrder(c.Request.Context(), req.OrderID)
		if getErr != nil {
			h.logger.Warn("Failed to fetch Razorpay order metadata - proceeding without metadata",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Error(getErr))
		}

		// Derive shipping values from notes
		notes := map[string]interface{}{}
		var shippingAddress string = ""
		var shippingAddressID string = ""
		var subtotal float64 = 0
		var cartTotal float64 = 0
		var shippingCharges float64 = 0
		var etaMin, etaMax int = 0, 0
		var firstName, lastName, mobile, mobileSecondary string
		if rzpOrder != nil && rzpOrder.Notes != nil {
			notes = rzpOrder.Notes
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

		// Use atomic order creation to prevent race conditions and duplicates
		h.logger.Info("Creating database order atomically",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("razorpay_order_id", req.OrderID),
			zap.Float64("shipping_charges", shippingCharges))

		// Create normalized customer and address data for the order
		// Provide default last name if not available
		if lastName == "" {
			lastName = "Customer" // Default value for required field
		}

		customerData := models.NewOrderCustomer(
			firstName,
			lastName,
			mobile,
			"", // phoneAlt not available
			"", // email not available
		)

		// Create normalized address data - extract structured address from Razorpay notes
		var addressData *models.OrderAddress

		// Try to extract structured address data from Razorpay notes first
		if rzpOrder != nil && rzpOrder.Notes != nil {
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
			if addr, err := h.orderService.GetAddressByID(c.Request.Context(), userID, shippingAddressID); err == nil && addr != nil {
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

		// Final fallback to string parsing
		if addressData == nil {
			h.logger.Warn("No structured address data available, falling back to string parsing",
				zap.String("request_id", requestID),
				zap.String("shipping_address", shippingAddress))
			addressData = h.parseShippingAddressToOrderAddress(shippingAddress)
		}

		// Use the new normalized order creation method
		order, err := h.orderService.CreateOrder(
			c.Request.Context(),
			userID,
			customerData,
			addressData,
			"razorpay",
			shippingCharges,
		)
		if err != nil {
			h.logger.Error("Failed to create database order after payment verification",
				zap.String("request_id", requestID),
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("payment_id", req.PaymentID),
				zap.String("razorpay_order_id", req.OrderID),
				zap.String("error_type", "order_creation_error"))

			// Check if this is a duplicate order error
			if strings.Contains(err.Error(), "already exists for razorpay order ID") ||
				strings.Contains(err.Error(), "duplicate key") ||
				strings.Contains(err.Error(), "UNIQUE constraint") ||
				strings.Contains(err.Error(), "Duplicate entry") {
				h.logger.Error("Duplicate order creation detected",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.String("payment_id", req.PaymentID),
					zap.String("razorpay_order_id", req.OrderID),
					zap.String("error_type", "duplicate_order_error"))
				c.JSON(http.StatusConflict, gin.H{
					"error": "This payment has already been processed. Please check your order history.",
					"code":  "PAYMENT_ALREADY_PROCESSED"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment verified but failed to create order"})
			}
			return
		}

		// Update the order with Razorpay order ID and payment details
		order.RazorpayOrderID = req.OrderID
		order.PaymentStatus = "paid"
		order.Status = models.OrderStatusConfirmed

		if err := h.orderService.UpdateOrder(c.Request.Context(), order); err != nil {
			h.logger.Error("Failed to update order with Razorpay details",
				zap.String("request_id", requestID),
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("order_id", order.ID),
				zap.String("razorpay_order_id", req.OrderID))
			// Don't fail the entire operation, just log the error
		}

		h.logger.Info("Database order created successfully after payment verification",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("order_id", order.ID),
			zap.String("payment_id", req.PaymentID),
			zap.String("razorpay_order_id", req.OrderID),
			zap.Float64("total_amount", order.TotalAmount))

		// Update order with payment details and confirm the order
		if err := h.orderService.UpdatePaymentStatusAndOrderStatus(c.Request.Context(), order.ID, "successful", req.PaymentID, models.OrderStatusConfirmed); err != nil {
			h.logger.Error("Failed to update payment and order status",
				zap.String("request_id", requestID),
				zap.Error(err),
				zap.String("order_id", order.ID),
				zap.String("payment_id", req.PaymentID),
				zap.String("error_type", "status_update_error"))
			// Don't fail the verification, but log the error
		} else {
			h.logger.Info("Order status updated after successful payment",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("payment_id", req.PaymentID),
				zap.String("order_status", "confirmed"),
				zap.String("payment_status", "successful"))

			// Fetch the updated order to get the current status
			updatedOrder, err := h.orderService.GetOrderByID(c.Request.Context(), order.ID)
			if err != nil {
				h.logger.Error("Failed to fetch updated order",
					zap.String("request_id", requestID),
					zap.Error(err),
					zap.String("order_id", order.ID),
					zap.String("error_type", "order_fetch_error"))
			} else {
				order = updatedOrder // Use the updated order with correct status
				h.logger.Info("Updated order fetched successfully",
					zap.String("request_id", requestID),
					zap.String("order_id", order.ID),
					zap.String("payment_status", order.PaymentStatus),
					zap.String("order_status", string(order.Status)))
			}
		}

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
			_ = h.orderService.UpdateOrderNotes(c.Request.Context(), order.ID, notesStr)
		}

		// Generate invoice and attach URL
		if url, invErr := h.orderService.GenerateAndAttachInvoice(c.Request.Context(), order.ID); invErr != nil {
			h.logger.Warn("Failed to generate invoice",
				zap.String("request_id", requestID),
				zap.Error(invErr),
				zap.String("order_id", order.ID))
		} else if url != "" {
			h.logger.Info("Invoice generated successfully",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("invoice_url", url))
		}

		// Create Delhivery shipment after successful order creation with enhanced error handling
		h.logger.Info("=== STARTING DELHIVERY SHIPMENT CREATION FLOW ===",
			zap.String("request_id", requestID),
			zap.String("order_id", order.ID),
			zap.String("user_id", userID),
			zap.String("flow_step", "shipment_creation_start"))

		var awbNumber, trackingURL string
		orderWithItems, orderItems, getOrderErr := h.orderService.GetOrderWithItems(c.Request.Context(), order.ID)

		h.logger.Info("Order items retrieval result",
			zap.String("request_id", requestID),
			zap.String("order_id", order.ID),
			zap.Error(getOrderErr),
			zap.Bool("has_error", getOrderErr != nil),
			zap.Bool("order_is_nil", orderWithItems == nil),
			zap.Int("items_count", len(orderItems)),
			zap.String("flow_step", "order_items_retrieval"))

		if getOrderErr != nil {
			h.logger.Error("Failed to get order items for shipment creation",
				zap.String("request_id", requestID),
				zap.Error(getOrderErr),
				zap.String("order_id", order.ID),
				zap.String("user_id", userID),
				zap.String("error_type", "order_items_retrieval_error"),
				zap.String("flow_step", "order_items_retrieval_failed"))
			// Continue without shipment creation - order is still valid
		} else if len(orderItems) == 0 {
			h.logger.Error("No order items found for shipment creation",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("user_id", userID),
				zap.String("error_type", "empty_order_items_error"),
				zap.String("flow_step", "empty_order_items"))
			// Continue without shipment creation - order is still valid
		} else {
			h.logger.Info("=== PROCEEDING WITH SHIPMENT CREATION ===",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("flow_step", "shipment_creation_proceed"))

			// Enhanced shipment creation with detailed logging
			h.logger.Info("Creating Delhivery shipment - detailed context",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("user_id", userID),
				zap.Int("item_count", len(orderItems)),
				zap.String("shipping_address", orderWithItems.ShippingAddr),
				zap.String("shipping_addr_id", orderWithItems.ShippingAddrID),
				zap.String("customer_name", orderWithItems.CustomerFirstName+" "+orderWithItems.CustomerLastName),
				zap.String("customer_phone", orderWithItems.CustomerPhone),
				zap.Float64("total_amount", orderWithItems.TotalAmount),
				zap.Float64("shipping_charge", orderWithItems.ShippingCharge),
				zap.String("flow_step", "calling_delivery_service"))

			shipmentResult, shipmentErr := h.deliveryService.CreateShipment(c.Request.Context(), orderWithItems, orderItems)

			h.logger.Info("=== SHIPMENT CREATION SERVICE RESPONSE ===",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.Bool("has_error", shipmentErr != nil),
				zap.Bool("result_is_nil", shipmentResult == nil),
				zap.String("flow_step", "shipment_service_response"))

			// Enhanced shipment result logging
			if shipmentErr != nil {
				h.logger.Error("=== DELHIVERY SHIPMENT CREATION FAILED WITH ERROR ===",
					zap.String("request_id", requestID),
					zap.Error(shipmentErr),
					zap.String("order_id", order.ID),
					zap.String("user_id", userID),
					zap.String("error_type", "shipment_creation_error"),
					zap.String("error_details", shipmentErr.Error()),
					zap.String("error_message", shipmentErr.Error()),
					zap.String("flow_step", "shipment_creation_error"))

				// Even if shipment fails, payment is successful - notify user
				h.logger.Warn("Order created successfully but shipment creation failed - manual intervention required",
					zap.String("request_id", requestID),
					zap.String("order_id", order.ID),
					zap.String("flow_step", "shipment_failed_but_order_success"))
			} else if shipmentResult == nil {
				h.logger.Error("=== DELHIVERY SHIPMENT RESULT IS NIL ===",
					zap.String("request_id", requestID),
					zap.String("order_id", order.ID),
					zap.String("error_type", "nil_shipment_result"),
					zap.String("flow_step", "nil_shipment_result"))
			} else {
				h.logger.Info("=== ANALYZING SHIPMENT RESULT ===",
					zap.String("request_id", requestID),
					zap.String("order_id", order.ID),
					zap.Bool("result_success", shipmentResult.Success),
					zap.String("result_message", shipmentResult.Message),
					zap.String("result_waybill", shipmentResult.Waybill),
					zap.String("result_tracking_url", shipmentResult.TrackingURL),
					zap.String("result_status", shipmentResult.Status),
					zap.String("flow_step", "shipment_result_analysis"))

				if !shipmentResult.Success {
					h.logger.Error("=== DELHIVERY API RETURNED FAILURE STATUS ===",
						zap.String("request_id", requestID),
						zap.String("order_id", order.ID),
						zap.String("message", shipmentResult.Message),
						zap.String("error_type", "shipment_api_failure"),
						zap.String("flow_step", "api_failure_status"))
				} else if shipmentResult.Waybill == "" {
					h.logger.Error("=== EMPTY WAYBILL RECEIVED FROM DELHIVERY ===",
						zap.String("request_id", requestID),
						zap.String("order_id", order.ID),
						zap.String("message", shipmentResult.Message),
						zap.String("status", shipmentResult.Status),
						zap.String("tracking_url", shipmentResult.TrackingURL),
						zap.String("error_type", "empty_waybill_error"),
						zap.String("flow_step", "empty_waybill_error"))
				} else {
					h.logger.Info("=== DELHIVERY SHIPMENT CREATED SUCCESSFULLY ===",
						zap.String("request_id", requestID),
						zap.String("order_id", order.ID),
						zap.String("awb_number", shipmentResult.Waybill),
						zap.String("tracking_url", shipmentResult.TrackingURL),
						zap.String("status", shipmentResult.Status),
						zap.String("flow_step", "shipment_success"))

					// Shipment created successfully
					awbNumber = shipmentResult.Waybill
					trackingURL = shipmentResult.TrackingURL

					// Ensure tracking URL is not empty - generate if needed
					if trackingURL == "" && awbNumber != "" {
						trackingURL = fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", awbNumber)
						h.logger.Info("Generated tracking URL for AWB",
							zap.String("request_id", requestID),
							zap.String("awb", awbNumber),
							zap.String("tracking_url", trackingURL),
							zap.String("flow_step", "tracking_url_generated"))
					}

					h.logger.Info("=== UPDATING ORDER WITH SHIPMENT DETAILS ===",
						zap.String("request_id", requestID),
						zap.String("order_id", order.ID),
						zap.String("waybill", shipmentResult.Waybill),
						zap.String("tracking_url", trackingURL),
						zap.String("status", shipmentResult.Status),
						zap.String("flow_step", "updating_order_with_shipment"))

					// Update order with shipment details atomically
					if updateErr := h.orderService.UpdateOrderWithShipmentAtomic(
						c.Request.Context(),
						order.ID,
						shipmentResult.Waybill,
						trackingURL,
						shipmentResult.Status,
					); updateErr != nil {
						h.logger.Error("=== FAILED TO UPDATE ORDER WITH SHIPMENT DETAILS ===",
							zap.String("request_id", requestID),
							zap.Error(updateErr),
							zap.String("order_id", order.ID),
							zap.String("waybill", shipmentResult.Waybill),
							zap.String("error_type", "shipment_update_error"),
							zap.String("flow_step", "order_update_failed"))
					} else {
						h.logger.Info("=== ORDER UPDATED WITH SHIPMENT DETAILS SUCCESSFULLY ===",
							zap.String("request_id", requestID),
							zap.String("order_id", order.ID),
							zap.String("waybill", shipmentResult.Waybill),
							zap.String("tracking_url", trackingURL),
							zap.String("status", shipmentResult.Status),
							zap.String("flow_step", "order_update_success"))
					}
				}
			}
		}

		// Clear user's cart after successful order creation
		h.logger.Info("Clearing user cart after successful order creation",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("order_id", order.ID))

		if err := h.orderService.ClearUserCart(c.Request.Context(), userID); err != nil {
			h.logger.Warn("Failed to clear cart after payment",
				zap.String("request_id", requestID),
				zap.Error(err),
				zap.String("user_id", userID))
		} else {
			h.logger.Info("User cart cleared successfully",
				zap.String("request_id", requestID),
				zap.String("user_id", userID))
		}

		// Final verification: Fetch the order one more time to ensure payment status is correct
		finalOrder, err := h.orderService.GetOrderByID(c.Request.Context(), order.ID)
		if err != nil {
			h.logger.Error("Failed to fetch final order for verification",
				zap.String("request_id", requestID),
				zap.Error(err),
				zap.String("order_id", order.ID),
				zap.String("error_type", "final_order_fetch_error"))
		} else {
			order = finalOrder // Use the final verified order
			h.logger.Info("Final order verification completed",
				zap.String("request_id", requestID),
				zap.String("order_id", order.ID),
				zap.String("payment_status", order.PaymentStatus),
				zap.String("order_status", string(order.Status)))
		}

		// Prepare comprehensive response
		h.logger.Info("=== PREPARING PAYMENT VERIFICATION RESPONSE ===",
			zap.String("request_id", requestID),
			zap.String("order_id", order.ID),
			zap.String("awb_number", awbNumber),
			zap.String("tracking_url", trackingURL),
			zap.Bool("has_awb", awbNumber != ""),
			zap.Bool("has_tracking_url", trackingURL != ""),
			zap.String("flow_step", "preparing_response"))

		responseMessage := "Payment verified and order created successfully"
		if awbNumber != "" {
			responseMessage += fmt.Sprintf(". Shipment AWB: %s", awbNumber)
		} else {
			responseMessage += ". Shipment will be processed shortly"
		}

		response = VerifyPaymentResponse{
			Success:     true,
			Message:     responseMessage,
			PaymentID:   req.PaymentID,
			OrderID:     order.ID,
			AWBNumber:   awbNumber,
			TrackingURL: trackingURL,
		}

		// Comprehensive success logging
		h.logger.Info("Payment verification and order creation completed successfully",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("order_id", order.ID),
			zap.String("payment_id", req.PaymentID),
			zap.String("razorpay_order_id", req.OrderID),
			zap.String("awb_number", awbNumber),
			zap.String("tracking_url", trackingURL),
			zap.Float64("total_amount", order.TotalAmount),
			zap.Float64("shipping_charges", order.ShippingCharge))
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

	// Read request body
	body, err := c.GetRawData()
	if err != nil {
		h.logger.Error("Failed to read webhook body", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}

	// Process webhook
	if err := h.paymentService.ProcessWebhook(c.Request.Context(), body, signature); err != nil {
		h.logger.Error("Webhook processing failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Webhook processing failed"})
		return
	}

	h.logger.Info("Webhook processed successfully")
	c.JSON(http.StatusOK, gin.H{"message": "Webhook processed successfully"})
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
