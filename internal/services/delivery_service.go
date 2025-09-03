package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
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
	CreateShipment(ctx context.Context, order *models.Order, orderItems []models.OrderItem) (string, error)
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
	// Cache for 30 minutes
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

// CheckServiceability checks if delivery is available to the given pincode using structured entities
func (s *DeliveryService) CheckServiceability(ctx context.Context, pincode string) (bool, error) {
	if pincode == "" {
		return false, fmt.Errorf("pincode cannot be empty")
	}

	// Ensure pincode is exactly 6 digits
	if len(pincode) != 6 {
		return false, fmt.Errorf("pincode must be exactly 6 digits")
	}

	// Create structured request
	serviceabilityReq := requests.NewServiceabilityRequest(pincode)
	if err := serviceabilityReq.IsValid(); err != nil {
		return false, fmt.Errorf("invalid serviceability request: %w", err)
	}

	endpoint := serviceabilityReq.BuildServiceabilityURL(s.config.BaseURL)
	s.logger.Info("Delhivery serviceability check", zap.String("endpoint", endpoint),
		zap.String("pincode", pincode))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Delhivery serviceability check failed", zap.Error(err), zap.String("pincode", pincode))
		return false, fmt.Errorf("failed to check serviceability: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("Delhivery serviceability API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
			zap.String("pincode", pincode))

		return false, fmt.Errorf("Delhivery serviceability API failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	s.logger.Info("Delhivery serviceability API response", zap.String("response", string(body)))

	// Parse response using structured entity
	serviceabilityResp, err := responses.ParseServiceabilityResponse(body)
	if err != nil {
		s.logger.Error("Failed to decode serviceability response", zap.Error(err), zap.String("body", string(body)))
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check serviceability using helper method
	serviceable := serviceabilityResp.IsServiceable(pincode)

	if serviceable {
		info := serviceabilityResp.GetServiceabilityInfo(pincode)
		s.logger.Info("Pincode is serviceable", zap.String("pincode", pincode),
			zap.String("prepaid", info.PrePaid),
			zap.String("cash", info.Cash),
			zap.String("cod", info.Cod),
			zap.String("district", info.District),
			zap.String("state", info.State),
			zap.String("city", info.City))
	} else {
		s.logger.Info("Pincode is not serviceable", zap.String("pincode", pincode),
			zap.Int("delivery_codes_count", len(serviceabilityResp.DeliveryCode)))
	}

	return serviceable, nil
}

// getShippingCharges gets shipping charges from Delhivery using structured entities - CORRECTED VERSION
func (s *DeliveryService) getShippingCharges(ctx context.Context, destinationPin string, weightGrams int, codAmount float64) (float64, error) {
	// Validate inputs
	if len(destinationPin) != 6 {
		return 0, fmt.Errorf("destination pincode must be 6 digits, got: %s", destinationPin)
	}

	if weightGrams <= 0 {
		weightGrams = 100 // Default to 100g minimum
	}

	// Create structured charges request with corrected parameters
	var chargesReq *requests.DelhiveryChargesRequest

	if codAmount > 0 {
		// COD request
		chargesReq = requests.NewCODChargesRequest(s.config.OriginPincode, destinationPin, weightGrams, codAmount)
	} else {
		// Prepaid request
		chargesReq = requests.NewChargesRequest(s.config.OriginPincode, destinationPin, weightGrams)
		// Ensure Surface mode and service status from config
		chargesReq.Mode = "S"
		if s.config.ServiceStatus != "" {
			chargesReq.ServiceStatus = s.config.ServiceStatus
		}
	}

	if err := chargesReq.IsValid(); err != nil {
		return 0, fmt.Errorf("invalid charges request: %w", err)
	}

	endpoint := chargesReq.BuildChargesURL(s.config.BaseURL)
	s.logger.Info("Delhivery charges API request - CORRECTED",
		zap.String("endpoint", endpoint),
		zap.String("destination_pin", destinationPin),
		zap.String("origin_pin", s.config.OriginPincode),
		zap.Int("weight_grams", weightGrams),
		zap.Float64("weight_kg", float64(chargesReq.ChargedWeight)/1000.0),
		zap.String("mode", chargesReq.Mode),
		zap.String("service_status", chargesReq.ServiceStatus),
		zap.String("payment_type", chargesReq.PaymentType),
		zap.Float64("cod_amount", chargesReq.CODAmount))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Delhivery charges API failed", zap.Error(err))
		return 0, fmt.Errorf("failed to get charges: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	s.logger.Info("Delhivery charges API raw response",
		zap.String("response", string(body)),
		zap.Int("status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("Delhivery charges API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
			zap.String("endpoint", endpoint))
		return 0, fmt.Errorf("charges API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response using structured entity
	chargesResp, err := responses.ParseChargesResponse(body)
	if err != nil {
		s.logger.Error("Failed to decode charges response",
			zap.Error(err),
			zap.String("body", string(body)))
		return 0, fmt.Errorf("failed to decode charges response: %w", err)
	}

	// Get total amount using helper method
	totalAmount, err := chargesResp.GetTotalShippingCost()
	if err != nil {
		s.logger.Error("Failed to extract total amount", zap.Error(err))
		return 0, fmt.Errorf("failed to extract total amount: %w", err)
	}

	// Log additional details for debugging
	if detail := chargesResp.GetFirstChargeDetail(); detail != nil {
		s.logger.Info("Delhivery charges parsed successfully",
			zap.Float64("total_amount", totalAmount),
			zap.Float64("base_cost", detail.BaseCost),
			zap.Float64("fuel_charges", detail.FuelCharges),
			zap.Float64("cod_charges", detail.CodCharges),
			zap.Float64("da_charges", detail.DaCharges),
			zap.Float64("handling_charges", detail.HandlingCharges),
			zap.Float64("other_charges", detail.OtherCharges),
			zap.Float64("gross_amount", detail.GrossAmount),
			zap.String("zone", detail.Zone),
			zap.String("weight_slab", detail.WeightSlab),
			zap.Float64("charged_weight", detail.ChargedWeight),
			zap.Float64("actual_weight", detail.ActualWeight))
	}

	return totalAmount, nil
}

// getEstimatedTAT gets estimated delivery time from Delhivery using structured entities
func (s *DeliveryService) getEstimatedTAT(ctx context.Context, destinationPin string) (int, error) {
	// Create structured TAT request
	tatReq := requests.NewTATRequest(s.config.OriginPincode, destinationPin)
	if err := tatReq.IsValid(); err != nil {
		return 0, fmt.Errorf("invalid TAT request: %w", err)
	}

	endpoint := tatReq.BuildTATURL(s.config.BaseURL)
	s.logger.Info("Delhivery TAT API request", zap.String("endpoint", endpoint),
		zap.String("destination_pin", destinationPin))

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Delhivery TAT API failed", zap.Error(err))
		return 0, fmt.Errorf("failed to get TAT from Delhivery API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Error("Failed to read TAT response body", zap.Error(err))
		return 0, fmt.Errorf("failed to read TAT response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("Delhivery TAT API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
			zap.String("endpoint", endpoint))
		return 0, fmt.Errorf("Delhivery TAT API failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	s.logger.Info("Delhivery TAT API response", zap.String("response", string(body)))

	// Parse response using structured entity
	tatResp, err := responses.ParseTATResponse(body)
	if err != nil {
		s.logger.Error("Failed to decode TAT response", zap.Error(err), zap.String("body", string(body)))
		return 0, fmt.Errorf("failed to decode TAT response: %w", err)
	}

	// Validate TAT response
	if !tatResp.IsValidTAT() {
		s.logger.Warn("Invalid TAT received from API", zap.Any("response", tatResp))
		return 0, fmt.Errorf("invalid TAT received from API")
	}

	s.logger.Info("Delhivery TAT parsed successfully",
		zap.Int("days", tatResp.Data.TAT),
		zap.String("expected_date", tatResp.Data.ExpectedDeliveryDate))

	return tatResp.Data.TAT, nil
}

// extractPincode extracts and validates pincode from postal_code field
func (s *DeliveryService) extractPincode(postalCode string) string {
	// Remove any non-numeric characters
	cleaned := strings.ReplaceAll(postalCode, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	// Extract numeric characters only
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

	s.logger.Warn("Invalid pincode format",
		zap.String("original", postalCode),
		zap.String("extracted", result))
	return ""
}

// Estimate computes delivery options using Delhivery APIs - CORRECTED VERSION
func (s *DeliveryService) Estimate(ctx context.Context, userID string, req requests.EstimateShippingRequest) (*models.DeliveryEstimate, error) {
	// Extract pincode from postal_code
	pincode := s.extractPincode(req.Address.PostalCode)
	if pincode == "" {
		return nil, fmt.Errorf("invalid postal code format: %s", req.Address.PostalCode)
	}

	s.logger.Info("Starting delivery estimation",
		zap.String("user_id", userID),
		zap.String("pincode", pincode),
		zap.String("original_postal_code", req.Address.PostalCode))

	// Get cart subtotal for weight estimation (must be real values; no fallbacks)
	var subtotal float64
	var totalItems int

	if s.repoManager == nil || s.repoManager.CartRepository == nil {
		return nil, fmt.Errorf("cart repository unavailable")
	}

	cart, err := s.repoManager.CartRepository.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to load cart: %w", err)
	}

	subtotal = cart.TotalPrice
	totalItems = cart.TotalItems
	if totalItems <= 0 {
		return nil, fmt.Errorf("cart is empty; add items before estimating shipping")
	}

	s.logger.Info("Cart details",
		zap.Float64("subtotal", subtotal),
		zap.Int("items_count", totalItems))

	// Check serviceability first
	serviceable, err := s.CheckServiceability(ctx, pincode)
	if err != nil {
		s.logger.Error("Failed to check serviceability", zap.Error(err), zap.String("pincode", pincode))
		return nil, fmt.Errorf("failed to check serviceability: %w", err)
	}

	if !serviceable {
		s.logger.Info("Pincode not serviceable", zap.String("pincode", pincode))
		return nil, fmt.Errorf("delivery service is not available to pincode %s. We currently don't deliver to this location", pincode)
	}

	// Calculate weight based on total quantity of items (100g per item)
	const PRODUCT_WEIGHT_GRAMS = 100
	weight := totalItems * PRODUCT_WEIGHT_GRAMS

	s.logger.Info("Weight calculation",
		zap.Int("total_quantity", totalItems),
		zap.Int("weight_per_item_grams", PRODUCT_WEIGHT_GRAMS),
		zap.Int("total_weight_grams", weight))

	// Get shipping charges from Delhivery for prepaid (no COD for now)
	charges, err := s.getShippingCharges(ctx, pincode, weight, 0) // 0 = prepaid
	if err != nil {
		s.logger.Error("Failed to get shipping charges from Delhivery API",
			zap.Error(err),
			zap.String("pincode", pincode),
			zap.Int("weight", weight))

		// Return error instead of using fallback
		return nil, fmt.Errorf("failed to get shipping charges from Delhivery API: %w", err)
	}

	s.logger.Info("Successfully got charges from Delhivery API", zap.Float64("charges", charges))

	// Get estimated TAT from Delhivery (non-blocking for pricing)
	deliveryDays, err := s.getEstimatedTAT(ctx, pincode)
	if err != nil {
		s.logger.Warn("Proceeding without TAT due to API error", zap.Error(err))
		deliveryDays = 3 // Default fallback
	}

	// Build delivery options - only Surface option with correct charges
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

	// Keep original subtotal separate from shipping
	result := &models.DeliveryEstimate{
		UserID:   userID,
		Subtotal: subtotal, // Original cart subtotal
		Options:  options,
	}

	// Save compact estimate for payment flow reuse
	// Build address string from available parts gracefully
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
		Subtotal:        subtotal,
		Total:           subtotal + surfaceCharges,
		Currency:        "INR", // Always INR
		EtaMinDays:      deliveryDays,
		EtaMaxDays:      deliveryDays + 2,
		Pincode:         pincode,
		Timestamp:       time.Now().Unix(),
	}
	s.saveLastEstimate(ctx, userID, cached)

	s.logger.Info("Delivery estimation completed",
		zap.Int("options_count", len(options)),
		zap.String("pincode", pincode),
		zap.Float64("cart_subtotal", subtotal),
		zap.Float64("shipping_charges", surfaceCharges),
		zap.Float64("total_with_shipping", subtotal+surfaceCharges))

	return result, nil
}

// CreateShipment creates a Delhivery shipment after successful payment
func (s *DeliveryService) CreateShipment(ctx context.Context, order *models.Order, orderItems []models.OrderItem) (string, error) {
	if order == nil {
		return "", fmt.Errorf("order cannot be nil")
	}

	// Parse shipping address to extract delivery details
	addressParts := strings.Split(order.ShippingAddr, ",")
	if len(addressParts) < 3 {
		return "", fmt.Errorf("invalid shipping address format")
	}

	// Extract pincode from shipping address (assuming it's at the end)
	pincode := s.extractPincode(order.ShippingAddr)
	if pincode == "" {
		return "", fmt.Errorf("could not extract pincode from shipping address")
	}

	// Build products description
	productDescs := []string{}
	totalWeight := 0
	for _, item := range orderItems {
		productDescs = append(productDescs, fmt.Sprintf("Product %s (Qty: %d)", item.ProductID, item.Quantity))
		totalWeight += item.Quantity * 100 // 100g per item
	}
	if totalWeight < 100 {
		totalWeight = 100 // Minimum weight
	}

	// Create shipment request
	shipmentReq := requests.NewShipmentRequest(s.config.WarehouseName)

	// Parse customer details from shipping address
	customerName := "Customer"    // Default, could be extracted from address
	customerPhone := "9999999999" // Default, should come from user profile

	// Create shipment detail
	shipmentDetail := requests.DelhiveryShipmentDetail{
		Name:          customerName,
		Add:           order.ShippingAddr,
		Pin:           pincode,
		City:          s.extractCityFromAddress(order.ShippingAddr),
		State:         s.extractStateFromAddress(order.ShippingAddr),
		Country:       "India",
		Phone:         customerPhone,
		Order:         order.ID,
		PaymentMode:   "Prepaid", // Assuming prepaid since payment is already done
		ReturnPin:     s.config.OriginPincode,
		ReturnCity:    s.config.OriginCity,
		ReturnName:    s.config.WarehouseName,
		ReturnAdd:     s.config.OriginAddress,
		ReturnState:   s.config.OriginState,
		ReturnCountry: "India",
		ReturnPhone:   s.config.OriginPhone,
		ProductsDesc:  strings.Join(productDescs, ", "),
		HsnCode:       "1234", // Default HSN code
		TotalAmount:   fmt.Sprintf("%.0f", order.TotalAmount),
		SellerName:    "Kisanlink",
		SellerInv:     order.ID,
		Quantity:      fmt.Sprintf("%d", len(orderItems)),
		Weight:        fmt.Sprintf("%d", totalWeight),
		ShippingMode:  "Surface",
		AddressType:   "home",
		ShipWidth:     "10",
		ShipHeight:    "10",
	}

	shipmentReq.AddShipmentDetail(shipmentDetail)

	// Validate request
	if err := shipmentReq.IsValid(); err != nil {
		return "", fmt.Errorf("invalid shipment request: %w", err)
	}

	// Build endpoint URL
	endpoint := shipmentReq.BuildShipmentURL(s.config.BaseURL)
	s.logger.Info("Creating Delhivery shipment",
		zap.String("endpoint", endpoint),
		zap.String("order_id", order.ID))

	// Marshal request body
	reqBody, err := json.Marshal(shipmentReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal shipment request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(string(reqBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Make API call
	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Error("Delhivery shipment API failed", zap.Error(err))
		return "", fmt.Errorf("failed to create shipment: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	s.logger.Info("Delhivery shipment API response",
		zap.String("response", string(body)),
		zap.Int("status_code", resp.StatusCode))

	if resp.StatusCode != http.StatusOK {
		s.logger.Error("Delhivery shipment API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return "", fmt.Errorf("shipment API failed: HTTP %d - %s", resp.StatusCode, string(body))
	}

	// Parse response
	shipmentResp, err := responses.ParseShipmentResponse(body)
	if err != nil {
		s.logger.Error("Failed to parse shipment response", zap.Error(err))
		return "", fmt.Errorf("failed to parse shipment response: %w", err)
	}

	// Extract waybill (AWB)
	waybill := shipmentResp.GetWaybill()
	if waybill == "" {
		return "", fmt.Errorf("no waybill received from Delhivery")
	}

	s.logger.Info("Delhivery shipment created successfully",
		zap.String("order_id", order.ID),
		zap.String("waybill", waybill),
		zap.String("status", shipmentResp.GetStatus()))

	return waybill, nil
}

// Helper methods for address parsing
func (s *DeliveryService) extractCityFromAddress(address string) string {
	parts := strings.Split(address, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if len(part) > 2 && !s.isNumeric(part) {
			return part
		}
	}
	return "Unknown"
}

func (s *DeliveryService) extractStateFromAddress(address string) string {
	parts := strings.Split(address, ",")
	for i := len(parts) - 2; i >= 0; i-- {
		part := strings.TrimSpace(parts[i])
		if len(part) > 2 && !s.isNumeric(part) && !strings.Contains(part, "India") {
			return part
		}
	}
	return "Unknown"
}

func (s *DeliveryService) isNumeric(str string) bool {
	for _, char := range str {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}
