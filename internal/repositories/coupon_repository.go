package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// CouponRepositoryInterface defines the interface for coupon repository operations
type CouponRepositoryInterface interface {
	Create(ctx context.Context, coupon *models.Coupon) error
	GetByID(ctx context.Context, id string, coupon *models.Coupon) (*models.Coupon, error)
	GetByCode(ctx context.Context, code string, coupon *models.Coupon) (*models.Coupon, error)
	GetAll(ctx context.Context, limit, offset int) ([]*models.Coupon, error)
	GetActive(ctx context.Context, limit, offset int) ([]*models.Coupon, error)
	Update(ctx context.Context, coupon *models.Coupon) error
	Delete(ctx context.Context, id string) error
	IncrementUsage(ctx context.Context, code string) error
	GetUsageStats(ctx context.Context, code string) (int, int, error)
	GetWithRules(ctx context.Context, id string) (*models.Coupon, error)
	GetByCodeWithRules(ctx context.Context, code string) (*models.Coupon, error)
	GetAllWithRules(ctx context.Context, limit, offset int) ([]*models.Coupon, error)
	GetActiveWithRules(ctx context.Context, limit, offset int) ([]*models.Coupon, error)
}

// CouponRepository handles database operations for coupons
type CouponRepository struct {
	*base.BaseFilterableRepository[*models.Coupon]
	dbManager db.DBManager
	logger    *zap.Logger
}

// NewCouponRepository creates a new coupon repository
func NewCouponRepository(dbManager db.DBManager, logger *zap.Logger) CouponRepositoryInterface {
	repo := &CouponRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Coupon](),
		dbManager:                dbManager,
		logger:                   logger,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Create creates a new coupon
func (r *CouponRepository) Create(ctx context.Context, coupon *models.Coupon) error {
	return r.dbManager.Create(ctx, coupon)
}

// GetByID retrieves a coupon by ID
func (r *CouponRepository) GetByID(ctx context.Context, id string, coupon *models.Coupon) (*models.Coupon, error) {
	if err := r.dbManager.GetByID(ctx, id, coupon); err != nil {
		return nil, err
	}
	return coupon, nil
}

// GetByCode retrieves a coupon by code
func (r *CouponRepository) GetByCode(ctx context.Context, code string, coupon *models.Coupon) (*models.Coupon, error) {
	// Use filter to find coupon by code
	filter := base.NewFilterBuilder().
		Where("code", base.OpEqual, code).
		Build()

	coupons, err := r.BaseFilterableRepository.Find(ctx, filter)
	if err != nil {
		r.logger.Error("Failed to get coupon by code",
			zap.String("code", code),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get coupon: %w", err)
	}

	if len(coupons) == 0 {
		return nil, fmt.Errorf("coupon not found")
	}

	*coupon = *coupons[0]
	return coupon, nil
}

// GetAll retrieves all coupons with pagination
func (r *CouponRepository) GetAll(ctx context.Context, limit, offset int) ([]*models.Coupon, error) {
	// Use filter with pagination
	filter := base.NewFilterBuilder().
		Sort("created_at", "desc").
		Page(offset/limit+1, limit).
		Build()

	return r.BaseFilterableRepository.Find(ctx, filter)
}

// GetActive retrieves all active coupons with pagination
func (r *CouponRepository) GetActive(ctx context.Context, limit, offset int) ([]*models.Coupon, error) {
	// Use filter to find active coupons with pagination
	filter := base.NewFilterBuilder().
		Where("is_active", base.OpEqual, true).
		Sort("created_at", "desc").
		Page(offset/limit+1, limit).
		Build()

	return r.BaseFilterableRepository.Find(ctx, filter)
}

// Update updates an existing coupon
func (r *CouponRepository) Update(ctx context.Context, coupon *models.Coupon) error {
	return r.dbManager.Update(ctx, coupon)
}

// Delete soft deletes a coupon
func (r *CouponRepository) Delete(ctx context.Context, id string) error {
	return r.dbManager.Delete(ctx, id, &models.Coupon{})
}

// IncrementUsage increments the usage count of a coupon
func (r *CouponRepository) IncrementUsage(ctx context.Context, code string) error {
	// First get the coupon to update its usage count
	var coupon models.Coupon
	_, err := r.GetByCode(ctx, code, &coupon)
	if err != nil {
		return fmt.Errorf("coupon not found: %w", err)
	}

	// Increment the usage count
	coupon.UsedCount++

	// Update the coupon
	return r.dbManager.Update(ctx, &coupon)
}

// GetUsageStats returns the usage statistics for a coupon
func (r *CouponRepository) GetUsageStats(ctx context.Context, code string) (int, int, error) {
	// Get the coupon to retrieve usage stats
	var coupon models.Coupon
	_, err := r.GetByCode(ctx, code, &coupon)
	if err != nil {
		return 0, 0, fmt.Errorf("coupon not found: %w", err)
	}

	return coupon.Count, coupon.UsedCount, nil
}

// GetWithRules retrieves a coupon by ID with its rules
func (r *CouponRepository) GetWithRules(ctx context.Context, id string) (*models.Coupon, error) {
	var coupon models.Coupon
	if err := r.dbManager.GetByID(ctx, id, &coupon); err != nil {
		return nil, err
	}

	// Load rules for the coupon
	ruleRepo := NewCouponRuleRepository(r.dbManager, r.logger)
	rules, err := ruleRepo.GetByCouponID(ctx, id)
	if err != nil {
		r.logger.Error("Failed to get rules for coupon",
			zap.String("coupon_id", id),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Convert to slice of CouponRule (not pointers)
	coupon.Rules = make([]models.CouponRule, len(rules))
	for i, rule := range rules {
		coupon.Rules[i] = *rule
	}

	return &coupon, nil
}

// GetByCodeWithRules retrieves a coupon by code with its rules
func (r *CouponRepository) GetByCodeWithRules(ctx context.Context, code string) (*models.Coupon, error) {
	// First get the coupon by code
	var coupon models.Coupon
	_, err := r.GetByCode(ctx, code, &coupon)
	if err != nil {
		return nil, err
	}

	// Load rules for the coupon
	ruleRepo := NewCouponRuleRepository(r.dbManager, r.logger)
	rules, err := ruleRepo.GetByCouponID(ctx, coupon.ID)
	if err != nil {
		r.logger.Error("Failed to get rules for coupon",
			zap.String("coupon_id", coupon.ID),
			zap.String("code", code),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get rules: %w", err)
	}

	// Convert to slice of CouponRule (not pointers)
	coupon.Rules = make([]models.CouponRule, len(rules))
	for i, rule := range rules {
		coupon.Rules[i] = *rule
	}

	return &coupon, nil
}

// GetAllWithRules retrieves all coupons with their rules
func (r *CouponRepository) GetAllWithRules(ctx context.Context, limit, offset int) ([]*models.Coupon, error) {
	// Get all coupons first
	coupons, err := r.GetAll(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Load rules for each coupon
	ruleRepo := NewCouponRuleRepository(r.dbManager, r.logger)
	for _, coupon := range coupons {
		rules, err := ruleRepo.GetByCouponID(ctx, coupon.ID)
		if err != nil {
			r.logger.Error("Failed to get rules for coupon",
				zap.String("coupon_id", coupon.ID),
				zap.Error(err))
			continue
		}

		// Convert to slice of CouponRule (not pointers)
		coupon.Rules = make([]models.CouponRule, len(rules))
		for i, rule := range rules {
			coupon.Rules[i] = *rule
		}
	}

	return coupons, nil
}

// GetActiveWithRules retrieves all active coupons with their rules
func (r *CouponRepository) GetActiveWithRules(ctx context.Context, limit, offset int) ([]*models.Coupon, error) {
	// Get active coupons first
	coupons, err := r.GetActive(ctx, limit, offset)
	if err != nil {
		return nil, err
	}

	// Load rules for each coupon
	ruleRepo := NewCouponRuleRepository(r.dbManager, r.logger)
	for _, coupon := range coupons {
		rules, err := ruleRepo.GetByCouponID(ctx, coupon.ID)
		if err != nil {
			r.logger.Error("Failed to get rules for coupon",
				zap.String("coupon_id", coupon.ID),
				zap.Error(err))
			continue
		}

		// Convert to slice of CouponRule (not pointers)
		coupon.Rules = make([]models.CouponRule, len(rules))
		for i, rule := range rules {
			coupon.Rules[i] = *rule
		}
	}

	return coupons, nil
}
