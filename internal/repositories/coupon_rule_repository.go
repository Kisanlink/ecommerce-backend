package repositories

import (
	"context"
	"fmt"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// CouponRuleRepositoryInterface defines the interface for coupon rule repository operations
type CouponRuleRepositoryInterface interface {
	Create(ctx context.Context, rule *models.CouponRule) error
	GetByID(ctx context.Context, id string, rule *models.CouponRule) (*models.CouponRule, error)
	GetByCouponID(ctx context.Context, couponID string) ([]*models.CouponRule, error)
	GetByCouponIDAndType(ctx context.Context, couponID string, ruleType models.CouponRuleType) ([]*models.CouponRule, error)
	GetAll(ctx context.Context, limit, offset int) ([]*models.CouponRule, error)
	GetActive(ctx context.Context, limit, offset int) ([]*models.CouponRule, error)
	Update(ctx context.Context, rule *models.CouponRule) error
	Delete(ctx context.Context, id string) error
	DeleteByCouponID(ctx context.Context, couponID string) error
	GetValidRulesForCart(ctx context.Context, cartTotal float64, userID string, isFirstTime bool) ([]*models.CouponRule, error)
	GetRulesByCategory(ctx context.Context, category string) ([]*models.CouponRule, error)
	GetRulesByProduct(ctx context.Context, productID string) ([]*models.CouponRule, error)
}

// CouponRuleRepository handles database operations for coupon rules
type CouponRuleRepository struct {
	*base.BaseFilterableRepository[*models.CouponRule]
	dbManager db.DBManager
	logger    *zap.Logger
}

// NewCouponRuleRepository creates a new coupon rule repository
func NewCouponRuleRepository(dbManager db.DBManager, logger *zap.Logger) CouponRuleRepositoryInterface {
	repo := &CouponRuleRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.CouponRule](),
		dbManager:                dbManager,
		logger:                   logger,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Create creates a new coupon rule
func (r *CouponRuleRepository) Create(ctx context.Context, rule *models.CouponRule) error {
	return r.dbManager.Create(ctx, rule)
}

// GetByID retrieves a coupon rule by ID
func (r *CouponRuleRepository) GetByID(ctx context.Context, id string, rule *models.CouponRule) (*models.CouponRule, error) {
	if err := r.dbManager.GetByID(ctx, id, rule); err != nil {
		return nil, err
	}
	return rule, nil
}

// GetByCouponID retrieves all rules for a specific coupon
func (r *CouponRuleRepository) GetByCouponID(ctx context.Context, couponID string) ([]*models.CouponRule, error) {
	filter := base.NewFilterBuilder().
		Where("coupon_id", base.OpEqual, couponID).
		Sort("created_at", "asc").
		Build()

	return r.BaseFilterableRepository.Find(ctx, filter)
}

// GetByCouponIDAndType retrieves rules for a specific coupon and type
func (r *CouponRuleRepository) GetByCouponIDAndType(ctx context.Context, couponID string, ruleType models.CouponRuleType) ([]*models.CouponRule, error) {
	filter := base.NewFilterBuilder().
		Where("coupon_id", base.OpEqual, couponID).
		Where("rule_type", base.OpEqual, string(ruleType)).
		Sort("created_at", "asc").
		Build()

	return r.BaseFilterableRepository.Find(ctx, filter)
}

// GetAll retrieves all coupon rules with pagination
func (r *CouponRuleRepository) GetAll(ctx context.Context, limit, offset int) ([]*models.CouponRule, error) {
	filter := base.NewFilterBuilder().
		Sort("created_at", "desc").
		Page(offset/limit+1, limit).
		Build()

	return r.BaseFilterableRepository.Find(ctx, filter)
}

// GetActive retrieves all active coupon rules with pagination
func (r *CouponRuleRepository) GetActive(ctx context.Context, limit, offset int) ([]*models.CouponRule, error) {
	now := time.Now()
	filter := base.NewFilterBuilder().
		Where("valid_from", base.OpLessEqual, now).
		Where("valid_until", base.OpGreaterEqual, now).
		Sort("created_at", "desc").
		Page(offset/limit+1, limit).
		Build()

	return r.BaseFilterableRepository.Find(ctx, filter)
}

// Update updates an existing coupon rule
func (r *CouponRuleRepository) Update(ctx context.Context, rule *models.CouponRule) error {
	return r.dbManager.Update(ctx, rule)
}

// Delete soft deletes a coupon rule
func (r *CouponRuleRepository) Delete(ctx context.Context, id string) error {
	return r.dbManager.Delete(ctx, id, &models.CouponRule{})
}

// DeleteByCouponID deletes all rules for a specific coupon
func (r *CouponRuleRepository) DeleteByCouponID(ctx context.Context, couponID string) error {
	// Get all rules for the coupon first
	rules, err := r.GetByCouponID(ctx, couponID)
	if err != nil {
		return fmt.Errorf("failed to get rules for coupon: %w", err)
	}

	// Delete each rule
	for _, rule := range rules {
		if err := r.dbManager.Delete(ctx, rule.ID, &models.CouponRule{}); err != nil {
			r.logger.Error("Failed to delete coupon rule",
				zap.String("rule_id", rule.ID),
				zap.String("coupon_id", couponID),
				zap.Error(err))
			return fmt.Errorf("failed to delete rule %s: %w", rule.ID, err)
		}
	}

	return nil
}

// GetValidRulesForCart retrieves rules that are valid for a specific cart
func (r *CouponRuleRepository) GetValidRulesForCart(ctx context.Context, cartTotal float64, userID string, isFirstTime bool) ([]*models.CouponRule, error) {
	now := time.Now()

	// Build filter for valid rules
	filter := base.NewFilterBuilder().
		Where("valid_from", base.OpLessEqual, now).
		Where("valid_until", base.OpGreaterEqual, now).
		Where("min_amount", base.OpLessEqual, cartTotal).
		Sort("discount_value", "desc").
		Build()

	rules, err := r.BaseFilterableRepository.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get valid rules: %w", err)
	}

	// Filter rules based on additional criteria
	var validRules []*models.CouponRule
	for _, rule := range rules {
		if rule.CanApplyToCart(cartTotal, userID, isFirstTime) {
			validRules = append(validRules, rule)
		}
	}

	return validRules, nil
}

// GetRulesByCategory retrieves rules that apply to a specific category
func (r *CouponRuleRepository) GetRulesByCategory(ctx context.Context, category string) ([]*models.CouponRule, error) {
	// This would require a more complex query to check JSON array contents
	// For now, we'll get all rules and filter in memory
	// In a production system, you might want to use PostgreSQL's JSON operators
	filter := base.NewFilterBuilder().
		Where("categories", base.OpNotEqual, nil).
		Sort("created_at", "desc").
		Build()

	allRules, err := r.BaseFilterableRepository.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules by category: %w", err)
	}

	var categoryRules []*models.CouponRule
	for _, rule := range allRules {
		if rule.AppliesToCategory(category) {
			categoryRules = append(categoryRules, rule)
		}
	}

	return categoryRules, nil
}

// GetRulesByProduct retrieves rules that apply to a specific product
func (r *CouponRuleRepository) GetRulesByProduct(ctx context.Context, productID string) ([]*models.CouponRule, error) {
	// This would require a more complex query to check JSON array contents
	// For now, we'll get all rules and filter in memory
	// In a production system, you might want to use PostgreSQL's JSON operators
	filter := base.NewFilterBuilder().
		Where("products", base.OpNotEqual, nil).
		Sort("created_at", "desc").
		Build()

	allRules, err := r.BaseFilterableRepository.Find(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get rules by product: %w", err)
	}

	var productRules []*models.CouponRule
	for _, rule := range allRules {
		if rule.AppliesToProduct(productID) {
			productRules = append(productRules, rule)
		}
	}

	return productRules, nil
}
