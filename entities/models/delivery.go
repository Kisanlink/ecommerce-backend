package models

// DeliveryMethod represents a shipping method name
type DeliveryMethod string

const (
	DeliveryMethodStandard DeliveryMethod = "standard"
	DeliveryMethodExpress  DeliveryMethod = "express"
)

// DeliveryOption represents a single delivery option with price and ETA
type DeliveryOption struct {
	Method       DeliveryMethod `json:"method"`
	Carrier      string         `json:"carrier"`
	ServiceLevel string         `json:"service_level"`
	Price        float64        `json:"price"`
	Currency     string         `json:"currency"`
	EtaDaysMin   int            `json:"eta_days_min"`
	EtaDaysMax   int            `json:"eta_days_max"`
}

// DeliveryEstimate contains available options for a user/address
type DeliveryEstimate struct {
	UserID    string           `json:"user_id"`
	AddressID string           `json:"address_id,omitempty"` // ID of the saved address used for estimation
	Options   []DeliveryOption `json:"options"`
	Subtotal  float64          `json:"subtotal"`
}
