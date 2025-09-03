package requests

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// DelhiveryServiceabilityRequest represents a request to check pincode serviceability
type DelhiveryServiceabilityRequest struct {
	ProductType string `json:"product_type" validate:"required,oneof=Heavy Light"`
	Pincode     string `json:"pincode" validate:"required,len=6,numeric"`
}

// DelhiveryChargesRequest represents a request to get shipping charges
type DelhiveryChargesRequest struct {
	Mode           string  `json:"md" validate:"required,oneof=E S"`        // E=Express, S=Surface
	ServiceStatus  string  `json:"ss" validate:"required"`                  // Delivered/RTO/DTO
	DestinationPin string  `json:"d_pin" validate:"required,len=6,numeric"` // Destination PIN
	OriginPin      string  `json:"o_pin" validate:"required,len=6,numeric"` // Origin PIN
	ChargedWeight  int     `json:"cgm" validate:"required,min=1"`           // Weight in grams
	PaymentType    string  `json:"pt" validate:"required,oneof=Pre-paid COD"`
	CODAmount      float64 `json:"cod" validate:"min=0"`                    // COD amount in rupees
	DeclaredValue  int     `json:"dv,omitempty" validate:"omitempty,min=0"` // Optional declared value
}

// DelhiveryTATRequest represents a request to get estimated delivery time
type DelhiveryTATRequest struct {
	OriginPin          string `json:"origin_pin" validate:"required,len=6,numeric"`
	DestinationPin     string `json:"destination_pin" validate:"required,len=6,numeric"`
	ModeOfTransport    string `json:"mot" validate:"required,oneof=S A"` // S=Surface, A=Air
	ProductType        string `json:"pdt" validate:"required,oneof=B2C B2B"`
	ExpectedPickupDate string `json:"expected_pickup_date" validate:"required"` // YYYY-MM-DD
}

// DelhiveryShipmentRequest represents a request to create a shipment
type DelhiveryShipmentRequest struct {
	ShipmentDetails []DelhiveryShipmentDetail `json:"shipments"`
	PickupLocation  DelhiveryPickupLocation   `json:"pickup_location"`
}

// DelhiveryShipmentDetail represents individual shipment details
type DelhiveryShipmentDetail struct {
	Name          string `json:"name" validate:"required"`
	Add           string `json:"add" validate:"required"`
	Pin           string `json:"pin" validate:"required,len=6"`
	City          string `json:"city" validate:"required"`
	State         string `json:"state" validate:"required"`
	Country       string `json:"country" validate:"required"`
	Phone         string `json:"phone" validate:"required"`
	Order         string `json:"order" validate:"required"`        // Order ID
	PaymentMode   string `json:"payment_mode" validate:"required"` // Prepaid/COD
	ReturnPin     string `json:"return_pin" validate:"required,len=6"`
	ReturnCity    string `json:"return_city" validate:"required"`
	ReturnName    string `json:"return_name" validate:"required"`
	ReturnAdd     string `json:"return_add" validate:"required"`
	ReturnState   string `json:"return_state" validate:"required"`
	ReturnCountry string `json:"return_country" validate:"required"`
	ReturnPhone   string `json:"return_phone" validate:"required"`
	ProductsDesc  string `json:"products_desc" validate:"required"`
	HsnCode       string `json:"hsn_code"`
	CODAmount     string `json:"cod_amount,omitempty"`
	OrderDate     string `json:"order_date,omitempty"`
	TotalAmount   string `json:"total_amount" validate:"required"`
	SellerAdd     string `json:"seller_add,omitempty"`
	SellerName    string `json:"seller_name,omitempty"`
	SellerInv     string `json:"seller_inv,omitempty"`
	Quantity      string `json:"quantity" validate:"required"`
	Waybill       string `json:"waybill"` // Leave empty for auto-generation
	ShipWidth     string `json:"shipment_width"`
	ShipHeight    string `json:"shipment_height"`
	Weight        string `json:"weight" validate:"required"`
	ShippingMode  string `json:"shipping_mode" validate:"required"` // Surface/Express
	AddressType   string `json:"address_type"`                      // home/office
}

// DelhiveryPickupLocation represents pickup location details
type DelhiveryPickupLocation struct {
	Name string `json:"name" validate:"required"`
}

// BuildServiceabilityURL constructs the serviceability check URL with query parameters
func (r *DelhiveryServiceabilityRequest) BuildServiceabilityURL(baseURL string) string {
	params := url.Values{}
	params.Set("filter_codes", r.Pincode)

	return fmt.Sprintf("%s/c/api/pin-codes/json/?%s", baseURL, params.Encode())
}

// BuildChargesURL constructs the charges API URL with query parameters
func (r *DelhiveryChargesRequest) BuildChargesURL(baseURL string) string {
	params := url.Values{}
	params.Set("md", r.Mode)
	params.Set("ss", r.ServiceStatus)
	params.Set("d_pin", r.DestinationPin)
	params.Set("o_pin", r.OriginPin)
	// weight in grams as integer
	params.Set("cgm", strconv.Itoa(r.ChargedWeight))
	params.Set("pt", r.PaymentType)
	// Format COD and declared value as integers
	params.Set("cod", strconv.Itoa(int(r.CODAmount)))
	if r.DeclaredValue > 0 {
		params.Set("dv", strconv.Itoa(r.DeclaredValue))
	}

	return fmt.Sprintf("%s/api/kinko/v1/invoice/charges/.json?%s", baseURL, params.Encode())
}

// BuildTATURL constructs the TAT API URL with query parameters
func (r *DelhiveryTATRequest) BuildTATURL(baseURL string) string {
	params := url.Values{}
	params.Set("origin_pin", r.OriginPin)
	params.Set("destination_pin", r.DestinationPin)
	params.Set("mot", r.ModeOfTransport)
	params.Set("pdt", r.ProductType)
	params.Set("expected_pickup_date", r.ExpectedPickupDate)

	return fmt.Sprintf("%s/api/dc/expected_tat?%s", baseURL, params.Encode())
}

// NewServiceabilityRequest creates a new serviceability request with defaults
func NewServiceabilityRequest(pincode string) *DelhiveryServiceabilityRequest {
	return &DelhiveryServiceabilityRequest{
		ProductType: "Heavy",
		Pincode:     pincode,
	}
}

// NewChargesRequest creates a new charges request with common defaults - CORRECTED VERSION
func NewChargesRequest(originPin, destinationPin string, weightGrams int) *DelhiveryChargesRequest {
	if weightGrams < 1 {
		weightGrams = 1
	}
	return &DelhiveryChargesRequest{
		Mode:           "S",         // Surface by default
		ServiceStatus:  "Delivered", // Valid values: Delivered, RTO, DTO
		DestinationPin: destinationPin,
		OriginPin:      originPin,
		ChargedWeight:  weightGrams, // grams
		PaymentType:    "Pre-paid",
		CODAmount:      0,
		DeclaredValue:  0,
	}
}

// NewExpressChargesRequest creates a new express charges request
func NewExpressChargesRequest(originPin, destinationPin string, weightGrams int) *DelhiveryChargesRequest {
	req := NewChargesRequest(originPin, destinationPin, weightGrams)
	req.Mode = "E" // Express
	return req
}

// NewCODChargesRequest creates a new COD charges request
func NewCODChargesRequest(originPin, destinationPin string, weightGrams int, codAmount float64) *DelhiveryChargesRequest {
	req := NewChargesRequest(originPin, destinationPin, weightGrams)
	req.PaymentType = "COD"
	req.CODAmount = codAmount
	return req
}

// NewTATRequest creates a new TAT request with defaults
func NewTATRequest(originPin, destinationPin string) *DelhiveryTATRequest {
	return &DelhiveryTATRequest{
		OriginPin:          originPin,
		DestinationPin:     destinationPin,
		ModeOfTransport:    "S", // Surface by default
		ProductType:        "B2C",
		ExpectedPickupDate: time.Now().Format("2006-01-02 15:04"), // Fixed format: YYYY-MM-DD HH:MM
	}
}

// NewShipmentRequest creates a new shipment creation request
func NewShipmentRequest(warehouseName string) *DelhiveryShipmentRequest {
	return &DelhiveryShipmentRequest{
		ShipmentDetails: []DelhiveryShipmentDetail{},
		PickupLocation: DelhiveryPickupLocation{
			Name: warehouseName,
		},
	}
}

// AddShipmentDetail adds a shipment detail to the request
func (r *DelhiveryShipmentRequest) AddShipmentDetail(detail DelhiveryShipmentDetail) {
	r.ShipmentDetails = append(r.ShipmentDetails, detail)
}

// BuildShipmentURL returns the shipment creation endpoint
func (r *DelhiveryShipmentRequest) BuildShipmentURL(baseURL string) string {
	return fmt.Sprintf("%s/api/cmu/create.json", baseURL)
}

// Validation methods

// IsValid validates the serviceability request
func (r *DelhiveryServiceabilityRequest) IsValid() error {
	if r.Pincode == "" {
		return fmt.Errorf("pincode is required")
	}
	if len(r.Pincode) != 6 {
		return fmt.Errorf("pincode must be exactly 6 digits")
	}
	if r.ProductType != "Heavy" && r.ProductType != "Light" {
		return fmt.Errorf("product_type must be either 'Heavy' or 'Light'")
	}
	return nil
}

// IsValid validates the charges request - CORRECTED VERSION
func (r *DelhiveryChargesRequest) IsValid() error {
	if r.DestinationPin == "" || len(r.DestinationPin) != 6 {
		return fmt.Errorf("destination_pin must be exactly 6 digits")
	}
	if r.OriginPin == "" || len(r.OriginPin) != 6 {
		return fmt.Errorf("origin_pin must be exactly 6 digits")
	}
	if r.ChargedWeight <= 0 {
		return fmt.Errorf("charged_weight must be positive")
	}
	if r.Mode != "E" && r.Mode != "S" {
		return fmt.Errorf("mode must be either 'E' (Express) or 'S' (Surface)")
	}
	if r.PaymentType != "Pre-paid" && r.PaymentType != "COD" {
		return fmt.Errorf("payment_type must be either 'Pre-paid' or 'COD'")
	}
	if r.ServiceStatus != "Delivered" && r.ServiceStatus != "RTO" && r.ServiceStatus != "DTO" {
		return fmt.Errorf("service_status must be one of 'Delivered','RTO','DTO'")
	}
	return nil
}

// IsValid validates the TAT request
func (r *DelhiveryTATRequest) IsValid() error {
	if r.OriginPin == "" || len(r.OriginPin) != 6 {
		return fmt.Errorf("origin_pin must be exactly 6 digits")
	}
	if r.DestinationPin == "" || len(r.DestinationPin) != 6 {
		return fmt.Errorf("destination_pin must be exactly 6 digits")
	}
	if r.ModeOfTransport != "S" && r.ModeOfTransport != "A" {
		return fmt.Errorf("mode_of_transport must be either 'S' (Surface) or 'A' (Air)")
	}
	if r.ProductType != "B2C" && r.ProductType != "B2B" {
		return fmt.Errorf("product_type must be either 'B2C' or 'B2B'")
	}
	if r.ExpectedPickupDate == "" {
		return fmt.Errorf("expected_pickup_date is required")
	}
	return nil
}

// IsValid validates the shipment request
func (r *DelhiveryShipmentRequest) IsValid() error {
	if len(r.ShipmentDetails) == 0 {
		return fmt.Errorf("at least one shipment detail is required")
	}
	if r.PickupLocation.Name == "" {
		return fmt.Errorf("pickup location name is required")
	}
	for i, detail := range r.ShipmentDetails {
		if err := detail.IsValid(); err != nil {
			return fmt.Errorf("shipment detail %d: %w", i, err)
		}
	}
	return nil
}

// IsValid validates a shipment detail
func (r *DelhiveryShipmentDetail) IsValid() error {
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.Add == "" {
		return fmt.Errorf("address is required")
	}
	if len(r.Pin) != 6 {
		return fmt.Errorf("pin must be exactly 6 digits")
	}
	if r.Phone == "" {
		return fmt.Errorf("phone is required")
	}
	if r.Order == "" {
		return fmt.Errorf("order ID is required")
	}
	if r.PaymentMode != "Prepaid" && r.PaymentMode != "COD" {
		return fmt.Errorf("payment_mode must be either 'Prepaid' or 'COD'")
	}
	return nil
}
