package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
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
}

// CouponService handles business logic for coupons
type CouponService struct {
	couponRepo repositories.CouponRepositoryInterface
	logger     *zap.Logger
}

// NewCouponService creates a new coupon service
func NewCouponService(couponRepo repositories.CouponRepositoryInterface, logger *zap.Logger) CouponServiceInterface {
	return &CouponService{
		couponRepo: couponRepo,
		logger:     logger,
	}
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

	// Create coupon rules
	rules := models.CouponRule{
		Type:          req.RuleType,
		DiscountValue: req.Value,
		MinAmount:     req.MinAmount,
		MaxDiscount:   req.MaxDiscount,
		Categories:    req.Categories,
		Products:      req.Products,
		ValidFrom:     req.ValidFrom,
		ValidUntil:    req.ValidUntil,
		UserLimit:     req.UserLimit,
		FirstTimeOnly: req.FirstTimeOnly,
	}

	// Create new coupon
	coupon := models.NewCoupon(req.Code, req.Description, req.CreatedBy, rules)
	coupon.Count = req.Count

	// Save to database
	if err := s.couponRepo.Create(ctx, coupon); err != nil {
		s.logger.Error("Failed to create coupon",
			zap.String("code", req.Code),
			zap.Error(err))
		return base.NewErrorResponse("Failed to create coupon", base.NewInternalServerError("Database error", err.Error())), nil
	}

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

	var coupon models.Coupon
	_, err := s.couponRepo.GetByCode(ctx, code, &coupon)
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
	coupons, err := s.couponRepo.GetAll(ctx, limit, offset)
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
	coupons, err := s.couponRepo.GetActive(ctx, limit, offset)
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
	// Get existing coupon
	var coupon models.Coupon
	_, err := s.couponRepo.GetByID(ctx, req.ID, &coupon)
	if err != nil {
		return base.NewErrorResponse("Coupon not found", base.NewNotFoundError("Coupon", req.ID)), nil
	}

	// Update fields if provided
	if req.Description != "" {
		coupon.Description = req.Description
	}
	if req.Count > 0 {
		coupon.Count = req.Count
	}
	if req.IsActive != nil {
		coupon.IsActive = *req.IsActive
	}

	// Update rules if provided
	if req.RuleType != "" {
		coupon.Rules.Type = req.RuleType
	}
	if req.Value > 0 {
		coupon.Rules.DiscountValue = req.Value
	}
	if req.MinAmount >= 0 {
		coupon.Rules.MinAmount = req.MinAmount
	}
	if req.MaxDiscount >= 0 {
		coupon.Rules.MaxDiscount = req.MaxDiscount
	}
	if req.Categories != nil {
		coupon.Rules.Categories = req.Categories
	}
	if req.Products != nil {
		coupon.Rules.Products = req.Products
	}
	if req.ValidFrom != nil {
		coupon.Rules.ValidFrom = req.ValidFrom
	}
	if req.ValidUntil != nil {
		coupon.Rules.ValidUntil = req.ValidUntil
	}
	if req.UserLimit >= 0 {
		coupon.Rules.UserLimit = req.UserLimit
	}
	if req.FirstTimeOnly != nil {
		coupon.Rules.FirstTimeOnly = *req.FirstTimeOnly
	}

	// Save updated coupon
	if err := s.couponRepo.Update(ctx, &coupon); err != nil {
		s.logger.Error("Failed to update coupon",
			zap.String("id", req.ID),
			zap.Error(err))
		return base.NewErrorResponse("Failed to update coupon", base.NewInternalServerError("Database error", err.Error())), nil
	}

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

	// Get coupon
	var coupon models.Coupon
	_, err := s.couponRepo.GetByCode(ctx, code, &coupon)
	if err != nil {
		return base.NewErrorResponse("Invalid coupon", base.NewNotFoundError("Coupon", code)), nil
	}

	// Check if user is first time (simplified check)
	isFirstTime := s.isFirstTimeUser(ctx, userID)

	// Validate coupon
	if !coupon.CanUse(cartTotal, userID, isFirstTime) {
		return base.NewErrorResponse("Invalid coupon", base.NewValidationError("coupon", "Coupon cannot be used for this cart")), nil
	}

	// Calculate discount
	discount := coupon.CalculateDiscount(cartTotal)

	response := map[string]interface{}{
		"coupon":        coupon,
		"discount":      discount,
		"is_valid":      true,
		"is_first_time": isFirstTime,
	}

	return base.NewSuccessResponse("Coupon is valid", response), nil
}

// ApplyCouponToUser applies a coupon and increments usage count
func (s *CouponService) ApplyCouponToUser(ctx context.Context, code string, userID string) (*base.BaseResponse, error) {
	// Get coupon first to check if it exists and is valid
	var coupon models.Coupon
	_, err := s.couponRepo.GetByCode(ctx, code, &coupon)
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

	// Increment usage count
	if err := s.couponRepo.IncrementUsage(ctx, code); err != nil {
		s.logger.Error("Failed to increment coupon usage",
			zap.String("code", code),
			zap.String("user_id", userID),
			zap.Error(err))
		return base.NewErrorResponse("Failed to apply coupon", base.NewInternalServerError("Database error", err.Error())), nil
	}

	s.logger.Info("Coupon applied successfully",
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

// isFirstTimeUser checks if the user is a first-time user (simplified implementation)
func (s *CouponService) isFirstTimeUser(ctx context.Context, userID string) bool {
	// This is a simplified check - in a real implementation, you might want to check
	// if the user has any previous orders or coupon usage history
	// For now, we'll assume all users are first-time users
	return true
}
