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
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// DeliveryServiceInterface defines delivery-related business behavior
type DeliveryServiceInterface interface {
	Estimate(ctx context.Context, userID string, req requests.EstimateShippingRequest) (*models.DeliveryEstimate, error)
	CheckServiceability(ctx context.Context, pincode string) (bool, error)
	CreateShipment(ctx context.Context, order *models.Order, orderItems []models.OrderItem) (*ShipmentResult, error)
	GetTrackingInfo(ctx context.Context, waybill string) (*TrackingInfo, error)
	SyncDeliveryStatus(ctx context.Context, orderID string) (*TrackingInfo, error)
	GetShippingCharges(ctx context.Context, destinationPin string, weightGrams int, codAmount float64) (float64, error)
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
	delhiveryConfig := config.GetDelhiveryConfig()

	// Validate configuration and log any issues
	if issues := config.ValidateDelhiveryConfig(delhiveryConfig); len(issues) > 0 {
		for _, issue := range issues {
			logger.Error("Delhivery configuration issue", zap.String("issue", issue))
		}
		logger.Warn("Delhivery service may not work properly due to configuration issues")
	}

	// Log configuration (with masked sensitive data)
	logger.Info("Delhivery service initialized with configuration",
		zap.String("base_url", delhiveryConfig.BaseURL),
		zap.Bool("is_production", delhiveryConfig.IsProduction),
		zap.String("warehouse_name", delhiveryConfig.WarehouseName),
		zap.String("origin_pincode", delhiveryConfig.OriginPincode),
		zap.String("origin_city", delhiveryConfig.OriginCity),
		zap.String("origin_state", delhiveryConfig.OriginState),
		zap.String("api_token_preview", delhiveryConfig.APIToken[:8]+"..."))

	return &DeliveryService{
		repoManager: repoManager,
		dbManager:   repoManager.GetDBManager(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		config: delhiveryConfig,
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

// GetShippingCharges gets shipping charges from Delhivery API based on destination pincode and weight
func (s *DeliveryService) GetShippingCharges(ctx context.Context, destinationPin string, weightGrams int, codAmount float64) (float64, error) {
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

	// GetUserCart can return (nil, nil) when the user has no cart yet; guard before
	// dereferencing it (#24) so an estimate for an empty cart returns a clean error
	// instead of panicking.
	if cart == nil || cart.TotalItems <= 0 {
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

	charges, err := s.GetShippingCharges(ctx, pincode, weight, 0)
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

// isPlaceholderDestination reports whether a resolved shipping destination is a
// placeholder/invalid address that must never be shipped to (#8/#6). It checks the
// structured destination (pincode/city/state) plus the full address string for the
// placeholder markers used across the codebase ("Unknown", "000000", "not provided").
func isPlaceholderDestination(pincode, city, state, addressString string) bool {
	p := strings.TrimSpace(pincode)
	if p == "" || p == "000000" {
		return true
	}
	if c := strings.TrimSpace(city); c == "" || strings.EqualFold(c, "Unknown") {
		return true
	}
	if st := strings.TrimSpace(state); st == "" || strings.EqualFold(st, "Unknown") {
		return true
	}
	a := strings.ToLower(addressString)
	if strings.Contains(a, "address not provided") ||
		strings.Contains(a, "unknown") ||
		strings.Contains(a, "000000") {
		return true
	}
	return false
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
	var shippingAddressString string

	// Log address IDs for debugging
	s.logger.Info("CreateShipment: Starting address resolution",
		zap.String("order_id", order.ID),
		zap.String("shipping_address_id", order.ShippingAddressID),
		zap.String("shipping_addr_id_legacy", order.ShippingAddrID),
		zap.String("shipping_addr_string", order.ShippingAddr))

	// PRIORITY 1: Try to use OrderAddress (normalized address from order)
	// This is used for Magic Checkout and new orders
	// IMPORTANT: Only use this if ShippingAddressID has OADR prefix (order_addresses table)
	// Legacy ADDR prefix belongs to the 'addresses' table and should use Priority 2
	if order.ShippingAddressID != "" && strings.HasPrefix(order.ShippingAddressID, "OADR") && s.repoManager != nil && s.repoManager.OrderRepository != nil {
		orderAddr, addrErr := s.repoManager.OrderRepository.GetOrderAddress(ctx, order.ShippingAddressID)
		if addrErr == nil && orderAddr != nil {
			pincode = orderAddr.PostalCode
			destCity = orderAddr.City
			destState = orderAddr.State
			// Build address string from OrderAddress
			line2 := ""
			if orderAddr.AddressLine2 != "" {
				line2 = ", " + orderAddr.AddressLine2
			}
			shippingAddressString = fmt.Sprintf("%s%s, %s, %s %s, %s",
				orderAddr.AddressLine1, line2, orderAddr.City, orderAddr.State, orderAddr.PostalCode, orderAddr.Country)

			s.logger.Info("Using OrderAddress for shipment",
				zap.String("order_id", order.ID),
				zap.String("address_id", order.ShippingAddressID),
				zap.String("city", destCity),
				zap.String("state", destState),
				zap.String("pincode", pincode))
		} else {
			s.logger.Warn("Failed to fetch OrderAddress, trying next option",
				zap.String("order_id", order.ID),
				zap.String("address_id", order.ShippingAddressID),
				zap.Error(addrErr))
		}
	} else if order.ShippingAddressID != "" && !strings.HasPrefix(order.ShippingAddressID, "OADR") {
		// ShippingAddressID has wrong prefix - log this as it indicates a bug
		s.logger.Warn("ShippingAddressID has incorrect prefix (expected OADR), skipping to legacy lookup",
			zap.String("order_id", order.ID),
			zap.String("shipping_address_id", order.ShippingAddressID),
			zap.String("expected_prefix", "OADR"))
	}

	// PRIORITY 2: Try to use legacy Address table (for backward compatibility)
	if pincode == "" && order.ShippingAddrID != "" && s.repoManager != nil && s.repoManager.AddressRepository != nil {
		addr, addrErr := s.repoManager.AddressRepository.GetAddressByID(ctx, order.UserID, order.ShippingAddrID)
		if addrErr == nil && addr != nil {
			pincode = addr.PostalCode
			destCity = addr.City
			destState = addr.State
			// Build address string from Address
			line2 := ""
			if addr.AddressLine2 != "" {
				line2 = ", " + addr.AddressLine2
			}
			name := addr.FirstName
			if addr.LastName != "" {
				name = name + " " + addr.LastName
			}
			shippingAddressString = fmt.Sprintf("%s, %s%s, %s, %s %s, %s",
				name, addr.AddressLine1, line2, addr.City, addr.State, addr.PostalCode, addr.Country)

			s.logger.Info("Using legacy Address for shipment",
				zap.String("order_id", order.ID),
				zap.String("address_id", order.ShippingAddrID),
				zap.String("city", destCity),
				zap.String("state", destState),
				zap.String("pincode", pincode))
		} else {
			s.logger.Warn("Failed to fetch legacy address, falling back to string parsing",
				zap.String("order_id", order.ID),
				zap.String("address_id", order.ShippingAddrID),
				zap.Error(addrErr))
		}
	}

	// PRIORITY 3: Fallback to parsing address string if structured data not available
	if pincode == "" {
		// Use existing ShippingAddr string if available
		if order.ShippingAddr != "" {
			shippingAddressString = order.ShippingAddr
		}

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

		// If address string is empty, build a basic one
		if shippingAddressString == "" {
			shippingAddressString = fmt.Sprintf("Address, %s, %s %s", destCity, destState, pincode)
		}
	}

	// SECURITY (#8/#6): refuse to ship to a placeholder/invalid address, at the single choke
	// point every shipment path goes through (verify/webhook via finalizeFulfillment AND the
	// admin manual retry via CreateOrderShipment). This guards the resolved destination, not
	// just the order's address string, so a "Unknown / 000000" parcel can never be dispatched.
	if isPlaceholderDestination(pincode, destCity, destState, shippingAddressString) {
		s.logger.Error("CreateShipment: refusing shipment to placeholder/invalid address",
			zap.String("order_id", order.ID),
			zap.String("pincode", pincode),
			zap.String("city", destCity),
			zap.String("state", destState))
		return &ShipmentResult{
			Success: false,
			Message: "shipping address is incomplete or invalid; cannot create shipment",
		}, fmt.Errorf("placeholder or invalid shipping address for order %s", order.ID)
	}

	// Fetch customer data from order_customers table (PRIORITY 1)
	// Falls back to legacy order-level fields if not available
	// IMPORTANT: Only lookup if CustomerID has OCUS prefix (order_customers table)
	var custName, custPhone string

	if order.CustomerID != "" && strings.HasPrefix(order.CustomerID, "OCUS") && s.repoManager != nil && s.repoManager.OrderRepository != nil {
		orderCustomer, custErr := s.repoManager.OrderRepository.GetOrderCustomer(ctx, order.CustomerID)
		if custErr == nil && orderCustomer != nil {
			custName = strings.TrimSpace(orderCustomer.FirstName + " " + orderCustomer.LastName)
			custPhone = strings.TrimSpace(orderCustomer.Phone)

			s.logger.Info("Using OrderCustomer for shipment",
				zap.String("order_id", order.ID),
				zap.String("customer_id", order.CustomerID),
				zap.String("customer_name", custName),
				zap.String("customer_phone", custPhone))
		} else {
			s.logger.Warn("Failed to fetch OrderCustomer, falling back to legacy fields",
				zap.String("order_id", order.ID),
				zap.String("customer_id", order.CustomerID),
				zap.Error(custErr))
		}
	} else if order.CustomerID != "" && !strings.HasPrefix(order.CustomerID, "OCUS") {
		// CustomerID has wrong prefix - log warning
		s.logger.Warn("CustomerID has incorrect prefix (expected OCUS), using legacy fields",
			zap.String("order_id", order.ID),
			zap.String("customer_id", order.CustomerID),
			zap.String("expected_prefix", "OCUS"))
	}

	// Fallback to legacy order-level customer fields
	if custName == "" {
		custName = strings.TrimSpace(order.CustomerFirstName + " " + order.CustomerLastName)
	}
	if custPhone == "" {
		custPhone = strings.TrimSpace(order.CustomerPhone)
	}

	if custName == "" || custPhone == "" {
		return &ShipmentResult{Success: false, Message: "Customer name and phone are required on the order"}, fmt.Errorf("missing customer name/phone on order")
	}

	// Normalize primary phone to last 10 digits
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

	// Build products description with actual product names
	productDescs := []string{}
	totalWeight := 0

	// Collect unique product IDs
	productIDSet := make(map[string]bool)
	for _, item := range orderItems {
		productIDSet[item.ProductID] = true
	}

	// Fetch all products to get their names
	var productIDs []string
	for productID := range productIDSet {
		productIDs = append(productIDs, productID)
	}

	// Get all products at once for better performance
	productMap := make(map[string]models.Product)
	if len(productIDs) > 0 && s.repoManager != nil && s.repoManager.ProductRepository != nil {
		productFilter := &base.Filter{
			Group: base.FilterGroup{
				Conditions: []base.FilterCondition{
					{Field: "id", Operator: base.OpIn, Value: productIDs},
				},
				Logic: base.LogicAnd,
			},
		}
		if products, err := s.repoManager.ProductRepository.Find(ctx, productFilter); err == nil {
			// Create a map for quick product lookup
			for _, product := range products {
				productMap[product.ID] = *product
			}
		}
	}

	// Build product descriptions with actual names
	for _, item := range orderItems {
		productName := item.ProductID // fallback to product ID
		if product, exists := productMap[item.ProductID]; exists {
			productName = product.Name
		}
		productDescs = append(productDescs, fmt.Sprintf("%s (Qty: %d)", productName, item.Quantity))
		totalWeight += item.Quantity * 100 // 100g per item
	}

	if totalWeight < 100 {
		totalWeight = 100
	}

	// Create shipment request
	shipmentReq := requests.NewShipmentRequest(s.config.WarehouseName)

	// #9 dedupe: use a DETERMINISTIC courier reference (the order ID) rather than appending a
	// timestamp. A stable reference lets Delhivery reject a duplicate manifest for the same order,
	// a second line of defense against double shipments on top of the atomic shipment claim.
	uniqueOrderID := order.ID

	// Use the address string we built (from OrderAddress, Address, or parsed string)
	// Include phone number subtly in the address - not highlighted, just there if needed
	enhancedAddress := shippingAddressString
	if enhancedAddress == "" {
		enhancedAddress = order.ShippingAddr // Final fallback
	}
	if custPhone != "" && enhancedAddress != "" {
		enhancedAddress = fmt.Sprintf("%s, %s", enhancedAddress, custPhone)
	}

	shipmentDetail := requests.DelhiveryShipmentDetail{
		Name:          custName,
		Add:           enhancedAddress,
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
		SellerName:    "Amrti",
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

	// Enhanced logging for shipment request preparation
	s.logger.Info("Preparing Delhivery shipment request with detailed config",
		zap.String("order_id", order.ID),
		zap.String("api_token", s.config.APIToken[:8]+"..."+s.config.APIToken[len(s.config.APIToken)-8:]), // Masked token
		zap.String("base_url", s.config.BaseURL),
		zap.Bool("is_production", s.config.IsProduction),
		zap.String("warehouse_name", s.config.WarehouseName),
		zap.String("origin_pincode", s.config.OriginPincode),
		zap.String("origin_city", s.config.OriginCity),
		zap.String("origin_state", s.config.OriginState),
		zap.String("dest_pincode", pincode),
		zap.String("dest_city", destCity),
		zap.String("dest_state", destState),
		zap.String("customer_name", custName),
		zap.String("customer_phone", custPhone),
		zap.String("unique_order_id", uniqueOrderID))

	s.logger.Info("Sending shipment creation request to Delhivery",
		zap.String("order_id", order.ID),
		zap.String("endpoint", endpoint),
		zap.String("unique_order_id", uniqueOrderID),
		zap.String("request_data", string(dataJSON)))

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

	// Log request headers for debugging (without exposing full token)
	s.logger.Info("HTTP request details for Delhivery API",
		zap.String("order_id", order.ID),
		zap.String("method", "POST"),
		zap.String("endpoint", endpoint),
		zap.String("content_type", "application/x-www-form-urlencoded"),
		zap.String("authorization_prefix", "Token "+s.config.APIToken[:8]+"..."),
		zap.Int("form_data_length", len(formData.Encode())))

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

	// Enhanced response logging for debugging AWS issues
	s.logger.Info("Raw Delhivery API response received",
		zap.String("order_id", order.ID),
		zap.Int("response_length", len(body)),
		zap.Int("status_code", resp.StatusCode),
		zap.String("content_type", resp.Header.Get("Content-Type")))

	// Always log the full response body for debugging
	s.logger.Info("Full Delhivery API response body",
		zap.String("order_id", order.ID),
		zap.String("response_body", string(body)))

	shipmentResp, err := responses.ParseShipmentResponse(body)
	if err != nil {
		s.logger.Error("Failed to parse Delhivery shipment response - JSON parsing error",
			zap.Error(err),
			zap.String("order_id", order.ID),
			zap.String("response_body", string(body)),
			zap.String("error_type", "json_parse_error"))
		return &ShipmentResult{
			Success: false,
			Message: "Failed to parse Delhivery response",
		}, fmt.Errorf("failed to parse shipment response: %w", err)
	}

	// Enhanced response validation with detailed logging
	s.logger.Info("Analyzing Delhivery response structure",
		zap.String("order_id", order.ID),
		zap.Bool("response_success", shipmentResp.Success),
		zap.Int("packages_count", len(shipmentResp.PackageData)),
		zap.String("rmk_field", shipmentResp.RMK),
		zap.Float64("cash_total", shipmentResp.CashTotal),
		zap.Float64("cod_total", shipmentResp.CODTotal),
		zap.Float64("prepaid", shipmentResp.Prepaid))

	if !shipmentResp.IsSuccessful() {
		errorMsg := shipmentResp.GetErrorMessage()

		// Enhanced error logging with configuration details
		s.logger.Error("Delhivery shipment creation failed - API returned failure status",
			zap.String("order_id", order.ID),
			zap.String("error_message", errorMsg),
			zap.Bool("success_flag", shipmentResp.Success),
			zap.Int("packages_received", len(shipmentResp.PackageData)),
			zap.String("rmk", shipmentResp.RMK),
			zap.String("error_type", "delhivery_api_failure"))

		// Log configuration to help debug
		s.logger.Error("Delhivery configuration at time of failure - CHECK THESE VALUES",
			zap.String("order_id", order.ID),
			zap.String("warehouse_name", s.config.WarehouseName),
			zap.String("origin_pincode", s.config.OriginPincode),
			zap.String("origin_city", s.config.OriginCity),
			zap.String("origin_state", s.config.OriginState),
			zap.String("origin_address", s.config.OriginAddress),
			zap.String("origin_phone", s.config.OriginPhone),
			zap.String("base_url", s.config.BaseURL),
			zap.Bool("is_production", s.config.IsProduction),
			zap.String("dest_pincode", pincode),
			zap.String("dest_city", destCity),
			zap.String("dest_state", destState),
			zap.String("customer_name", custName),
			zap.String("customer_phone", custPhone),
			zap.String("shipping_address", shippingAddressString),
			zap.String("HINT", "Warehouse name must EXACTLY match pickup location name in Delhivery dashboard"))

		return &ShipmentResult{
			Success: false,
			Message: fmt.Sprintf("Failed to create shipment: %s", errorMsg),
		}, fmt.Errorf("shipment creation failed: %s", errorMsg)
	}

	waybill := shipmentResp.GetWaybill()
	s.logger.Info("Extracted waybill from response",
		zap.String("order_id", order.ID),
		zap.String("waybill", waybill),
		zap.Int("packages_count", len(shipmentResp.PackageData)))

	if waybill == "" {
		// Enhanced debugging for empty waybill
		if len(shipmentResp.PackageData) > 0 {
			pkg := shipmentResp.PackageData[0]
			s.logger.Error("Waybill extraction failed despite having package data",
				zap.String("order_id", order.ID),
				zap.String("package_status", pkg.Status),
				zap.String("package_client", pkg.Client),
				zap.String("package_sort_code", pkg.SortCode),
				zap.String("package_waybill", pkg.Waybill),
				zap.String("package_refnum", pkg.RefNum),
				zap.String("package_payment_mode", pkg.PaymentMode),
				zap.String("package_remarks", pkg.Remarks.String()),
				zap.String("error_type", "empty_waybill_with_packages"))
		}

		s.logger.Error("No waybill received from Delhivery despite successful response",
			zap.String("order_id", order.ID),
			zap.String("response_body", string(body)),
			zap.String("error_type", "no_waybill_received"))
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

// GetTrackingInfo retrieves real-time tracking information for a waybill from Delhivery API
// SyncDeliveryStatus pulls the latest courier status for an order's shipment and reflects it on
// the order: it refreshes DelhiveryStatus and advances Order.Status to delivered once the courier
// reports delivery. Safe to call repeatedly (admin action or scheduled job). No-op (returns an
// error) for orders without a waybill.
func (s *DeliveryService) SyncDeliveryStatus(ctx context.Context, orderID string) (*TrackingInfo, error) {
	if s.dbManager == nil {
		return nil, fmt.Errorf("db manager unavailable")
	}

	var order models.Order
	if err := s.dbManager.GetByID(ctx, orderID, &order); err != nil {
		return nil, fmt.Errorf("failed to load order %s: %w", orderID, err)
	}

	waybill := strings.TrimSpace(order.DelhiveryWaybill)
	if waybill == "" {
		return nil, fmt.Errorf("order %s has no shipment/waybill to track", orderID)
	}

	info, err := s.GetTrackingInfo(ctx, waybill)
	if err != nil || info == nil {
		return nil, fmt.Errorf("failed to get tracking info for waybill %s: %w", waybill, err)
	}

	changed := false
	if info.Status != "" && info.Status != order.DelhiveryStatus {
		order.DelhiveryStatus = info.Status
		changed = true
	}
	// #16: advance to delivered ONLY on a true customer delivery. A plain substring match on
	// "delivered" also matched "Undelivered" and "RTO/DTO Delivered" (parcel returned to origin),
	// which wrongly advanced undelivered/returned orders to delivered. Require "delivered" while
	// excluding the undelivered and return-to-origin variants.
	normalized := strings.ToLower(strings.TrimSpace(info.Status))
	isTrueDelivery := strings.Contains(normalized, "delivered") &&
		!strings.Contains(normalized, "undelivered") &&
		!strings.Contains(normalized, "rto") &&
		!strings.Contains(normalized, "dto")
	if isTrueDelivery && order.Status != models.OrderStatusDelivered {
		order.Status = models.OrderStatusDelivered
		changed = true
	}

	if changed {
		if err := s.dbManager.Update(ctx, &order); err != nil {
			return nil, fmt.Errorf("failed to update order %s delivery status: %w", orderID, err)
		}
		s.logger.Info("Synced delivery status",
			zap.String("order_id", orderID),
			zap.String("waybill", waybill),
			zap.String("delhivery_status", order.DelhiveryStatus),
			zap.String("order_status", string(order.Status)))
	}

	return info, nil
}

func (s *DeliveryService) GetTrackingInfo(ctx context.Context, waybill string) (*TrackingInfo, error) {
	if waybill == "" {
		return nil, fmt.Errorf("waybill cannot be empty")
	}

	// Build tracking API URL
	// Delhivery tracking API endpoint: https://track.delhivery.com/api/v1/packages/json/
	trackingEndpoint := fmt.Sprintf("https://track.delhivery.com/api/v1/packages/json/?token=%s&waybill=%s",
		s.config.APIToken, waybill)

	s.logger.Info("Fetching tracking info from Delhivery API",
		zap.String("waybill", waybill),
		zap.String("endpoint", "track.delhivery.com/api/v1/packages/json"))

	req, err := http.NewRequestWithContext(ctx, "GET", trackingEndpoint, nil)
	if err != nil {
		s.logger.Error("Failed to create tracking request",
			zap.String("waybill", waybill),
			zap.Error(err))
		return s.getStaticTrackingInfo(waybill), nil
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		s.logger.Warn("Failed to fetch tracking from Delhivery API, returning static info",
			zap.String("waybill", waybill),
			zap.Error(err))
		return s.getStaticTrackingInfo(waybill), nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		s.logger.Warn("Failed to read tracking response",
			zap.String("waybill", waybill),
			zap.Error(err))
		return s.getStaticTrackingInfo(waybill), nil
	}

	if resp.StatusCode != http.StatusOK {
		s.logger.Warn("Delhivery tracking API returned non-200 status",
			zap.String("waybill", waybill),
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)))
		return s.getStaticTrackingInfo(waybill), nil
	}

	// Parse tracking response
	trackingResp, err := responses.ParseTrackingResponse(body)
	if err != nil {
		s.logger.Warn("Failed to parse tracking response",
			zap.String("waybill", waybill),
			zap.Error(err),
			zap.String("response", string(body)))
		return s.getStaticTrackingInfo(waybill), nil
	}

	// Extract tracking details
	shipment := trackingResp.GetShipmentTracking()
	if shipment == nil {
		s.logger.Warn("No shipment data found in tracking response",
			zap.String("waybill", waybill))
		return s.getStaticTrackingInfo(waybill), nil
	}

	status := trackingResp.GetCurrentStatus()
	if status == "" {
		status = "In Transit"
	}

	lastUpdate := trackingResp.GetLastUpdate()
	if lastUpdate == "" {
		lastUpdate = time.Now().Format("2006-01-02 15:04:05")
	}

	trackingURL := fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", waybill)

	s.logger.Info("Successfully retrieved tracking info from Delhivery",
		zap.String("waybill", waybill),
		zap.String("status", status),
		zap.String("last_update", lastUpdate),
		zap.String("expected_delivery", trackingResp.GetExpectedDelivery()))

	return &TrackingInfo{
		Waybill:     waybill,
		Status:      status,
		TrackingURL: trackingURL,
		LastUpdate:  lastUpdate,
	}, nil
}

// getStaticTrackingInfo returns a basic tracking info with just the URL (fallback)
func (s *DeliveryService) getStaticTrackingInfo(waybill string) *TrackingInfo {
	return &TrackingInfo{
		Waybill:     waybill,
		Status:      "Shipped",
		TrackingURL: fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", waybill),
		LastUpdate:  time.Now().Format("2006-01-02 15:04:05"),
	}
}

// ValidateWarehouse checks if the warehouse/pickup location is registered with Delhivery
func (s *DeliveryService) ValidateWarehouse(ctx context.Context) (*WarehouseValidationResult, error) {
	result := &WarehouseValidationResult{
		WarehouseName: s.config.WarehouseName,
		IsValid:       false,
		Issues:        []string{},
	}

	// Check configuration issues first
	configIssues := config.ValidateDelhiveryConfig(s.config)
	if len(configIssues) > 0 {
		result.Issues = append(result.Issues, configIssues...)
	}

	// Check if origin address is still default
	if s.config.OriginAddress == "Default Address" || s.config.OriginAddress == "" {
		result.Issues = append(result.Issues, "CRITICAL: DELHIVERY_ORIGIN_ADDRESS is not set to your actual pickup address")
	}

	// Try to get warehouses from Delhivery API
	endpoint := fmt.Sprintf("%s/api/backend/clientwarehouse/edit/", s.config.BaseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to create warehouse check request: %v", err))
		return result, nil
	}

	req.Header.Set("Authorization", "Token "+s.config.APIToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		result.Issues = append(result.Issues, fmt.Sprintf("Failed to connect to Delhivery: %v", err))
		return result, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == http.StatusOK {
		// Try to parse warehouse list
		var warehouseResp struct {
			Data []struct {
				Name   string `json:"name"`
				Active bool   `json:"active"`
			} `json:"data"`
		}

		if err := json.Unmarshal(body, &warehouseResp); err == nil {
			result.AvailableWarehouses = []string{}
			for _, wh := range warehouseResp.Data {
				result.AvailableWarehouses = append(result.AvailableWarehouses, wh.Name)
				if wh.Name == s.config.WarehouseName {
					result.IsValid = true
					if !wh.Active {
						result.Issues = append(result.Issues, fmt.Sprintf("Warehouse '%s' exists but is NOT ACTIVE", s.config.WarehouseName))
						result.IsValid = false
					}
				}
			}

			if !result.IsValid && len(result.AvailableWarehouses) > 0 {
				result.Issues = append(result.Issues, fmt.Sprintf("Warehouse '%s' NOT FOUND. Available warehouses: %v", s.config.WarehouseName, result.AvailableWarehouses))
			}
		}
	} else {
		s.logger.Warn("Could not fetch warehouse list from Delhivery",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response", string(body)))
	}

	// Log the validation result
	s.logger.Info("Delhivery warehouse validation completed",
		zap.Bool("is_valid", result.IsValid),
		zap.String("warehouse_name", result.WarehouseName),
		zap.Strings("issues", result.Issues),
		zap.Strings("available_warehouses", result.AvailableWarehouses))

	return result, nil
}

// WarehouseValidationResult contains the result of warehouse validation
type WarehouseValidationResult struct {
	WarehouseName       string   `json:"warehouse_name"`
	IsValid             bool     `json:"is_valid"`
	Issues              []string `json:"issues"`
	AvailableWarehouses []string `json:"available_warehouses,omitempty"`
}

// GetConfigurationStatus returns the current Delhivery configuration status for debugging
func (s *DeliveryService) GetConfigurationStatus() map[string]interface{} {
	issues := config.ValidateDelhiveryConfig(s.config)

	return map[string]interface{}{
		"is_production":   s.config.IsProduction,
		"base_url":        s.config.BaseURL,
		"warehouse_name":  s.config.WarehouseName,
		"origin_pincode":  s.config.OriginPincode,
		"origin_city":     s.config.OriginCity,
		"origin_state":    s.config.OriginState,
		"origin_address":  s.config.OriginAddress,
		"origin_phone":    s.config.OriginPhone,
		"has_api_token":   s.config.APIToken != "",
		"api_token_len":   len(s.config.APIToken),
		"config_issues":   issues,
		"is_valid_config": len(issues) == 0,
	}
}
