package requests

import (
	"fmt"
	"strings"
)

// AddToCartRequest represents the request to add an item to the cart
type AddToCartRequest struct {
	ProductID string  `json:"product_id" binding:"required" validate:"required,min=1"`
	Quantity  int     `json:"quantity" binding:"required,min=1" validate:"required,min=1,max=1000"`
	UnitPrice float64 `json:"unit_price,omitempty" validate:"omitempty,min=0"` // Optional, will be fetched from product if not provided
}

// UpdateCartItemRequest represents the request to update a cart item
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=0" validate:"required,min=0,max=1000"` // 0 quantity means remove item
}

// IncrementCartItemRequest represents the request to increment cart item quantity
type IncrementCartItemRequest struct {
	Amount int `json:"amount" binding:"omitempty,min=1" validate:"omitempty,min=1,max=100"` // Default to 1 if not provided or 0
}

// DecrementCartItemRequest represents the request to decrement cart item quantity
type DecrementCartItemRequest struct {
	Amount int `json:"amount" binding:"omitempty,min=1" validate:"omitempty,min=1,max=100"` // Default to 1 if not provided or 0
}

// RemoveFromCartRequest represents the request to remove an item from the cart
type RemoveFromCartRequest struct {
	ProductID string `json:"product_id" binding:"required" validate:"required,min=1"`
}

// ClearCartRequest represents the request to clear the entire cart
type ClearCartRequest struct {
	UserID string `json:"user_id" binding:"required" validate:"required,min=1"`
}

// ApplyCouponRequest represents the request to apply a coupon to cart
type ApplyCouponRequest struct {
	CouponCode string `json:"coupon_code" binding:"required" validate:"required,min=1,max=50"`
}

// EstimateShippingRequest represents the request to estimate shipping costs
type EstimateShippingRequest struct {
	// Option 1: Use existing saved address
	AddressID string `json:"address_id,omitempty" validate:"omitempty,uuid"`

	// Option 2: Provide new address details (will be automatically saved for future use)
	Address struct {
		Street     string `json:"street,omitempty" validate:"omitempty,min=1,max=200"`
		City       string `json:"city,omitempty" validate:"omitempty,min=1,max=100"`
		State      string `json:"state,omitempty" validate:"omitempty,min=1,max=100"`
		PostalCode string `json:"postal_code,omitempty" validate:"omitempty,min=1,max=20"`
		Country    string `json:"country,omitempty" validate:"omitempty,min=1,max=100"`
	} `json:"address,omitempty"`

	// Address metadata (optional - will use defaults if not provided)
	AddressType  string `json:"address_type,omitempty" validate:"omitempty,oneof=home work other shipping billing"`
	FirstName    string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName     string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Phone        string `json:"phone,omitempty" validate:"omitempty,min=10,max=15"`
	SetAsDefault bool   `json:"set_as_default,omitempty"` // Whether to set as default address

	ShippingMethod string `json:"shipping_method,omitempty" validate:"omitempty,oneof=standard express overnight"`
}

//Validation helper methods

// IsValid checks if EstimateShippingRequest is valid
func (r *EstimateShippingRequest) IsValid() error {
	// Must provide either address_id or address details
	if r.AddressID == "" && (r.Address.Street == "" || r.Address.City == "" || r.Address.State == "" || r.Address.PostalCode == "") {
		return fmt.Errorf("either address_id or complete address details (street, city, state, postal_code) must be provided")
	}

	return nil
}

// Sanitize cleans string fields
func (r *EstimateShippingRequest) Sanitize() {
	r.AddressID = strings.TrimSpace(r.AddressID)
	r.Address.Street = strings.TrimSpace(r.Address.Street)
	r.Address.City = strings.TrimSpace(r.Address.City)
	r.Address.State = strings.TrimSpace(r.Address.State)
	r.Address.PostalCode = strings.TrimSpace(r.Address.PostalCode)
	r.Address.Country = strings.TrimSpace(r.Address.Country)
	r.AddressType = strings.TrimSpace(r.AddressType)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Phone = strings.TrimSpace(r.Phone)
	r.ShippingMethod = strings.TrimSpace(r.ShippingMethod)

	// Set defaults
	if r.Address.Country == "" && r.AddressID == "" {
		r.Address.Country = "India"
	}
	if r.AddressType == "" && r.AddressID == "" && r.Address.Street != "" {
		r.AddressType = "shipping"
	}
}

// IsValid checks if AddToCartRequest is valid
func (r *AddToCartRequest) IsValid() error {
	if r.ProductID == "" {
		return fmt.Errorf("product ID is required")
	}
	if r.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}
	if r.Quantity > 1000 {
		return fmt.Errorf("quantity cannot exceed 1000")
	}
	if r.UnitPrice < 0 {
		return fmt.Errorf("unit price cannot be negative")
	}
	return nil
}

// IsValid checks if UpdateCartItemRequest is valid
func (r *UpdateCartItemRequest) IsValid() error {
	if r.Quantity < 0 {
		return fmt.Errorf("quantity cannot be negative")
	}
	if r.Quantity > 1000 {
		return fmt.Errorf("quantity cannot exceed 1000")
	}
	return nil
}

// IsValid checks if IncrementCartItemRequest is valid and sets defaults
func (r *IncrementCartItemRequest) IsValid() error {
	if r.Amount <= 0 {
		r.Amount = 1 // Default to 1
	}
	if r.Amount > 100 {
		return fmt.Errorf("increment amount cannot exceed 100")
	}
	return nil
}

// IsValid checks if DecrementCartItemRequest is valid and sets defaults
func (r *DecrementCartItemRequest) IsValid() error {
	if r.Amount <= 0 {
		r.Amount = 1 // Default to 1
	}
	if r.Amount > 100 {
		return fmt.Errorf("decrement amount cannot exceed 100")
	}
	return nil
}

// Sanitize removes any potentially harmful characters from string fields
func (r *AddToCartRequest) Sanitize() {
	r.ProductID = strings.TrimSpace(r.ProductID)
}

func (r *UpdateCartItemRequest) Sanitize() {
	// No string fields to sanitize
}

func (r *IncrementCartItemRequest) Sanitize() {
	// No string fields to sanitize
}

func (r *DecrementCartItemRequest) Sanitize() {
	// No string fields to sanitize
}

func (r *RemoveFromCartRequest) Sanitize() {
	r.ProductID = strings.TrimSpace(r.ProductID)
}

func (r *ClearCartRequest) Sanitize() {
	r.UserID = strings.TrimSpace(r.UserID)
}

func (r *ApplyCouponRequest) Sanitize() {
	r.CouponCode = strings.TrimSpace(strings.ToUpper(r.CouponCode))
}
