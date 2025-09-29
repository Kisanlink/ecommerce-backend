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

	// Send hello_world template (same as working curl command)
	return w.sendHelloWorldTemplate(ctx, formattedPhone)
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

	// Send hello_world template (same as working curl command)
	return w.sendHelloWorldTemplate(ctx, formattedPhone)
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

	// Send hello_world template (same as working curl command)
	return w.sendHelloWorldTemplate(ctx, formattedPhone)
}

// sendHelloWorldTemplate sends the hello_world template message (exactly like the working curl command)
func (w *WhatsAppService) sendHelloWorldTemplate(ctx context.Context, phoneNumber string) error {
	url := fmt.Sprintf("%s/%s/messages", w.config.APIBaseURL, w.config.PhoneNumberID)

	// Exact same structure as your working curl command
	requestBody := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                phoneNumber,
		"type":              "template",
		"template": map[string]interface{}{
			"name": "hello_world",
			"language": map[string]string{
				"code": "en_US",
			},
		},
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

	w.logger.Info("Sending WhatsApp hello_world template (exact curl match)",
		zap.String("phone_number", phoneNumber),
		zap.String("url", url),
		zap.String("api_base_url", w.config.APIBaseURL),
		zap.String("phone_number_id", w.config.PhoneNumberID))

	resp, err := w.client.Do(req)
	if err != nil {
		w.logger.Error("WhatsApp API request failed",
			zap.String("phone_number", phoneNumber),
			zap.String("url", url),
			zap.Error(err))
		return fmt.Errorf("failed to send WhatsApp message: %w", err)
	}
	defer resp.Body.Close()

	// Read response body for debugging
	bodyBytes, _ := io.ReadAll(resp.Body)
	w.logger.Info("WhatsApp API response",
		zap.String("phone_number", phoneNumber),
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

	w.logger.Info("WhatsApp hello_world template sent successfully",
		zap.String("phone_number", phoneNumber),
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
	// For now, redirect to template message using hello_world template
	// This is because text messages have restrictions on WhatsApp Business API
	return w.sendTemplateMessage(ctx, phoneNumber, "hello_world", "en_US", nil)
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

// IsEnabled returns whether WhatsApp service is enabled
func (w *WhatsAppService) IsEnabled() bool {
	return w.config.Enabled
}

// GetCustomerPhoneFromOrder extracts phone number from order with fallback logic
func (w *WhatsAppService) GetCustomerPhoneFromOrder(order *models.Order) string {
	// Priority 1: Customer phone from order (legacy)
	if order.CustomerPhone != "" {
		return order.CustomerPhone
	}

	// Priority 2: Customer phone from normalized order customer
	if order.Customer != nil && order.Customer.Phone != "" {
		return order.Customer.Phone
	}

	// Priority 3: Alternative phone from order
	if order.CustomerPhoneAlt != "" {
		return order.CustomerPhoneAlt
	}

	// Priority 4: Alternative phone from normalized order customer
	if order.Customer != nil && order.Customer.PhoneAlt != "" {
		return order.Customer.PhoneAlt
	}

	return ""
}
