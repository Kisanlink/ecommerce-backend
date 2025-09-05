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
	GetLastEstimate(ctx context.Context, userID string) (*LastEstimate, error)
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

// LastEstimate represents a cached delivery estimate summary for reuse in payment
type LastEstimate struct {
	ShippingAddress string  `json:"shipping_address"`
	ShippingCharges float64 `json:"shipping_charges"`
	Subtotal        float64 `json:"subtotal"`
	Total           float64 `json:"total"`
	Currency        string  `json:"currency"`
	EtaMinDays      int     `json:"eta_min_days"`
	EtaMaxDays      int     `json:"eta_max_days"`
	Pincode         string  `json:"pincode"`
	Timestamp       int64   `json:"ts"`
}

func (s *DeliveryService) estimateCacheKey(userID string) string {
	return fmt.Sprintf("user:%s:last_estimate", userID)
}

func (s *DeliveryService) saveLastEstimate(ctx context.Context, userID string, est *LastEstimate) {
	if s.repoManager == nil || s.repoManager.RedisManager == nil || est == nil {
		return
	}
	b, err := json.Marshal(est)
	if err != nil {
		s.logger.Warn("Failed to marshal last estimate for cache", zap.Error(err))
		return
	}
	_ = s.repoManager.RedisManager.Set(s.estimateCacheKey(userID), string(b), 30*time.Minute)
}

// GetLastEstimate retrieves the cached estimate for a user if available
func (s *DeliveryService) GetLastEstimate(ctx context.Context, userID string) (*LastEstimate, error) {
	if s.repoManager == nil || s.repoManager.RedisManager == nil {
		return nil, fmt.Errorf("estimate cache unavailable")
	}
	val, err := s.repoManager.RedisManager.Get(s.estimateCacheKey(userID))
	if err != nil || val == "" {
		return nil, fmt.Errorf("no cached estimate")
	}
	var est LastEstimate
	if e := json.Unmarshal([]byte(val), &est); e != nil {
		return nil, e
	}
	return &est, nil
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
		UserID:   userID,
		Subtotal: cart.TotalPrice,
		Options:  options,
	}

	// Build address string for caching
	parts := []string{}
	if s := strings.TrimSpace(req.Address.Street); s != "" {
		parts = append(parts, s)
	}
	cityState := strings.TrimSpace(strings.TrimSpace(req.Address.City) + ", " + strings.TrimSpace(req.Address.State))
	cityState = strings.Trim(cityState, ", ")
	if cityState != "" {
		parts = append(parts, cityState)
	}
	if p := strings.TrimSpace(req.Address.PostalCode); p != "" {
		parts = append(parts, p)
	}
	if c := strings.TrimSpace(req.Address.Country); c != "" {
		parts = append(parts, c)
	}
	addressStr := strings.Join(parts, ", ")

	cached := &LastEstimate{
		ShippingAddress: addressStr,
		ShippingCharges: surfaceCharges,
		Subtotal:        cart.TotalPrice,
		Total:           cart.TotalPrice + surfaceCharges,
		Currency:        "INR",
		EtaMinDays:      deliveryDays,
		EtaMaxDays:      deliveryDays + 2,
		Pincode:         pincode,
		Timestamp:       time.Now().Unix(),
	}
	s.saveLastEstimate(ctx, userID, cached)

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

	pincode := s.extractPincode(order.ShippingAddr)
	if pincode == "" {
		return &ShipmentResult{
			Success: false,
			Message: "Could not extract valid pincode from shipping address",
		}, fmt.Errorf("could not extract pincode from shipping address")
	}

	// Get customer details (best-effort). Fall back to address or safe defaults.
	customerName := "Customer"
	customerPhone := ""
	// Prefer order-level fields captured at checkout
	if strings.TrimSpace(order.CustomerFirstName) != "" || strings.TrimSpace(order.CustomerLastName) != "" {
		fn := strings.TrimSpace(order.CustomerFirstName)
		ln := strings.TrimSpace(order.CustomerLastName)
		name := strings.TrimSpace(strings.TrimSpace(fn+" "+ln))
		if name != "" { customerName = name }
	}
	if strings.TrimSpace(order.CustomerPhone) != "" {
		ph := strings.TrimSpace(order.CustomerPhone)
		if len(ph) > 10 { ph = ph[len(ph)-10:] }
		customerPhone = ph
	}
	if s.repoManager != nil && s.repoManager.UserRepository != nil {
		user := &models.User{}
		if res, err := s.repoManager.UserRepository.GetByID(ctx, order.UserID, user); err == nil && res != nil {
			if strings.TrimSpace(res.Name) != "" {
				customerName = strings.TrimSpace(res.Name)
			}
			if strings.TrimSpace(res.PhoneNumber) != "" {
				customerPhone = strings.TrimSpace(res.PhoneNumber)
			}
		} else {
			s.logger.Warn("Proceeding without user record for shipment", zap.String("order_id", order.ID), zap.Error(err))
		}
	} else {
		s.logger.Warn("User repository unavailable; falling back to address for contact details", zap.String("order_id", order.ID))
	}
	// Try name/phone from the formatted shipping address
	if customerName == "Customer" {
		if n := s.extractNameFromAddress(order.ShippingAddr); n != "" {
			customerName = n
		}
	}
	if customerPhone == "" {
		if p := s.extractPhoneFromText(order.ShippingAddr); p != "" {
			customerPhone = p
		}
	}
	// Final fallback for phone: use a generic 10-digit placeholder acceptable by Delhivery
	if customerPhone == "" {
		customerPhone = "9999999999"
	}

	// Get city and state from API
	destCity, destState, err := s.getPincodeInfo(ctx, pincode)
	if err != nil {
		return &ShipmentResult{
			Success: false,
			Message: "Could not get city/state information for pincode",
		}, fmt.Errorf("could not get city/state for pincode: %w", err)
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
		Name:          customerName,
		Add:           order.ShippingAddr,
		Pin:           pincode,
		City:          destCity,
		State:         destState,
		Country:       "India",
		Phone:         customerPhone,
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
		return &ShipmentResult{
			Success: false,
			Message: fmt.Sprintf("Invalid shipment request: %v", err),
		}, fmt.Errorf("invalid shipment request: %w", err)
	}

	endpoint := shipmentReq.BuildShipmentURL(s.config.BaseURL)

	dataJSON, err := json.Marshal(shipmentReq)
	if err != nil {
		return &ShipmentResult{
			Success: false,
			Message: "Failed to prepare shipment request data",
		}, fmt.Errorf("failed to marshal shipment request: %w", err)
	}

	formData := url.Values{}
	formData.Set("format", "json")
	formData.Set("data", string(dataJSON))

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(formData.Encode()))
	if err != nil {
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
		return &ShipmentResult{
			Success: false,
			Message: "Failed to connect to Delhivery API",
		}, fmt.Errorf("failed to create shipment: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return &ShipmentResult{
			Success: false,
			Message: "Failed to read response from Delhivery API",
		}, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return &ShipmentResult{
			Success: false,
			Message: fmt.Sprintf("Delhivery API returned error status %d", resp.StatusCode),
		}, fmt.Errorf("shipment API failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	shipmentResp, err := responses.ParseShipmentResponse(body)
	if err != nil {
		return &ShipmentResult{
			Success: false,
			Message: "Failed to parse Delhivery response",
		}, fmt.Errorf("failed to parse shipment response: %w", err)
	}

	if !shipmentResp.IsSuccessful() {
		errorMsg := shipmentResp.GetErrorMessage()
		return &ShipmentResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create shipment: %s", errorMsg),
		}, fmt.Errorf("shipment creation failed: %s", errorMsg)
	}

	waybill := shipmentResp.GetWaybill()
	if waybill == "" {
		return &ShipmentResult{
			Success: false,
			Message: "No AWB/Waybill number received from Delhivery",
		}, fmt.Errorf("no waybill received from Delhivery")
	}

	trackingURL := shipmentResp.GetTrackingURL()
	status := shipmentResp.GetStatus()

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

	trackingURL := fmt.Sprintf("https://track.delhivery.com/p/%s", waybill)

	return &TrackingInfo{
		Waybill:     waybill,
		Status:      "Shipped",
		TrackingURL: trackingURL,
		LastUpdate:  time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}

// extractPhoneFromText scans a string and returns the last 10-digit sequence it finds
func (s *DeliveryService) extractPhoneFromText(text string) string {
	digits := make([]rune, 0, 16)
	var last10 string
	for _, ch := range text {
		if ch >= '0' && ch <= '9' {
			digits = append(digits, ch)
			if len(digits) >= 10 {
				// Keep only the last 10 digits window as candidate
				last10 = string(digits[len(digits)-10:])
			}
		}
	}
	if len(last10) == 10 {
		return last10
	}
	return ""
}

// extractNameFromAddress tries to extract the name as the token before the first comma
func (s *DeliveryService) extractNameFromAddress(address string) string {
	parts := strings.Split(address, ",")
	if len(parts) == 0 {
		return ""
	}
	name := strings.TrimSpace(parts[0])
	// Avoid taking generic labels
	if strings.EqualFold(name, "address") || strings.EqualFold(name, "addr") || strings.EqualFold(name, "home") {
		return ""
	}
	return name
}