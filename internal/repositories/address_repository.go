package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
)

// AddressRepository handles address-related database operations
type AddressRepository struct {
	*base.BaseFilterableRepository[*models.Address]
	dbManager db.DBManager
}

// NewAddressRepository creates a new address repository
func NewAddressRepository(dbManager db.DBManager) *AddressRepository {
	repo := &AddressRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Address](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// GetUserAddresses retrieves all addresses for a specific user
func (r *AddressRepository) GetUserAddresses(ctx context.Context, userID string) ([]*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Build()

	return r.Find(ctx, filter)
}

// GetUserAddressByID retrieves an address by ID for a specific user
func (r *AddressRepository) GetUserAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error) {
	if userID == "" || addressID == "" {
		return nil, fmt.Errorf("user ID and address ID cannot be empty")
	}

	filter := base.NewFilterBuilder().
		Where("id", base.OpEqual, addressID).
		Where("user_id", base.OpEqual, userID).
		Build()

	return r.FindOne(ctx, filter)
}

// CreateAddress creates a new address
func (r *AddressRepository) CreateAddress(ctx context.Context, address *models.Address) error {
	if address == nil {
		return fmt.Errorf("address cannot be nil")
	}

	// If this is marked as default, unset other defaults for the user first
	if address.IsDefault {
		if err := r.unsetOtherDefaults(ctx, address.UserID, ""); err != nil {
			return fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	return r.dbManager.Create(ctx, address)
}

// UpdateAddress updates an existing address
func (r *AddressRepository) UpdateAddress(ctx context.Context, address *models.Address) error {
	if address == nil {
		return fmt.Errorf("address cannot be nil")
	}

	// If this is being set as default, unset other defaults for the user
	if address.IsDefault {
		if err := r.unsetOtherDefaults(ctx, address.UserID, address.ID); err != nil {
			return fmt.Errorf("failed to unset other defaults: %w", err)
		}
	}

	return r.dbManager.Update(ctx, address)
}

// SetDefaultAddress sets an address as the default for a user
func (r *AddressRepository) SetDefaultAddress(ctx context.Context, userID, addressID string) error {
	if userID == "" || addressID == "" {
		return fmt.Errorf("user ID and address ID cannot be empty")
	}

	// First unset all defaults for the user
	if err := r.unsetOtherDefaults(ctx, userID, ""); err != nil {
		return fmt.Errorf("failed to unset other defaults: %w", err)
	}

	// Then get and update the specific address
	address, err := r.GetUserAddressByID(ctx, userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	address.IsDefault = true
	return r.dbManager.Update(ctx, address)
}

// GetDefaultAddress retrieves the default address for a user
func (r *AddressRepository) GetDefaultAddress(ctx context.Context, userID string) (*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Where("is_default", base.OpEqual, true).
		Build()

	return r.FindOne(ctx, filter)
}

// DeleteAddress soft deletes an address
func (r *AddressRepository) DeleteAddress(ctx context.Context, userID, addressID string) error {
	if userID == "" || addressID == "" {
		return fmt.Errorf("user ID and address ID cannot be empty")
	}

	// Get the address first to verify ownership
	address, err := r.GetUserAddressByID(ctx, userID, addressID)
	if err != nil {
		return fmt.Errorf("failed to get address: %w", err)
	}

	// If this is the default address, we might want to set another as default
	// For now, just delete it
	// Note: dbManager.Delete expects (ctx, id, model) — pass id first, then model
	return r.dbManager.Delete(ctx, address.ID, address)
}

// unsetOtherDefaults unsets default flag for other addresses of the user
func (r *AddressRepository) unsetOtherDefaults(ctx context.Context, userID, excludeAddressID string) error {
	// Get all default addresses for the user
	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Where("is_default", base.OpEqual, true).
		Build()

	if excludeAddressID != "" {
		filter = base.NewFilterBuilder().
			Where("user_id", base.OpEqual, userID).
			Where("is_default", base.OpEqual, true).
			Where("id", base.OpNotEqual, excludeAddressID).
			Build()
	}

	addresses, err := r.Find(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to get addresses to unset: %w", err)
	}

	// Update each address to not be default
	for _, addr := range addresses {
		addr.IsDefault = false
		if err := r.dbManager.Update(ctx, addr); err != nil {
			return fmt.Errorf("failed to unset default for address %s: %w", addr.ID, err)
		}
	}

	return nil
}

// CountUserAddresses counts the total number of addresses for a user
func (r *AddressRepository) CountUserAddresses(ctx context.Context, userID string) (int64, error) {
	if userID == "" {
		return 0, fmt.Errorf("user ID cannot be empty")
	}

	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Build()

	var address models.Address
	return r.Count(ctx, filter, &address)
}

// GetAddressesByType retrieves addresses by type for a specific user
func (r *AddressRepository) GetAddressesByType(ctx context.Context, userID string, addressType models.AddressType) ([]*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Where("type", base.OpEqual, addressType).
		Build()

	return r.Find(ctx, filter)
}
