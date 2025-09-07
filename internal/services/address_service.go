package services

import (
	"context"
	"fmt"
	"log"

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

	// Create new address
	address = &models.Address{
		BaseModel:    baseModel,
		UserID:       userID,
		Type:         models.AddressTypeShipping,
		FirstName:    "Default",
		LastName:     "User",
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
