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

// CouponRule represents a normalized coupon rule configuration
type CouponRule struct {
	base.BaseModel
	CouponID      string         `json:"coupon_id" gorm:"not null;index"`
	RuleType      CouponRuleType `json:"rule_type" gorm:"not null;check:rule_type IN ('percentage', 'fixed', 'free_shipping')"`
	DiscountValue float64        `json:"discount_value" gorm:"type:decimal(10,2);not null;check:discount_value >= 0"`
	MinAmount     float64        `json:"min_amount" gorm:"type:decimal(10,2);default:0;check:min_amount >= 0"`
	MaxDiscount   *float64       `json:"max_discount,omitempty" gorm:"type:decimal(10,2)"`
	ValidFrom     *time.Time     `json:"valid_from,omitempty"`
	ValidUntil    *time.Time     `json:"valid_until,omitempty"`
	UserLimit     *int           `json:"user_limit,omitempty"`
	FirstTimeOnly bool           `json:"first_time_only" gorm:"default:false"`
	Categories    CategoriesList `json:"categories,omitempty" gorm:"type:jsonb"`
	Products      ProductsList   `json:"products,omitempty" gorm:"type:jsonb"`
}

// CategoriesList represents a list of category names
type CategoriesList []string

// ProductsList represents a list of product IDs
type ProductsList []string

// Value implements the driver.Valuer interface for database storage
func (cl CategoriesList) Value() (driver.Value, error) {
	if len(cl) == 0 {
		return nil, nil
	}
	return json.Marshal(cl)
}

// Scan implements the sql.Scanner interface for database retrieval
func (cl *CategoriesList) Scan(value interface{}) error {
	if value == nil {
		*cl = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into CategoriesList", value)
	}

	return json.Unmarshal(bytes, cl)
}

// Value implements the driver.Valuer interface for database storage
func (pl ProductsList) Value() (driver.Value, error) {
	if len(pl) == 0 {
		return nil, nil
	}
	return json.Marshal(pl)
}

// Scan implements the sql.Scanner interface for database retrieval
func (pl *ProductsList) Scan(value interface{}) error {
	if value == nil {
		*pl = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into ProductsList", value)
	}

	return json.Unmarshal(bytes, pl)
}

// NewCouponRule creates a new coupon rule instance
func NewCouponRule(couponID string, ruleType CouponRuleType, discountValue float64) *CouponRule {
	return &CouponRule{
		BaseModel:     *base.NewBaseModel("RULE", hash.Medium),
		CouponID:      couponID,
		RuleType:      ruleType,
		DiscountValue: discountValue,
		MinAmount:     0,
		FirstTimeOnly: false,
	}
}

// IsValid checks if the coupon rule is valid based on its configuration
func (cr *CouponRule) IsValid() bool {
	now := time.Now()

	// Check validity period
	if cr.ValidFrom != nil && now.Before(*cr.ValidFrom) {
		return false
	}

	if cr.ValidUntil != nil && now.After(*cr.ValidUntil) {
		return false
	}

	// Check discount value
	if cr.DiscountValue <= 0 {
		return false
	}

	// Check rule type
	if cr.RuleType != CouponRuleTypePercentage &&
		cr.RuleType != CouponRuleTypeFixed &&
		cr.RuleType != CouponRuleTypeFreeShipping {
		return false
	}

	return true
}

// CanApplyToCart checks if the rule can be applied to a specific cart
func (cr *CouponRule) CanApplyToCart(cartTotal float64, userID string, isFirstTime bool) bool {
	if !cr.IsValid() {
		return false
	}

	// Check minimum amount
	if cr.MinAmount > 0 && cartTotal < cr.MinAmount {
		return false
	}

	// Check first time only rule
	if cr.FirstTimeOnly && !isFirstTime {
		return false
	}

	return true
}

// CalculateDiscount calculates the discount amount for a given cart total
func (cr *CouponRule) CalculateDiscount(cartTotal float64) float64 {
	if !cr.IsValid() {
		return 0
	}

	var discount float64

	switch cr.RuleType {
	case CouponRuleTypePercentage:
		discount = cartTotal * cr.DiscountValue / 100
		// Apply maximum discount limit if set
		if cr.MaxDiscount != nil && discount > *cr.MaxDiscount {
			discount = *cr.MaxDiscount
		}
	case CouponRuleTypeFixed:
		discount = cr.DiscountValue
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

// AppliesToCategory checks if the rule applies to a specific category
func (cr *CouponRule) AppliesToCategory(category string) bool {
	if len(cr.Categories) == 0 {
		return true // No category restriction
	}

	for _, cat := range cr.Categories {
		if cat == category {
			return true
		}
	}

	return false
}

// AppliesToProduct checks if the rule applies to a specific product
func (cr *CouponRule) AppliesToProduct(productID string) bool {
	if len(cr.Products) == 0 {
		return true // No product restriction
	}

	for _, prod := range cr.Products {
		if prod == productID {
			return true
		}
	}

	return false
}

// SetMinAmount sets the minimum amount for the rule
func (cr *CouponRule) SetMinAmount(amount float64) {
	cr.MinAmount = amount
}

// SetMaxDiscount sets the maximum discount for the rule
func (cr *CouponRule) SetMaxDiscount(discount float64) {
	cr.MaxDiscount = &discount
}

// SetValidityPeriod sets the validity period for the rule
func (cr *CouponRule) SetValidityPeriod(validFrom, validUntil *time.Time) {
	cr.ValidFrom = validFrom
	cr.ValidUntil = validUntil
}

// SetUserLimit sets the user limit for the rule
func (cr *CouponRule) SetUserLimit(limit int) {
	cr.UserLimit = &limit
}

// SetFirstTimeOnly sets the first time only flag
func (cr *CouponRule) SetFirstTimeOnly(firstTimeOnly bool) {
	cr.FirstTimeOnly = firstTimeOnly
}

// AddCategory adds a category to the rule
func (cr *CouponRule) AddCategory(category string) {
	if cr.Categories == nil {
		cr.Categories = make(CategoriesList, 0)
	}

	// Check if category already exists
	for _, cat := range cr.Categories {
		if cat == category {
			return
		}
	}

	cr.Categories = append(cr.Categories, category)
}

// AddProduct adds a product to the rule
func (cr *CouponRule) AddProduct(productID string) {
	if cr.Products == nil {
		cr.Products = make(ProductsList, 0)
	}

	// Check if product already exists
	for _, prod := range cr.Products {
		if prod == productID {
			return
		}
	}

	cr.Products = append(cr.Products, productID)
}

// RemoveCategory removes a category from the rule
func (cr *CouponRule) RemoveCategory(category string) {
	if cr.Categories == nil {
		return
	}

	for i, cat := range cr.Categories {
		if cat == category {
			cr.Categories = append(cr.Categories[:i], cr.Categories[i+1:]...)
			break
		}
	}
}

// RemoveProduct removes a product from the rule
func (cr *CouponRule) RemoveProduct(productID string) {
	if cr.Products == nil {
		return
	}

	for i, prod := range cr.Products {
		if prod == productID {
			cr.Products = append(cr.Products[:i], cr.Products[i+1:]...)
			break
		}
	}
}
