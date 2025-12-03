package responses

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// StringOrInt is a JSON field that may come as string or number; we store it as string
type StringOrInt string

func (s *StringOrInt) UnmarshalJSON(b []byte) error {
	// Try string first
	var str string
	if err := json.Unmarshal(b, &str); err == nil {
		*s = StringOrInt(str)
		return nil
	}
	// Try int
	var i int
	if err := json.Unmarshal(b, &i); err == nil {
		*s = StringOrInt(strconv.Itoa(i))
		return nil
	}
	// Try float
	var f float64
	if err := json.Unmarshal(b, &f); err == nil {
		// If it is an integer-like float, format without decimal
		if f == math.Trunc(f) {
			*s = StringOrInt(strconv.FormatInt(int64(f), 10))
		} else {
			// Use 'g' to avoid trailing zeros and scientific notation for typical values
			*s = StringOrInt(strconv.FormatFloat(f, 'f', -1, 64))
		}
		return nil
	}
	return fmt.Errorf("StringOrInt: unsupported JSON: %s", string(b))
}

func (s StringOrInt) String() string { return string(s) }

// StringOrSlice handles fields that may be a single string or an array of strings
type StringOrSlice []string

func (s *StringOrSlice) UnmarshalJSON(b []byte) error {
	// Try single string
	var single string
	if err := json.Unmarshal(b, &single); err == nil {
		*s = []string{single}
		return nil
	}
	// Try array of strings
	var arr []string
	if err := json.Unmarshal(b, &arr); err == nil {
		*s = arr
		return nil
	}
	return fmt.Errorf("StringOrSlice: unsupported JSON: %s", string(b))
}

func (s StringOrSlice) String() string { return strings.Join([]string(s), "; ") }

// DelhiveryServiceabilityResponse represents Delhivery pincode serviceability check response
// This matches the production API format: /c/api/pin-codes/json/
type DelhiveryServiceabilityResponse struct {
	DeliveryCode []DelhiveryServiceabilityWrapper `json:"delivery_codes"`
}

// DelhiveryServiceabilityWrapper wraps the postal_code data
type DelhiveryServiceabilityWrapper struct {
	PostalCode DelhiveryServiceabilityData `json:"postal_code"`
}

// DelhiveryServiceabilityData represents individual pincode serviceability data
// This matches the production API structure
type DelhiveryServiceabilityData struct {
	Pin         StringOrInt `json:"pin"`
	City        string      `json:"city"`
	District    string      `json:"district"`
	State       string      `json:"state_code"`
	PrePaid     string      `json:"pre_paid"` // "Y" or "N"
	Cash        string      `json:"cash"`     // "Y" or "N"
	Pickup      string      `json:"pickup"`   // "Y" or "N"
	Repl        string      `json:"repl"`     // "Y" or "N"
	Cod         string      `json:"cod"`      // "Y" or "N"
	IsODA       string      `json:"is_oda"`   // "Y" or "N"
	CountryCode string      `json:"country_code"`
	Remarks     string      `json:"remarks"` // Empty = serviceable, "Embargo" = temporary NSZ
}

// Legacy struct for backwards compatibility
type DelhiveryServiceabilityCode struct {
	PostalCode DelhiveryPostalCode `json:"postal_code"`
	Cash       bool                `json:"cash"`     // COD availability
	PrePaid    bool                `json:"pre_paid"` // Prepaid availability
}

// DelhiveryPostalCode represents postal code information
type DelhiveryPostalCode struct {
	Pin          string `json:"pin"`
	DistrictName string `json:"district_name"`
	StateName    string `json:"state_name"`
}

// DelhiveryChargesResponse represents Delhivery charges API response
type DelhiveryChargesResponse []DelhiveryChargeDetail

// DelhiveryChargeDetail represents individual charge calculation
type DelhiveryChargeDetail struct {
	ActualWeight    float64 `json:"actual_weight"`
	BaseCost        float64 `json:"base_cost"`
	ChargedWeight   float64 `json:"charged_weight"`
	CodCharges      float64 `json:"cod_charges"`
	DaCharges       float64 `json:"da_charges"`
	DestinationPin  string  `json:"destination_pin"`
	FuelCharges     float64 `json:"fuel_charges"`
	HandlingCharges float64 `json:"handling_charges"`
	OtherCharges    float64 `json:"other_charges"`
	OriginPin       string  `json:"origin_pin"`
	RtoCharges      float64 `json:"rto_charges"`
	TotalAmount     float64 `json:"total_amount"`
	Zone            string  `json:"zone"`
	WeightSlab      string  `json:"weight_slab"`
	Status          string  `json:"status"`
	ChargeDL        float64 `json:"charge_DL"`
	GrossAmount     float64 `json:"gross_amount"`
}

// DelhiveryTATResponse represents Delhivery TAT (delivery time) API response
type DelhiveryTATResponse struct {
	Message string `json:"msg"`
	Data    struct {
		TAT                  int    `json:"tat"`
		ExpectedDeliveryDate string `json:"expected_delivery_date"`
	} `json:"data"`
	Success         bool   `json:"success"`
	OriginPin       string `json:"origin_pin,omitempty"`
	DestinationPin  string `json:"destination_pin,omitempty"`
	ModeOfTransport string `json:"mot,omitempty"`
}

// DelhiveryErrorResponse represents error response from Delhivery APIs
type DelhiveryErrorResponse struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`
	Status  string `json:"status,omitempty"`
}

// Helper methods for DelhiveryServiceabilityResponse

// IsServiceable checks if the pincode is serviceable based on the production API format
func (r *DelhiveryServiceabilityResponse) IsServiceable(pincode string) bool {
	// If API response is empty list, pincode is non-serviceable (NSZ)
	if len(r.DeliveryCode) == 0 {
		return false
	}

	for _, wrapper := range r.DeliveryCode {
		data := wrapper.PostalCode
		// Check if pincode matches
		if data.Pin.String() == pincode {
			// Check for embargo (temporary NSZ)
			if data.Remarks == "Embargo" {
				return false
			}

			// For prepaid only service, check if prepaid is available
			// Remarks should be blank/empty for serviceable pincode
			if data.PrePaid == "Y" && data.Remarks == "" {
				return true
			}
		}
	}
	return false
}

// GetServiceabilityInfo returns serviceability details for a specific pincode
func (r *DelhiveryServiceabilityResponse) GetServiceabilityInfo(pincode string) *DelhiveryServiceabilityData {
	for _, wrapper := range r.DeliveryCode {
		data := wrapper.PostalCode
		if data.Pin.String() == pincode {
			return &data
		}
	}
	return nil
}

// SupportsCOD checks if COD is available for the pincode
func (r *DelhiveryServiceabilityResponse) SupportsCOD(pincode string) bool {
	info := r.GetServiceabilityInfo(pincode)
	if info == nil {
		return false
	}
	// Check if COD/Cash is available
	return info.Cash == "Y" || info.Cod == "Y"
}

// SupportsPrePaid checks if prepaid is available for the pincode
func (r *DelhiveryServiceabilityResponse) SupportsPrePaid(pincode string) bool {
	info := r.GetServiceabilityInfo(pincode)
	if info == nil {
		return false
	}
	// Check if PrePaid is available
	return info.PrePaid == "Y"
}

// Helper methods for DelhiveryChargesResponse

// GetTotalAmount returns the total shipping amount as float64
func (detail *DelhiveryChargeDetail) GetTotalAmount() (float64, error) {
	return detail.TotalAmount, nil
}

// GetBaseCost returns the base cost as float64
func (detail *DelhiveryChargeDetail) GetBaseCost() (float64, error) {
	return detail.BaseCost, nil
}

// GetFuelCharges returns the fuel charges as float64
func (detail *DelhiveryChargeDetail) GetFuelCharges() (float64, error) {
	return detail.FuelCharges, nil
}

// GetCODCharges returns the COD charges as float64
func (detail *DelhiveryChargeDetail) GetCODCharges() (float64, error) {
	return detail.CodCharges, nil
}

// GetChargedWeight returns the charged weight as float64
func (detail *DelhiveryChargeDetail) GetChargedWeight() (float64, error) {
	return detail.ChargedWeight, nil
}

// GetActualWeight returns the actual weight as float64
func (detail *DelhiveryChargeDetail) GetActualWeight() (float64, error) {
	return detail.ActualWeight, nil
}

// GetFirstChargeDetail returns the first (and typically only) charge detail
func (r *DelhiveryChargesResponse) GetFirstChargeDetail() *DelhiveryChargeDetail {
	if len(*r) > 0 {
		return &(*r)[0]
	}
	return nil
}

// GetTotalShippingCost returns the total shipping cost from the first charge detail
func (r *DelhiveryChargesResponse) GetTotalShippingCost() (float64, error) {
	detail := r.GetFirstChargeDetail()
	if detail == nil {
		return 0, fmt.Errorf("no charge details available")
	}
	return detail.GetTotalAmount()
}

// Helper methods for DelhiveryTATResponse

// GetExpectedDeliveryTime parses and returns the expected delivery date
func (r *DelhiveryTATResponse) GetExpectedDeliveryTime() (*time.Time, error) {
	if r.Data.ExpectedDeliveryDate == "" {
		return nil, fmt.Errorf("expected delivery date not provided")
	}

	// Parse the date (assuming YYYY-MM-DD format)
	t, err := time.Parse("2006-01-02", r.Data.ExpectedDeliveryDate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse delivery date: %w", err)
	}

	return &t, nil
}

// IsValidTAT checks if the TAT response contains valid delivery information
func (r *DelhiveryTATResponse) IsValidTAT() bool {
	return r.Success && r.Data.TAT > 0 && r.Data.ExpectedDeliveryDate != ""
}

// Unified response wrapper for all Delhivery API responses

// DelhiveryAPIResponse represents a unified response wrapper
type DelhiveryAPIResponse struct {
	Success        bool                             `json:"success"`
	StatusCode     int                              `json:"status_code"`
	Serviceability *DelhiveryServiceabilityResponse `json:"serviceability,omitempty"`
	Charges        *DelhiveryChargesResponse        `json:"charges,omitempty"`
	TAT            *DelhiveryTATResponse            `json:"tat,omitempty"`
	Error          *DelhiveryErrorResponse          `json:"error,omitempty"`
}

// IsSuccess checks if the API response indicates success
func (r *DelhiveryAPIResponse) IsSuccess() bool {
	return r.Success && r.StatusCode >= 200 && r.StatusCode < 300
}

// HasError checks if there's an error in the response
func (r *DelhiveryAPIResponse) HasError() bool {
	return r.Error != nil || (!r.Success) || r.StatusCode >= 400
}

// GetErrorMessage returns the error message if available
func (r *DelhiveryAPIResponse) GetErrorMessage() string {
	if r.Error != nil {
		if r.Error.Message != "" {
			return r.Error.Message
		}
		if r.Error.Error != "" {
			return r.Error.Error
		}
		if r.Error.Details != "" {
			return r.Error.Details
		}
	}
	if !r.Success {
		return fmt.Sprintf("API call failed with status code: %d", r.StatusCode)
	}
	return ""
}

// Factory functions for creating responses

// NewServiceabilityResponse creates a successful serviceability response
func NewServiceabilityResponse(data *DelhiveryServiceabilityResponse) *DelhiveryAPIResponse {
	return &DelhiveryAPIResponse{
		Success:        true,
		StatusCode:     200,
		Serviceability: data,
	}
}

// NewChargesResponse creates a successful charges response
func NewChargesResponse(data *DelhiveryChargesResponse) *DelhiveryAPIResponse {
	return &DelhiveryAPIResponse{
		Success:    true,
		StatusCode: 200,
		Charges:    data,
	}
}

// NewTATResponse creates a successful TAT response
func NewTATResponse(data *DelhiveryTATResponse) *DelhiveryAPIResponse {
	return &DelhiveryAPIResponse{
		Success:    true,
		StatusCode: 200,
		TAT:        data,
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(statusCode int, message string) *DelhiveryAPIResponse {
	return &DelhiveryAPIResponse{
		Success:    false,
		StatusCode: statusCode,
		Error: &DelhiveryErrorResponse{
			Message: message,
			Status:  strconv.Itoa(statusCode),
		},
	}
}

// ParseServiceabilityResponse parses raw JSON response into serviceability response
func ParseServiceabilityResponse(data []byte) (*DelhiveryServiceabilityResponse, error) {
	var response DelhiveryServiceabilityResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse serviceability response: %w", err)
	}
	return &response, nil
}

// ParseChargesResponse parses raw JSON response into charges response
func ParseChargesResponse(data []byte) (*DelhiveryChargesResponse, error) {
	var response DelhiveryChargesResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse charges response: %w", err)
	}
	return &response, nil
}

// ParseTATResponse parses raw JSON response into TAT response
func ParseTATResponse(data []byte) (*DelhiveryTATResponse, error) {
	var response DelhiveryTATResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse TAT response: %w", err)
	}
	return &response, nil
}

// DelhiveryShipmentResponse represents response from shipment creation
type DelhiveryShipmentResponse struct {
	PackageData []DelhiveryPackageInfo `json:"packages"`
	CashTotal   float64                `json:"cash_total"`
	CODTotal    float64                `json:"cod_total"`
	Success     bool                   `json:"success"`
	Prepaid     float64                `json:"prepaid"`
	RMK         string                 `json:"rmk"`
}

// Packages returns the package data (alias for compatibility)
func (r *DelhiveryShipmentResponse) Packages() []DelhiveryPackageInfo {
	return r.PackageData
}

// IsSuccessful checks if the shipment creation was successful
func (r *DelhiveryShipmentResponse) IsSuccessful() bool {
	return r.Success && len(r.PackageData) > 0 && r.GetWaybill() != ""
}

// GetErrorMessage returns error message from RMK field or a generic message
func (r *DelhiveryShipmentResponse) GetErrorMessage() string {
	if r.RMK != "" {
		return r.RMK
	}
	if !r.Success {
		return "Shipment creation failed - no success status from Delhivery"
	}
	if len(r.PackageData) == 0 {
		return "Shipment creation failed - no package data received"
	}
	if r.GetWaybill() == "" {
		return "Shipment creation failed - no waybill number received"
	}
	return "Unknown error occurred during shipment creation"
}

// DelhiveryPackageInfo represents individual package information in response
type DelhiveryPackageInfo struct {
	Status      string        `json:"status"`
	Client      string        `json:"client"`
	SortCode    string        `json:"sort_code"`
	Remarks     StringOrSlice `json:"remarks"`
	Waybill     string        `json:"waybill"`
	RefNum      string        `json:"refnum"`
	CODAmount   StringOrInt   `json:"cod_amount"`
	PaymentMode string        `json:"payment_mode"`
}

// GetWaybill extracts the first waybill number from the response
func (r *DelhiveryShipmentResponse) GetWaybill() string {
	if len(r.PackageData) > 0 {
		return r.PackageData[0].Waybill
	}
	return ""
}

// GetStatus extracts the first package status from the response
func (r *DelhiveryShipmentResponse) GetStatus() string {
	if len(r.PackageData) > 0 {
		return r.PackageData[0].Status
	}
	return ""
}

// GetTrackingURL generates Delhivery tracking URL
func (r *DelhiveryShipmentResponse) GetTrackingURL() string {
	waybill := r.GetWaybill()
	if waybill == "" {
		return ""
	}
	// Use B2C tracking URL pattern
	return fmt.Sprintf("https://www.delhivery.com/track-v2/package/%s", waybill)
}

// ParseShipmentResponse parses raw JSON response into shipment response
func ParseShipmentResponse(data []byte) (*DelhiveryShipmentResponse, error) {
	var response DelhiveryShipmentResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse shipment response: %w", err)
	}
	return &response, nil
}

// DelhiveryTrackingResponse represents the tracking API response from Delhivery
type DelhiveryTrackingResponse struct {
	ShipmentData []DelhiveryShipmentTracking `json:"ShipmentData"`
}

// DelhiveryShipmentTracking represents tracking data for a single shipment
type DelhiveryShipmentTracking struct {
	Shipment DelhiveryShipmentDetail `json:"Shipment"`
}

// DelhiveryShipmentDetail contains detailed tracking information
type DelhiveryShipmentDetail struct {
	AWB               string                    `json:"AWB"`
	Status            DelhiveryTrackingStatus   `json:"Status"`
	Scans             []DelhiveryTrackingScan   `json:"Scans"`
	PickUpDate        string                    `json:"PickUpDate"`
	DeliveryDate      string                    `json:"DeliveryDate"`
	ExpectedDelivery  string                    `json:"ExpectedDelivery"`
	DestRecieveDate   string                    `json:"DestRecieveDate"`
	Origin            string                    `json:"Origin"`
	Destination       string                    `json:"Destination"`
	Consignee         DelhiveryConsignee        `json:"Consignee"`
	ReferenceNo       string                    `json:"ReferenceNo"`
	OrderType         string                    `json:"OrderType"`
	ReturnedDate      string                    `json:"ReturnedDate"`
	SenderName        string                    `json:"SenderName"`
	ChargedWeight     interface{}               `json:"ChargedWeight"` // Can be string or float
}

// DelhiveryTrackingStatus contains the current status of the shipment
type DelhiveryTrackingStatus struct {
	Status          string `json:"Status"`
	StatusDateTime  string `json:"StatusDateTime"`
	StatusLocation  string `json:"StatusLocation"`
	StatusCode      string `json:"StatusCode"`
	StatusType      string `json:"StatusType"`
	RecievedBy      string `json:"RecievedBy"`
	Instructions    string `json:"Instructions"`
}

// DelhiveryTrackingScan represents a single scan/event in the tracking history
type DelhiveryTrackingScan struct {
	ScanDetail DelhiveryScanDetail `json:"ScanDetail"`
}

// DelhiveryScanDetail contains details of a tracking scan
type DelhiveryScanDetail struct {
	Scan              string `json:"Scan"`
	ScanDateTime      string `json:"ScanDateTime"`
	ScanType          string `json:"ScanType"`
	ScannedLocation   string `json:"ScannedLocation"`
	StatusDateTime    string `json:"StatusDateTime"`
	StatusCode        string `json:"StatusCode"`
	Instructions      string `json:"Instructions"`
}

// DelhiveryConsignee contains recipient information
type DelhiveryConsignee struct {
	Name    string `json:"Name"`
	City    string `json:"City"`
	State   string `json:"State"`
	PinCode string `json:"PinCode"`
	Address string `json:"Address1"`
	Phone   string `json:"Telephone1"`
}

// ParseTrackingResponse parses raw JSON response into tracking response
func ParseTrackingResponse(data []byte) (*DelhiveryTrackingResponse, error) {
	var response DelhiveryTrackingResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse tracking response: %w", err)
	}
	return &response, nil
}

// GetShipmentTracking returns the tracking data for the first (or only) shipment
func (r *DelhiveryTrackingResponse) GetShipmentTracking() *DelhiveryShipmentDetail {
	if len(r.ShipmentData) > 0 {
		return &r.ShipmentData[0].Shipment
	}
	return nil
}

// GetCurrentStatus returns the current status of the shipment
func (r *DelhiveryTrackingResponse) GetCurrentStatus() string {
	shipment := r.GetShipmentTracking()
	if shipment != nil {
		return shipment.Status.Status
	}
	return ""
}

// GetStatusCode returns the current status code
func (r *DelhiveryTrackingResponse) GetStatusCode() string {
	shipment := r.GetShipmentTracking()
	if shipment != nil {
		return shipment.Status.StatusCode
	}
	return ""
}

// GetLastUpdate returns the last status update time
func (r *DelhiveryTrackingResponse) GetLastUpdate() string {
	shipment := r.GetShipmentTracking()
	if shipment != nil {
		return shipment.Status.StatusDateTime
	}
	return ""
}

// GetExpectedDelivery returns the expected delivery date
func (r *DelhiveryTrackingResponse) GetExpectedDelivery() string {
	shipment := r.GetShipmentTracking()
	if shipment != nil {
		return shipment.ExpectedDelivery
	}
	return ""
}

// GetTrackingHistory returns all tracking scans/events
func (r *DelhiveryTrackingResponse) GetTrackingHistory() []DelhiveryScanDetail {
	shipment := r.GetShipmentTracking()
	if shipment == nil || len(shipment.Scans) == 0 {
		return nil
	}

	history := make([]DelhiveryScanDetail, len(shipment.Scans))
	for i, scan := range shipment.Scans {
		history[i] = scan.ScanDetail
	}
	return history
}

// IsDelivered checks if the shipment has been delivered
func (r *DelhiveryTrackingResponse) IsDelivered() bool {
	statusCode := r.GetStatusCode()
	// Delhivery status codes: DL = Delivered
	return statusCode == "DL" || statusCode == "DELIVERED"
}

// IsInTransit checks if the shipment is in transit
func (r *DelhiveryTrackingResponse) IsInTransit() bool {
	statusCode := r.GetStatusCode()
	// Common in-transit status codes
	inTransitCodes := []string{"IT", "OT", "RT", "CC", "ST", "UD", "OC", "RAD", "OFD", "PP", "OP"}
	for _, code := range inTransitCodes {
		if statusCode == code {
			return true
		}
	}
	return false
}
