package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// PhoneVerification represents an OTP verification session for WhatsApp phone auth
type PhoneVerification struct {
	*base.BaseModel `gorm:"embedded"`
	PhoneNumber     string    `gorm:"index;size:20;not null" json:"phone_number"`
	CodeHash        string    `gorm:"not null" json:"-"`
	SessionInfo     string    `gorm:"uniqueIndex;size:100;not null" json:"session_info"`
	ExpiresAt       time.Time `gorm:"not null;index" json:"expires_at"`
	Attempts        int       `gorm:"default:0" json:"attempts"`
	IsUsed          bool      `gorm:"default:false" json:"is_used"`
}

// NewPhoneVerification creates a new PhoneVerification instance with a base model
func NewPhoneVerification(phoneNumber, codeHash, sessionInfo string, expiresAt time.Time) *PhoneVerification {
	baseModel := base.NewBaseModel("PV", hash.Small)
	return &PhoneVerification{
		BaseModel:   baseModel,
		PhoneNumber: phoneNumber,
		CodeHash:    codeHash,
		SessionInfo: sessionInfo,
		ExpiresAt:   expiresAt,
		Attempts:    0,
		IsUsed:      false,
	}
}
