package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
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

// SimplifiedOrderItem represents essential order item info for list view
type SimplifiedOrderItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	ImageURL    string  `json:"image_url"`
	Quantity    int     `json:"quantity"`
	UnitPrice   float64 `json:"unit_price"`
	TotalPrice  float64 `json:"total_price"`
}

// SimplifiedOrder represents essential order info for list view
type SimplifiedOrder struct {
	ID            string                `json:"id"`
	OrderID       string                `json:"order_id"` // Human-readable order ID
	Status        string                `json:"status"`
	PaymentStatus string                `json:"payment_status"`
	TotalAmount   float64               `json:"total_amount"`
	ItemCount     int                   `json:"item_count"`
	Items         []SimplifiedOrderItem `json:"items"` // First item or summary
	CreatedAt     time.Time             `json:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at"`
}

// GetUserOrdersResponse represents the response when retrieving user orders
type GetUserOrdersResponse struct {
	Orders     []SimplifiedOrder `json:"orders"`
	Pagination PaginationInfo    `json:"pagination"`
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

// EnhancedProductInfo represents enhanced product information for order details
type EnhancedProductInfo struct {
	Name        string   `json:"name"`
	ImageURL    string   `json:"image_url"`  // Primary image
	ImageURLs   []string `json:"image_urls"` // All product images
	Description string   `json:"description,omitempty"`
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

// OrderItemWithEnhancedProduct represents an order item with enhanced product information
type OrderItemWithEnhancedProduct struct {
	*base.BaseModel `json:",inline"`
	OrderID         string               `json:"order_id"`
	ProductID       string               `json:"product_id"`
	Quantity        int                  `json:"quantity"`
	UnitPrice       float64              `json:"unit_price"`
	TotalPrice      float64              `json:"total_price"`
	Product         *EnhancedProductInfo `json:"product,omitempty"`
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

// OrderWithEnhancedProductInfo represents an order with enhanced product information and all details
type OrderWithEnhancedProductInfo struct {
	*base.BaseModel     `json:",inline"`
	UserID              string                         `json:"user_id"`
	TotalAmount         float64                        `json:"total_amount"`
	Status              string                         `json:"status"`
	CustomerID          string                         `json:"customer_id"`
	ShippingAddressID   string                         `json:"shipping_address_id"`
	ShippingAddr        string                         `json:"shipping_address"`
	ShippingAddrID      string                         `json:"shipping_address_id_legacy"`
	CustomerFirstName   string                         `json:"customer_first_name"`
	CustomerLastName    string                         `json:"customer_last_name"`
	CustomerPhone       string                         `json:"customer_phone"`
	CustomerPhoneAlt    string                         `json:"customer_phone_alt"`
	PaymentMethod       string                         `json:"payment_method"`
	PaymentStatus       string                         `json:"payment_status"`
	Notes               string                         `json:"notes"`
	RazorpayOrderID     string                         `json:"razorpay_order_id"`
	RazorpayReceipt     string                         `json:"razorpay_receipt"`
	RazorpayAmountPaisa int                            `json:"razorpay_amount_paisa"`
	RazorpayNotesJSON   string                         `json:"razorpay_notes_json"`
	ShippingCarrier     string                         `json:"shipping_carrier"`
	ShippingMode        string                         `json:"shipping_mode"`
	ShippingCharge      float64                        `json:"shipping_charge"`
	EtaMinDays          int                            `json:"eta_min_days"`
	EtaMaxDays          int                            `json:"eta_max_days"`
	DelhiveryWaybill    string                         `json:"delhivery_waybill"`
	DelhiveryStatus     string                         `json:"delhivery_status"`
	TrackingURL         string                         `json:"tracking_url"`
	InvoiceURL          string                         `json:"invoice_url"`
	CouponCode          string                         `json:"coupon_code"`
	DiscountAmount      float64                        `json:"discount_amount"`
	Customer            *models.OrderCustomer          `json:"customer,omitempty"`
	ShippingAddress     *models.OrderAddress           `json:"shipping_address_normalized,omitempty"`
	OrderItems          []OrderItemWithEnhancedProduct `json:"order_items,omitempty"`
	// Payment details
	PaymentDetails map[string]interface{} `json:"payment_details,omitempty"`
	// Tracking details
	TrackingDetails map[string]interface{} `json:"tracking_details,omitempty"`
	// Order summary
	Subtotal   float64 `json:"subtotal"`
	Discount   float64 `json:"discount"`
	Shipping   float64 `json:"shipping"`
	Tax        float64 `json:"tax"`
	GrandTotal float64 `json:"grand_total"`
}

// GetOrderDetailsResponse represents the response when retrieving order details
type GetOrderDetailsResponse struct {
	Order OrderWithMinimalProductInfo `json:"order"`
}

// GetOrderDetailsResponseEnhanced represents the enhanced response with all product images
type GetOrderDetailsResponseEnhanced struct {
	Order OrderWithEnhancedProductInfo `json:"order"`
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

// NewGetUserOrdersResponse creates a simplified GetUserOrdersResponse with only essential info
func NewGetUserOrdersResponse(orders []models.Order, page, limit int) *GetUserOrdersResponse {
	total := len(orders)
	hasMore := (page * limit) < total

	// Convert to simplified orders
	simplifiedOrders := make([]SimplifiedOrder, 0, len(orders))
	for _, order := range orders {
		// Get first item or create summary
		var items []SimplifiedOrderItem
		if len(order.OrderItems) > 0 {
			// Show first item with image, or summary if multiple items
			for i, item := range order.OrderItems {
				if i >= 3 { // Limit to 3 items for list view
					break
				}
				simplifiedItem := SimplifiedOrderItem{
					ProductID:  item.ProductID,
					Quantity:   item.Quantity,
					UnitPrice:  item.UnitPrice,
					TotalPrice: item.TotalPrice,
				}
				if item.Product != nil {
					simplifiedItem.ProductName = item.Product.Name
					simplifiedItem.ImageURL = item.Product.ImageURL
				}
				items = append(items, simplifiedItem)
			}
		}

		simplifiedOrder := SimplifiedOrder{
			ID:            order.ID,
			OrderID:       order.ID, // Can be formatted as ORDER-XXX if needed
			Status:        string(order.Status),
			PaymentStatus: order.PaymentStatus,
			TotalAmount:   order.TotalAmount,
			ItemCount:     len(order.OrderItems),
			Items:         items,
			CreatedAt:     order.CreatedAt,
			UpdatedAt:     order.UpdatedAt,
		}
		simplifiedOrders = append(simplifiedOrders, simplifiedOrder)
	}

	return &GetUserOrdersResponse{
		Orders: simplifiedOrders,
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

// NewGetOrderDetailsResponseEnhanced creates an enhanced GetOrderDetailsResponse with all product images and details
func NewGetOrderDetailsResponseEnhanced(order *models.Order, enrichedItems []repositories.OrderItemWithProduct) *GetOrderDetailsResponseEnhanced {
	// Convert enriched items to enhanced product format
	var enhancedOrderItems []OrderItemWithEnhancedProduct
	for _, enrichedItem := range enrichedItems {
		enhancedItem := OrderItemWithEnhancedProduct{
			BaseModel:  enrichedItem.OrderItem.BaseModel,
			OrderID:    enrichedItem.OrderID,
			ProductID:  enrichedItem.ProductID,
			Quantity:   enrichedItem.Quantity,
			UnitPrice:  enrichedItem.UnitPrice,
			TotalPrice: enrichedItem.TotalPrice,
		}

		// Add enhanced product information with all images
		if enrichedItem.ProductName != "" || len(enrichedItem.ProductImageURLs) > 0 {
			enhancedItem.Product = &EnhancedProductInfo{
				Name:        enrichedItem.ProductName,
				ImageURL:    enrichedItem.ProductImageURL,  // Primary image
				ImageURLs:   enrichedItem.ProductImageURLs, // All images
				Description: enrichedItem.ProductDescription,
			}
		}

		enhancedOrderItems = append(enhancedOrderItems, enhancedItem)
	}

	// Calculate order summary
	subtotal := order.TotalAmount
	discount := order.DiscountAmount
	shipping := order.ShippingCharge
	tax := 0.0 // Tax is included in total or can be calculated
	grandTotal := order.TotalAmount

	// Build payment details
	paymentDetails := map[string]interface{}{
		"method":           order.PaymentMethod,
		"status":           order.PaymentStatus,
		"razorpay_id":      order.RazorpayOrderID,
		"razorpay_receipt": order.RazorpayReceipt,
	}
	if order.RazorpayOrderID != "" {
		paymentDetails["razorpay_order_id"] = order.RazorpayOrderID
	}

	// Build tracking details
	trackingDetails := map[string]interface{}{
		"carrier":      order.ShippingCarrier,
		"mode":         order.ShippingMode,
		"waybill":      order.DelhiveryWaybill,
		"status":       order.DelhiveryStatus,
		"tracking_url": order.TrackingURL,
		"eta_min_days": order.EtaMinDays,
		"eta_max_days": order.EtaMaxDays,
	}

	// Create enhanced order with all details
	enhancedOrder := OrderWithEnhancedProductInfo{
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
		OrderItems:          enhancedOrderItems,
		PaymentDetails:      paymentDetails,
		TrackingDetails:     trackingDetails,
		Subtotal:            subtotal - discount - shipping, // Base amount before discount and shipping
		Discount:            discount,
		Shipping:            shipping,
		Tax:                 tax,
		GrandTotal:          grandTotal,
	}

	return &GetOrderDetailsResponseEnhanced{
		Order: enhancedOrder,
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
