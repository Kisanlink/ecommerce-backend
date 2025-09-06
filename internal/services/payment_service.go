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

	"math"

	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"go.uber.org/zap"
)

// PaymentServiceInterface defines the contract for payment operations
type PaymentServiceInterface interface {
	CreateOrder(ctx context.Context, userID string, amount float64, currency string, cartID string, notes map[string]interface{}) (*RazorpayOrder, error)
	VerifyPayment(ctx context.Context, orderID, paymentID, signature string) (bool, error)
	ProcessWebhook(ctx context.Context, body []byte, signature string) error
	GetPaymentStatus(ctx context.Context, paymentID string) (*RazorpayPayment, error)
	GetOrder(ctx context.Context, orderID string) (*RazorpayOrder, error)
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
	ID         string                 `json:"id"`
	Entity     string                 `json:"entity"`
	Amount     int                    `json:"amount"`      // Amount in paisa
	AmountPaid int                    `json:"amount_paid"` // Amount paid in paisa
	AmountDue  int                    `json:"amount_due"`  // Amount due in paisa
	Currency   string                 `json:"currency"`
	Receipt    string                 `json:"receipt"`
	OfferID    string                 `json:"offer_id,omitempty"`
	Status     string                 `json:"status"`
	Attempts   int                    `json:"attempts"`
	Notes      map[string]interface{} `json:"notes"`
	CreatedAt  int64                  `json:"created_at"`
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
	amountPaisa := int(math.Round(amount * 100))

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
		s.logger.Error("Razorpay API error", zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
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
		s.logger.Warn("Payment signature verification FAILED - detailed debug info",
			zap.String("order_id", orderID),
			zap.String("payment_id", paymentID),
			zap.String("message", message),
			zap.String("expected_signature", expectedSignature),
			zap.String("received_signature", signature),
			zap.String("key_secret_length", fmt.Sprintf("%d", len(s.config.KeySecret))),
			zap.Bool("signatures_equal", expectedSignature == signature))
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
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var payment RazorpayPayment
	if err := json.Unmarshal(body, &payment); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &payment, nil
}

// GetOrder fetches a Razorpay order by ID
func (s *PaymentService) GetOrder(ctx context.Context, orderID string) (*RazorpayOrder, error) {
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
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var order RazorpayOrder
	if err := json.Unmarshal(body, &order); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &order, nil
}

// ProcessWebhook processes Razorpay webhook events
func (s *PaymentService) ProcessWebhook(ctx context.Context, body []byte, signature string) error {
	// Verify webhook signature
	expectedSignature := s.generateSignature(string(body), s.config.WebhookSecret)
	if !hmac.Equal([]byte(expectedSignature), []byte(signature)) {
		s.logger.Error("Webhook signature verification failed",
			zap.String("expected", expectedSignature),
			zap.String("received", signature))
		return fmt.Errorf("invalid webhook signature")
	}

	// Parse webhook payload
	var webhook RazorpayWebhookPayload
	if err := json.Unmarshal(body, &webhook); err != nil {
		return fmt.Errorf("failed to parse webhook: %w", err)
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

// Helper function to convert paisa to rupees
func PaisaToRupees(paisa int) float64 {
	return float64(paisa) / 100.0
}

// Helper function to convert rupees to paisa
func RupeesToPaisa(rupees float64) int {
	return int(rupees * 100)
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
