package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// User represents a user in the system
type User struct {
	*base.BaseModel `gorm:"embedded"`
	Username        string `json:"username" gorm:"unique;not null"`
	Email           string `json:"email" gorm:"unique;not null"`
	Password        string `json:"-" gorm:"not null"`
	Role            string `json:"role" gorm:"default:user"`
	FirebaseUID     string `json:"firebase_uid,omitempty" gorm:"unique"`
	FirstName       string `json:"first_name"`
	LastName        string `json:"last_name"`
	PhoneNumber     string `json:"phone_number"`
	IsVerified      bool   `json:"is_verified" gorm:"default:false"`
}

// NewUser creates a new user with base model
func NewUser(username, email, password, role string) *User {
	baseModel := base.NewBaseModel("USER", hash.Small)
	return &User{
		BaseModel:   baseModel,
		Username:    username,
		Email:       email,
		Password:    password,
		Role:        role,
		IsVerified:  false,
	}
}

// TableName returns the table name for GORM
func (u *User) TableName() string {
	return "users"
}
