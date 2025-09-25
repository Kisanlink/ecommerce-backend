package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Coupon represents a discount coupon with normalized structure
type Coupon struct {
	base.BaseModel
	Code        string `json:"code" gorm:"uniqueIndex;not null"`
	Description string `json:"description"`
	Count       int    `json:"count" gorm:"default:0;check:count >= 0"`
	UsedCount   int    `json:"used_count" gorm:"default:0;check:used_count >= 0 AND used_count <= count"`
	IsActive    bool   `json:"is_active" gorm:"default:true"`
	CreatedBy   string `json:"created_by"`

	// Relationship to coupon rules
	Rules []CouponRule `json:"rules" gorm:"foreignKey:CouponID;constraint:OnDelete:CASCADE"`
}

// NewCoupon creates a new coupon instance
func NewCoupon(code, description, createdBy string) *Coupon {
	return &Coupon{
		BaseModel:   *base.NewBaseModel("COUP", hash.Medium),
		Code:        code,
		Description: description,
		Count:       0,
		UsedCount:   0,
		IsActive:    true,
		CreatedBy:   createdBy,
		Rules:       make([]CouponRule, 0),
	}
}

// IsValid checks if the coupon is valid based on its rules
func (c *Coupon) IsValid() bool {
	if !c.IsActive {
		return false
	}

	// Check usage limit
	if c.Count > 0 && c.UsedCount >= c.Count {
		return false
	}

	// Check if any rule is valid
	if len(c.Rules) == 0 {
		return false
	}

	// At least one rule must be valid
	for _, rule := range c.Rules {
		if rule.IsValid() {
			return true
		}
	}

	return false
}

// CanUse checks if the coupon can be used for a specific cart
func (c *Coupon) CanUse(cartTotal float64, userID string, isFirstTime bool) bool {
	if !c.IsValid() {
		return false
	}

	// Check if any rule can be applied to this cart
	for _, rule := range c.Rules {
		if rule.CanApplyToCart(cartTotal, userID, isFirstTime) {
			return true
		}
	}

	return false
}

// CanUseWithItems checks if the coupon can be used for a specific cart with items
func (c *Coupon) CanUseWithItems(cartTotal float64, userID string, isFirstTime bool, cartItems []string) bool {
	if !c.IsValid() {
		return false
	}

	// Check if any rule can be applied to this cart with items
	for _, rule := range c.Rules {
		if rule.CanApplyToCartWithItems(cartTotal, userID, isFirstTime, cartItems) {
			return true
		}
	}

	return false
}

// CalculateDiscount calculates the discount amount for a given cart total
func (c *Coupon) CalculateDiscount(cartTotal float64) float64 {
	if !c.IsValid() {
		return 0
	}

	var maxDiscount float64

	// Find the rule that gives the maximum discount
	for _, rule := range c.Rules {
		if rule.CanApplyToCart(cartTotal, "", true) { // Simplified check
			discount := rule.CalculateDiscount(cartTotal)
			if discount > maxDiscount {
				maxDiscount = discount
			}
		}
	}

	return maxDiscount
}

// GetBestRule returns the rule that provides the best discount for the given cart total
func (c *Coupon) GetBestRule(cartTotal float64, userID string, isFirstTime bool) *CouponRule {
	if !c.IsValid() {
		return nil
	}

	var bestRule *CouponRule
	var maxDiscount float64

	for i := range c.Rules {
		rule := &c.Rules[i]
		if rule.CanApplyToCart(cartTotal, userID, isFirstTime) {
			discount := rule.CalculateDiscount(cartTotal)
			if discount > maxDiscount {
				maxDiscount = discount
				bestRule = rule
			}
		}
	}

	return bestRule
}

// GetBestRuleWithItems returns the rule that provides the best discount for the given cart with items
func (c *Coupon) GetBestRuleWithItems(cartTotal float64, userID string, isFirstTime bool, cartItems []string) *CouponRule {
	if !c.IsValid() {
		return nil
	}

	var bestRule *CouponRule
	var maxDiscount float64

	for i := range c.Rules {
		rule := &c.Rules[i]
		if rule.CanApplyToCartWithItems(cartTotal, userID, isFirstTime, cartItems) {
			discount := rule.CalculateDiscount(cartTotal)
			if discount > maxDiscount {
				maxDiscount = discount
				bestRule = rule
			}
		}
	}

	return bestRule
}

// IncrementUsage increments the used count
func (c *Coupon) IncrementUsage() {
	c.UsedCount++
	c.UpdatedAt = time.Now()
}

// AddRule adds a new rule to the coupon
func (c *Coupon) AddRule(rule *CouponRule) {
	if c.Rules == nil {
		c.Rules = make([]CouponRule, 0)
	}

	rule.CouponID = c.ID
	c.Rules = append(c.Rules, *rule)
}

// RemoveRule removes a rule from the coupon
func (c *Coupon) RemoveRule(ruleID string) {
	if c.Rules == nil {
		return
	}

	for i, rule := range c.Rules {
		if rule.ID == ruleID {
			c.Rules = append(c.Rules[:i], c.Rules[i+1:]...)
			break
		}
	}
}

// GetRuleByID returns a rule by its ID
func (c *Coupon) GetRuleByID(ruleID string) *CouponRule {
	for i := range c.Rules {
		if c.Rules[i].ID == ruleID {
			return &c.Rules[i]
		}
	}
	return nil
}

// GetRulesByType returns all rules of a specific type
func (c *Coupon) GetRulesByType(ruleType CouponRuleType) []CouponRule {
	var rules []CouponRule
	for _, rule := range c.Rules {
		if rule.RuleType == ruleType {
			rules = append(rules, rule)
		}
	}
	return rules
}

// HasActiveRules checks if the coupon has any active rules
func (c *Coupon) HasActiveRules() bool {
	for _, rule := range c.Rules {
		if rule.IsValid() {
			return true
		}
	}
	return false
}

// GetUsagePercentage returns the usage percentage of the coupon
func (c *Coupon) GetUsagePercentage() float64 {
	if c.Count == 0 {
		return 0
	}
	return float64(c.UsedCount) / float64(c.Count) * 100
}

// IsFullyUsed checks if the coupon has reached its usage limit
func (c *Coupon) IsFullyUsed() bool {
	return c.Count > 0 && c.UsedCount >= c.Count
}

// GetRemainingUses returns the number of remaining uses
func (c *Coupon) GetRemainingUses() int {
	if c.Count == 0 {
		return -1 // Unlimited uses
	}
	remaining := c.Count - c.UsedCount
	if remaining < 0 {
		return 0
	}
	return remaining
}
