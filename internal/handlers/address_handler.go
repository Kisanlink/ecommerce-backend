package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type AddressHandler struct {
	addressService services.AddressServiceInterface
	authContext    *auth.AuthContext
	logger         *zap.Logger
}

func NewAddressHandler(addressService services.AddressServiceInterface, logger *zap.Logger) *AddressHandler {
	return &AddressHandler{
		addressService: addressService,
		authContext:    auth.NewAuthContext(),
		logger:         logger,
	}
}

// GetUserAddresses gets all saved addresses for a user
// GET /api/v1/addresses
// @Summary Get user's saved addresses
// @Description Get all saved addresses for the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Addresses retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to get addresses"
// @Router /addresses [get]
func (h *AddressHandler) GetUserAddresses(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addresses, err := h.addressService.GetUserAddresses(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user addresses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get addresses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"addresses": addresses,
		"count":     len(addresses),
	})
}

// GetLast4Addresses gets the last 4 addresses used by the user
// GET /api/v1/addresses/history
// @Summary Get user's last 4 addresses
// @Description Get the last 4 addresses used by the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Addresses retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to get address history"
// @Router /addresses/history [get]
func (h *AddressHandler) GetLast4Addresses(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addresses, err := h.addressService.GetLast4Addresses(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get last 4 addresses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get address history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"addresses": addresses,
		"count":     len(addresses),
		"max_count": 4,
	})
}

// GetAddressHistory gets address history from previous orders
// GET /api/v1/addresses/order-history
// @Summary Get address history from orders
// @Description Get all unique addresses used in previous orders
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Address history retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to get address history"
// @Router /addresses/order-history [get]
func (h *AddressHandler) GetAddressHistory(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addresses, err := h.addressService.GetAddressHistoryFromOrders(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get address history from orders", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get address history"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"addresses": addresses,
		"count":     len(addresses),
	})
}

// GetAllAddresses gets both saved addresses and address history
// GET /api/v1/addresses/all
// @Summary Get all user addresses
// @Description Get both saved addresses and address history for the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "All addresses retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to get all addresses"
// @Router /addresses/all [get]
func (h *AddressHandler) GetAllAddresses(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addresses, err := h.addressService.GetAllUserAddresses(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get all addresses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get all addresses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"addresses": addresses,
		"count":     len(addresses),
	})
}

// SaveAddressFromOrder saves an address from order history to user's saved addresses
// POST /api/v1/addresses/save-from-order
// @Summary Save address from order history
// @Description Save an address from order history to user's saved addresses
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body object{order_address_id=string} true "Order address ID to save"
// @Success 201 {object} map[string]interface{} "Address saved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to save address"
// @Router /addresses/save-from-order [post]
func (h *AddressHandler) SaveAddressFromOrder(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req struct {
		OrderAddressID string `json:"order_address_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	address, err := h.addressService.SaveAddressFromOrder(c.Request.Context(), userID, req.OrderAddressID)
	if err != nil {
		h.logger.Error("Failed to save address from order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save address"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"address": address,
		"message": "Address saved successfully",
	})
}

// CreateAddress creates a new address
// POST /api/v1/addresses
// @Summary Create new address
// @Description Create a new address for the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body models.Address true "Address data"
// @Success 201 {object} map[string]interface{} "Address created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Failed to create address"
// @Router /addresses [post]
func (h *AddressHandler) CreateAddress(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var address models.Address
	if err := c.ShouldBindJSON(&address); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdAddress, err := h.addressService.CreateAddress(c.Request.Context(), userID, &address)
	if err != nil {
		h.logger.Error("Failed to create address", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create address"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"address": createdAddress,
		"message": "Address created successfully",
	})
}

// UpdateAddress updates an existing address
// PUT /api/v1/addresses/:id
// @Summary Update address
// @Description Update an existing address for the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Address ID"
// @Param request body models.Address true "Updated address data"
// @Success 200 {object} map[string]interface{} "Address updated successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Failed to update address"
// @Router /addresses/{id} [put]
func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	var address models.Address
	if err := c.ShouldBindJSON(&address); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Set the ID from the URL parameter
	address.ID = addressID

	updatedAddress, err := h.addressService.UpdateAddress(c.Request.Context(), userID, &address)
	if err != nil {
		h.logger.Error("Failed to update address", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"address": updatedAddress,
		"message": "Address updated successfully",
	})
}

// SetDefaultAddress sets an address as default
// PUT /api/v1/addresses/:id/default
// @Summary Set default address
// @Description Set an address as the default for the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Address ID"
// @Success 200 {object} map[string]interface{} "Default address updated successfully"
// @Failure 400 {object} map[string]interface{} "Address ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Failed to set default address"
// @Router /addresses/{id}/default [put]
func (h *AddressHandler) SetDefaultAddress(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	err := h.addressService.SetDefaultAddress(c.Request.Context(), userID, addressID)
	if err != nil {
		h.logger.Error("Failed to set default address", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Default address updated successfully",
	})
}

// DeleteAddress deletes an address
// DELETE /api/v1/addresses/:id
// @Summary Delete address
// @Description Delete an address for the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Address ID"
// @Success 200 {object} map[string]interface{} "Address deleted successfully"
// @Failure 400 {object} map[string]interface{} "Address ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Failed to delete address"
// @Router /addresses/{id} [delete]
func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	err := h.addressService.DeleteAddress(c.Request.Context(), userID, addressID)
	if err != nil {
		h.logger.Error("Failed to delete address", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Address deleted successfully",
	})
}

// GetAddressByID gets a specific address by ID
// GET /api/v1/addresses/:id
// @Summary Get address by ID
// @Description Get a specific address by ID for the authenticated user
// @Tags Addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Address ID"
// @Success 200 {object} map[string]interface{} "Address retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Address ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Failed to get address"
// @Router /addresses/{id} [get]
func (h *AddressHandler) GetAddressByID(c *gin.Context) {
	userID, exists := h.authContext.GetAuthenticatedUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	address, err := h.addressService.GetAddressByID(c.Request.Context(), userID, addressID)
	if err != nil {
		h.logger.Error("Failed to get address by ID", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"address": address,
	})
}
