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

// Fulfillment status values track the post-payment lifecycle (order + shipment + inventory),
// independent of the courier-reported delivery Status above. Used to find and recover
// paid-but-unshipped orders.
const (
	FulfillmentPendingShipment = "pending_shipment"         // paid, shipment not yet created
	FulfillmentShipmentFailed  = "shipment_failed"          // shipment attempt failed (see ShipmentError)
	FulfillmentNeedsManual     = "needs_manual_fulfillment" // paid but order could not be built (e.g. cart gone)
	FulfillmentFulfilled       = "fulfilled"                // order + shipment + inventory all done
)

// OrderCustomer represents customer information captured at order time
type OrderCustomer struct {
	*base.BaseModel `gorm:"embedded"`
	FirstName       string `json:"first_name" gorm:"not null"`
	LastName        string `json:"last_name" gorm:"not null"`
	Phone           string `json:"phone" gorm:"not null"`
	PhoneAlt        string `json:"phone_alt"`
	Email           string `json:"email"`
}

// OrderAddress represents shipping address captured at order time
type OrderAddress struct {
	*base.BaseModel `gorm:"embedded"`
	AddressLine1    string `json:"address_line_1" gorm:"not null"`
	AddressLine2    string `json:"address_line_2"`
	City            string `json:"city" gorm:"not null"`
	State           string `json:"state" gorm:"not null"`
	PostalCode      string `json:"postal_code" gorm:"not null"`
	Country         string `json:"country" gorm:"not null;default:'India'"`
}

// Order represents an order in the system
type Order struct {
	*base.BaseModel `gorm:"embedded"`
	UserID          string      `json:"user_id" gorm:"not null;index"`
	TotalAmount     float64     `json:"total_amount" gorm:"not null"`
	Status          OrderStatus `json:"status" gorm:"default:'pending'"`

	// NORMALIZED: Foreign key references (nullable during migration)
	CustomerID        string `json:"customer_id" gorm:"index"`         // FK to order_customers
	ShippingAddressID string `json:"shipping_address_id" gorm:"index"` // FK to order_addresses

	// LEGACY: Keep for backward compatibility during migration
	ShippingAddr      string `json:"shipping_address"`
	ShippingAddrID    string `json:"shipping_address_id_legacy" gorm:"index"` // Reference to addresses table
	CustomerFirstName string `json:"customer_first_name"`
	CustomerLastName  string `json:"customer_last_name"`
	CustomerPhone     string `json:"customer_phone"`
	CustomerPhoneAlt  string `json:"customer_phone_alt"`
	PaymentMethod     string `json:"payment_method"`
	PaymentStatus     string `json:"payment_status" gorm:"default:'pending'"`
	Notes             string `json:"notes"`
	// Payment metadata (Razorpay)
	RazorpayOrderID     string `json:"razorpay_order_id" gorm:"index;unique"`
	RazorpayPaymentID   string `json:"razorpay_payment_id" gorm:"index"` // Captured pay_... ID, for reconciliation
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

	// Fulfillment state (for recovery/visibility of paid-but-unshipped orders)
	// Values: "" | "pending_shipment" | "shipment_failed" | "needs_manual_fulfillment" | "fulfilled"
	FulfillmentStatus string `json:"fulfillment_status" gorm:"index"`
	ShipmentError     string `json:"shipment_error"`                          // Last shipment failure reason (for support)
	InventoryDeducted bool   `json:"inventory_deducted" gorm:"default:false"` // Idempotency guard for stock decrement

	// Invoice metadata
	InvoiceURL string `json:"invoice_url"`

	// Discount information
	CouponCode     string  `json:"coupon_code" gorm:"index"`
	DiscountAmount float64 `json:"discount_amount" gorm:"default:0"`

	// GORM Relationships
	Customer        *OrderCustomer `json:"customer,omitempty" gorm:"foreignKey:CustomerID"`
	ShippingAddress *OrderAddress  `json:"shipping_address_normalized,omitempty" gorm:"foreignKey:ShippingAddressID"`
	OrderItems      []OrderItem    `json:"order_items,omitempty" gorm:"foreignKey:OrderID"`
}

// OrderItem represents an item in an order
type OrderItem struct {
	*base.BaseModel `gorm:"embedded"`
	OrderID         string  `json:"order_id" gorm:"not null;index"`
	ProductID       string  `json:"product_id" gorm:"not null;index"`
	Quantity        int     `json:"quantity" gorm:"not null"`
	UnitPrice       float64 `json:"unit_price" gorm:"not null"`
	TotalPrice      float64 `json:"total_price" gorm:"not null"`

	// GORM Relationship
	Product *Product `json:"product,omitempty" gorm:"foreignKey:ProductID"`
}

// NewOrderCustomer creates a new order customer with base model
func NewOrderCustomer(firstName, lastName, phone, phoneAlt, email string) *OrderCustomer {
	baseModel := base.NewBaseModel("OCUS", hash.Medium)
	return &OrderCustomer{
		BaseModel: baseModel,
		FirstName: firstName,
		LastName:  lastName,
		Phone:     phone,
		PhoneAlt:  phoneAlt,
		Email:     email,
	}
}

// NewOrderAddress creates a new order address with base model
func NewOrderAddress(addressLine1, addressLine2, city, state, postalCode, country string) *OrderAddress {
	baseModel := base.NewBaseModel("OADR", hash.Medium)
	return &OrderAddress{
		BaseModel:    baseModel,
		AddressLine1: addressLine1,
		AddressLine2: addressLine2,
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Country:      country,
	}
}

// NewOrder creates a new order with base model (normalized version)
func NewOrder(userID, customerID, shippingAddressID string, totalAmount float64, paymentMethod string, status OrderStatus, paymentStatus string) *Order {
	baseModel := base.NewBaseModel("ORDR", hash.Large)
	return &Order{
		BaseModel:         baseModel,
		UserID:            userID,
		CustomerID:        customerID,
		ShippingAddressID: shippingAddressID,
		TotalAmount:       totalAmount,
		Status:            status,
		PaymentMethod:     paymentMethod,
		PaymentStatus:     paymentStatus,
	}
}

// NewOrderLegacy creates a new order with base model (legacy version for backward compatibility)
func NewOrderLegacy(userID string, totalAmount float64, shippingAddr, paymentMethod string, status OrderStatus, paymentStatus string) *Order {
	baseModel := base.NewBaseModel("ORDR", hash.Large)
	return &Order{
		BaseModel:     baseModel,
		UserID:        userID,
		TotalAmount:   totalAmount,
		Status:        status,
		ShippingAddr:  shippingAddr,
		PaymentMethod: paymentMethod,
		PaymentStatus: paymentStatus,
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
func (oc *OrderCustomer) TableName() string {
	return "order_customers"
}

// TableName returns the table name for GORM
func (oa *OrderAddress) TableName() string {
	return "order_addresses"
}

// TableName returns the table name for GORM
func (o *Order) TableName() string {
	return "orders"
}

// TableName returns the table name for GORM
func (oi *OrderItem) TableName() string {
	return "order_items"
}
