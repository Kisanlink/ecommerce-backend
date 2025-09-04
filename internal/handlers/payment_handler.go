package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"

	"fmt"
	"html/template"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
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
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	PaymentID   string `json:"payment_id,omitempty"`
	OrderID     string `json:"order_id,omitempty"`
	AWBNumber   string `json:"awb_number,omitempty"`
	TrackingURL string `json:"tracking_url,omitempty"`
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

	// Always use INR currency and prefer last delivery estimate for address and totals
	useCurrency := "INR" // Always INR for all transactions
	shippingAddress := ""
	finalAmount := 0.0
	shippingCharges := 0.0

	// First priority: Use cached estimate from shipping estimation if available
	if h.deliveryService != nil {
		if last, err := h.deliveryService.GetLastEstimate(c.Request.Context(), userID); err == nil && last != nil {
			shippingAddress = last.ShippingAddress
			shippingCharges = last.ShippingCharges
			finalAmount = last.Total // This includes subtotal + shipping
			// Currency is always INR from estimate, but force it anyway
			useCurrency = "INR"

			h.logger.Info("Using cached delivery estimate",
				zap.String("user_id", userID),
				zap.String("cached_address", shippingAddress),
				zap.Float64("cached_total", finalAmount),
				zap.Float64("cached_shipping", shippingCharges))
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

	// Step 2: Calculate totals - prefer cached estimate, fallback to dynamic calculation
	cartTotal := 0.0
	for _, item := range cart.Items {
		cartTotal += item.TotalPrice
	}

	// Force INR currency always
	req.Currency = useCurrency // Always "INR"

	// Use cached estimate totals if available, otherwise calculate dynamically
	if finalAmount <= 0 {
		// No cached estimate available - calculate from request or fallback
		subtotal := req.Subtotal
		if subtotal > 0 {
			shippingCharges = subtotal - cartTotal
			if shippingCharges < 0 {
				shippingCharges = 0
			}
			finalAmount = subtotal
		} else {
			// Get shipping charges dynamically from delivery service API
			if h.deliveryService != nil {
				// Try to get real shipping charges from API
				if dynamicCharges, err := h.calculateShippingFromAddress(c.Request.Context(), userID, shippingAddress, cartTotal, len(cart.Items)); err == nil {
					shippingCharges = dynamicCharges
				} else {
					h.logger.Warn("Failed to get dynamic shipping charges, using fallback", zap.Error(err))
					shippingCharges = 50.0 // Keep fallback for safety
				}
			} else {
				h.logger.Warn("Delivery service not available, using fallback shipping charges")
				shippingCharges = 50.0 // Keep fallback for safety
			}
			finalAmount = cartTotal + shippingCharges
		}

		h.logger.Info("Using dynamic calculation (no cached estimate)",
			zap.String("user_id", userID),
			zap.Float64("cart_total", cartTotal),
			zap.Float64("calculated_shipping", shippingCharges),
			zap.Float64("final_amount", finalAmount))
	}

	h.logger.Info("Calculated payment amounts",
		zap.Float64("cart_total", cartTotal),
		zap.Float64("shipping_charges", shippingCharges),
		zap.Float64("final_amount", finalAmount),
		zap.String("user_id", userID))

	// Step 3: Create ONLY Razorpay payment order (NO database order!)
	// Store all necessary information in notes for later order creation
	notes := map[string]interface{}{
		"user_id":             userID,
		"shipping_address":    shippingAddress,
		"shipping_address_id": req.ShippingAddressID,
		"cart_total":          cartTotal,
		"shipping_charges":    shippingCharges,
		"subtotal":            finalAmount, // Total including shipping
		"cart_items_count":    len(cart.Items),
		"currency":            "INR", // Always INR
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

		// Check if this payment has already been processed to avoid duplicate orders
		existingOrders, getOrdersErr := h.orderService.GetUserOrders(c.Request.Context(), userID)
		if getOrdersErr == nil {
			for _, existingOrder := range existingOrders {
				if existingOrder.RazorpayOrderID == req.OrderID {
					h.logger.Warn("Order already exists for this Razorpay order ID",
						zap.String("user_id", userID),
						zap.String("existing_order_id", existingOrder.ID),
						zap.String("razorpay_order_id", req.OrderID))

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
		}

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

			// Check if this is a duplicate key error
			if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
				h.logger.Error("Duplicate order creation attempted - this payment may have already been processed",
					zap.String("user_id", userID),
					zap.String("payment_id", req.PaymentID),
					zap.String("razorpay_order_id", req.OrderID))
				c.JSON(http.StatusConflict, gin.H{
					"error": "This payment has already been processed. Please check your order history.",
					"code":  "PAYMENT_ALREADY_PROCESSED"})
			} else {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Payment verified but failed to create order"})
			}
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

		// Create Delhivery shipment after successful order creation
		var awbNumber, trackingURL string
		orderWithItems, orderItems, getOrderErr := h.orderService.GetOrderWithItems(c.Request.Context(), order.ID)
		if getOrderErr != nil {
			h.logger.Error("Failed to get order items for shipment creation",
				zap.Error(getOrderErr),
				zap.String("order_id", order.ID))
		} else {
			// Create Delhivery shipment
			shipmentResult, shipmentErr := h.deliveryService.CreateShipment(c.Request.Context(), orderWithItems, orderItems)
			if shipmentErr != nil {
				h.logger.Error("Failed to create Delhivery shipment",
					zap.Error(shipmentErr),
					zap.String("order_id", order.ID),
					zap.String("user_id", userID))
				// Don't fail the payment verification, just log the error
			} else if shipmentResult != nil && shipmentResult.Success && shipmentResult.Waybill != "" {
				// Update order with Delhivery waybill number
				orderWithItems.DelhiveryWaybill = shipmentResult.Waybill
				orderWithItems.DelhiveryStatus = shipmentResult.Status
				// Use tracking URL from shipment result
				orderWithItems.TrackingURL = shipmentResult.TrackingURL

				// Store for response
				awbNumber = shipmentResult.Waybill
				trackingURL = shipmentResult.TrackingURL

				if updateErr := h.orderService.UpdateOrder(c.Request.Context(), orderWithItems); updateErr != nil {
					h.logger.Error("Failed to update order with Delhivery waybill",
						zap.Error(updateErr),
						zap.String("order_id", order.ID),
						zap.String("waybill", shipmentResult.Waybill))
				} else {
					h.logger.Info("Delhivery shipment created successfully",
						zap.String("order_id", order.ID),
						zap.String("waybill", shipmentResult.Waybill),
						zap.String("tracking_url", orderWithItems.TrackingURL))
				}
			}
		}

		// Clear user's cart after successful order creation
		if err := h.orderService.ClearUserCart(c.Request.Context(), userID); err != nil {
			h.logger.Warn("Failed to clear cart after payment", zap.Error(err), zap.String("user_id", userID))
		}

		response = VerifyPaymentResponse{
			Success:     true,
			Message:     "Payment verified and order created successfully",
			PaymentID:   req.PaymentID,
			OrderID:     order.ID,
			AWBNumber:   awbNumber,
			TrackingURL: trackingURL,
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
