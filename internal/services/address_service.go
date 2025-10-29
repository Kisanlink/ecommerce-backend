package services

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"go.uber.org/zap"
)

type AddressServiceInterface interface {
	CreateAddress(ctx context.Context, userID string, req *models.Address) (*models.Address, error)
	GetAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error)
	GetUserAddresses(ctx context.Context, userID string) ([]*models.Address, error)
	UpdateAddress(ctx context.Context, userID string, req *models.Address) (*models.Address, error)
	DeleteAddress(ctx context.Context, userID, addressID string) error
	SetDefaultAddress(ctx context.Context, userID, addressID string) error
	FindOrCreateAddress(ctx context.Context, userID, street, city, state, postalCode, country string) (*models.Address, error)

	// NEW: Address history methods
	GetLast4Addresses(ctx context.Context, userID string) ([]*models.Address, error)
	GetAddressHistoryFromOrders(ctx context.Context, userID string) ([]*models.Address, error)
	GetAllUserAddresses(ctx context.Context, userID string) ([]*models.Address, error)
	SaveAddressFromOrder(ctx context.Context, userID string, orderAddressID string) (*models.Address, error)
	UpdateAddressFromOrder(ctx context.Context, userID, addressID string, req *models.Address) (*models.Address, error)
	ManageAddressLimit(ctx context.Context, userID string) error
}

type AddressService struct {
	addressRepo repositories.AddressRepositoryInterface
	repoManager *repositories.RepositoryManager
	logger      *zap.Logger
}

func NewAddressService(repoManager *repositories.RepositoryManager, logger *zap.Logger) *AddressService {
	service := &AddressService{
		addressRepo: repoManager.AddressRepository,
		repoManager: repoManager,
		logger:      logger,
	}

	// Initialize the ID generator with existing addresses
	service.initializeIDGenerator()

	return service
}

func (s *AddressService) initializeIDGenerator() {
	ctx := context.Background()

	// Get all existing addresses to initialize ID generator
	filter := &base.Filter{
		Page:     1,
		PageSize: 10000, // Get all addresses
	}

	var addresses []*models.Address
	if err := s.repoManager.GetDBManager().List(ctx, filter, &addresses); err != nil {
		s.logger.Error("Failed to get existing addresses for ID generator", zap.Error(err))
		return
	}

	existingIDs := make([]string, len(addresses))
	for i, addr := range addresses {
		existingIDs[i] = addr.ID
	}

	// Initialize the global generator with existing IDs
	hash.InitializeGlobalCountersFromDatabase("ADDR", existingIDs, hash.Medium)
	log.Printf("Initialized address ID generator with %d existing address IDs", len(existingIDs))
}

func (s *AddressService) CreateAddress(ctx context.Context, userID string, req *models.Address) (*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Generate new ID using the hash generator
	newID, err := hash.GenerateRandomID("ADDR", hash.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address ID: %w", err)
	}

	// Create base model with generated ID
	baseModel := base.NewBaseModel("ADDR", hash.Medium)
	baseModel.ID = newID

	// Create new address
	address := &models.Address{
		BaseModel:    baseModel,
		UserID:       userID,
		Type:         req.Type,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Company:      req.Company,
		AddressLine1: req.AddressLine1,
		AddressLine2: req.AddressLine2,
		City:         req.City,
		State:        req.State,
		PostalCode:   req.PostalCode,
		Country:      req.Country,
		Phone:        req.Phone,
		IsDefault:    req.IsDefault,
	}

	if err := s.addressRepo.CreateAddress(ctx, address); err != nil {
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	return address, nil
}

func (s *AddressService) GetAddressByID(ctx context.Context, userID, addressID string) (*models.Address, error) {
	return s.addressRepo.GetAddressByID(ctx, userID, addressID)
}

func (s *AddressService) GetUserAddresses(ctx context.Context, userID string) ([]*models.Address, error) {
	return s.addressRepo.GetUserAddresses(ctx, userID)
}

func (s *AddressService) UpdateAddress(ctx context.Context, userID string, req *models.Address) (*models.Address, error) {
	// First verify the address belongs to the user
	existing, err := s.addressRepo.GetAddressByID(ctx, userID, req.ID)
	if err != nil {
		return nil, fmt.Errorf("address not found: %w", err)
	}

	// Update fields
	existing.Type = req.Type
	existing.FirstName = req.FirstName
	existing.LastName = req.LastName
	existing.Company = req.Company
	existing.AddressLine1 = req.AddressLine1
	existing.AddressLine2 = req.AddressLine2
	existing.City = req.City
	existing.State = req.State
	existing.PostalCode = req.PostalCode
	existing.Country = req.Country
	existing.Phone = req.Phone
	existing.IsDefault = req.IsDefault

	if err := s.addressRepo.UpdateAddress(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update address: %w", err)
	}

	return existing, nil
}

func (s *AddressService) DeleteAddress(ctx context.Context, userID, addressID string) error {
	return s.addressRepo.DeleteAddress(ctx, userID, addressID)
}

func (s *AddressService) SetDefaultAddress(ctx context.Context, userID, addressID string) error {
	return s.addressRepo.SetDefaultAddress(ctx, userID, addressID)
}

func (s *AddressService) FindOrCreateAddress(ctx context.Context, userID, street, city, state, postalCode, country string) (*models.Address, error) {
	// First try to find existing address using repository
	address, err := s.addressRepo.FindOrCreateAddress(ctx, userID, street, city, state, postalCode, country)
	if err == nil {
		return address, nil
	}

	// If FindOrCreate in repo failed, create manually with proper ID generation
	newID, err := hash.GenerateRandomID("ADDR", hash.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to generate address ID: %w", err)
	}

	// Create base model with generated ID
	baseModel := base.NewBaseModel("ADDR", hash.Medium)
	baseModel.ID = newID

	// Get user's actual name from profile
	firstName, lastName := s.getUserNames(ctx, userID)

	// Create new address with actual user names
	address = &models.Address{
		BaseModel:    baseModel,
		UserID:       userID,
		Type:         models.AddressTypeShipping,
		FirstName:    firstName,
		LastName:     lastName,
		Company:      "",
		AddressLine1: street,
		AddressLine2: "",
		City:         city,
		State:        state,
		PostalCode:   postalCode,
		Country:      country,
		Phone:        "",
		IsDefault:    false,
	}

	if err := s.addressRepo.CreateAddress(ctx, address); err != nil {
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	return address, nil
}

// getUserNames retrieves the user's first and last name from their profile
func (s *AddressService) getUserNames(ctx context.Context, userID string) (string, string) {
	// Try to get user profile first
	profile, err := s.repoManager.ProfileRepository.GetByUserID(ctx, userID)
	if err == nil && profile != nil {
		firstName := strings.TrimSpace(profile.FirstName)
		lastName := strings.TrimSpace(profile.LastName)

		// If we have valid names, return them
		if firstName != "" && lastName != "" {
			return firstName, lastName
		} else if firstName != "" {
			return firstName, ""
		} else if lastName != "" {
			return "", lastName
		}
	}

	// Fallback: try to get user info from users table
	user := &models.User{}
	_, err = s.repoManager.UserRepository.GetByID(ctx, userID, user)
	if err == nil && user.Name != "" {
		// Extract names from display name
		names := strings.Fields(strings.TrimSpace(user.Name))
		if len(names) > 0 {
			firstName := names[0]
			lastName := ""
			if len(names) > 1 {
				lastName = strings.Join(names[1:], " ")
			}
			return firstName, lastName
		}
	}

	// Final fallback: return generic names
	return "User", ""
}

// GetLast4Addresses retrieves the last 4 addresses used by the user
func (s *AddressService) GetLast4Addresses(ctx context.Context, userID string) ([]*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Get all user addresses ordered by updated_at desc, limit to 4
	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Sort("updated_at", "desc").
		Page(1, 4).
		Build()

	var addresses []*models.Address
	if err := s.repoManager.GetDBManager().List(ctx, filter, &addresses); err != nil {
		return nil, fmt.Errorf("failed to get last 4 addresses: %w", err)
	}

	return addresses, nil
}

// GetAddressHistoryFromOrders retrieves all unique addresses used in previous orders
func (s *AddressService) GetAddressHistoryFromOrders(ctx context.Context, userID string) ([]*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Get all orders for the user
	orders, err := s.repoManager.OrderRepository.GetOrdersByUserWithDetails(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user orders: %w", err)
	}

	// Extract unique addresses from orders
	addressMap := make(map[string]*models.Address)

	for _, order := range orders {
		if order.ShippingAddress != nil {
			// Create a unique key based on address components
			addressKey := fmt.Sprintf("%s|%s|%s|%s|%s",
				order.ShippingAddress.AddressLine1,
				order.ShippingAddress.City,
				order.ShippingAddress.State,
				order.ShippingAddress.PostalCode,
				order.ShippingAddress.Country)

			// Convert OrderAddress to Address for consistency
			baseModel := &base.BaseModel{}
			baseModel.ID = order.ShippingAddress.ID
			address := &models.Address{
				BaseModel:    baseModel,
				UserID:       userID,
				Type:         models.AddressTypeShipping,
				FirstName:    "Previous Order", // You might want to get this from order customer
				LastName:     "",
				AddressLine1: order.ShippingAddress.AddressLine1,
				AddressLine2: order.ShippingAddress.AddressLine2,
				City:         order.ShippingAddress.City,
				State:        order.ShippingAddress.State,
				PostalCode:   order.ShippingAddress.PostalCode,
				Country:      order.ShippingAddress.Country,
				Phone:        "",
				IsDefault:    false,
			}

			addressMap[addressKey] = address
		}
	}

	// Convert map to slice
	var addresses []*models.Address
	for _, addr := range addressMap {
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// GetAllUserAddresses retrieves both saved addresses and address history
func (s *AddressService) GetAllUserAddresses(ctx context.Context, userID string) ([]*models.Address, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Get saved addresses
	savedAddresses, err := s.GetUserAddresses(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get saved addresses: %w", err)
	}

	// Get address history from orders
	historyAddresses, err := s.GetAddressHistoryFromOrders(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get address history: %w", err)
	}

	// Combine and deduplicate
	addressMap := make(map[string]*models.Address)

	// Add saved addresses first
	for _, addr := range savedAddresses {
		addressKey := fmt.Sprintf("%s|%s|%s|%s|%s",
			addr.AddressLine1, addr.City, addr.State, addr.PostalCode, addr.Country)
		addressMap[addressKey] = addr
	}

	// Add history addresses (won't overwrite saved addresses)
	for _, addr := range historyAddresses {
		addressKey := fmt.Sprintf("%s|%s|%s|%s|%s",
			addr.AddressLine1, addr.City, addr.State, addr.PostalCode, addr.Country)
		if _, exists := addressMap[addressKey]; !exists {
			addressMap[addressKey] = addr
		}
	}

	// Convert map to slice
	var addresses []*models.Address
	for _, addr := range addressMap {
		addresses = append(addresses, addr)
	}

	return addresses, nil
}

// SaveAddressFromOrder saves an address from order history to user's saved addresses
func (s *AddressService) SaveAddressFromOrder(ctx context.Context, userID string, orderAddressID string) (*models.Address, error) {
	if userID == "" || orderAddressID == "" {
		return nil, fmt.Errorf("user ID and order address ID are required")
	}

	// Get the order address
	orderAddress, err := s.repoManager.OrderRepository.GetOrderAddress(ctx, orderAddressID)
	if err != nil {
		return nil, fmt.Errorf("failed to get order address: %w", err)
	}

	// Check if similar address already exists
	existingAddresses, err := s.GetUserAddresses(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing addresses: %w", err)
	}

	// Check for duplicates
	for _, existing := range existingAddresses {
		if existing.AddressLine1 == orderAddress.AddressLine1 &&
			existing.City == orderAddress.City &&
			existing.State == orderAddress.State &&
			existing.PostalCode == orderAddress.PostalCode {
			// Address already exists, return it
			return existing, nil
		}
	}

	// Create new address from order address
	firstName, lastName := s.getUserNames(ctx, userID)

	newAddress := &models.Address{
		UserID:       userID,
		Type:         models.AddressTypeShipping,
		FirstName:    firstName,
		LastName:     lastName,
		AddressLine1: orderAddress.AddressLine1,
		AddressLine2: orderAddress.AddressLine2,
		City:         orderAddress.City,
		State:        orderAddress.State,
		PostalCode:   orderAddress.PostalCode,
		Country:      orderAddress.Country,
		Phone:        "",
		IsDefault:    false,
	}

	// Create the address
	createdAddress, err := s.CreateAddress(ctx, userID, newAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to create address from order: %w", err)
	}

	// Manage address limit (keep only last 4)
	if err := s.ManageAddressLimit(ctx, userID); err != nil {
		s.logger.Warn("Failed to manage address limit", zap.Error(err))
	}

	return createdAddress, nil
}

// UpdateAddressFromOrder updates an existing address from order data
func (s *AddressService) UpdateAddressFromOrder(ctx context.Context, userID, addressID string, req *models.Address) (*models.Address, error) {
	if userID == "" || addressID == "" {
		return nil, fmt.Errorf("user ID and address ID are required")
	}

	// Get existing address
	existing, err := s.GetAddressByID(ctx, userID, addressID)
	if err != nil {
		return nil, fmt.Errorf("address not found: %w", err)
	}

	// Update fields
	existing.Type = req.Type
	existing.FirstName = req.FirstName
	existing.LastName = req.LastName
	existing.Company = req.Company
	existing.AddressLine1 = req.AddressLine1
	existing.AddressLine2 = req.AddressLine2
	existing.City = req.City
	existing.State = req.State
	existing.PostalCode = req.PostalCode
	existing.Country = req.Country
	existing.Phone = req.Phone

	// Update in database
	if err := s.addressRepo.UpdateAddress(ctx, existing); err != nil {
		return nil, fmt.Errorf("failed to update address: %w", err)
	}

	return existing, nil
}

// ManageAddressLimit keeps only the last 4 addresses for a user
func (s *AddressService) ManageAddressLimit(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	// Get all addresses for user ordered by updated_at desc
	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Sort("updated_at", "desc").
		Build()

	var addresses []*models.Address
	if err := s.repoManager.GetDBManager().List(ctx, filter, &addresses); err != nil {
		return fmt.Errorf("failed to get user addresses: %w", err)
	}

	// If we have more than 4 addresses, delete the oldest ones
	if len(addresses) > 4 {
		addressesToDelete := addresses[4:] // Keep first 4, delete the rest

		for _, addr := range addressesToDelete {
			if err := s.DeleteAddress(ctx, userID, addr.ID); err != nil {
				s.logger.Warn("Failed to delete old address",
					zap.String("address_id", addr.ID),
					zap.Error(err))
			}
		}

		s.logger.Info("Cleaned up old addresses",
			zap.String("user_id", userID),
			zap.Int("deleted_count", len(addressesToDelete)))
	}

	return nil
}
