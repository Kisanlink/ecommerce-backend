package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// DeliveryServiceInterface defines delivery-related business behavior
type DeliveryServiceInterface interface {
	Estimate(ctx context.Context, userID string, req requests.EstimateShippingRequest) (*models.DeliveryEstimate, error)
	CheckServiceability(ctx context.Context, pincode string) (bool, error)
	CreateShipment(ctx context.Context, order *models.Order, orderItems []models.OrderItem) (*ShipmentResult, error)
	GetTrackingInfo(ctx context.Context, waybill string) (*TrackingInfo, error)
}

// ShipmentResult contains detailed shipment creation results
type ShipmentResult struct {
	Success     bool   `json:"success"`
	Message     string `json:"message"`
	Waybill     string `json:"waybill"`
	TrackingURL string `json:"tracking_url"`
	Status      string `json:"status"`
}

// TrackingInfo contains tracking information for a shipment
type TrackingInfo struct {
	Waybill     string `json:"waybill"`
	Status      string `json:"status"`
	TrackingURL string `json:"tracking_url"`
	LastUpdate  string `json:"last_update"`
}

type DeliveryService struct {
	repoManager *repositories.RepositoryManager
	dbManager   db.DBManager
	httpClient  *http.Client
	config      config.DelhiveryConfig
	logger      *zap.Logger
}

func NewDeliveryService(repoManager *repositories.RepositoryManager, logger *zap.Logger) *DeliveryService {
	return &DeliveryService{
		repoManager: repoManager,
		dbManager:   repoManager.GetDBManager(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: config.GetDelhiveryConfig(),
		logger: logger,
	}
}

// CheckServiceability checks if delivery is available to the given pincode
func (s *DeliveryService) CheckServiceability(ctx context.Context, pincode string) (bool, error) {
	if pincode == "" {
		return false, fmt.Errorf("pincode cannot be empty")
	}

	if len(pincode) != 6 {
		return false, fmt.Errorf("pincode must be exactly 6 digits")
	}

	serviceabilityReq := requests.NewServiceabilityRequest(pincode)
	if err := serviceabilityReq.IsValid(); err != nil {
		return false, fmt.Errorf("invalid serviceability request: %w", err)
	}

	endpoint := serviceabilityReq.BuildServiceabilityURL(s.config.BaseURL)
	s.logger.Info("Delhivery serviceability check", zap.String("endpoint", endpoint))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to check serviceability: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("Delhivery serviceability API failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	serviceabilityResp, err := responses.ParseServiceabilityResponse(body)
	if err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return serviceabilityResp.IsServiceable(pincode), nil
}

// getPincodeInfo fetches city/state info for a pincode from Delhivery serviceability API
func (s *DeliveryService) getPincodeInfo(ctx context.Context, pincode string) (city string, state string, err error) {
	if pincode == "" || len(pincode) != 6 {
		return "", "", fmt.Errorf("invalid pincode")
	}

	serviceabilityReq := requests.NewServiceabilityRequest(pincode)
	endpoint := serviceabilityReq.BuildServiceabilityURL(s.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("serviceability HTTP %d", resp.StatusCode)
	}

	serviceabilityResp, err := responses.ParseServiceabilityResponse(body)
	if err != nil {
		return "", "", err
	}

	info := serviceabilityResp.GetServiceabilityInfo(pincode)
	if info == nil {
		return "", "", fmt.Errorf("pincode not found")
	}

	return strings.TrimSpace(info.City), strings.TrimSpace(info.State), nil
}

// getShippingCharges gets shipping charges from Delhivery
func (s *DeliveryService) getShippingCharges(ctx context.Context, destinationPin string, weightGrams int, codAmount float64) (float64, error) {
	if len(destinationPin) != 6 {
		return 0, fmt.Errorf("destination pincode must be 6 digits, got: %s", destinationPin)
	}

	if weightGrams <= 0 {
		return 0, fmt.Errorf("weight must be positive")
	}

	var chargesReq *requests.DelhiveryChargesRequest

	if codAmount > 0 {
		chargesReq = requests.NewCODChargesRequest(s.config.OriginPincode, destinationPin, weightGrams, codAmount)
	} else {
		chargesReq = requests.NewChargesRequest(s.config.OriginPincode, destinationPin, weightGrams)
		chargesReq.Mode = "S"
		if s.config.ServiceStatus != "" {
			chargesReq.ServiceStatus = s.config.ServiceStatus
		}
	}

	if err := chargesReq.IsValid(); err != nil {
		return 0, fmt.Errorf("invalid charges request: %w", err)
	}

	endpoint := chargesReq.BuildChargesURL(s.config.BaseURL)
	s.logger.Info("Delhivery charges API request", zap.String("endpoint", endpoint))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get charges: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("charges API returned status %d: %s", resp.StatusCode, string(body))
	}

	chargesResp, err := responses.ParseChargesResponse(body)
	if err != nil {
		return 0, fmt.Errorf("failed to decode charges response: %w", err)
	}

	totalAmount, err := chargesResp.GetTotalShippingCost()
	if err != nil {
		return 0, fmt.Errorf("failed to extract total amount: %w", err)
	}

	return totalAmount, nil
}

// getEstimatedTAT gets estimated delivery time from Delhivery
func (s *DeliveryService) getEstimatedTAT(ctx context.Context, destinationPin string) (int, error) {
	tatReq := requests.NewTATRequest(s.config.OriginPincode, destinationPin)
	if err := tatReq.IsValid(); err != nil {
		return 0, fmt.Errorf("invalid TAT request: %w", err)
	}

	endpoint := tatReq.BuildTATURL(s.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to get TAT from Delhivery API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read TAT response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("Delhivery TAT API failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	tatResp, err := responses.ParseTATResponse(body)
	if err != nil {
		return 0, fmt.Errorf("failed to decode TAT response: %w", err)
	}

	if !tatResp.IsValidTAT() {
		return 0, fmt.Errorf("invalid TAT received from API")
	}

	return tatResp.Data.TAT, nil
}

// extractPincode extracts and validates pincode from postal_code field
func (s *DeliveryService) extractPincode(postalCode string) string {
	cleaned := strings.ReplaceAll(postalCode, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	var pincode strings.Builder
	for _, char := range cleaned {
		if char >= '0' && char <= '9' {
			pincode.WriteRune(char)
			if pincode.Len() >= 6 {
				break
			}
		}
	}

	result := pincode.String()
	if len(result) == 6 {
		return result
	}

	return ""
}

// Estimate computes delivery options using Delhivery APIs
func (s *DeliveryService) Estimate(ctx context.Context, userID string, req requests.EstimateShippingRequest) (*models.DeliveryEstimate, error) {
	pincode := s.extractPincode(req.Address.PostalCode)
	if pincode == "" {
		return nil, fmt.Errorf("invalid postal code format: %s", req.Address.PostalCode)
	}

	if s.repoManager == nil || s.repoManager.CartRepository == nil {
		return nil, fmt.Errorf("cart repository unavailable")
	}

	cart, err := s.repoManager.CartRepository.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart: %w", err)
	}

	if cart.TotalItems <= 0 {
		return nil, fmt.Errorf("cart is empty; add items before estimating shipping")
	}

	serviceable, err := s.CheckServiceability(ctx, pincode)
	if err != nil {
		return nil, fmt.Errorf("failed to check serviceability: %w", err)
	}

	if !serviceable {
		return nil, fmt.Errorf("delivery service is not available to pincode %s", pincode)
	}

	const PRODUCT_WEIGHT_GRAMS = 100
	weight := cart.TotalItems * PRODUCT_WEIGHT_GRAMS

	charges, err := s.getShippingCharges(ctx, pincode, weight, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get shipping charges from Delhivery API: %w", err)
	}

	deliveryDays, err := s.getEstimatedTAT(ctx, pincode)
	if err != nil {
		return nil, fmt.Errorf("failed to get estimated delivery time: %w", err)
	}

	surfaceCharges := math.Round(charges*100) / 100
	options := []models.DeliveryOption{
		{
			Method:       models.DeliveryMethodStandard,
			Carrier:      "Delhivery",
			ServiceLevel: "Surface",
			Price:        surfaceCharges,
			Currency:     "INR",
			EtaDaysMin:   deliveryDays,
			EtaDaysMax:   deliveryDays + 2,
		},
	}

	result := &models.DeliveryEstimate{
		UserID:    userID,
		AddressID: "", // Will be set below if address is saved successfully
		Subtotal:  cart.TotalPrice,
		Options:   options,
	}

	// Store or find the address in the database and return address_id
	var addressID string

	if s.repoManager != nil && s.repoManager.AddressRepository != nil {
		s.logger.Info("Attempting to save address during estimation",
			zap.String("user_id", userID),
			zap.String("street", strings.TrimSpace(req.Address.Street)),
			zap.String("city", strings.TrimSpace(req.Address.City)),
			zap.String("state", strings.TrimSpace(req.Address.State)),
			zap.String("postal_code", strings.TrimSpace(req.Address.PostalCode)),
			zap.String("country", strings.TrimSpace(req.Address.Country)))

		// Save the address to database (or find existing one)
		savedAddress, addrErr := s.repoManager.AddressRepository.FindOrCreateAddress(
			ctx, userID,
			strings.TrimSpace(req.Address.Street),
			strings.TrimSpace(req.Address.City),
			strings.TrimSpace(req.Address.State),
			strings.TrimSpace(req.Address.PostalCode),
			strings.TrimSpace(req.Address.Country),
		)

		if addrErr != nil {
			s.logger.Error("Failed to save address during estimation", zap.Error(addrErr), zap.String("user_id", userID))
			// Don't fail the estimation, but log the error
		} else {
			addressID = savedAddress.ID
			result.AddressID = addressID // Include in response for frontend
			s.logger.Info("Address saved during shipping estimation",
				zap.String("user_id", userID),
				zap.String("address_id", addressID),
				zap.String("city", savedAddress.City),
				zap.String("pincode", savedAddress.PostalCode))
		}
	} else {
		s.logger.Warn("Address repository not available, cannot save address during estimation",
			zap.String("user_id", userID))
	}

	return result, nil
}

// CreateShipment creates a Delhivery shipment after successful payment
func (s *DeliveryService) CreateShipment(ctx context.Context, order *models.Order, orderItems []models.OrderItem) (*ShipmentResult, error) {
	if order == nil {
		return &ShipmentResult{
			Success: false,
			Message: "Order cannot be nil",
		}, fmt.Errorf("order cannot be nil")
	}

	var pincode, destCity, destState string

	// Try to use structured address data first (preferred method)
	if order.ShippingAddrID != "" && s.repoManager != nil && s.repoManager.AddressRepository != nil {
		addr, addrErr := s.repoManager.AddressRepository.GetAddressByID(ctx, order.UserID, order.ShippingAddrID)
		if addrErr == nil && addr != nil {
			pincode = addr.PostalCode
			destCity = addr.City
			destState = addr.State
			s.logger.Info("Using structured address data for shipment",
				zap.String("order_id", order.ID),
				zap.String("address_id", order.ShippingAddrID),
				zap.String("city", destCity),
				zap.String("state", destState),
				zap.String("pincode", pincode))
		} else {
			s.logger.Warn("Failed to fetch structured address, falling back to string parsing",
				zap.String("order_id", order.ID),
				zap.String("address_id", order.ShippingAddrID),
				zap.Error(addrErr))
		}
	}

	// Fallback to parsing address string if structured data not available
	if pincode == "" {
		pincode = s.extractPincode(order.ShippingAddr)
		if pincode == "" {
			return &ShipmentResult{
				Success: false,
				Message: "Could not extract valid pincode from shipping address",
			}, fmt.Errorf("could not extract pincode from shipping address")
		}

		// Get city and state from API (only needed for fallback)
		var err error
		destCity, destState, err = s.getPincodeInfo(ctx, pincode)
		if err != nil {
			return &ShipmentResult{
				Success: false,
				Message: "Could not get city/state information for pincode",
			}, fmt.Errorf("could not get city/state for pincode: %w", err)
		}
	}

	// Use only order-level customer fields (no fallbacks)
	custName := strings.TrimSpace(strings.TrimSpace(order.CustomerFirstName + " " + order.CustomerLastName))
	custPhone := strings.TrimSpace(order.CustomerPhone)
	if custName == "" || custPhone == "" {
		return &ShipmentResult{Success: false, Message: "Customer name and phone are required on the order"}, fmt.Errorf("missing customer name/phone on order")
	}
	// Normalize phone to last 10 digits
	{
		var digits strings.Builder
		digits.Grow(len(custPhone))
		for _, ch := range custPhone {
			if ch >= '0' && ch <= '9' {
				digits.WriteRune(ch)
			}
		}
		d := digits.String()
		if len(d) >= 10 {
			custPhone = d[len(d)-10:]
		} else {
			return &ShipmentResult{Success: false, Message: "Customer phone must contain at least 10 digits"}, fmt.Errorf("invalid customer phone")
		}
	}

	// Build products description
	productDescs := []string{}
	totalWeight := 0
	for _, item := range orderItems {
		productDescs = append(productDescs, fmt.Sprintf("Product %s (Qty: %d)", item.ProductID, item.Quantity))
		totalWeight += item.Quantity * 100 // 100g per item
	}

	if totalWeight < 100 {
		totalWeight = 100
	}

	// Create shipment request
	shipmentReq := requests.NewShipmentRequest(s.config.WarehouseName)

	uniqueOrderID := fmt.Sprintf("%s_%d", order.ID, time.Now().Unix())

	shipmentDetail := requests.DelhiveryShipmentDetail{
		Name:          custName,
		Add:           order.ShippingAddr,
		Pin:           pincode,
		City:          destCity,
		State:         destState,
		Country:       "India",
		Phone:         custPhone,
		Order:         uniqueOrderID,
		PaymentMode:   "Prepaid",
		ReturnPin:     s.config.OriginPincode,
		ReturnCity:    s.config.OriginCity,
		ReturnName:    s.config.WarehouseName,
		ReturnAdd:     s.config.OriginAddress,
		ReturnState:   s.config.OriginState,
		ReturnCountry: "India",
		ReturnPhone:   s.config.OriginPhone,
		ProductsDesc:  strings.Join(productDescs, ", "),
		HsnCode:       "1234",
		TotalAmount:   fmt.Sprintf("%.0f", order.TotalAmount),
		SellerName:    "Kisanlink",
		SellerInv:     order.ID,
		Quantity:      fmt.Sprintf("%d", len(orderItems)),
		Weight:        fmt.Sprintf("%d", totalWeight),
		ShippingMode:  "Surface",
		AddressType:   "home",
		ShipWidth:     "15",
		ShipHeight:    "15",
	}

	shipmentReq.AddShipmentDetail(shipmentDetail)

	if err := shipmentReq.IsValid(); err != nil {
		s.logger.Error("Invalid shipment request validation failed",
			zap.Error(err),
			zap.String("order_id", order.ID))
		return &ShipmentResult{
			Success: false,
			Message: fmt.Sprintf("Invalid shipment request: %v", err),
		}, fmt.Errorf("invalid shipment request: %w", err)
	}

	endpoint := shipmentReq.BuildShipmentURL(s.config.BaseURL)

	dataJSON, err := json.Marshal(shipmentReq)
	if err != nil {
		s.logger.Error("Failed to marshal shipment request",
			zap.Error(err),
			zap.String("order_id", order.ID))
		return &ShipmentResult{
			Success: false,
			Message: "Failed to prepare shipment request data",
		}, fmt.Errorf("failed to marshal shipment request: %w", err)
	}

	s.logger.Info("Sending shipment creation request to Delhivery",
		zap.String("order_id", order.ID),
		zap.String("endpoint", endpoint),
		zap.String("unique_order_id", uniqueOrderID))

	formData := url.Values{}
	formData.Set("format", "json")
	formData.Set("data", string(dataJSON))

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
		s.logger.Error("Failed to create HTTP request",
			zap.Error(err),
			zap.String("order_id", order.ID))
		return &ShipmentResult{
			Success: false,
			Message: "Failed to create HTTP request to Delhivery API",
		}, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("HTTP request to Delhivery API failed",
			zap.Error(err),
			zap.String("order_id", order.ID),
			zap.String("endpoint", endpoint))
		return &ShipmentResult{
			Success: false,
			Message: "Failed to connect to Delhivery API",
		}, fmt.Errorf("failed to create shipment: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read Delhivery API response",
			zap.Error(err),
			zap.String("order_id", order.ID))
		return &ShipmentResult{
			Success: false,
			Message: "Failed to read response from Delhivery API",
		}, fmt.Errorf("failed to read response body: %w", err)
	}

	s.logger.Info("Received response from Delhivery API",
		zap.String("order_id", order.ID),
		zap.Int("status_code", resp.StatusCode),
		zap.String("response_length", fmt.Sprintf("%d", len(body))))

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("Delhivery API returned error status",
			zap.String("order_id", order.ID),
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)))
		return &ShipmentResult{
			Success: false,
			Message: fmt.Sprintf("Delhivery API returned error status %d", resp.StatusCode),
		}, fmt.Errorf("shipment API failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// Log response for debugging
	s.logger.Debug("Raw Delhivery API response",
		zap.String("order_id", order.ID),
		zap.String("response_body", string(body)))

	shipmentResp, err := responses.ParseShipmentResponse(body)
	if err != nil {
		s.logger.Error("Failed to parse Delhivery shipment response",
			zap.Error(err),
			zap.String("order_id", order.ID),
			zap.String("response_body", string(body)))
		return &ShipmentResult{
			Success: false,
			Message: "Failed to parse Delhivery response",
		}, fmt.Errorf("failed to parse shipment response: %w", err)
	}

	if !shipmentResp.IsSuccessful() {
		errorMsg := shipmentResp.GetErrorMessage()
		s.logger.Error("Delhivery shipment creation failed",
			zap.String("order_id", order.ID),
			zap.String("error_message", errorMsg))
		return &ShipmentResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create shipment: %s", errorMsg),
		}, fmt.Errorf("shipment creation failed: %s", errorMsg)
	}

	waybill := shipmentResp.GetWaybill()
	if waybill == "" {
		s.logger.Error("No waybill received from Delhivery despite successful response",
			zap.String("order_id", order.ID),
			zap.String("response_body", string(body)))
		return &ShipmentResult{
			Success: false,
			Message: "No AWB/Waybill number received from Delhivery",
		}, fmt.Errorf("no waybill received from Delhivery")
	}

	trackingURL := shipmentResp.GetTrackingURL()
	status := shipmentResp.GetStatus()

	// Ensure tracking URL is properly formatted
	if trackingURL == "" {
		trackingURL = fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", waybill)
	}

	s.logger.Info("Delhivery shipment created successfully",
		zap.String("order_id", order.ID),
		zap.String("waybill", waybill),
		zap.String("tracking_url", trackingURL),
		zap.String("status", status))

	return &ShipmentResult{
		Success:     true,
		Message:     fmt.Sprintf("Shipment created successfully! Your AWB number is %s", waybill),
		Waybill:     waybill,
		TrackingURL: trackingURL,
		Status:      status,
	}, nil
}

// GetTrackingInfo retrieves tracking information for a waybill
func (s *DeliveryService) GetTrackingInfo(ctx context.Context, waybill string) (*TrackingInfo, error) {
	if waybill == "" {
		return nil, fmt.Errorf("waybill cannot be empty")
	}

	trackingURL := fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", waybill)

	return &TrackingInfo{
		Waybill:     waybill,
		Status:      "Shipped",
		TrackingURL: trackingURL,
		LastUpdate:  time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}
