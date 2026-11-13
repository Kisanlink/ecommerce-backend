package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"go.uber.org/zap"
)

// CouponServiceInterface defines the interface for coupon service operations
type CouponServiceInterface interface {
	CreateCoupon(ctx context.Context, req *requests.CreateCouponRequest) (*base.BaseResponse, error)
	GetCouponByCode(ctx context.Context, code string) (*base.BaseResponse, error)
	GetAllCoupons(ctx context.Context, limit, offset int) (*base.BaseResponse, error)
	GetActiveCoupons(ctx context.Context, limit, offset int) (*base.BaseResponse, error)
	UpdateCoupon(ctx context.Context, req *requests.UpdateCouponRequest) (*base.BaseResponse, error)
	DeleteCoupon(ctx context.Context, id string) (*base.BaseResponse, error)
	ValidateCoupon(ctx context.Context, code string, cartTotal float64, userID string) (*base.BaseResponse, error)
	ApplyCouponToUser(ctx context.Context, code string, userID string) (*base.BaseResponse, error)
	GetCouponUsageStats(ctx context.Context, code string) (*base.BaseResponse, error)
	CreateCouponRule(ctx context.Context, req *requests.CreateCouponRuleRequest) (*base.BaseResponse, error)
	UpdateCouponRule(ctx context.Context, req *requests.UpdateCouponRuleRequest) (*base.BaseResponse, error)
	DeleteCouponRule(ctx context.Context, id string) (*base.BaseResponse, error)
}

// CouponService handles business logic for coupons
type CouponService struct {
	couponRepo     repositories.CouponRepositoryInterface
	couponRuleRepo repositories.CouponRuleRepositoryInterface
	logger         *zap.Logger
}

// NewCouponService creates a new coupon service
func NewCouponService(couponRepo repositories.CouponRepositoryInterface, couponRuleRepo repositories.CouponRuleRepositoryInterface, logger *zap.Logger) CouponServiceInterface {
	service := &CouponService{
		couponRepo:     couponRepo,
		couponRuleRepo: couponRuleRepo,
		logger:         logger,
	}

	// Initialize ID generators from existing database records
	service.initializeCouponIDGenerator()
	service.initializeCouponRuleIDGenerator()

	return service
}

// CreateCoupon creates a new coupon
func (s *CouponService) CreateCoupon(ctx context.Context, req *requests.CreateCouponRequest) (*base.BaseResponse, error) {
	// Validate request
	if err := s.validateCreateCouponRequest(req); err != nil {
		return base.NewErrorResponse("Invalid request", base.NewValidationError("request", err.Error())), nil
	}

	// Check if coupon code already exists
	var existingCoupon models.Coupon
	_, err := s.couponRepo.GetByCode(ctx, req.Code, &existingCoupon)
	if err == nil {
		return base.NewErrorResponse("Coupon already exists", base.NewValidationError("code", "A coupon with this code already exists")), nil
	}

	// Create new coupon (without rules initially)
	coupon := models.NewCoupon(req.Code, req.Description, req.CreatedBy)
	coupon.Count = req.Count

	// Save coupon to database
	if err := s.couponRepo.Create(ctx, coupon); err != nil {
		s.logger.Error("Failed to create coupon",
			zap.String("code", req.Code),
			zap.Error(err))
		return base.NewErrorResponse("Failed to create coupon", base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Create coupon rule
	rule := models.NewCouponRule(coupon.ID, req.RuleType, req.Value)
	rule.SetMinAmount(req.MinAmount)
	if req.MaxDiscount > 0 {
		rule.SetMaxDiscount(req.MaxDiscount)
	}
	rule.SetValidityPeriod(req.ValidFrom, req.ValidUntil)
	if req.UserLimit > 0 {
		rule.SetUserLimit(req.UserLimit)
	}
	rule.SetFirstTimeOnly(req.FirstTimeOnly)

	// Add categories and products
	for _, category := range req.Categories {
		rule.AddCategory(category)
	}
	for _, product := range req.Products {
		rule.AddProduct(product)
	}

	// Save rule to database
	if err := s.couponRuleRepo.Create(ctx, rule); err != nil {
		s.logger.Error("Failed to create coupon rule",
			zap.String("coupon_id", coupon.ID),
			zap.Error(err))
		// Rollback coupon creation
		s.couponRepo.Delete(ctx, coupon.ID)
		return base.NewErrorResponse("Failed to create coupon rule", base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Add rule to coupon for response
	coupon.AddRule(rule)

	s.logger.Info("Coupon created successfully",
		zap.String("id", coupon.ID),
		zap.String("code", coupon.Code))

	return base.NewSuccessResponse("Coupon created successfully", coupon), nil
}

// GetCouponByCode retrieves a coupon by code
func (s *CouponService) GetCouponByCode(ctx context.Context, code string) (*base.BaseResponse, error) {
	if strings.TrimSpace(code) == "" {
		return base.NewErrorResponse("Invalid request", base.NewValidationError("code", "Coupon code is required")), nil
	}

	coupon, err := s.couponRepo.GetByCodeWithRules(ctx, code)
	if err != nil {
		s.logger.Error("Failed to get coupon by code",
			zap.String("code", code),
			zap.Error(err))
		return base.NewErrorResponse("Coupon not found", base.NewNotFoundError("Coupon", code)), nil
	}

	return base.NewSuccessResponse("Coupon retrieved successfully", coupon), nil
}

// GetAllCoupons retrieves all coupons with pagination
func (s *CouponService) GetAllCoupons(ctx context.Context, limit, offset int) (*base.BaseResponse, error) {
	coupons, err := s.couponRepo.GetAllWithRules(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get all coupons",
			zap.Error(err))
		return base.NewErrorResponse("Failed to retrieve coupons", base.NewInternalServerError("Database error", err.Error())), nil
	}

	response := map[string]interface{}{
		"coupons": coupons,
		"count":   len(coupons),
		"limit":   limit,
		"offset":  offset,
	}

	return base.NewSuccessResponse("Coupons retrieved successfully", response), nil
}

// GetActiveCoupons retrieves all active coupons with pagination
func (s *CouponService) GetActiveCoupons(ctx context.Context, limit, offset int) (*base.BaseResponse, error) {
	coupons, err := s.couponRepo.GetActiveWithRules(ctx, limit, offset)
	if err != nil {
		s.logger.Error("Failed to get active coupons",
			zap.Error(err))
		return base.NewErrorResponse("Failed to retrieve active coupons", base.NewInternalServerError("Database error", err.Error())), nil
	}

	response := map[string]interface{}{
		"coupons": coupons,
		"count":   len(coupons),
		"limit":   limit,
		"offset":  offset,
	}

	return base.NewSuccessResponse("Active coupons retrieved successfully", response), nil
}

// UpdateCoupon updates an existing coupon
func (s *CouponService) UpdateCoupon(ctx context.Context, req *requests.UpdateCouponRequest) (*base.BaseResponse, error) {
	// Get existing coupon with rules
	coupon, err := s.couponRepo.GetWithRules(ctx, req.ID)
	if err != nil {
		return base.NewErrorResponse("Coupon not found", base.NewNotFoundError("Coupon", req.ID)), nil
	}

	// Update coupon fields if provided
	if req.Description != "" {
		coupon.Description = req.Description
	}
	if req.Count > 0 {
		coupon.Count = req.Count
	}
	if req.IsActive != nil {
		coupon.IsActive = *req.IsActive
	}

	// Save updated coupon
	if err := s.couponRepo.Update(ctx, coupon); err != nil {
		s.logger.Error("Failed to update coupon",
			zap.String("id", req.ID),
			zap.Error(err))
		return base.NewErrorResponse("Failed to update coupon", base.NewInternalServerError("Database error", err.Error())), nil
	}

	// Note: Rule updates should be done through the separate rule management endpoints
	// This maintains separation of concerns and follows the normalized structure

	s.logger.Info("Coupon updated successfully",
		zap.String("id", coupon.ID),
		zap.String("code", coupon.Code))

	return base.NewSuccessResponse("Coupon updated successfully", coupon), nil
}

// DeleteCoupon deletes a coupon
func (s *CouponService) DeleteCoupon(ctx context.Context, id string) (*base.BaseResponse, error) {
	if strings.TrimSpace(id) == "" {
		return base.NewErrorResponse("Invalid request", base.NewValidationError("id", "Coupon ID is required")), nil
	}

	if err := s.couponRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete coupon",
			zap.String("id", id),
			zap.Error(err))
		return base.NewErrorResponse("Failed to delete coupon", base.NewInternalServerError("Database error", err.Error())), nil
	}

	s.logger.Info("Coupon deleted successfully",
		zap.String("id", id))

	return base.NewSuccessResponse("Coupon deleted successfully", nil), nil
}

// ValidateCoupon validates a coupon for a specific cart and user
func (s *CouponService) ValidateCoupon(ctx context.Context, code string, cartTotal float64, userID string) (*base.BaseResponse, error) {
	if strings.TrimSpace(code) == "" {
		return base.NewErrorResponse("Invalid request", base.NewValidationError("code", "Coupon code is required")), nil
	}

	// Get coupon with rules
	coupon, err := s.couponRepo.GetByCodeWithRules(ctx, code)
	if err != nil {
		return base.NewErrorResponse("Invalid coupon", base.NewNotFoundError("Coupon", code)), nil
	}

	// Check if user is first time (simplified check)
	isFirstTime := s.isFirstTimeUser(ctx, userID)

	// Validate coupon (without product restrictions for validation endpoint)
	if !coupon.CanUse(cartTotal, userID, isFirstTime) {
		return base.NewErrorResponse("Invalid coupon", base.NewValidationError("coupon", "Coupon cannot be used for this cart")), nil
	}

	// Get the best rule for this cart
	bestRule := coupon.GetBestRule(cartTotal, userID, isFirstTime)
	if bestRule == nil {
		return base.NewErrorResponse("Invalid coupon", base.NewValidationError("coupon", "No applicable rules found for this cart")), nil
	}

	// Calculate discount
	discount := bestRule.CalculateDiscount(cartTotal)

	response := map[string]interface{}{
		"coupon":        coupon,
		"best_rule":     bestRule,
		"discount":      discount,
		"is_valid":      true,
		"is_first_time": isFirstTime,
	}

	return base.NewSuccessResponse("Coupon is valid", response), nil
}

// ApplyCouponToUser validates that a coupon is applicable for a user. It does NOT increment the
// coupon's usage count: usage is owned solely by order creation (OrderService.ReduceCouponUsage),
// so a coupon counts exactly once per confirmed order. Incrementing here as well double-counted
// usage whenever an order followed an /coupons/apply call.
func (s *CouponService) ApplyCouponToUser(ctx context.Context, code string, userID string) (*base.BaseResponse, error) {
	// Get coupon with rules first to check if it exists and is valid
	coupon, err := s.couponRepo.GetByCodeWithRules(ctx, code)
	if err != nil {
		return base.NewErrorResponse("Invalid coupon", base.NewNotFoundError("Coupon", code)), nil
	}

	// Check if coupon is valid
	if !coupon.IsValid() {
		return base.NewErrorResponse("Coupon is not valid", base.NewValidationError("coupon", "Coupon is expired or inactive")), nil
	}

	// Check if coupon has remaining uses
	if coupon.Count > 0 && coupon.UsedCount >= coupon.Count {
		return base.NewErrorResponse("Coupon usage limit reached", base.NewValidationError("coupon", "Coupon has reached its usage limit")), nil
	}

	// Usage is intentionally NOT incremented here — see the doc comment. Order creation owns it.
	s.logger.Info("Coupon validated for user (usage counted at order creation)",
		zap.String("code", code),
		zap.String("user_id", userID))

	return base.NewSuccessResponse("Coupon applied successfully", coupon), nil
}

// GetCouponUsageStats returns usage statistics for a coupon
func (s *CouponService) GetCouponUsageStats(ctx context.Context, code string) (*base.BaseResponse, error) {
	if strings.TrimSpace(code) == "" {
		return base.NewErrorResponse("Invalid request", base.NewValidationError("code", "Coupon code is required")), nil
	}

	totalCount, usedCount, err := s.couponRepo.GetUsageStats(ctx, code)
	if err != nil {
		s.logger.Error("Failed to get coupon usage stats",
			zap.String("code", code),
			zap.Error(err))
		return base.NewErrorResponse("Failed to get usage statistics", base.NewInternalServerError("Database error", err.Error())), nil
	}

	response := map[string]interface{}{
		"code":       code,
		"total":      totalCount,
		"used":       usedCount,
		"remaining":  totalCount - usedCount,
		"usage_rate": float64(usedCount) / float64(totalCount) * 100,
	}

	return base.NewSuccessResponse("Usage statistics retrieved successfully", response), nil
}

// validateCreateCouponRequest validates the create coupon request
func (s *CouponService) validateCreateCouponRequest(req *requests.CreateCouponRequest) error {
	if strings.TrimSpace(req.Code) == "" {
		return fmt.Errorf("coupon code is required")
	}
	if strings.TrimSpace(req.Description) == "" {
		return fmt.Errorf("coupon description is required")
	}
	if req.RuleType == "" {
		return fmt.Errorf("rule type is required")
	}
	if req.Value <= 0 {
		return fmt.Errorf("rule value must be greater than 0")
	}
	if req.Count <= 0 {
		return fmt.Errorf("count must be greater than 0")
	}
	return nil
}

// CreateCouponRule creates a new coupon rule
func (s *CouponService) CreateCouponRule(ctx context.Context, req *requests.CreateCouponRuleRequest) (*base.BaseResponse, error) {
	// Validate request
	if err := s.validateCreateCouponRuleRequest(req); err != nil {
		return base.NewErrorResponse("Invalid request", base.NewValidationError("request", err.Error())), nil
	}

	// Check if coupon exists
	var coupon models.Coupon
	_, err := s.couponRepo.GetByID(ctx, req.CouponID, &coupon)
	if err != nil {
		return base.NewErrorResponse("Coupon not found", base.NewNotFoundError("Coupon", req.CouponID)), nil
	}

	// Create new coupon rule
	rule := models.NewCouponRule(req.CouponID, req.RuleType, req.DiscountValue)
	rule.SetMinAmount(req.MinAmount)
	if req.MaxDiscount > 0 {
		rule.SetMaxDiscount(req.MaxDiscount)
	}
	rule.SetValidityPeriod(req.ValidFrom, req.ValidUntil)
	if req.UserLimit > 0 {
		rule.SetUserLimit(req.UserLimit)
	}
	rule.SetFirstTimeOnly(req.FirstTimeOnly)

	// Add categories and products
	for _, category := range req.Categories {
		rule.AddCategory(category)
	}
	for _, product := range req.Products {
		rule.AddProduct(product)
	}

	// Save rule to database
	if err := s.couponRuleRepo.Create(ctx, rule); err != nil {
		s.logger.Error("Failed to create coupon rule",
			zap.String("coupon_id", req.CouponID),
			zap.Error(err))
		return base.NewErrorResponse("Failed to create coupon rule", base.NewInternalServerError("Database error", err.Error())), nil
	}

	s.logger.Info("Coupon rule created successfully",
		zap.String("rule_id", rule.ID),
		zap.String("coupon_id", req.CouponID))

	return base.NewSuccessResponse("Coupon rule created successfully", rule), nil
}

// UpdateCouponRule updates an existing coupon rule
func (s *CouponService) UpdateCouponRule(ctx context.Context, req *requests.UpdateCouponRuleRequest) (*base.BaseResponse, error) {
	// Get existing rule
	var rule models.CouponRule
	_, err := s.couponRuleRepo.GetByID(ctx, req.ID, &rule)
	if err != nil {
		return base.NewErrorResponse("Coupon rule not found", base.NewNotFoundError("CouponRule", req.ID)), nil
	}

	// Update fields if provided
	if req.RuleType != "" {
		rule.RuleType = req.RuleType
	}
	if req.DiscountValue > 0 {
		rule.DiscountValue = req.DiscountValue
	}
	if req.MinAmount >= 0 {
		rule.SetMinAmount(req.MinAmount)
	}
	if req.MaxDiscount >= 0 {
		rule.SetMaxDiscount(req.MaxDiscount)
	}
	if req.ValidFrom != nil {
		rule.ValidFrom = req.ValidFrom
	}
	if req.ValidUntil != nil {
		rule.ValidUntil = req.ValidUntil
	}
	if req.UserLimit >= 0 {
		rule.SetUserLimit(req.UserLimit)
	}
	if req.FirstTimeOnly != nil {
		rule.SetFirstTimeOnly(*req.FirstTimeOnly)
	}
	if req.Categories != nil {
		rule.Categories = req.Categories
	}
	if req.Products != nil {
		rule.Products = req.Products
	}

	// Save updated rule
	if err := s.couponRuleRepo.Update(ctx, &rule); err != nil {
		s.logger.Error("Failed to update coupon rule",
			zap.String("rule_id", req.ID),
			zap.Error(err))
		return base.NewErrorResponse("Failed to update coupon rule", base.NewInternalServerError("Database error", err.Error())), nil
	}

	s.logger.Info("Coupon rule updated successfully",
		zap.String("rule_id", rule.ID))

	return base.NewSuccessResponse("Coupon rule updated successfully", rule), nil
}

// DeleteCouponRule deletes a coupon rule
func (s *CouponService) DeleteCouponRule(ctx context.Context, id string) (*base.BaseResponse, error) {
	if strings.TrimSpace(id) == "" {
		return base.NewErrorResponse("Invalid request", base.NewValidationError("id", "Rule ID is required")), nil
	}

	if err := s.couponRuleRepo.Delete(ctx, id); err != nil {
		s.logger.Error("Failed to delete coupon rule",
			zap.String("id", id),
			zap.Error(err))
		return base.NewErrorResponse("Failed to delete coupon rule", base.NewInternalServerError("Database error", err.Error())), nil
	}

	s.logger.Info("Coupon rule deleted successfully",
		zap.String("id", id))

	return base.NewSuccessResponse("Coupon rule deleted successfully", nil), nil
}

// validateCreateCouponRuleRequest validates the create coupon rule request
func (s *CouponService) validateCreateCouponRuleRequest(req *requests.CreateCouponRuleRequest) error {
	if strings.TrimSpace(req.CouponID) == "" {
		return fmt.Errorf("coupon ID is required")
	}
	if req.RuleType == "" {
		return fmt.Errorf("rule type is required")
	}
	if req.DiscountValue <= 0 {
		return fmt.Errorf("discount value must be greater than 0")
	}
	return nil
}

// isFirstTimeUser checks if the user is a first-time user (simplified implementation)
func (s *CouponService) isFirstTimeUser(ctx context.Context, userID string) bool {
	// This is a simplified check - in a real implementation, you might want to check
	// if the user has any previous orders or coupon usage history
	// For now, we'll assume all users are first-time users
	return true
}

// initializeCouponIDGenerator initializes the coupon ID generator from existing database records
func (s *CouponService) initializeCouponIDGenerator() {
	ctx := context.Background()
	s.logger.Info("Initializing coupon ID generator")

	// Query all existing coupon IDs to prevent conflicts (exclude soft-deleted)
	coupons, err := s.couponRepo.GetAll(ctx, 10000, 0)
	if err != nil {
		s.logger.Error("Failed to get existing coupons for ID initialization", zap.Error(err))
		// Fallback to empty initialization
		hash.InitializeGlobalCountersFromDatabase("COUP", []string{}, hash.Medium)
	} else {
		// Extract IDs
		existingIDs := make([]string, len(coupons))
		for i, coupon := range coupons {
			if coupon != nil {
				existingIDs[i] = coupon.GetID()
			}
		}
		s.logger.Info("Found existing coupon IDs", zap.Int("count", len(existingIDs)))
		hash.InitializeGlobalCountersFromDatabase("COUP", existingIDs, hash.Medium)
	}

	s.logger.Info("Coupon ID generator initialized")
}

// initializeCouponRuleIDGenerator initializes the coupon rule ID generator from existing database records
func (s *CouponService) initializeCouponRuleIDGenerator() {
	ctx := context.Background()
	s.logger.Info("Initializing coupon rule ID generator")

	// Query all existing coupon rule IDs to prevent conflicts
	rules, err := s.couponRuleRepo.GetAll(ctx, 10000, 0)
	if err != nil {
		s.logger.Error("Failed to get existing coupon rules for ID initialization", zap.Error(err))
		// Fallback to empty initialization
		hash.InitializeGlobalCountersFromDatabase("RULE", []string{}, hash.Medium)
	} else {
		// Extract IDs
		existingIDs := make([]string, len(rules))
		for i, rule := range rules {
			if rule != nil {
				existingIDs[i] = rule.GetID()
			}
		}
		s.logger.Info("Found existing coupon rule IDs", zap.Int("count", len(existingIDs)))
		hash.InitializeGlobalCountersFromDatabase("RULE", existingIDs, hash.Medium)
	}

	s.logger.Info("Coupon rule ID generator initialized")
}
