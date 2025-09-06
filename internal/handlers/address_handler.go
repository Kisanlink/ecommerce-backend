package handlers

import (
	"context"
	"net/http"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// AddressHandler handles address-related HTTP requests
type AddressHandler struct {
	addressRepo *repositories.AddressRepository
	logger      *zap.Logger
}

// NewAddressHandler creates a new address handler
func NewAddressHandler(addressRepo *repositories.AddressRepository, logger *zap.Logger) *AddressHandler {
	return &AddressHandler{
		addressRepo: addressRepo,
		logger:      logger,
	}
}

// CreateAddressRequest represents the request to create an address
type CreateAddressRequest struct {
	Type         string `json:"type,omitempty" validate:"omitempty,oneof=home work other shipping billing"`
	FirstName    string `json:"first_name" binding:"required" validate:"required,min=1,max=100"`
	LastName     string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Company      string `json:"company,omitempty" validate:"omitempty,max=100"`
	AddressLine1 string `json:"address_line_1" binding:"required" validate:"required,min=1,max=255"`
	AddressLine2 string `json:"address_line_2,omitempty" validate:"omitempty,max=255"`
	City         string `json:"city" binding:"required" validate:"required,min=1,max=100"`
	State        string `json:"state" binding:"required" validate:"required,min=1,max=100"`
	PostalCode   string `json:"postal_code" binding:"required" validate:"required,min=1,max=20"`
	Country      string `json:"country,omitempty" validate:"omitempty,min=1,max=100"`
	Phone        string `json:"phone,omitempty" validate:"omitempty,min=10,max=15"`
	IsDefault    bool   `json:"is_default,omitempty"`
}

// Sanitize cleans string fields
func (r *CreateAddressRequest) Sanitize() {
	r.Type = strings.TrimSpace(r.Type)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Company = strings.TrimSpace(r.Company)
	r.AddressLine1 = strings.TrimSpace(r.AddressLine1)
	r.AddressLine2 = strings.TrimSpace(r.AddressLine2)
	r.City = strings.TrimSpace(r.City)
	r.State = strings.TrimSpace(r.State)
	r.PostalCode = strings.TrimSpace(r.PostalCode)
	r.Country = strings.TrimSpace(r.Country)
	r.Phone = strings.TrimSpace(r.Phone)

	// Set defaults
	if r.Type == "" {
		r.Type = "shipping"
	}
	if r.Country == "" {
		r.Country = "India"
	}
}

// UpdateAddressRequest represents the request to update an address
type UpdateAddressRequest struct {
	Type         string `json:"type,omitempty" validate:"omitempty,oneof=home work other shipping billing"`
	FirstName    string `json:"first_name,omitempty" validate:"omitempty,min=1,max=100"`
	LastName     string `json:"last_name,omitempty" validate:"omitempty,min=1,max=100"`
	Company      string `json:"company,omitempty" validate:"omitempty,max=100"`
	AddressLine1 string `json:"address_line_1,omitempty" validate:"omitempty,min=1,max=255"`
	AddressLine2 string `json:"address_line_2,omitempty" validate:"omitempty,max=255"`
	City         string `json:"city,omitempty" validate:"omitempty,min=1,max=100"`
	State        string `json:"state,omitempty" validate:"omitempty,min=1,max=100"`
	PostalCode   string `json:"postal_code,omitempty" validate:"omitempty,min=1,max=20"`
	Country      string `json:"country,omitempty" validate:"omitempty,min=1,max=100"`
	Phone        string `json:"phone,omitempty" validate:"omitempty,min=10,max=15"`
	// Use pointer so we can detect if the client omitted this field
	IsDefault *bool `json:"is_default,omitempty"`
}

// Sanitize cleans string fields
func (r *UpdateAddressRequest) Sanitize() {
	r.Type = strings.TrimSpace(r.Type)
	r.FirstName = strings.TrimSpace(r.FirstName)
	r.LastName = strings.TrimSpace(r.LastName)
	r.Company = strings.TrimSpace(r.Company)
	r.AddressLine1 = strings.TrimSpace(r.AddressLine1)
	r.AddressLine2 = strings.TrimSpace(r.AddressLine2)
	r.City = strings.TrimSpace(r.City)
	r.State = strings.TrimSpace(r.State)
	r.PostalCode = strings.TrimSpace(r.PostalCode)
	r.Country = strings.TrimSpace(r.Country)
	r.Phone = strings.TrimSpace(r.Phone)
}

// GetUserAddresses retrieves all addresses for the authenticated user
// @Summary Get user addresses
// @Description Get all addresses associated with the authenticated user
// @Tags addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Success 200 {array} models.Address "List of user addresses"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /addresses [get]
func (h *AddressHandler) GetAddresses(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	ctx := context.Background()
	addresses, err := h.addressRepo.GetUserAddresses(ctx, userID.(string))
	if err != nil {
		h.logger.Error("Failed to get user addresses", zap.Error(err), zap.String("user_id", userID.(string)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve addresses"})
		return
	}

	c.JSON(http.StatusOK, addresses)
}

// GetAddress retrieves a specific address by ID
// @Summary Get address by ID
// @Description Get a specific address by its ID for the authenticated user
// @Tags addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Address ID"
// @Success 200 {object} models.Address "Address details"
// @Failure 400 {object} map[string]interface{} "Invalid address ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /addresses/{id} [get]
func (h *AddressHandler) GetAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	ctx := context.Background()
	address, err := h.addressRepo.GetUserAddressByID(ctx, userID.(string), addressID)
	if err != nil {
		h.logger.Error("Failed to get address", zap.Error(err), zap.String("user_id", userID.(string)), zap.String("address_id", addressID))
		if err.Error() == "address not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve address"})
		return
	}

	c.JSON(http.StatusOK, address)
}

// CreateAddress creates a new address for the user
// @Summary Create new address
// @Description Create a new address for the authenticated user
// @Tags addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body CreateAddressRequest true "Address details"
// @Success 201 {object} models.Address "Created address"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /addresses [post]
func (h *AddressHandler) CreateAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req CreateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data: " + err.Error()})
		return
	}

	req.Sanitize()

	// Create address model using correct constructor
	address := models.NewAddress(
		userID.(string),
		req.AddressLine1,
		req.City,
		req.State,
		req.PostalCode,
		req.Country,
		models.AddressType(req.Type),
	)

	// Set additional fields
	address.FirstName = req.FirstName
	address.LastName = req.LastName
	address.Company = req.Company
	address.AddressLine2 = req.AddressLine2
	address.Phone = req.Phone
	address.IsDefault = req.IsDefault
	address.Source = "manual" // User created manually

	ctx := context.Background()
	err := h.addressRepo.CreateAddress(ctx, address)
	if err != nil {
		h.logger.Error("Failed to create address", zap.Error(err), zap.String("user_id", userID.(string)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create address"})
		return
	}

	// If this should be the default address, set it
	if req.IsDefault {
		err := h.addressRepo.SetDefaultAddress(ctx, userID.(string), address.ID)
		if err != nil {
			h.logger.Error("Failed to set default address", zap.Error(err), zap.String("user_id", userID.(string)), zap.String("address_id", address.ID))
			// Don't fail the request, just log the error
		}
	}

	c.JSON(http.StatusCreated, address)
}

// UpdateAddress updates an existing address
// @Summary Update address
// @Description Update an existing address for the authenticated user
// @Tags addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Address ID"
// @Param request body UpdateAddressRequest true "Updated address details"
// @Success 200 {object} models.Address "Updated address"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /addresses/{id} [put]
func (h *AddressHandler) UpdateAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	var req UpdateAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data: " + err.Error()})
		return
	}

	req.Sanitize()

	ctx := context.Background()
	// Get existing address
	existingAddress, err := h.addressRepo.GetUserAddressByID(ctx, userID.(string), addressID)
	if err != nil {
		h.logger.Error("Failed to get existing address", zap.Error(err), zap.String("user_id", userID.(string)), zap.String("address_id", addressID))
		if err.Error() == "address not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve address"})
		return
	}

	// Update fields if provided
	if req.Type != "" {
		existingAddress.Type = models.AddressType(req.Type)
	}
	if req.FirstName != "" {
		existingAddress.FirstName = req.FirstName
	}
	if req.LastName != "" {
		existingAddress.LastName = req.LastName
	}
	if req.Company != "" {
		existingAddress.Company = req.Company
	}
	if req.AddressLine1 != "" {
		existingAddress.AddressLine1 = req.AddressLine1
	}
	if req.AddressLine2 != "" {
		existingAddress.AddressLine2 = req.AddressLine2
	}
	if req.City != "" {
		existingAddress.City = req.City
	}
	if req.State != "" {
		existingAddress.State = req.State
	}
	if req.PostalCode != "" {
		existingAddress.PostalCode = req.PostalCode
	}
	if req.Country != "" {
		existingAddress.Country = req.Country
	}
	if req.Phone != "" {
		existingAddress.Phone = req.Phone
	}

	// Only update IsDefault if it was provided in the request
	if req.IsDefault != nil {
		existingAddress.IsDefault = *req.IsDefault
	}

	err = h.addressRepo.UpdateAddress(ctx, existingAddress)
	if err != nil {
		h.logger.Error("Failed to update address", zap.Error(err), zap.String("user_id", userID.(string)), zap.String("address_id", addressID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	// If the request explicitly set IsDefault to true, ensure it's set as default
	if req.IsDefault != nil && *req.IsDefault {
		if err := h.addressRepo.SetDefaultAddress(ctx, userID.(string), addressID); err != nil {
			h.logger.Error("Failed to set default address", zap.Error(err), zap.String("user_id", userID.(string)), zap.String("address_id", addressID))
			// Don't fail the request, just log the error
		}
	}

	c.JSON(http.StatusOK, existingAddress)
}

// DeleteAddress deletes an address
// @Summary Delete address
// @Description Delete an address for the authenticated user
// @Tags addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Address ID"
// @Success 200 {object} map[string]interface{} "Address deleted successfully"
// @Failure 400 {object} map[string]interface{} "Invalid address ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /addresses/{id} [delete]
func (h *AddressHandler) DeleteAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	ctx := context.Background()
	err := h.addressRepo.DeleteAddress(ctx, userID.(string), addressID)
	if err != nil {
		h.logger.Error("Failed to delete address", zap.Error(err), zap.String("user_id", userID.(string)), zap.String("address_id", addressID))
		if err.Error() == "address not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Address deleted successfully",
		"data":    nil,
	})
}

// SetDefaultAddress sets an address as the default for the user
// @Summary Set default address
// @Description Set a specific address as the default for the authenticated user
// @Tags addresses
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param id path string true "Address ID"
// @Success 200 {object} map[string]interface{} "Address set as default successfully"
// @Failure 400 {object} map[string]interface{} "Invalid address ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Address not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /addresses/{id}/default [put]
func (h *AddressHandler) SetDefaultAddress(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	addressID := c.Param("id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Address ID is required"})
		return
	}

	ctx := context.Background()
	err := h.addressRepo.SetDefaultAddress(ctx, userID.(string), addressID)
	if err != nil {
		h.logger.Error("Failed to set default address", zap.Error(err), zap.String("user_id", userID.(string)), zap.String("address_id", addressID))
		if err.Error() == "address not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to set default address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Address set as default successfully",
		"data":    nil,
	})
}
