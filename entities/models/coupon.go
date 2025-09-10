package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// CouponRuleType represents the type of coupon rule
type CouponRuleType string

const (
	CouponRuleTypePercentage   CouponRuleType = "percentage"
	CouponRuleTypeFixed        CouponRuleType = "fixed"
	CouponRuleTypeFreeShipping CouponRuleType = "free_shipping"
)

// CouponRule represents a coupon rule configuration
type CouponRule struct {
	Type          CouponRuleType `json:"type"`
	DiscountValue float64        `json:"value"`
	MinAmount     float64        `json:"min_amount,omitempty"`
	MaxDiscount   float64        `json:"max_discount,omitempty"`
	Categories    []string       `json:"categories,omitempty"`
	Products      []string       `json:"products,omitempty"`
	ValidFrom     *time.Time     `json:"valid_from,omitempty"`
	ValidUntil    *time.Time     `json:"valid_until,omitempty"`
	UserLimit     int            `json:"user_limit,omitempty"`
	FirstTimeOnly bool           `json:"first_time_only,omitempty"`
}

// Value implements the driver.Valuer interface for database storage
func (cr CouponRule) Value() (driver.Value, error) {
	return json.Marshal(cr)
}

// Scan implements the sql.Scanner interface for database retrieval
func (cr *CouponRule) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into CouponRule", value)
	}

	return json.Unmarshal(bytes, cr)
}

// Coupon represents a discount coupon
type Coupon struct {
	base.BaseModel
	Code        string     `json:"code" gorm:"uniqueIndex;not null"`
	Description string     `json:"description"`
	Rules       CouponRule `json:"rules" gorm:"type:jsonb"`
	Count       int        `json:"count" gorm:"default:0"`
	UsedCount   int        `json:"used_count" gorm:"default:0"`
	IsActive    bool       `json:"is_active" gorm:"default:true"`
	CreatedBy   string     `json:"created_by"`
}

// NewCoupon creates a new coupon instance
func NewCoupon(code, description, createdBy string, rules CouponRule) *Coupon {
	return &Coupon{
		BaseModel:   *base.NewBaseModel("COUP", hash.Medium),
		Code:        code,
		Description: description,
		Rules:       rules,
		Count:       0,
		UsedCount:   0,
		IsActive:    true,
		CreatedBy:   createdBy,
	}
}

// IsValid checks if the coupon is valid based on its rules
func (c *Coupon) IsValid() bool {
	if !c.IsActive {
		return false
	}

	now := time.Now()

	// Check validity period
	if c.Rules.ValidFrom != nil && now.Before(*c.Rules.ValidFrom) {
		return false
	}

	if c.Rules.ValidUntil != nil && now.After(*c.Rules.ValidUntil) {
		return false
	}

	// Check usage limit
	if c.Count > 0 && c.UsedCount >= c.Count {
		return false
	}

	return true
}

// CanUse checks if the coupon can be used for a specific cart
func (c *Coupon) CanUse(cartTotal float64, userID string, isFirstTime bool) bool {
	if !c.IsValid() {
		return false
	}

	// Check minimum amount
	if c.Rules.MinAmount > 0 && cartTotal < c.Rules.MinAmount {
		return false
	}

	// Check first time only rule
	if c.Rules.FirstTimeOnly && !isFirstTime {
		return false
	}

	return true
}

// CalculateDiscount calculates the discount amount for a given cart total
func (c *Coupon) CalculateDiscount(cartTotal float64) float64 {
	if !c.IsValid() {
		return 0
	}

	var discount float64

	switch c.Rules.Type {
	case CouponRuleTypePercentage:
		discount = cartTotal * c.Rules.DiscountValue / 100
		// Apply maximum discount limit if set
		if c.Rules.MaxDiscount > 0 && discount > c.Rules.MaxDiscount {
			discount = c.Rules.MaxDiscount
		}
	case CouponRuleTypeFixed:
		discount = c.Rules.DiscountValue
		// Don't exceed cart total
		if discount > cartTotal {
			discount = cartTotal
		}
	case CouponRuleTypeFreeShipping:
		// This would be handled separately in shipping calculation
		discount = 0
	}

	return discount
}

// IncrementUsage increments the used count
func (c *Coupon) IncrementUsage() {
	c.UsedCount++
	c.UpdatedAt = time.Now()
}
