package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// CreateOrderResponse represents the response when creating a new order
type CreateOrderResponse struct {
	Message         string    `json:"message"`
	OrderID         string    `json:"order_id"`
	CustomerID      string    `json:"customer_id"`
	AddressID       string    `json:"address_id"`
	TotalAmount     float64   `json:"total_amount"`
	Status          string    `json:"status"`
	PaymentMethod   string    `json:"payment_method"`
	PaymentStatus   string    `json:"payment_status"`
	ShippingCharges float64   `json:"shipping_charges"`
	CreatedAt       time.Time `json:"created_at"`
}

// GetUserOrdersResponse represents the response when retrieving user orders
type GetUserOrdersResponse struct {
	Orders     []models.Order `json:"orders"`
	Pagination PaginationInfo `json:"pagination"`
}

// PaginationInfo represents pagination metadata
type PaginationInfo struct {
	Page    int  `json:"page"`
	Limit   int  `json:"limit"`
	Total   int  `json:"total"`
	HasMore bool `json:"has_more"`
}

// GetOrderDetailsResponse represents the response when retrieving order details
type GetOrderDetailsResponse struct {
	Order models.Order `json:"order"`
}

// GetOrderStatusResponse represents the response when retrieving order status
type GetOrderStatusResponse struct {
	OrderID       string    `json:"order_id"`
	Status        string    `json:"status"`
	PaymentStatus string    `json:"payment_status"`
	TotalAmount   float64   `json:"total_amount"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// GetOrderTrackingResponse represents the response when retrieving order tracking
type GetOrderTrackingResponse struct {
	OrderID     string `json:"order_id"`
	Waybill     string `json:"waybill"`
	Status      string `json:"status"`
	TrackingURL string `json:"tracking_url"`
	Carrier     string `json:"carrier"`
	Mode        string `json:"mode"`
	EtaMinDays  int    `json:"eta_min_days"`
	EtaMaxDays  int    `json:"eta_max_days"`
}

// GetOrderDetailsDebugResponse represents the response for debug order details
type GetOrderDetailsDebugResponse struct {
	OrderID         string    `json:"order_id"`
	UserID          string    `json:"user_id"`
	Status          string    `json:"status"`
	PaymentStatus   string    `json:"payment_status"`
	TotalAmount     float64   `json:"total_amount"`
	ShippingAddress string    `json:"shipping_address"`
	Waybill         string    `json:"waybill"`
	TrackingURL     string    `json:"tracking_url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse represents a standard success response
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewCreateOrderResponse creates a new CreateOrderResponse from an order
func NewCreateOrderResponse(order *models.Order) *CreateOrderResponse {
	return &CreateOrderResponse{
		Message:         "Order created successfully",
		OrderID:         order.ID,
		CustomerID:      order.CustomerID,
		AddressID:       order.ShippingAddressID,
		TotalAmount:     order.TotalAmount,
		Status:          string(order.Status),
		PaymentMethod:   order.PaymentMethod,
		PaymentStatus:   order.PaymentStatus,
		ShippingCharges: order.ShippingCharge,
		CreatedAt:       order.CreatedAt,
	}
}

// NewGetUserOrdersResponse creates a new GetUserOrdersResponse
func NewGetUserOrdersResponse(orders []models.Order, page, limit int) *GetUserOrdersResponse {
	total := len(orders)
	hasMore := (page * limit) < total

	return &GetUserOrdersResponse{
		Orders: orders,
		Pagination: PaginationInfo{
			Page:    page,
			Limit:   limit,
			Total:   total,
			HasMore: hasMore,
		},
	}
}

// NewGetOrderDetailsResponse creates a new GetOrderDetailsResponse
func NewGetOrderDetailsResponse(order *models.Order) *GetOrderDetailsResponse {
	return &GetOrderDetailsResponse{
		Order: *order,
	}
}

// NewGetOrderStatusResponse creates a new GetOrderStatusResponse
func NewGetOrderStatusResponse(order *models.Order) *GetOrderStatusResponse {
	return &GetOrderStatusResponse{
		OrderID:       order.ID,
		Status:        string(order.Status),
		PaymentStatus: order.PaymentStatus,
		TotalAmount:   order.TotalAmount,
		CreatedAt:     order.CreatedAt,
		UpdatedAt:     order.UpdatedAt,
	}
}

// NewGetOrderTrackingResponse creates a new GetOrderTrackingResponse
func NewGetOrderTrackingResponse(order *models.Order) *GetOrderTrackingResponse {
	return &GetOrderTrackingResponse{
		OrderID:     order.ID,
		Waybill:     order.DelhiveryWaybill,
		Status:      order.DelhiveryStatus,
		TrackingURL: order.TrackingURL,
		Carrier:     order.ShippingCarrier,
		Mode:        order.ShippingMode,
		EtaMinDays:  order.EtaMinDays,
		EtaMaxDays:  order.EtaMaxDays,
	}
}

// NewGetOrderDetailsDebugResponse creates a new GetOrderDetailsDebugResponse
func NewGetOrderDetailsDebugResponse(order *models.Order) *GetOrderDetailsDebugResponse {
	return &GetOrderDetailsDebugResponse{
		OrderID:         order.ID,
		UserID:          order.UserID,
		Status:          string(order.Status),
		PaymentStatus:   order.PaymentStatus,
		TotalAmount:     order.TotalAmount,
		ShippingAddress: order.ShippingAddr,
		Waybill:         order.DelhiveryWaybill,
		TrackingURL:     order.TrackingURL,
		CreatedAt:       order.CreatedAt,
		UpdatedAt:       order.UpdatedAt,
	}
}
