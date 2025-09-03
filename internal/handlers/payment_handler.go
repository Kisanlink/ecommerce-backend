package handlers

import (
	"net/http"

	"fmt"
	"html/template"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

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
	Subtotal          float64 `json:"subtotal,omitempty"`            // Ignored if cached estimate available
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
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	PaymentID string `json:"payment_id,omitempty"`
}

// CheckoutPage serves a minimal Razorpay Checkout HTML to initiate payment
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
func (h *PaymentHandler) CreateOrder(c *gin.Context) {
	// Get user ID from context (works with both Firebase and JWT auth)
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid create order request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Prefer last delivery estimate for address and totals to avoid re-entry
	useCurrency := "INR" // hardcode
	shippingAddress := req.ShippingAddress
	finalAmount := 0.0
	shippingCharges := 0.0
	if h.deliveryService != nil {
		if last, err := h.deliveryService.GetLastEstimate(c.Request.Context(), userID); err == nil && last != nil {
			shippingAddress = last.ShippingAddress
			shippingCharges = last.ShippingCharges
			finalAmount = last.Total
			useCurrency = last.Currency
		}
	}
	// Fallback to provided address or address ID if cache not present
	if shippingAddress == "" {
		if req.ShippingAddressID != "" {
			if addr, err := h.orderService.GetAddressByID(c.Request.Context(), userID, req.ShippingAddressID); err == nil && addr != nil {
				shippingAddress = h.orderService.FormatAddress(addr)
			} else {
				h.logger.Error("Invalid shipping_address_id", zap.String("user_id", userID), zap.String("addr_id", req.ShippingAddressID), zap.Error(err))
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid shipping_address_id"})
				return
			}
		} else if req.ShippingAddress != "" {
			shippingAddress = req.ShippingAddress
		}
	}
	if shippingAddress == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Shipping address is required"})
		return
	}

	h.logger.Info("Creating payment intent for user", zap.String("user_id", userID))

	// Step 1: Get user's cart to calculate total (NO order creation yet!)
	cart, err := h.orderService.GetUserCart(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user cart",
			zap.Error(err),
			zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get cart"})
		return
	}

	if len(cart.Items) == 0 {
		h.logger.Error("User cart is empty", zap.String("user_id", userID))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cart is empty"})
		return
	}

	// Step 2: Calculate totals
	cartTotal := 0.0
	for _, item := range cart.Items {
		cartTotal += item.TotalPrice
	}

	// Prefer cached subtotal (includes shipping). If not, compute from cart + dynamic shipping.
	subtotal := req.Subtotal
	req.Currency = useCurrency // force INR

	// Compute shipping charges from subtotal - cartTotal when subtotal provided
	if finalAmount <= 0 {
		if subtotal > 0 {
			shippingCharges = subtotal - cartTotal
			if shippingCharges < 0 {
				shippingCharges = 0
			}
			finalAmount = subtotal
		} else {
			// Last-resort fallback: fixed shipping
			shippingCharges = 50.0
			finalAmount = cartTotal + shippingCharges
		}
	}

	h.logger.Info("Calculated payment amounts",
		zap.Float64("cart_total", cartTotal),
		zap.Float64("shipping_charges", shippingCharges),
		zap.Float64("final_amount", finalAmount),
		zap.String("user_id", userID))

	// Step 3: Create ONLY Razorpay payment order (NO database order!)
	// We'll store cart info and shipping address in notes for later use
	notes := map[string]interface{}{
		"user_id":             userID,
		"shipping_address":    shippingAddress,
		"shipping_address_id": req.ShippingAddressID,
		"cart_total":          cartTotal,
		"shipping_charges":    shippingCharges,
		"subtotal":            finalAmount,
		"cart_items_count":    len(cart.Items),
	}

	razorpayOrder, err := h.paymentService.CreateOrder(c.Request.Context(), userID, finalAmount, req.Currency, "payment_intent_"+userID, notes)
	if err != nil {
		h.logger.Error("Failed to create payment order",
			zap.Error(err),
			zap.String("user_id", userID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create payment order", "details": err.Error()})
		return
	}

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

	response := CreateOrderResponse{
		OrderID:     razorpayOrder.ID,
		Amount:      amountInRupees,
		Currency:    razorpayOrder.Currency,
		KeyID:       cfg.KeyID,
		Description: "Kisanlink Order Payment",
		Notes:       notes,
		CheckoutURL: checkoutURL,
	}

	h.logger.Info("Payment intent created successfully - NO database order yet",
		zap.String("razorpay_order_id", razorpayOrder.ID),
		zap.String("user_id", userID),
		zap.Float64("final_amount", amountInRupees),
		zap.Float64("shipping_charges", shippingCharges))

	c.JSON(http.StatusOK, response)
}

// VerifyPayment verifies Razorpay payment signature
func (h *PaymentHandler) VerifyPayment(c *gin.Context) {
	// Get user ID from context
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		h.logger.Error("User ID not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req VerifyPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid verify payment request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// Verify payment signature first
	isValid, err := h.paymentService.VerifyPayment(c.Request.Context(), req.OrderID, req.PaymentID, req.Signature)
	if err != nil {
		h.logger.Error("Payment verification error",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("order_id", req.OrderID),
			zap.String("payment_id", req.PaymentID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment verification failed"})
		return
	}

	var response VerifyPaymentResponse
	if isValid {
		h.logger.Info("Payment verified successfully - NOW creating order",
			zap.String("user_id", userID),
			zap.String("payment_id", req.PaymentID),
			zap.String("razorpay_order_id", req.OrderID))

		// Fetch Razorpay order to get notes (subtotal, cart_total, shipping_charges, shipping_address)
		rzpOrder, getErr := h.paymentService.GetOrder(c.Request.Context(), req.OrderID)
		if getErr != nil {
			h.logger.Warn("Failed to fetch Razorpay order for metadata", zap.Error(getErr))
		}

		// Derive shipping values from notes
		notes := map[string]interface{}{}
		var shippingAddress string = ""
		var shippingAddressID string = ""
		var subtotal float64 = 0
		var cartTotal float64 = 0
		var shippingCharges float64 = 0
		var etaMin, etaMax int = 0, 0
		if rzpOrder != nil && rzpOrder.Notes != nil {
			notes = rzpOrder.Notes
			if v, ok := notes["shipping_address"].(string); ok {
				shippingAddress = v
			}
			if v, ok := notes["shipping_address_id"].(string); ok {
				shippingAddressID = v
			}
			if v, ok := notes["subtotal"].(float64); ok {
				subtotal = v
			} else if v, ok := notes["subtotal"].(int); ok {
				subtotal = float64(v)
			}
			if v, ok := notes["cart_total"].(float64); ok {
				cartTotal = v
			} else if v, ok := notes["cart_total"].(int); ok {
				cartTotal = float64(v)
			}
			if v, ok := notes["shipping_charges"].(float64); ok {
				shippingCharges = v
			} else if v, ok := notes["shipping_charges"].(int); ok {
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
		}
		// If address string missing but address ID present, resolve again (safety)
		if shippingAddress == "" && shippingAddressID != "" {
			if addr, err := h.orderService.GetAddressByID(c.Request.Context(), userID, shippingAddressID); err == nil && addr != nil {
				shippingAddress = h.orderService.FormatAddress(addr)
			}
		}
		if shippingCharges < 0 {
			shippingCharges = 0
		}

		order, err := h.orderService.CreateOrderFromCartWithShipping(
			c.Request.Context(),
			userID,
			shippingAddress,
			"razorpay",
			shippingCharges,
		)
		if err != nil {
			h.logger.Error("Failed to create order after payment verification",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("payment_id", req.PaymentID))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment verified but failed to create order"})
			return
		}

		// Update order with payment details
		if err := h.orderService.UpdatePaymentStatus(c.Request.Context(), order.ID, "paid", req.PaymentID); err != nil {
			h.logger.Error("Failed to update payment status",
				zap.Error(err),
				zap.String("order_id", order.ID),
				zap.String("payment_id", req.PaymentID))
			// Don't fail the verification, but log the error
		}

		// Persist full payment and shipping metadata on order and notes
		if rzpOrder != nil {
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
			if updErr := h.orderService.UpdateOrder(c.Request.Context(), order); updErr != nil {
				h.logger.Warn("Failed to persist order metadata", zap.Error(updErr), zap.String("order_id", order.ID))
			}

			// Update notes compact string
			notesStr := fmt.Sprintf("rzp_order:%s, amount_paisa:%d, carrier:%s, mode:%s, charges:%.2f",
				rzpOrder.ID, rzpOrder.Amount, order.ShippingCarrier, order.ShippingMode, order.ShippingCharge)
			_ = h.orderService.UpdateOrderNotes(c.Request.Context(), order.ID, notesStr)
		}

		// Generate invoice and attach URL
		if url, invErr := h.orderService.GenerateAndAttachInvoice(c.Request.Context(), order.ID); invErr != nil {
			h.logger.Warn("Failed to generate invoice", zap.Error(invErr), zap.String("order_id", order.ID))
		} else if url != "" {
			h.logger.Info("Invoice generated", zap.String("order_id", order.ID), zap.String("invoice_url", url))
		}

		// Clear user's cart after successful order creation
		if err := h.orderService.ClearUserCart(c.Request.Context(), userID); err != nil {
			h.logger.Warn("Failed to clear cart after payment", zap.Error(err), zap.String("user_id", userID))
		}

		response = VerifyPaymentResponse{
			Success:   true,
			Message:   "Payment verified and order created successfully",
			PaymentID: req.PaymentID,
		}

		h.logger.Info("Order created successfully after payment verification",
			zap.String("user_id", userID),
			zap.String("order_id", order.ID),
			zap.String("payment_id", req.PaymentID),
			zap.String("razorpay_order_id", req.OrderID))
	} else {
		response = VerifyPaymentResponse{
			Success: false,
			Message: "Payment verification failed - invalid signature",
		}
		h.logger.Warn("Payment verification failed",
			zap.String("user_id", userID),
			zap.String("payment_id", req.PaymentID),
			zap.String("order_id", req.OrderID))
	}

	c.JSON(http.StatusOK, response)
}

// ProcessWebhook handles Razorpay webhook events
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
