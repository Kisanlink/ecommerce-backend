package requests

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// CreateCouponRequest represents the request to create a new coupon
type CreateCouponRequest struct {
	Code          string                `json:"code" binding:"required"`
	Description   string                `json:"description" binding:"required"`
	RuleType      models.CouponRuleType `json:"rule_type" binding:"required"`
	Value         float64               `json:"value" binding:"required,gt=0"`
	MinAmount     float64               `json:"min_amount,omitempty"`
	MaxDiscount   float64               `json:"max_discount,omitempty"`
	Categories    []string              `json:"categories,omitempty"`
	Products      []string              `json:"products,omitempty"`
	ValidFrom     *time.Time            `json:"valid_from,omitempty"`
	ValidUntil    *time.Time            `json:"valid_until,omitempty"`
	UserLimit     int                   `json:"user_limit,omitempty"`
	FirstTimeOnly bool                  `json:"first_time_only,omitempty"`
	Count         int                   `json:"count" binding:"required,gt=0"`
	CreatedBy     string                `json:"created_by" binding:"required"`
}

// UpdateCouponRequest represents the request to update an existing coupon
type UpdateCouponRequest struct {
	ID            string                `json:"id,omitempty"`
	Description   string                `json:"description,omitempty"`
	RuleType      models.CouponRuleType `json:"rule_type,omitempty"`
	Value         float64               `json:"value,omitempty"`
	MinAmount     float64               `json:"min_amount,omitempty"`
	MaxDiscount   float64               `json:"max_discount,omitempty"`
	Categories    []string              `json:"categories,omitempty"`
	Products      []string              `json:"products,omitempty"`
	ValidFrom     *time.Time            `json:"valid_from,omitempty"`
	ValidUntil    *time.Time            `json:"valid_until,omitempty"`
	UserLimit     int                   `json:"user_limit,omitempty"`
	FirstTimeOnly *bool                 `json:"first_time_only,omitempty"`
	Count         int                   `json:"count,omitempty"`
	IsActive      *bool                 `json:"is_active,omitempty"`
}

// ValidateCouponRequest represents the request to validate a coupon
type ValidateCouponRequest struct {
	Code      string  `json:"code" binding:"required"`
	CartTotal float64 `json:"cart_total" binding:"required,gte=0"`
	UserID    string  `json:"user_id" binding:"required"`
}

// ApplyCouponToUserRequest represents the request to apply a coupon to a user
type ApplyCouponToUserRequest struct {
	Code   string `json:"code" binding:"required"`
	UserID string `json:"user_id" binding:"required"`
}

// GetCouponsRequest represents the request to get coupons with pagination
type GetCouponsRequest struct {
	Limit  int  `json:"limit,omitempty" form:"limit"`
	Offset int  `json:"offset,omitempty" form:"offset"`
	Active bool `json:"active,omitempty" form:"active"`
}

// CreateCouponRuleRequest represents the request to create a new coupon rule
type CreateCouponRuleRequest struct {
	CouponID      string                `json:"coupon_id" binding:"required"`
	RuleType      models.CouponRuleType `json:"rule_type" binding:"required"`
	DiscountValue float64               `json:"discount_value" binding:"required,gt=0"`
	MinAmount     float64               `json:"min_amount,omitempty"`
	MaxDiscount   float64               `json:"max_discount,omitempty"`
	Categories    []string              `json:"categories,omitempty"`
	Products      []string              `json:"products,omitempty"`
	ValidFrom     *time.Time            `json:"valid_from,omitempty"`
	ValidUntil    *time.Time            `json:"valid_until,omitempty"`
	UserLimit     int                   `json:"user_limit,omitempty"`
	FirstTimeOnly bool                  `json:"first_time_only,omitempty"`
}

// UpdateCouponRuleRequest represents the request to update an existing coupon rule
type UpdateCouponRuleRequest struct {
	ID            string                `json:"id,omitempty"`
	RuleType      models.CouponRuleType `json:"rule_type,omitempty"`
	DiscountValue float64               `json:"discount_value,omitempty"`
	MinAmount     float64               `json:"min_amount,omitempty"`
	MaxDiscount   float64               `json:"max_discount,omitempty"`
	Categories    []string              `json:"categories,omitempty"`
	Products      []string              `json:"products,omitempty"`
	ValidFrom     *time.Time            `json:"valid_from,omitempty"`
	ValidUntil    *time.Time            `json:"valid_until,omitempty"`
	UserLimit     int                   `json:"user_limit,omitempty"`
	FirstTimeOnly *bool                 `json:"first_time_only,omitempty"`
}
