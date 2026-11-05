package models

import (
	"encoding/json"
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Snapshot lifecycle states.
const (
	SnapshotStatusPending  = "pending"  // written at create-order, not yet turned into an order
	SnapshotStatusConsumed = "consumed" // an order has been rebuilt from this snapshot
)

// SnapshotLineItem is one cart line frozen at create-order time. It is stored as JSON
// inside PaymentCartSnapshot.ItemsJSON so the exact items/prices the customer paid for
// survive even when the live cart is later changed or the process restarts.
type SnapshotLineItem struct {
	ProductID string  `json:"product_id"`
	Quantity  int     `json:"quantity"`
	UnitPrice float64 `json:"unit_price"`
}

// PaymentCartSnapshot freezes everything needed to rebuild a confirmed order from only a
// razorpay_order_id. It is written synchronously when the Razorpay order (payment intent)
// is created and read back by the webhook when a payment is captured but no DB order
// exists (customer closed the tab before /verify ran). Because it lives in Postgres — not
// an in-memory cache — it survives a process restart / host idle-sleep (e.g. Render free
// tier), which is exactly when the webhook, not the browser, has to complete the order.
//
// It is NOT a confirmed order: it holds no money and ships nothing. It only lets
// fulfillFromWebhook rebuild the order the browser would have created at /verify.
type PaymentCartSnapshot struct {
	*base.BaseModel `gorm:"embedded"`

	// Keyed by the Razorpay order id so the webhook can find it from the payload alone.
	RazorpayOrderID string `json:"razorpay_order_id" gorm:"uniqueIndex;not null"`
	UserID          string `json:"user_id" gorm:"index;not null"`

	// "standard" (widget) or "magic" (1CC). Recorded for triage; both rebuild the same way.
	Flow string `json:"flow" gorm:"index"`

	// Frozen cart contents + money (rupees), plus the captured amount we expect in paisa.
	ItemsJSON           string  `json:"-" gorm:"type:text"`
	CartTotal           float64 `json:"cart_total"`
	DiscountAmount      float64 `json:"discount_amount" gorm:"default:0"`
	CouponCode          string  `json:"coupon_code"`
	ShippingCharge      float64 `json:"shipping_charge" gorm:"default:0"`
	FinalAmount         float64 `json:"final_amount"`
	ExpectedAmountPaisa int     `json:"expected_amount_paisa"`

	// Customer captured at create-order.
	CustomerFirstName string `json:"customer_first_name"`
	CustomerLastName  string `json:"customer_last_name"`
	CustomerPhone     string `json:"customer_phone"`
	CustomerPhoneAlt  string `json:"customer_phone_alt"`
	CustomerEmail     string `json:"customer_email"`

	// Shipping address captured at create-order (frozen copy, not a live FK).
	ShippingAddressID string `json:"shipping_address_id"`
	AddressLine1      string `json:"address_line_1"`
	AddressLine2      string `json:"address_line_2"`
	City              string `json:"city"`
	State             string `json:"state"`
	PostalCode        string `json:"postal_code"`
	Country           string `json:"country" gorm:"default:'India'"`

	// Lifecycle. Set to consumed once an order is rebuilt so it is actioned once.
	Status            string     `json:"status" gorm:"index;default:'pending'"`
	ConsumedAt        *time.Time `json:"consumed_at"`
	RazorpayPaymentID string     `json:"razorpay_payment_id"`
}

// NewPaymentCartSnapshot creates a snapshot in the pending state.
func NewPaymentCartSnapshot(razorpayOrderID, userID string) *PaymentCartSnapshot {
	return &PaymentCartSnapshot{
		BaseModel:       base.NewBaseModel("PCS", hash.Medium),
		RazorpayOrderID: razorpayOrderID,
		UserID:          userID,
		Country:         "India",
		Status:          SnapshotStatusPending,
	}
}

// SetLineItems serializes the frozen cart lines into ItemsJSON.
func (p *PaymentCartSnapshot) SetLineItems(items []SnapshotLineItem) error {
	b, err := json.Marshal(items)
	if err != nil {
		return err
	}
	p.ItemsJSON = string(b)
	return nil
}

// LineItems deserializes the frozen cart lines. A blank/invalid blob yields an empty slice.
func (p *PaymentCartSnapshot) LineItems() []SnapshotLineItem {
	if p.ItemsJSON == "" {
		return nil
	}
	var items []SnapshotLineItem
	if err := json.Unmarshal([]byte(p.ItemsJSON), &items); err != nil {
		return nil
	}
	return items
}

// TableName returns the table name for GORM.
func (p *PaymentCartSnapshot) TableName() string { return "payment_cart_snapshots" }
