package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// User represents a user in the system (Authentication only)
type User struct {
	*base.BaseModel `gorm:"embedded"`
	Email           string `json:"email" gorm:"unique;not null"`
	Name            string `json:"name" gorm:"not null"`
	Password        string `json:"-" gorm:"not null"`
	FirebaseUID     string `json:"firebase_uid,omitempty" gorm:"unique"`
	PhoneNumber     string `json:"phone_number"`
	IsVerified      bool   `json:"is_verified" gorm:"default:false"`
	IsActive        bool   `json:"is_active" gorm:"default:true"`
	Role            string `json:"role" gorm:"type:varchar(20);default:user;index"`
}

// NewUser creates a new user with base model
func NewUser(email, name, password string) *User {
	baseModel := base.NewBaseModel("USER", hash.Small)
	return &User{
		BaseModel:  baseModel,
		Email:      email,
		Name:       name,
		Password:   password,
		IsVerified: false,
		IsActive:   true,
		Role:       "user",
	}
}

// TableName returns the table name for GORM
func (u *User) TableName() string {
	return "users"
}
