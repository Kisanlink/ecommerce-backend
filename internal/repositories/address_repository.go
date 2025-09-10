package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// AddressRepositoryInterface defines the contract for address data operations
type AddressRepositoryInterface interface {
	CreateAddress(ctx context.Context, address *models.Address) error
	GetAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error)
	GetUserAddresses(ctx context.Context, userID string) ([]*models.Address, error)
	UpdateAddress(ctx context.Context, address *models.Address) error
	DeleteAddress(ctx context.Context, userID, addressID string) error
	GetDefaultAddress(ctx context.Context, userID string) (*models.Address, error)
	SetDefaultAddress(ctx context.Context, userID, addressID string) error
	FindOrCreateAddress(ctx context.Context, userID string, street, city, state, postalCode, country string) (*models.Address, error)
	FormatAddressForDelhivery(address *models.Address) map[string]interface{}
}

// Extended methods for AddressRepository (defined in repository_manager.go)
// CreateAddress creates a new address
func (r *AddressRepository) CreateAddress(ctx context.Context, address *models.Address) error {
	if address == nil {
		return fmt.Errorf("address cannot be nil")
	}

	if err := r.dbManager.Create(ctx, address); err != nil {
		return fmt.Errorf("failed to create address: %w", err)
	}

	return nil
}

// GetAddressByID retrieves an address by ID for a specific user
func (r *AddressRepository) GetAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error) {
	if userID == "" || addressID == "" {
		return nil, fmt.Errorf("userID and addressID cannot be empty")
	}

	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "id", Operator: base.OpEqual, Value: addressID},
				{Field: "user_id", Operator: base.OpEqual, Value: userID},
			},
			Logic: base.LogicAnd,
		},
	}

	var addresses []*models.Address
	if err := r.dbManager.List(ctx, filter, &addresses); err != nil {
		return nil, fmt.Errorf("failed to get address: %w", err)
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("address not found")
	}

	return addresses[0], nil
}

// GetUserAddresses retrieves all addresses for a user
func (r *AddressRepository) GetUserAddresses(ctx context.Context, userID string) ([]*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "user_id", Operator: base.OpEqual, Value: userID},
			},
			Logic: base.LogicAnd,
		},
	}

	var addresses []*models.Address
	if err := r.dbManager.List(ctx, filter, &addresses); err != nil {
		return nil, fmt.Errorf("failed to get user addresses: %w", err)
	}

	return addresses, nil
}

// UpdateAddress updates an existing address
func (r *AddressRepository) UpdateAddress(ctx context.Context, address *models.Address) error {
	if address == nil {
		return fmt.Errorf("address cannot be nil")
	}

	if err := r.dbManager.Update(ctx, address); err != nil {
		return fmt.Errorf("failed to update address: %w", err)
	}

	return nil
}

// DeleteAddress deletes an address
func (r *AddressRepository) DeleteAddress(ctx context.Context, userID, addressID string) error {
	if userID == "" || addressID == "" {
		return fmt.Errorf("userID and addressID cannot be empty")
	}

	address := &models.Address{}
	if err := r.dbManager.SoftDelete(ctx, addressID, address, userID); err != nil {
		return fmt.Errorf("failed to delete address: %w", err)
	}

	return nil
}

// GetDefaultAddress retrieves the default address for a user
func (r *AddressRepository) GetDefaultAddress(ctx context.Context, userID string) (*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "user_id", Operator: base.OpEqual, Value: userID},
				{Field: "is_default", Operator: base.OpEqual, Value: true},
			},
			Logic: base.LogicAnd,
		},
	}

	var addresses []*models.Address
	if err := r.dbManager.List(ctx, filter, &addresses); err != nil {
		return nil, fmt.Errorf("failed to get default address: %w", err)
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("no default address found")
	}

	return addresses[0], nil
}

// SetDefaultAddress sets an address as the default for a user
func (r *AddressRepository) SetDefaultAddress(ctx context.Context, userID, addressID string) error {
	if userID == "" || addressID == "" {
		return fmt.Errorf("userID and addressID cannot be empty")
	}

	// First get the address to ensure it exists and belongs to the user
	address, err := r.GetAddressByID(ctx, userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to find address: %w", err)
	}

	// Set all addresses for this user to not default
	allAddresses, err := r.GetUserAddresses(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user addresses: %w", err)
	}

	for _, addr := range allAddresses {
		addr.IsDefault = false
		if err := r.UpdateAddress(ctx, addr); err != nil {
			return fmt.Errorf("failed to unset default for address %s: %w", addr.ID, err)
		}
	}

	// Set the target address as default
	address.IsDefault = true
	if err := r.UpdateAddress(ctx, address); err != nil {
		return fmt.Errorf("failed to set address as default: %w", err)
	}

	return nil
}

// FindOrCreateAddress finds an existing address or creates a new one based on address components
func (r *AddressRepository) FindOrCreateAddress(ctx context.Context, userID string, street, city, state, postalCode, country string) (*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("userID cannot be empty")
	}

	// Debug log incoming parameters
	fmt.Printf("FindOrCreateAddress called with: userID=%s, street=%s, city=%s, state=%s, postal=%s, country=%s\n",
		userID, street, city, state, postalCode, country)

	// First, try to find an existing address with the same components
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "user_id", Operator: base.OpEqual, Value: userID},
				{Field: "address_line1", Operator: base.OpEqual, Value: street},
				{Field: "city", Operator: base.OpEqual, Value: city},
				{Field: "state", Operator: base.OpEqual, Value: state},
				{Field: "postal_code", Operator: base.OpEqual, Value: postalCode},
				{Field: "country", Operator: base.OpEqual, Value: country},
			},
			Logic: base.LogicAnd,
		},
	}

	var addresses []*models.Address
	if err := r.dbManager.List(ctx, filter, &addresses); err == nil && len(addresses) > 0 {
		// Found existing address, return it
		fmt.Printf("Found existing address with ID: %s\n", addresses[0].ID)
		return addresses[0], nil
	}

	// No existing address found, create a new one
	fmt.Printf("Creating new address for user: %s\n", userID)
	// Note: We'll use generic names here as the repository doesn't have access to profile data
	// The service layer will handle getting actual user names
	newAddress := models.NewAddress(userID, "User", "", street, city, state, postalCode, country, models.AddressTypeShipping)

	fmt.Printf("New address created with ID: %s (before DB save)\n", newAddress.ID)

	if err := r.CreateAddress(ctx, newAddress); err != nil {
		fmt.Printf("Failed to save address to DB: %v\n", err)
		return nil, fmt.Errorf("failed to create new address: %w", err)
	}

	fmt.Printf("Address saved successfully with ID: %s\n", newAddress.ID)
	return newAddress, nil
}

// FormatAddressForDelhivery formats an address for Delhivery API
func (r *AddressRepository) FormatAddressForDelhivery(address *models.Address) map[string]interface{} {
	return map[string]interface{}{
		"address": map[string]string{
			"street":      address.AddressLine1,
			"city":        address.City,
			"state":       address.State,
			"postal_code": address.PostalCode,
			"country":     address.Country,
		},
	}
}
