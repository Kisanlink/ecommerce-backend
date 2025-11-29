package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// User represents a user in the system (Authentication only)
type User struct {
	*base.BaseModel `gorm:"embedded"`
	Email           *string `json:"email,omitempty" gorm:"unique"` // Made optional for phone-only users
	Name            string  `json:"name" gorm:"not null"`
	Password        string  `json:"-"` // Made optional for phone-only users
	FirebaseUID     *string `json:"firebase_uid,omitempty"`
	PhoneNumber     *string `json:"phone_number,omitempty" gorm:"unique"` // Nullable, unique when set
	IsVerified      bool    `json:"is_verified" gorm:"default:false"`
	IsActive        bool    `json:"is_active" gorm:"default:true"`
	Role            string  `json:"role" gorm:"type:varchar(20);default:user;index"`
}

// NewUser creates a new user with base model
func NewUser(email, name, password string) *User {
	baseModel := base.NewBaseModel("USER", hash.Small)
	return &User{
		BaseModel:   baseModel,
		Email:       &email,
		Name:        name,
		Password:    password,
		PhoneNumber: nil, // nil for email-based users (no phone number)
		IsVerified:  false,
		IsActive:    true,
		Role:        "user",
	}
}

// NewPhoneUser creates a new user with phone number (no email/password)
func NewPhoneUser(phoneNumber, name string) *User {
	baseModel := base.NewBaseModel("USER", hash.Small)
	phoneNum := phoneNumber // Convert to pointer
	return &User{
		BaseModel:   baseModel,
		Email:       nil, // nil for phone-only users (no email)
		PhoneNumber: &phoneNum,
		Name:        name,
		Password:    "",   // Empty string for phone-only users
		FirebaseUID: nil,  // Will be set later when Firebase user is created
		IsVerified:  true, // Phone verification means user is verified
		IsActive:    true,
		Role:        "user",
	}
}

// TableName returns the table name for GORM
func (u *User) TableName() string {
	return "users"
}
