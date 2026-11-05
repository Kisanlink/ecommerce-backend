package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"go.uber.org/zap"
)

// PaymentServiceInterface defines the contract for payment operations
type PaymentServiceInterface interface {
	CreateOrder(ctx context.Context, userID string, amount float64, currency string, cartID string, notes map[string]interface{}) (*RazorpayOrder, error)
	CreateOrderWithOptions(ctx context.Context, orderOptions map[string]interface{}) (*RazorpayOrder, error)
	VerifyPayment(ctx context.Context, orderID, paymentID, signature string) (bool, error)
	ProcessWebhook(ctx context.Context, body []byte, signature string) error
	ParseWebhook(body []byte, signature string) (*RazorpayWebhookPayload, error)
	GetPaymentStatus(ctx context.Context, paymentID string) (*RazorpayPayment, error)
	GetOrder(ctx context.Context, orderID string) (*RazorpayOrder, error)
	VerifyMagicCheckoutSignature(body []byte, signature string) (configured bool, valid bool)
}

// PaymentService implements payment business logic with Razorpay
type PaymentService struct {
	httpClient *http.Client
	config     config.RazorpayConfig
	logger     *zap.Logger
}

// NewPaymentService creates a new payment service with Razorpay integration
func NewPaymentService(logger *zap.Logger) PaymentServiceInterface {
	return &PaymentService{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config.GetRazorpayConfig(),
		logger: logger,
	}
}

// RazorpayOrder represents Razorpay order response
type RazorpayOrder struct {
	ID             string                   `json:"id"`
	Entity         string                   `json:"entity"`
	Amount         int                      `json:"amount"`      // Amount in paisa
	AmountPaid     int                      `json:"amount_paid"` // Amount paid in paisa
	AmountDue      int                      `json:"amount_due"`  // Amount due in paisa
	Currency       string                   `json:"currency"`
	Receipt        string                   `json:"receipt"`
	OfferID        string                   `json:"offer_id,omitempty"`
	Status         string                   `json:"status"`
	Attempts       int                      `json:"attempts"`
	Notes          map[string]interface{}   `json:"notes"`
	CreatedAt      int64                    `json:"created_at"`
	Description    string                   `json:"description,omitempty"`
	Checkout       map[string]interface{}   `json:"checkout,omitempty"`
	Promotions     []map[string]interface{} `json:"promotions,omitempty"`
	CODFee         int                      `json:"cod_fee,omitempty"`
	ShippingFee    int                      `json:"shipping_fee,omitempty"`
	LineItemsTotal int                      `json:"line_items_total,omitempty"`
	TaxDetails     map[string]interface{}   `json:"tax_details,omitempty"`
	// Customer details from Magic Checkout
	CustomerDetails *CustomerDetails `json:"customer_details,omitempty"`
}

// CustomerDetails represents customer details from Magic Checkout
type CustomerDetails struct {
	Email           string        `json:"email"`
	Contact         string        `json:"contact"`
	BillingAddress  *MagicAddress `json:"billing_address,omitempty"`
	ShippingAddress *MagicAddress `json:"shipping_address,omitempty"`
}

// MagicAddress represents address from Magic Checkout
type MagicAddress struct {
	ID      string `json:"id,omitempty"`
	Type    string `json:"type,omitempty"`
	Name    string `json:"name"`
	Contact string `json:"contact,omitempty"`
	Line1   string `json:"line1"`
	Line2   string `json:"line2,omitempty"`
	City    string `json:"city"`
	State   string `json:"state"`
	Country string `json:"country"`
	Zipcode string `json:"zipcode"`
	Tag     string `json:"tag,omitempty"`
}

// RazorpayPayment represents Razorpay payment response
type RazorpayPayment struct {
	ID               string                 `json:"id"`
	Entity           string                 `json:"entity"`
	Amount           int                    `json:"amount"`
	Currency         string                 `json:"currency"`
	Status           string                 `json:"status"`
	OrderID          string                 `json:"order_id"`
	Invoice          string                 `json:"invoice_id,omitempty"`
	International    bool                   `json:"international"`
	Method           string                 `json:"method"`
	AmountRefunded   int                    `json:"amount_refunded"`
	RefundStatus     string                 `json:"refund_status,omitempty"`
	Captured         bool                   `json:"captured"`
	Description      string                 `json:"description,omitempty"`
	CardID           string                 `json:"card_id,omitempty"`
	Bank             string                 `json:"bank,omitempty"`
	Wallet           string                 `json:"wallet,omitempty"`
	VPA              string                 `json:"vpa,omitempty"`
	Email            string                 `json:"email"`
	Contact          string                 `json:"contact"`
	Notes            map[string]interface{} `json:"notes"`
	Fee              int                    `json:"fee"`
	Tax              int                    `json:"tax"`
	ErrorCode        string                 `json:"error_code,omitempty"`
	ErrorDescription string                 `json:"error_description,omitempty"`
	CreatedAt        int64                  `json:"created_at"`
	// Magic Checkout fields - customer details with shipping address
	CustomerDetails *CustomerDetails `json:"customer_details,omitempty"`
}

// RazorpayWebhookPayload represents webhook event payload
type RazorpayWebhookPayload struct {
	Entity    string   `json:"entity"`
	AccountID string   `json:"account_id"`
	Event     string   `json:"event"`
	Contains  []string `json:"contains"`
	Payload   struct {
		Payment struct {
			Entity RazorpayPayment `json:"payment"`
		} `json:"payment,omitempty"`
		Order struct {
			Entity RazorpayOrder `json:"order"`
		} `json:"order,omitempty"`
	} `json:"payload"`
	CreatedAt int64 `json:"created_at"`
}

// CreateOrder creates a new Razorpay order
func (s *PaymentService) CreateOrder(ctx context.Context, userID string, amount float64, currency string, cartID string, notes map[string]interface{}) (*RazorpayOrder, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("invalid amount %.2f; must be greater than 0", amount)
	}
	if s.config.KeyID == "" || s.config.KeySecret == "" {
		return nil, fmt.Errorf("razorpay credentials missing: KeyID or KeySecret not configured")
	}
	if currency == "" {
		currency = "INR"
	}
	// Convert amount to paisa (Razorpay uses smallest currency unit)
	amountPaisa := utils.RupeesToPaisa(amount)

	// Generate compact receipt (<= 40 chars) per Razorpay limits
	receipt := s.makeCompactReceipt(cartID)

	// Merge provided notes with required keys
	finalNotes := map[string]interface{}{}
	for k, v := range notes {
		finalNotes[k] = v
	}
	finalNotes["user_id"] = userID
	finalNotes["cart_id"] = cartID

	orderData := map[string]interface{}{
		"amount":   amountPaisa,
		"currency": currency,
		"receipt":  receipt,
		"notes":    finalNotes,
	}

	jsonData, err := json.Marshal(orderData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order data: %w", err)
	}

	endpoint := fmt.Sprintf("%s/orders", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(s.config.KeyID, s.config.KeySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Razorpay order creation failed", zap.Error(err))
		return nil, fmt.Errorf("failed to create order: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// SECURITY (#27): do not log the raw upstream body (may echo customer/payment
		// details); log only the status and the body length.
		s.logger.Error("Razorpay API error", zap.Int("status", resp.StatusCode), zap.Int("body_len", len(body)))
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var order RazorpayOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	s.logger.Info("Razorpay order created successfully",
		zap.String("order_id", order.ID),
		zap.String("user_id", userID),
		zap.Float64("amount", amount),
		zap.String("receipt", receipt))

	return &order, nil
}

// CreateOrderWithOptions creates a Razorpay order with custom options including line_items for Magic Checkout
func (s *PaymentService) CreateOrderWithOptions(ctx context.Context, orderOptions map[string]interface{}) (*RazorpayOrder, error) {
	// Validate required fields
	amount, ok := orderOptions["amount"].(int)
	if !ok {
		return nil, fmt.Errorf("amount must be an integer (in paise)")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("invalid amount %d; must be greater than 0", amount)
	}

	currency, ok := orderOptions["currency"].(string)
	if !ok || currency == "" {
		currency = "INR"
	}

	if s.config.KeyID == "" || s.config.KeySecret == "" {
		return nil, fmt.Errorf("razorpay credentials missing: KeyID or KeySecret not configured")
	}

	// Generate receipt if not provided
	receipt, ok := orderOptions["receipt"].(string)
	if !ok || receipt == "" {
		receipt = s.makeCompactReceipt(fmt.Sprintf("order_%d", time.Now().Unix()))
	}

	// Prepare order data
	orderData := map[string]interface{}{
		"amount":   amount,
		"currency": currency,
		"receipt":  receipt,
	}

	// Add notes if provided
	if notes, ok := orderOptions["notes"].(map[string]interface{}); ok {
		orderData["notes"] = notes
	}

	// Add line_items_total and line_items for Magic Checkout (if provided)
	var lineItemsCount int
	if lineItemsTotal, ok := orderOptions["line_items_total"].(int); ok {
		orderData["line_items_total"] = lineItemsTotal
	}
	if lineItems, ok := orderOptions["line_items"].([]interface{}); ok && len(lineItems) > 0 {
		orderData["line_items"] = lineItems
		lineItemsCount = len(lineItems)
	}

	jsonData, err := json.Marshal(orderData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal order data: %w", err)
	}

	// Validate credentials before making request
	if s.config.KeyID == "" || s.config.KeySecret == "" {
		keySecretDisplay := ""
		if s.config.KeySecret != "" {
			keySecretDisplay = "***"
		}
		s.logger.Error("Razorpay credentials missing in CreateOrderWithOptions",
			zap.String("key_id", s.config.KeyID),
			zap.Bool("has_key_secret", s.config.KeySecret != ""),
			zap.String("base_url", s.config.BaseURL))
		return nil, fmt.Errorf("razorpay credentials missing: KeyID=%q, KeySecret=%q (check RAZORPAY_KEY_ID and RAZORPAY_KEY_SECRET env vars)",
			s.config.KeyID,
			keySecretDisplay)
	}

	// Log key prefix for debugging (first 10 chars only)
	keyIDPrefix := s.config.KeyID
	if len(keyIDPrefix) > 10 {
		keyIDPrefix = keyIDPrefix[:10] + "..."
	}
	s.logger.Info("Creating Razorpay order with options",
		zap.String("key_id_prefix", keyIDPrefix),
		zap.String("base_url", s.config.BaseURL),
		zap.Int("amount", amount),
		zap.Bool("has_line_items", lineItemsCount > 0))

	endpoint := fmt.Sprintf("%s/orders", s.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(s.config.KeyID, s.config.KeySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Razorpay order creation failed", zap.Error(err))
		return nil, fmt.Errorf("failed to create order: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// SECURITY (#27): do not log the raw upstream body; log only status + length.
		s.logger.Error("Razorpay API error",
			zap.Int("status", resp.StatusCode),
			zap.Int("body_len", len(body)),
			zap.String("key_id_prefix", keyIDPrefix))
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var order RazorpayOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	s.logger.Info("Razorpay order created successfully with options",
		zap.String("order_id", order.ID),
		zap.Int("amount", order.Amount),
		zap.String("receipt", receipt),
		zap.Bool("has_line_items", lineItemsCount > 0))

	return &order, nil
}

// VerifyPayment verifies Razorpay payment signature with enhanced debugging
func (s *PaymentService) VerifyPayment(ctx context.Context, orderID, paymentID, signature string) (bool, error) {
	// Validate inputs
	if orderID == "" || paymentID == "" || signature == "" {
		s.logger.Error("Invalid payment verification inputs",
			zap.String("order_id", orderID),
			zap.String("payment_id", paymentID),
			zap.Bool("signature_empty", signature == ""))
		return false, fmt.Errorf("order ID, payment ID, and signature are required")
	}

	// Validate configuration
	if s.config.KeySecret == "" {
		s.logger.Error("Razorpay key secret not configured")
		return false, fmt.Errorf("payment service not properly configured")
	}

	// Create expected signature
	message := orderID + "|" + paymentID
	expectedSignature := s.generateSignature(message, s.config.KeySecret)

	// Enhanced signature comparison with detailed logging
	isValid := hmac.Equal([]byte(expectedSignature), []byte(signature))

	if isValid {
		s.logger.Info("Payment signature verified successfully",
			zap.String("order_id", orderID),
			zap.String("payment_id", paymentID),
			zap.String("message", message))
	} else {
		// SECURITY (#27): never log the expected/received signatures or secret material —
		// they are the HMAC output over order|payment with the key secret, and logging them
		// aids forgery/correlation. Log only the (non-secret) ids and a boolean.
		s.logger.Warn("Payment signature verification failed",
			zap.String("order_id", orderID),
			zap.String("payment_id", paymentID),
			zap.Bool("signatures_equal", false))
	}

	return isValid, nil
}

// GetPaymentStatus fetches payment details from Razorpay
func (s *PaymentService) GetPaymentStatus(ctx context.Context, paymentID string) (*RazorpayPayment, error) {
	endpoint := fmt.Sprintf("%s/payments/%s", s.config.BaseURL, paymentID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(s.config.KeyID, s.config.KeySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// SECURITY (#27): do not echo the raw upstream body to logs or callers.
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var payment RazorpayPayment
	if err := json.Unmarshal(body, &payment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// SECURITY (#27): never log Razorpay customer_details (email/contact/shipping
	// address are PII). Log only whether they are present, for diagnostics.
	if payment.CustomerDetails != nil {
		s.logger.Info("Payment has customer_details (Magic Checkout)",
			zap.String("payment_id", paymentID),
			zap.Bool("has_shipping_address", payment.CustomerDetails.ShippingAddress != nil))
	}

	return &payment, nil
}

// GetOrder fetches a Razorpay order by ID
func (s *PaymentService) GetOrder(ctx context.Context, orderID string) (*RazorpayOrder, error) {
	// Note: Razorpay Orders API only supports expand for: payments, payments.card, virtual_account, transfers, offers
	// customer_details is NOT supported as expand parameter but IS returned for Magic Checkout orders
	endpoint := fmt.Sprintf("%s/orders/%s", s.config.BaseURL, orderID)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(s.config.KeyID, s.config.KeySecret)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get order: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// SECURITY (#27): do not echo the raw upstream body to logs or callers.
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	var order RazorpayOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// SECURITY (#27): never log Razorpay customer_details (PII). Log presence only.
	if order.CustomerDetails != nil {
		s.logger.Info("Razorpay order has customer_details (Magic Checkout)",
			zap.String("order_id", orderID),
			zap.Bool("has_shipping_address", order.CustomerDetails.ShippingAddress != nil))
	} else {
		s.logger.Info("Razorpay order does NOT have customer_details",
			zap.String("order_id", orderID),
			zap.String("status", order.Status))
	}

	s.logger.Info("Razorpay order fetched",
		zap.String("order_id", orderID),
		zap.String("status", order.Status),
		zap.Int("amount", order.Amount))

	return &order, nil
}

// ParseWebhook verifies the webhook signature and returns the parsed payload.
// Fulfillment is driven by the handler (which has the order/delivery dependencies); this keeps
// signature verification and parsing in one place.
func (s *PaymentService) ParseWebhook(body []byte, signature string) (*RazorpayWebhookPayload, error) {
	expectedSignature := s.generateSignature(string(body), s.config.WebhookSecret)
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		s.logger.Error("Webhook signature verification failed",
			zap.String("received", signature))
		return nil, fmt.Errorf("invalid webhook signature")
	}

	var webhook RazorpayWebhookPayload
	if err := json.Unmarshal(body, &webhook); err != nil {
		return nil, fmt.Errorf("failed to parse webhook: %w", err)
	}
	return &webhook, nil
}

// ProcessWebhook processes Razorpay webhook events
func (s *PaymentService) ProcessWebhook(ctx context.Context, body []byte, signature string) error {
	webhook, err := s.ParseWebhook(body, signature)
	if err != nil {
		return err
	}

	s.logger.Info("Processing Razorpay webhook",
		zap.String("event", webhook.Event),
		zap.String("account_id", webhook.AccountID))

	// Handle different webhook events
	switch webhook.Event {
	case "payment.captured":
		return s.handlePaymentCaptured(ctx, webhook.Payload.Payment.Entity)
	case "payment.failed":
		return s.handlePaymentFailed(ctx, webhook.Payload.Payment.Entity)
	case "order.paid":
		return s.handleOrderPaid(ctx, webhook.Payload.Order.Entity)
	default:
		s.logger.Info("Unhandled webhook event", zap.String("event", webhook.Event))
	}

	return nil
}

// handlePaymentCaptured processes successful payment
func (s *PaymentService) handlePaymentCaptured(ctx context.Context, payment RazorpayPayment) error {
	s.logger.Info("Payment captured successfully",
		zap.String("payment_id", payment.ID),
		zap.String("order_id", payment.OrderID),
		zap.Int("amount", payment.Amount))

	// TODO: Update order status to confirmed
	// TODO: Clear cart
	// TODO: Send confirmation email/SMS
	// TODO: Update inventory

	return nil
}

// handlePaymentFailed processes failed payment
func (s *PaymentService) handlePaymentFailed(ctx context.Context, payment RazorpayPayment) error {
	s.logger.Warn("Payment failed",
		zap.String("payment_id", payment.ID),
		zap.String("order_id", payment.OrderID),
		zap.String("error_code", payment.ErrorCode),
		zap.String("error_description", payment.ErrorDescription))

	// TODO: Update order status to failed
	// TODO: Send failure notification
	// TODO: Restore cart if needed

	return nil
}

// handleOrderPaid processes order paid event
func (s *PaymentService) handleOrderPaid(ctx context.Context, order RazorpayOrder) error {
	s.logger.Info("Order paid successfully",
		zap.String("order_id", order.ID),
		zap.Int("amount", order.Amount))

	// TODO: Confirm order
	// TODO: Trigger fulfillment process

	return nil
}

// generateSignature generates HMAC signature
func (s *PaymentService) generateSignature(message, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// VerifyMagicCheckoutSignature verifies a Magic Checkout (1CC) callback signature.
//
// It returns configured=false when no 1CC secret is set — verification is DORMANT and
// the caller should skip enforcement (the callbacks keep their other #7 protections).
// When configured=true, valid reports whether the supplied X-Razorpay-Signature equals
// HMAC-SHA256(rawBody, 1CC secret), hex-encoded — the same construction as the webhook
// check (generateSignature), but with the dedicated Magic Checkout secret
// (RAZORPAY_MAGIC_CHECKOUT_SECRET), which is distinct from the webhook secret.
func (s *PaymentService) VerifyMagicCheckoutSignature(body []byte, signature string) (configured bool, valid bool) {
	secret := s.config.MagicCheckoutSecret
	if secret == "" {
		return false, false
	}
	expected := s.generateSignature(string(body), secret)
	return true, hmac.Equal([]byte(expected), []byte(signature))
}

// Helper function to convert paisa to rupees
func PaisaToRupees(paisa int) float64 {
	return utils.PaisaToRupees(paisa)
}

// Helper function to convert rupees to paisa
func RupeesToPaisa(rupees float64) int {
	return utils.RupeesToPaisa(rupees)
}

// makeCompactReceipt generates a short, URL-safe receipt string within Razorpay's 40-char limit
func (s *PaymentService) makeCompactReceipt(base string) string {
	// Hash the input to a short hex and combine with timestamp
	sum := sha256.Sum256([]byte(base))
	short := hex.EncodeToString(sum[:5]) // 10 hex chars
	r := fmt.Sprintf("pi_%s_%d", short, time.Now().Unix())
	if len(r) > 40 {
		return r[:40]
	}
	return r
}
