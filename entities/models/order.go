package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusConfirmed OrderStatus = "confirmed"
	OrderStatusShipped   OrderStatus = "shipped"
	OrderStatusDelivered OrderStatus = "delivered"
)

// Order represents an order in the system
type Order struct {
	*base.BaseModel `gorm:"embedded"`
	UserID          string      `json:"user_id" gorm:"not null;index"`
	TotalAmount     float64     `json:"total_amount" gorm:"not null"`
	Status          OrderStatus `json:"status" gorm:"default:'pending'"`
	ShippingAddr    string      `json:"shipping_address"`
	PaymentMethod   string      `json:"payment_method"`
	PaymentStatus   string      `json:"payment_status" gorm:"default:'pending'"`
	Notes           string      `json:"notes"`
	// Payment metadata (Razorpay)
	RazorpayOrderID     string `json:"razorpay_order_id" gorm:"index"`
	RazorpayReceipt     string `json:"razorpay_receipt"`
	RazorpayAmountPaisa int    `json:"razorpay_amount_paisa"`
	RazorpayNotesJSON   string `json:"razorpay_notes_json"`
	// Shipping metadata
	ShippingCarrier string  `json:"shipping_carrier"`
	ShippingMode    string  `json:"shipping_mode"`
	ShippingCharge  float64 `json:"shipping_charge"`
	EtaMinDays      int     `json:"eta_min_days"`
	EtaMaxDays      int     `json:"eta_max_days"`

	// Delhivery tracking
	DelhiveryWaybill string `json:"delhivery_waybill" gorm:"index"` // AWB number
	DelhiveryStatus  string `json:"delhivery_status"`               // Shipment status
	TrackingURL      string `json:"tracking_url"`                   // Direct tracking link

	// Invoice metadata
	InvoiceURL string `json:"invoice_url"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	*base.BaseModel `gorm:"embedded"`
	OrderID         string  `json:"order_id" gorm:"not null;index"`
	ProductID       string  `json:"product_id" gorm:"not null;index"`
	Quantity        int     `json:"quantity" gorm:"not null"`
	UnitPrice       float64 `json:"unit_price" gorm:"not null"`
	TotalPrice      float64 `json:"total_price" gorm:"not null"`
}

// NewOrder creates a new order with base model
func NewOrder(userID string, totalAmount float64, shippingAddr, paymentMethod string) *Order {
	baseModel := base.NewBaseModel("ORDR", hash.Large)
	return &Order{
		BaseModel:     baseModel,
		UserID:        userID,
		TotalAmount:   totalAmount,
		Status:        OrderStatusPending,
		ShippingAddr:  shippingAddr,
		PaymentMethod: paymentMethod,
		PaymentStatus: "pending",
	}
}

// NewOrderItem creates a new order item with base model
func NewOrderItem(orderID, productID string, quantity int, unitPrice float64) *OrderItem {
	baseModel := base.NewBaseModel("OITM", hash.Large)
	return &OrderItem{
		BaseModel:  baseModel,
		OrderID:    orderID,
		ProductID:  productID,
		Quantity:   quantity,
		UnitPrice:  unitPrice,
		TotalPrice: float64(quantity) * unitPrice,
	}
}

// TableName returns the table name for GORM
func (o *Order) TableName() string {
	return "orders"
}

// TableName returns the table name for GORM
func (oi *OrderItem) TableName() string {
	return "order_items"
}
