package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// AddressType represents the type of address
type AddressType string

const (
	AddressTypeHome     AddressType = "home"
	AddressTypeWork     AddressType = "work"
	AddressTypeOther    AddressType = "other"
	AddressTypeShipping AddressType = "shipping"
	AddressTypeBilling  AddressType = "billing"
)

// Address represents an address in the system
type Address struct {
	*base.BaseModel `gorm:"embedded"`
	UserID          string      `json:"user_id" gorm:"not null;index"`
	Type            AddressType `json:"type" gorm:"default:'shipping'"`
	FirstName       string      `json:"first_name"`                              // Optional during estimate, can be added later
	LastName        string      `json:"last_name"`                               // Optional during estimate, can be added later
	Company         string      `json:"company"`                                 // Optional
	AddressLine1    string      `json:"address_line_1" gorm:"not null"`          // Main address line (from "street")
	AddressLine2    string      `json:"address_line_2"`                          // Optional secondary line
	City            string      `json:"city" gorm:"not null"`                    // Required
	State           string      `json:"state" gorm:"not null"`                   // Required
	PostalCode      string      `json:"postal_code" gorm:"not null;index"`       // Required, indexed for shipping lookups
	Country         string      `json:"country" gorm:"not null;default:'India'"` // Required, defaults to India
	Phone           string      `json:"phone"`                                   // Optional during estimate
	IsDefault       bool        `json:"is_default" gorm:"default:false"`         // Default address flag
	Source          string      `json:"source" gorm:"default:'estimate'"`        // Track how address was created (estimate, manual, etc)
	LastUsedAt      *time.Time  `json:"last_used_at"`                            // Track when address was last used
}

// NewAddress creates a new address with base model
func NewAddress(userID, addressLine1, city, state, postalCode, country string, addressType AddressType) *Address {
	baseModel := base.NewBaseModel("ADDR", hash.Medium)
	return &Address{
		BaseModel:    baseModel,
		UserID:       userID,
		Type:         addressType,
		AddressLine1: addressLine1,
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Country:      country,
		IsDefault:    false,
		Source:       "manual", // Default source
	}
}

// NewAddressFromEstimate creates a new address from shipping estimate data
func NewAddressFromEstimate(userID, street, city, state, postalCode, country string) *Address {
	baseModel := base.NewBaseModel("ADDR", hash.Medium)
	return &Address{
		BaseModel:    baseModel,
		UserID:       userID,
		Type:         AddressTypeShipping,
		AddressLine1: street,
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Country:      country,
		IsDefault:    false,
		Source:       "estimate", // Track that this came from estimate shipping
	}
}

// TableName returns the table name for GORM
func (a *Address) TableName() string {
	return "addresses"
}
