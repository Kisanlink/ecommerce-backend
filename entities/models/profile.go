package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Profile represents a user profile with personal information (no duplicate fields)
type Profile struct {
	*base.BaseModel `gorm:"embedded"`
	UserID          string     `json:"user_id" gorm:"uniqueIndex;not null"`
	FirstName       string     `json:"first_name" gorm:"size:100;not null"`
	LastName        string     `json:"last_name" gorm:"size:100;not null"`
	Email           string     `json:"email,omitempty" gorm:"size:255"` // Optional - phone users don't need email
	DateOfBirth     *time.Time `json:"date_of_birth" gorm:"type:date"`
	Gender          string     `json:"gender" gorm:"size:10"` // Male, Female, Other

	// Address Information
	AddressLine1 string `json:"address_line_1" gorm:"size:255"`
	AddressLine2 string `json:"address_line_2" gorm:"size:255"`
	City         string `json:"city" gorm:"size:100"`
	State        string `json:"state" gorm:"size:100"`
	Country      string `json:"country" gorm:"size:100"`
	Pincode      string `json:"pincode" gorm:"size:20"`

	// Profile Image (S3 Integration)
	ProfilePictureURL   string `json:"profile_picture_url" gorm:"type:text"`   // Full URL for access
	ProfilePictureS3Key string `json:"profile_picture_s3_key" gorm:"size:500"` // S3 key for storage
	Bio                 string `json:"bio" gorm:"type:text"`                   // User bio

	// Preferences
	NewsletterSubscribed bool   `json:"newsletter_subscribed" gorm:"default:false"`
	PreferredLanguage    string `json:"preferred_language" gorm:"size:10;default:'en'"`

	// Account Status
	IsActive   bool `json:"is_active" gorm:"default:true"`
	IsVerified bool `json:"is_verified" gorm:"default:false"`
}

// NewProfile creates a new profile instance
func NewProfile(userID, firstName, lastName, email string) *Profile {
	baseModel := base.NewBaseModel("PROF", hash.Small)
	return &Profile{
		BaseModel:            baseModel,
		UserID:               userID,
		FirstName:            firstName,
		LastName:             lastName,
		Email:                email, // Optional - can be empty for phone users
		IsActive:             true,
		IsVerified:           false,
		NewsletterSubscribed: false,
		PreferredLanguage:    "en",
		Country:              "India", // Default for KisanLink
	}
}

// GetFullName returns the full name of the user
func (p *Profile) GetFullName() string {
	return p.FirstName + " " + p.LastName
}

// GetFullAddress returns the complete address
func (p *Profile) GetFullAddress() string {
	address := p.AddressLine1
	if p.AddressLine2 != "" {
		address += ", " + p.AddressLine2
	}
	if p.City != "" {
		address += ", " + p.City
	}
	if p.State != "" {
		address += ", " + p.State
	}
	if p.Country != "" {
		address += ", " + p.Country
	}
	if p.Pincode != "" {
		address += " - " + p.Pincode
	}
	return address
}

// IsAddressComplete checks if the address is complete
func (p *Profile) IsAddressComplete() bool {
	return p.AddressLine1 != "" && p.City != "" && p.State != "" && p.Country != "" && p.Pincode != ""
}

// TableName returns the table name for GORM
func (Profile) TableName() string {
	return "profiles"
}
