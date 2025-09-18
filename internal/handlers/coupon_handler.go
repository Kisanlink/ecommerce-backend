package handlers

import (
	"net/http"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// CouponHandler handles HTTP requests for coupon operations
type CouponHandler struct {
	couponService services.CouponServiceInterface
	logger        *zap.Logger
}

// NewCouponHandler creates a new coupon handler
func NewCouponHandler(couponService services.CouponServiceInterface, logger *zap.Logger) *CouponHandler {
	return &CouponHandler{
		couponService: couponService,
		logger:        logger,
	}
}

// CreateCoupon creates a new coupon
// @Summary Create a new coupon
// @Description Create a new discount coupon with rules
// @Tags coupons
// @Accept json
// @Produce json
// @Param request body requests.CreateCouponRequest true "Coupon creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/coupons [post]
// @Security BearerAuth
func (h *CouponHandler) CreateCoupon(c *gin.Context) {
	var req requests.CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid create coupon request",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	response, err := h.couponService.CreateCoupon(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create coupon",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create coupon",
		})
		return
	}

	statusCode := http.StatusOK
	if response.Success {
		statusCode = http.StatusCreated
	}

	c.JSON(statusCode, response)
}

// GetCouponByCode retrieves a coupon by code
// @Summary Get coupon by code
// @Description Retrieve a coupon by its code
// @Tags coupons
// @Accept json
// @Produce json
// @Param code path string true "Coupon code"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/coupons/{code} [get]
func (h *CouponHandler) GetCouponByCode(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Coupon code is required",
		})
		return
	}

	response, err := h.couponService.GetCouponByCode(c.Request.Context(), code)
	if err != nil {
		h.logger.Error("Failed to get coupon by code",
			zap.String("code", code),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve coupon",
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusNotFound
	}

	c.JSON(statusCode, response)
}

// GetAllCoupons retrieves all coupons with pagination
// @Summary Get all coupons
// @Description Retrieve all coupons with pagination
// @Tags coupons
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/coupons [get]
// @Security BearerAuth
func (h *CouponHandler) GetAllCoupons(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	response, err := h.couponService.GetAllCoupons(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get all coupons",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve coupons",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetActiveCoupons retrieves all active coupons with pagination
// @Summary Get active coupons
// @Description Retrieve all active coupons with pagination
// @Tags coupons
// @Accept json
// @Produce json
// @Param limit query int false "Limit" default(10)
// @Param offset query int false "Offset" default(0)
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/coupons/active [get]
func (h *CouponHandler) GetActiveCoupons(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 0 {
		limit = 10
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	response, err := h.couponService.GetActiveCoupons(c.Request.Context(), limit, offset)
	if err != nil {
		h.logger.Error("Failed to get active coupons",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve active coupons",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// UpdateCoupon updates an existing coupon
// @Summary Update coupon
// @Description Update an existing coupon
// @Tags coupons
// @Accept json
// @Produce json
// @Param request body requests.UpdateCouponRequest true "Coupon update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/coupons [put]
// @Security BearerAuth
func (h *CouponHandler) UpdateCoupon(c *gin.Context) {
	couponId := c.Param("id")
	if couponId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Coupon ID is required",
		})
		return
	}

	var req requests.UpdateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid update coupon request",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Set the coupon ID from URL parameter
	req.ID = couponId

	response, err := h.couponService.UpdateCoupon(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to update coupon",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update coupon",
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusNotFound
	}

	c.JSON(statusCode, response)
}

// DeleteCoupon deletes a coupon
// @Summary Delete coupon
// @Description Delete a coupon by ID
// @Tags coupons
// @Accept json
// @Produce json
// @Param id path string true "Coupon ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/coupons/{id} [delete]
// @Security BearerAuth
func (h *CouponHandler) DeleteCoupon(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Coupon ID is required",
		})
		return
	}

	response, err := h.couponService.DeleteCoupon(c.Request.Context(), id)
	if err != nil {
		h.logger.Error("Failed to delete coupon",
			zap.String("id", id),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete coupon",
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ValidateCoupon validates a coupon for a specific cart and user
// @Summary Validate coupon
// @Description Validate a coupon for a specific cart total and user
// @Tags coupons
// @Accept json
// @Produce json
// @Param request body requests.ValidateCouponRequest true "Coupon validation request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/coupons/validate [post]
func (h *CouponHandler) ValidateCoupon(c *gin.Context) {
	var req requests.ValidateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid validate coupon request",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	response, err := h.couponService.ValidateCoupon(c.Request.Context(), req.Code, req.CartTotal, req.UserID)
	if err != nil {
		h.logger.Error("Failed to validate coupon",
			zap.String("code", req.Code),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to validate coupon",
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, response)
}

// ApplyCouponToUser applies a coupon and increments usage count
// @Summary Apply coupon to user
// @Description Apply a coupon to a user and increment its usage count
// @Tags coupons
// @Accept json
// @Produce json
// @Param request body requests.ApplyCouponToUserRequest true "Coupon application request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/coupons/apply [post]
// @Security BearerAuth
func (h *CouponHandler) ApplyCouponToUser(c *gin.Context) {
	var req requests.ApplyCouponToUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid apply coupon request",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	response, err := h.couponService.ApplyCouponToUser(c.Request.Context(), req.Code, req.UserID)
	if err != nil {
		h.logger.Error("Failed to apply coupon",
			zap.String("code", req.Code),
			zap.String("user_id", req.UserID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to apply coupon",
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusBadRequest
	}

	c.JSON(statusCode, response)
}

// GetCouponUsageStats returns usage statistics for a coupon
// @Summary Get coupon usage statistics
// @Description Get usage statistics for a specific coupon
// @Tags coupons
// @Accept json
// @Produce json
// @Param code path string true "Coupon code"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Router /api/v1/coupons/{code}/stats [get]
// @Security BearerAuth
func (h *CouponHandler) GetCouponUsageStats(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Coupon code is required",
		})
		return
	}

	response, err := h.couponService.GetCouponUsageStats(c.Request.Context(), code)
	if err != nil {
		h.logger.Error("Failed to get coupon usage stats",
			zap.String("code", code),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve usage statistics",
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusNotFound
	}

	c.JSON(statusCode, response)
}

// CreateCouponRule creates a new coupon rule
// @Summary Create a new coupon rule
// @Description Create a new rule for an existing coupon
// @Tags coupons
// @Accept json
// @Produce json
// @Param request body requests.CreateCouponRuleRequest true "Coupon rule creation request"
// @Success 201 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/coupons/rules [post]
// @Security BearerAuth
func (h *CouponHandler) CreateCouponRule(c *gin.Context) {
	couponId := c.Param("couponId")
	if couponId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Coupon ID is required",
		})
		return
	}

	var req requests.CreateCouponRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid create coupon rule request",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Set the coupon ID from URL parameter
	req.CouponID = couponId

	response, err := h.couponService.CreateCouponRule(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to create coupon rule",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to create coupon rule",
		})
		return
	}

	statusCode := http.StatusOK
	if response.Success {
		statusCode = http.StatusCreated
	}

	c.JSON(statusCode, response)
}

// UpdateCouponRule updates an existing coupon rule
// @Summary Update coupon rule
// @Description Update an existing coupon rule
// @Tags coupons
// @Accept json
// @Produce json
// @Param request body requests.UpdateCouponRuleRequest true "Coupon rule update request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/coupons/rules [put]
// @Security BearerAuth
func (h *CouponHandler) UpdateCouponRule(c *gin.Context) {
	couponId := c.Param("couponId")
	ruleId := c.Param("ruleId")

	if couponId == "" || ruleId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Coupon ID and Rule ID are required",
		})
		return
	}

	var req requests.UpdateCouponRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid update coupon rule request",
			zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Set the rule ID from URL parameter
	req.ID = ruleId

	response, err := h.couponService.UpdateCouponRule(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Failed to update coupon rule",
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to update coupon rule",
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusNotFound
	}

	c.JSON(statusCode, response)
}

// DeleteCouponRule deletes a coupon rule
// @Summary Delete coupon rule
// @Description Delete an existing coupon rule
// @Tags coupons
// @Accept json
// @Produce json
// @Param id path string true "Coupon rule ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]interface{}
// @Failure 404 {object} map[string]interface{}
// @Failure 500 {object} map[string]interface{}
// @Router /api/v1/coupons/rules/{id} [delete]
// @Security BearerAuth
func (h *CouponHandler) DeleteCouponRule(c *gin.Context) {
	couponId := c.Param("couponId")
	ruleId := c.Param("ruleId")

	if couponId == "" || ruleId == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Coupon ID and Rule ID are required",
		})
		return
	}

	response, err := h.couponService.DeleteCouponRule(c.Request.Context(), ruleId)
	if err != nil {
		h.logger.Error("Failed to delete coupon rule",
			zap.String("couponId", couponId),
			zap.String("ruleId", ruleId),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to delete coupon rule",
		})
		return
	}

	statusCode := http.StatusOK
	if !response.Success {
		statusCode = http.StatusNotFound
	}

	c.JSON(statusCode, response)
}
