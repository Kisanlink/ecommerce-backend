package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"go.uber.org/zap"
)

// WhatsAppService handles WhatsApp Business API integration
type WhatsAppService struct {
	config *config.WhatsAppEnvConfig
	client *http.Client
	logger *zap.Logger
}

// WhatsAppMessageRequest represents the request structure for sending WhatsApp messages
type WhatsAppMessageRequest struct {
	MessagingProduct string               `json:"messaging_product"`
	To               string               `json:"to"`
	Type             string               `json:"type"`
	Template         *WhatsAppTemplate    `json:"template,omitempty"`
	Text             *WhatsAppTextMessage `json:"text,omitempty"`
}

// WhatsAppTemplate represents a template message
type WhatsAppTemplate struct {
	Name       string              `json:"name"`
	Language   WhatsAppLanguage    `json:"language"`
	Components []WhatsAppComponent `json:"components,omitempty"`
}

// WhatsAppLanguage represents the language configuration
type WhatsAppLanguage struct {
	Code string `json:"code"`
}

// WhatsAppComponent represents a template component
type WhatsAppComponent struct {
	Type       string              `json:"type"`
	Parameters []WhatsAppParameter `json:"parameters,omitempty"`
}

// WhatsAppParameter represents a template parameter
type WhatsAppParameter struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// WhatsAppTextMessage represents a text message
type WhatsAppTextMessage struct {
	Body string `json:"body"`
}

// WhatsAppResponse represents the response from WhatsApp API
type WhatsAppResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// WhatsAppError represents an error response from WhatsApp API
type WhatsAppError struct {
	Error struct {
		Message   string `json:"message"`
		Type      string `json:"type"`
		Code      int    `json:"code"`
		ErrorData struct {
			MessagingProduct string `json:"messaging_product"`
			Details          string `json:"details"`
		} `json:"error_data"`
	} `json:"error"`
}

// NewWhatsAppService creates a new WhatsApp service
func NewWhatsAppService(logger *zap.Logger) *WhatsAppService {
	whatsappConfig := config.GetWhatsAppEnvConfig()

	return &WhatsAppService{
		config: &whatsappConfig,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger: logger,
	}
}

// SendOrderConfirmationMessage sends a WhatsApp message when an order is placed
func (w *WhatsAppService) SendOrderConfirmationMessage(ctx context.Context, order *models.Order, customerPhone string) error {
	if !w.config.Enabled {
		w.logger.Info("WhatsApp service is disabled, skipping order confirmation message")
		return nil
	}

	if customerPhone == "" {
		return fmt.Errorf("customer phone number is required for WhatsApp messaging")
	}

	// Format phone number (remove any non-digit characters and add country code if needed)
	formattedPhone := w.formatPhoneNumber(customerPhone)

	// Extract customer name, order number, and tracking URL from order
	customerName := w.extractCustomerName(order)
	orderNumber := w.extractOrderNumber(order)
	trackingURL := w.extractTrackingURL(order)

	w.logger.Info("Sending order confirmation message",
		zap.String("phone_number", formattedPhone),
		zap.String("customer_name", customerName),
		zap.String("order_id", order.ID),
		zap.String("order_number", orderNumber),
		zap.String("tracking_url", trackingURL))

	// Send order_management_amrti template with customer name, order number, and tracking URL
	return w.sendOrderManagementAmrtiTemplate(ctx, formattedPhone, customerName, orderNumber, trackingURL)
}

// SendPaymentOrderConfirmMessage sends a WhatsApp message using order_management_amrti template
func (w *WhatsAppService) SendPaymentOrderConfirmMessage(ctx context.Context, order *models.Order, customerPhone string) error {
	if !w.config.Enabled {
		w.logger.Info("WhatsApp service is disabled, skipping payment order confirmation message")
		return nil
	}

	if customerPhone == "" {
		return fmt.Errorf("customer phone number is required for WhatsApp messaging")
	}

	// Format phone number
	formattedPhone := w.formatPhoneNumber(customerPhone)

	// Extract customer name, order number, and tracking URL from order
	customerName := w.extractCustomerName(order)
	orderNumber := w.extractOrderNumber(order)
	trackingURL := w.extractTrackingURL(order)

	w.logger.Info("Sending payment order confirmation message",
		zap.String("phone_number", formattedPhone),
		zap.String("customer_name", customerName),
		zap.String("order_id", order.ID),
		zap.String("order_number", orderNumber),
		zap.String("tracking_url", trackingURL))

	// Send order_management_amrti template with customer name, order number, and tracking URL
	return w.sendOrderManagementAmrtiTemplate(ctx, formattedPhone, customerName, orderNumber, trackingURL)
}

// SendPaymentSuccessMessage sends a WhatsApp message when payment is successful
func (w *WhatsAppService) SendPaymentSuccessMessage(ctx context.Context, order *models.Order, customerPhone string) error {
	if !w.config.Enabled {
		w.logger.Info("WhatsApp service is disabled, skipping payment success message")
		return nil
	}

	if customerPhone == "" {
		return fmt.Errorf("customer phone number is required for WhatsApp messaging")
	}

	// Format phone number
	formattedPhone := w.formatPhoneNumber(customerPhone)

	// Extract customer name, order number, and tracking URL from order
	customerName := w.extractCustomerName(order)
	orderNumber := w.extractOrderNumber(order)
	trackingURL := w.extractTrackingURL(order)

	w.logger.Info("Sending payment confirmation message",
		zap.String("phone_number", formattedPhone),
		zap.String("customer_name", customerName),
		zap.String("order_id", order.ID),
		zap.String("order_number", orderNumber),
		zap.String("tracking_url", trackingURL))

	// Send order_management_amrti template with customer name, order number, and tracking URL
	return w.sendOrderManagementAmrtiTemplate(ctx, formattedPhone, customerName, orderNumber, trackingURL)
}

// SendTrackingMessage sends a WhatsApp message with tracking information
func (w *WhatsAppService) SendTrackingMessage(ctx context.Context, order *models.Order, customerPhone string) error {
	if !w.config.Enabled {
		w.logger.Info("WhatsApp service is disabled, skipping tracking message")
		return nil
	}

	if customerPhone == "" {
		return fmt.Errorf("customer phone number is required for WhatsApp messaging")
	}

	// Format phone number
	formattedPhone := w.formatPhoneNumber(customerPhone)

	// Extract customer name and order details
	customerName := w.extractCustomerName(order)
	orderNumber := w.extractOrderNumber(order)
	trackingURL := w.extractTrackingURL(order)

	w.logger.Info("Sending tracking message",
		zap.String("phone_number", formattedPhone),
		zap.String("customer_name", customerName),
		zap.String("order_number", orderNumber),
		zap.String("tracking_url", trackingURL),
		zap.String("order_id", order.ID))

	// Send order_management_amrti template with tracking information
	return w.sendOrderManagementAmrtiTemplate(ctx, formattedPhone, customerName, orderNumber, trackingURL)
}

// SendTrackingMessageWithURL sends a WhatsApp message with tracking information using provided tracking URL
func (w *WhatsAppService) SendTrackingMessageWithURL(ctx context.Context, order *models.Order, customerPhone, trackingURL string) error {
	if !w.config.Enabled {
		w.logger.Info("WhatsApp service is disabled, skipping tracking message")
		return nil
	}

	if customerPhone == "" {
		return fmt.Errorf("customer phone number is required for WhatsApp messaging")
	}

	if trackingURL == "" {
		w.logger.Warn("No tracking URL provided, skipping tracking message",
			zap.String("order_id", order.ID))
		return nil
	}

	// Format phone number
	formattedPhone := w.formatPhoneNumber(customerPhone)

	// Extract customer name and order details
	customerName := w.extractCustomerName(order)
	orderNumber := w.extractOrderNumber(order)

	w.logger.Info("Sending tracking message with provided URL",
		zap.String("phone_number", formattedPhone),
		zap.String("customer_name", customerName),
		zap.String("order_number", orderNumber),
		zap.String("tracking_url", trackingURL),
		zap.String("order_id", order.ID))

	// Send order_management_amrti template with tracking information
	return w.sendOrderManagementAmrtiTemplate(ctx, formattedPhone, customerName, orderNumber, trackingURL)
}

// sendOrderManagementAmrtiTemplate sends the order_management_amrti template message
// Template structure:
// - Body: 3 text parameters
//   - {{1}} = customer name
//   - {{2}} = order number
//   - {{3}} = tracking URL
func (w *WhatsAppService) sendOrderManagementAmrtiTemplate(ctx context.Context, phoneNumber, customerName, orderNumber, trackingURL string) error {
	if !w.config.Enabled {
		w.logger.Info("WhatsApp service is disabled, skipping order management message")
		return nil
	}

	if phoneNumber == "" {
		return fmt.Errorf("phone number is required for WhatsApp messaging")
	}

	// Format phone number
	formattedPhone := w.formatPhoneNumber(phoneNumber)

	url := fmt.Sprintf("%s/%s/messages", w.config.APIBaseURL, w.config.PhoneNumberID)

	// Template structure for order_management_amrti:
	// - Body: 3 text parameters (customer name {{1}}, order number {{2}}, tracking URL {{3}})
	requestBody := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                formattedPhone,
		"type":              "template",
		"template": map[string]interface{}{
			"name": "order_management_amrti",
			"language": map[string]string{
				"code": "en_US",
			},
			"components": []map[string]interface{}{
				{
					"type": "body",
					"parameters": []map[string]interface{}{
						{
							"type": "text",
							"text": customerName,
						},
						{
							"type": "text",
							"text": orderNumber,
						},
						{
							"type": "text",
							"text": trackingURL,
						},
					},
				},
			},
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		w.logger.Error("Failed to marshal WhatsApp order management request",
			zap.String("phone_number", formattedPhone),
			zap.String("customer_name", customerName),
			zap.String("order_number", orderNumber),
			zap.String("tracking_url", trackingURL),
			zap.Error(err))
		return fmt.Errorf("failed to marshal WhatsApp request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		w.logger.Error("Failed to create WhatsApp order management request",
			zap.String("phone_number", formattedPhone),
			zap.String("customer_name", customerName),
			zap.String("order_number", orderNumber),
			zap.String("tracking_url", trackingURL),
			zap.Error(err))
		return fmt.Errorf("failed to create WhatsApp request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.config.AccessToken)

	w.logger.Info("Sending WhatsApp order_management_amrti template",
		zap.String("phone_number", formattedPhone),
		zap.String("customer_name", customerName),
		zap.String("order_number", orderNumber),
		zap.String("tracking_url", trackingURL),
		zap.String("url", url),
		zap.String("request_body", string(jsonData)))

	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Error("WhatsApp order management API request failed",
			zap.String("phone_number", formattedPhone),
			zap.String("customer_name", customerName),
			zap.String("order_number", orderNumber),
			zap.String("tracking_url", trackingURL),
			zap.String("url", url),
			zap.Error(err))
		return fmt.Errorf("failed to send WhatsApp message: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	w.logger.Info("WhatsApp order management API response",
		zap.String("phone_number", formattedPhone),
		zap.String("customer_name", customerName),
		zap.String("order_number", orderNumber),
		zap.String("tracking_url", trackingURL),
		zap.Int("status_code", resp.StatusCode),
		zap.String("response_body", string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		var errorResp WhatsAppError
		if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
			w.logger.Error("Failed to decode WhatsApp error response",
				zap.String("phone_number", formattedPhone),
				zap.String("customer_name", customerName),
				zap.String("order_number", orderNumber),
				zap.String("tracking_url", trackingURL),
				zap.Int("status_code", resp.StatusCode),
				zap.String("response_body", string(bodyBytes)),
				zap.Error(err))
			return fmt.Errorf("WhatsApp API error (status %d): failed to decode error response", resp.StatusCode)
		}

		w.logger.Error("WhatsApp order management API error",
			zap.String("phone_number", formattedPhone),
			zap.String("customer_name", customerName),
			zap.String("order_number", orderNumber),
			zap.String("tracking_url", trackingURL),
			zap.Int("status_code", resp.StatusCode),
			zap.String("error_message", errorResp.Error.Message),
			zap.String("error_type", errorResp.Error.Type),
			zap.Int("error_code", errorResp.Error.Code))
		return fmt.Errorf("WhatsApp API error (status %d): %s", resp.StatusCode, errorResp.Error.Message)
	}

	var response WhatsAppResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		w.logger.Error("Failed to decode WhatsApp order management response",
			zap.String("phone_number", formattedPhone),
			zap.String("customer_name", customerName),
			zap.String("order_number", orderNumber),
			zap.String("tracking_url", trackingURL),
			zap.String("response_body", string(bodyBytes)),
			zap.Error(err))
		return fmt.Errorf("failed to decode WhatsApp response: %w", err)
	}

	if len(response.Messages) == 0 {
		w.logger.Error("No message ID returned from WhatsApp order management API",
			zap.String("phone_number", formattedPhone),
			zap.String("customer_name", customerName),
			zap.String("order_number", orderNumber),
			zap.String("tracking_url", trackingURL),
			zap.String("response_body", string(bodyBytes)))
		return fmt.Errorf("no message ID returned from WhatsApp API")
	}

	w.logger.Info("WhatsApp order_management_amrti template sent successfully",
		zap.String("phone_number", formattedPhone),
		zap.String("customer_name", customerName),
		zap.String("order_number", orderNumber),
		zap.String("tracking_url", trackingURL),
		zap.String("message_id", response.Messages[0].ID))

	return nil
}

// sendTemplateMessage sends a template message via WhatsApp Business API
func (w *WhatsAppService) sendTemplateMessage(ctx context.Context, phoneNumber, templateName, languageCode string, parameters []WhatsAppParameter) error {
	url := fmt.Sprintf("%s/%s/messages", w.config.APIBaseURL, w.config.PhoneNumberID)

	requestBody := WhatsAppMessageRequest{
		MessagingProduct: "whatsapp",
		To:               phoneNumber,
		Type:             "template",
		Template: &WhatsAppTemplate{
			Name:     templateName,
			Language: WhatsAppLanguage{Code: languageCode},
		},
	}

	// Add parameters if provided
	if len(parameters) > 0 {
		requestBody.Template.Components = []WhatsAppComponent{
			{
				Type:       "body",
				Parameters: parameters,
			},
		}
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return fmt.Errorf("failed to marshal WhatsApp request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create WhatsApp request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+w.config.AccessToken)

	w.logger.Info("Sending WhatsApp template message",
		zap.String("phone_number", phoneNumber),
		zap.String("template_name", templateName),
		zap.String("url", url),
		zap.String("api_base_url", w.config.APIBaseURL),
		zap.String("phone_number_id", w.config.PhoneNumberID))

	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Error("WhatsApp API request failed",
			zap.String("phone_number", phoneNumber),
			zap.String("template_name", templateName),
			zap.String("url", url),
			zap.Error(err))
		return fmt.Errorf("failed to send WhatsApp message: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	w.logger.Info("WhatsApp API response",
		zap.String("phone_number", phoneNumber),
		zap.String("template_name", templateName),
		zap.Int("status_code", resp.StatusCode),
		zap.String("response_body", string(bodyBytes)))

	if resp.StatusCode != http.StatusOK {
		var errorResp WhatsAppError
		if err := json.Unmarshal(bodyBytes, &errorResp); err != nil {
			return fmt.Errorf("WhatsApp API error (status %d): failed to decode error response", resp.StatusCode)
		}
		return fmt.Errorf("WhatsApp API error (status %d): %s", resp.StatusCode, errorResp.Error.Message)
	}

	var response WhatsAppResponse
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		return fmt.Errorf("failed to decode WhatsApp response: %w", err)
	}

	if len(response.Messages) == 0 {
		return fmt.Errorf("no message ID returned from WhatsApp API")
	}

	w.logger.Info("WhatsApp template message sent successfully",
		zap.String("phone_number", phoneNumber),
		zap.String("template_name", templateName),
		zap.String("message_id", response.Messages[0].ID))

	return nil
}

// sendTextMessage sends a text message via WhatsApp Business API (kept for future use)
func (w *WhatsAppService) sendTextMessage(ctx context.Context, phoneNumber, message string) error {
	// Text messages have restrictions on WhatsApp Business API
	// This method is kept for future use but currently not implemented
	return fmt.Errorf("text messages are not supported, use template messages instead")
}

// formatPhoneNumber formats phone number for WhatsApp (removes non-digits and ensures country code)
func (w *WhatsAppService) formatPhoneNumber(phone string) string {
	// Remove all non-digit characters
	re := regexp.MustCompile(`\D`)
	formatted := re.ReplaceAllString(phone, "")

	// If the number doesn't start with country code, assume it's Indian (+91)
	if len(formatted) == 10 {
		formatted = "91" + formatted
	}

	return formatted
}

// FormatPhoneNumber formats phone number for WhatsApp (public method for external use)
func (w *WhatsAppService) FormatPhoneNumber(phone string) string {
	return w.formatPhoneNumber(phone)
}

// IsEnabled returns whether WhatsApp service is enabled
func (w *WhatsAppService) IsEnabled() bool {
	return w.config.Enabled
}

// GetCustomerPhoneFromOrder extracts phone number from order with fallback logic
func (w *WhatsAppService) GetCustomerPhoneFromOrder(order *models.Order) string {
	// Priority 1: Customer phone from order (legacy)
	if order.CustomerPhone != "" {
		w.logger.Debug("Extracted phone from order.CustomerPhone",
			zap.String("order_id", order.ID),
			zap.String("phone", order.CustomerPhone),
			zap.String("source", "order.CustomerPhone"))
		return order.CustomerPhone
	}

	// Priority 2: Customer phone from normalized order customer
	if order.Customer != nil && order.Customer.Phone != "" {
		w.logger.Debug("Extracted phone from order.Customer.Phone",
			zap.String("order_id", order.ID),
			zap.String("phone", order.Customer.Phone),
			zap.String("source", "order.Customer.Phone"))
		return order.Customer.Phone
	}

	// Priority 3: Alternative phone from order
	if order.CustomerPhoneAlt != "" {
		w.logger.Debug("Extracted phone from order.CustomerPhoneAlt",
			zap.String("order_id", order.ID),
			zap.String("phone", order.CustomerPhoneAlt),
			zap.String("source", "order.CustomerPhoneAlt"))
		return order.CustomerPhoneAlt
	}

	// Priority 4: Alternative phone from normalized order customer
	if order.Customer != nil && order.Customer.PhoneAlt != "" {
		w.logger.Debug("Extracted phone from order.Customer.PhoneAlt",
			zap.String("order_id", order.ID),
			zap.String("phone", order.Customer.PhoneAlt),
			zap.String("source", "order.Customer.PhoneAlt"))
		return order.Customer.PhoneAlt
	}

	w.logger.Warn("No phone number found in order - all sources checked",
		zap.String("order_id", order.ID),
		zap.String("order_customer_phone", order.CustomerPhone),
		zap.String("order_customer_phone_alt", order.CustomerPhoneAlt),
		zap.Bool("has_customer_object", order.Customer != nil),
		zap.String("customer_phone", func() string {
			if order.Customer != nil {
				return order.Customer.Phone
			}
			return ""
		}()),
		zap.String("customer_phone_alt", func() string {
			if order.Customer != nil {
				return order.Customer.PhoneAlt
			}
			return ""
		}()))

	return ""
}

// extractCustomerName extracts customer name from order with fallback logic
func (w *WhatsAppService) extractCustomerName(order *models.Order) string {
	// Priority 1: First name from order (legacy)
	if order.CustomerFirstName != "" {
		return order.CustomerFirstName
	}

	// Priority 2: First name from normalized order customer
	if order.Customer != nil && order.Customer.FirstName != "" {
		return order.Customer.FirstName
	}

	// Priority 3: Fallback to "Customer" if no name found
	w.logger.Warn("No customer name found in order, using fallback",
		zap.String("order_id", order.ID))
	return "Customer"
}


// extractOrderNumber extracts order number from order with fallback logic
func (w *WhatsAppService) extractOrderNumber(order *models.Order) string {
	// Priority 1: Use the order ID directly
	if order.ID != "" {
		return order.ID
	}

	// Priority 2: Use Razorpay order ID if available
	if order.RazorpayOrderID != "" {
		return order.RazorpayOrderID
	}

	// Priority 3: Fallback to generic order number
	w.logger.Warn("No order number found, using fallback",
		zap.String("order_id", order.ID))
	return "ORDER-" + order.ID
}

// extractTrackingURL extracts tracking URL from order with fallback logic
func (w *WhatsAppService) extractTrackingURL(order *models.Order) string {
	// Priority 1: Use the tracking URL from order
	if order.TrackingURL != "" {
		return order.TrackingURL
	}

	// Priority 2: Generate tracking URL from waybill number
	if order.DelhiveryWaybill != "" {
		return fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", order.DelhiveryWaybill)
	}

	// Priority 3: Fallback to generic tracking message
	w.logger.Warn("No tracking URL found, using fallback",
		zap.String("order_id", order.ID))
	return "Tracking information will be available soon"
}

