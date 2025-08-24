package models

import (
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
	Type            AddressType `json:"type" gorm:"default:'home'"`
	FirstName       string      `json:"first_name" gorm:"not null"`
	LastName        string      `json:"last_name" gorm:"not null"`
	Company         string      `json:"company"`
	AddressLine1    string      `json:"address_line_1" gorm:"not null"`
	AddressLine2    string      `json:"address_line_2"`
	City            string      `json:"city" gorm:"not null"`
	State           string      `json:"state" gorm:"not null"`
	PostalCode      string      `json:"postal_code" gorm:"not null"`
	Country         string      `json:"country" gorm:"not null;default:'India'"`
	Phone           string      `json:"phone"`
	IsDefault       bool        `json:"is_default" gorm:"default:false"`
}

// NewAddress creates a new address with base model
func NewAddress(userID, firstName, lastName, addressLine1, city, state, postalCode, country string, addressType AddressType) *Address {
	baseModel := base.NewBaseModel("ADDR", hash.Medium)
	return &Address{
		BaseModel:    baseModel,
		UserID:       userID,
		Type:         addressType,
		FirstName:    firstName,
		LastName:     lastName,
		AddressLine1: addressLine1,
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Country:      country,
		IsDefault:    false,
	}
}

// TableName returns the table name for GORM
func (a *Address) TableName() string {
	return "addresses"
}
