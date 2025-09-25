package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
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

// MinimalProductInfo represents minimal product information for order items
type MinimalProductInfo struct {
	Name     string `json:"name"`
	ImageURL string `json:"image_url"`
}

// OrderItemWithMinimalProduct represents an order item with minimal product information
type OrderItemWithMinimalProduct struct {
	*base.BaseModel `json:",inline"`
	OrderID         string              `json:"order_id"`
	ProductID       string              `json:"product_id"`
	Quantity        int                 `json:"quantity"`
	UnitPrice       float64             `json:"unit_price"`
	TotalPrice      float64             `json:"total_price"`
	Product         *MinimalProductInfo `json:"product,omitempty"`
}

// OrderWithMinimalProductInfo represents an order with minimal product information in items
type OrderWithMinimalProductInfo struct {
	*base.BaseModel     `json:",inline"`
	UserID              string                        `json:"user_id"`
	TotalAmount         float64                       `json:"total_amount"`
	Status              string                        `json:"status"`
	CustomerID          string                        `json:"customer_id"`
	ShippingAddressID   string                        `json:"shipping_address_id"`
	ShippingAddr        string                        `json:"shipping_address"`
	ShippingAddrID      string                        `json:"shipping_address_id_legacy"`
	CustomerFirstName   string                        `json:"customer_first_name"`
	CustomerLastName    string                        `json:"customer_last_name"`
	CustomerPhone       string                        `json:"customer_phone"`
	CustomerPhoneAlt    string                        `json:"customer_phone_alt"`
	PaymentMethod       string                        `json:"payment_method"`
	PaymentStatus       string                        `json:"payment_status"`
	Notes               string                        `json:"notes"`
	RazorpayOrderID     string                        `json:"razorpay_order_id"`
	RazorpayReceipt     string                        `json:"razorpay_receipt"`
	RazorpayAmountPaisa int                           `json:"razorpay_amount_paisa"`
	RazorpayNotesJSON   string                        `json:"razorpay_notes_json"`
	ShippingCarrier     string                        `json:"shipping_carrier"`
	ShippingMode        string                        `json:"shipping_mode"`
	ShippingCharge      float64                       `json:"shipping_charge"`
	EtaMinDays          int                           `json:"eta_min_days"`
	EtaMaxDays          int                           `json:"eta_max_days"`
	DelhiveryWaybill    string                        `json:"delhivery_waybill"`
	DelhiveryStatus     string                        `json:"delhivery_status"`
	TrackingURL         string                        `json:"tracking_url"`
	InvoiceURL          string                        `json:"invoice_url"`
	CouponCode          string                        `json:"coupon_code"`
	DiscountAmount      float64                       `json:"discount_amount"`
	Customer            *models.OrderCustomer         `json:"customer,omitempty"`
	ShippingAddress     *models.OrderAddress          `json:"shipping_address_normalized,omitempty"`
	OrderItems          []OrderItemWithMinimalProduct `json:"order_items,omitempty"`
}

// GetOrderDetailsResponse represents the response when retrieving order details
type GetOrderDetailsResponse struct {
	Order OrderWithMinimalProductInfo `json:"order"`
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

// NewGetOrderDetailsResponse creates a new GetOrderDetailsResponse with minimal product information
func NewGetOrderDetailsResponse(order *models.Order) *GetOrderDetailsResponse {
	// Convert order items to minimal product format
	var minimalOrderItems []OrderItemWithMinimalProduct
	for _, item := range order.OrderItems {
		minimalItem := OrderItemWithMinimalProduct{
			BaseModel:  item.BaseModel,
			OrderID:    item.OrderID,
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			UnitPrice:  item.UnitPrice,
			TotalPrice: item.TotalPrice,
		}

		// Add minimal product information if available
		if item.Product != nil {
			minimalItem.Product = &MinimalProductInfo{
				Name:     item.Product.Name,
				ImageURL: item.Product.ImageURL,
			}
		}

		minimalOrderItems = append(minimalOrderItems, minimalItem)
	}

	// Create order with minimal product information
	minimalOrder := OrderWithMinimalProductInfo{
		BaseModel:           order.BaseModel,
		UserID:              order.UserID,
		TotalAmount:         order.TotalAmount,
		Status:              string(order.Status),
		CustomerID:          order.CustomerID,
		ShippingAddressID:   order.ShippingAddressID,
		ShippingAddr:        order.ShippingAddr,
		ShippingAddrID:      order.ShippingAddrID,
		CustomerFirstName:   order.CustomerFirstName,
		CustomerLastName:    order.CustomerLastName,
		CustomerPhone:       order.CustomerPhone,
		CustomerPhoneAlt:    order.CustomerPhoneAlt,
		PaymentMethod:       order.PaymentMethod,
		PaymentStatus:       order.PaymentStatus,
		Notes:               order.Notes,
		RazorpayOrderID:     order.RazorpayOrderID,
		RazorpayReceipt:     order.RazorpayReceipt,
		RazorpayAmountPaisa: order.RazorpayAmountPaisa,
		RazorpayNotesJSON:   order.RazorpayNotesJSON,
		ShippingCarrier:     order.ShippingCarrier,
		ShippingMode:        order.ShippingMode,
		ShippingCharge:      order.ShippingCharge,
		EtaMinDays:          order.EtaMinDays,
		EtaMaxDays:          order.EtaMaxDays,
		DelhiveryWaybill:    order.DelhiveryWaybill,
		DelhiveryStatus:     order.DelhiveryStatus,
		TrackingURL:         order.TrackingURL,
		InvoiceURL:          order.InvoiceURL,
		CouponCode:          order.CouponCode,
		DiscountAmount:      order.DiscountAmount,
		Customer:            order.Customer,
		ShippingAddress:     order.ShippingAddress,
		OrderItems:          minimalOrderItems,
	}

	return &GetOrderDetailsResponse{
		Order: minimalOrder,
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
