package repositories

import (
	"context"
	"fmt"
	"log"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
)

// ProfileRepository handles profile-related database operations
type ProfileRepository struct {
	*base.BaseFilterableRepository[*models.Profile]
	dbManager db.DBManager
}

// NewProfileRepository creates a new profile repository
func NewProfileRepository(dbManager db.DBManager) *ProfileRepository {
	repo := &ProfileRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Profile](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// GetByUserID retrieves a profile by user ID
func (r *ProfileRepository) GetByUserID(ctx context.Context, userID string) (*models.Profile, error) {
	log.Printf("Getting profile by user ID: %s", userID)

	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Build()

	profiles, err := r.BaseFilterableRepository.Find(ctx, filter)
	if err != nil {
		log.Printf("Error getting profile by user ID %s: %v", userID, err)
		return nil, fmt.Errorf("failed to get profile by user ID: %w", err)
	}

	if len(profiles) == 0 {
		log.Printf("No profile found for user ID: %s", userID)
		return nil, fmt.Errorf("record not found")
	}

	log.Printf("Successfully retrieved profile for user ID: %s", userID)
	return profiles[0], nil
}

// CreateProfile creates a new profile
func (r *ProfileRepository) CreateProfile(ctx context.Context, profile *models.Profile) error {
	log.Printf("Creating profile for user ID: %s", profile.UserID)

	if err := r.dbManager.Create(ctx, profile); err != nil {
		log.Printf("Error creating profile: %v", err)
		return fmt.Errorf("failed to create profile: %w", err)
	}

	log.Printf("Successfully created profile with ID: %s", profile.GetID())
	return nil
}

// UpdateProfile updates an existing profile
func (r *ProfileRepository) UpdateProfile(ctx context.Context, profile *models.Profile) error {
	log.Printf("Updating profile: %s", profile.GetID())

	if err := r.dbManager.Update(ctx, profile); err != nil {
		log.Printf("Error updating profile: %v", err)
		return fmt.Errorf("failed to update profile: %w", err)
	}

	log.Printf("Successfully updated profile: %s", profile.GetID())
	return nil
}

// ExistsByUserID checks if a profile exists for a user
func (r *ProfileRepository) ExistsByUserID(ctx context.Context, userID string) (bool, error) {
	log.Printf("Checking if profile exists for user: %s", userID)

	_, err := r.GetByUserID(ctx, userID)
	if err != nil {
		if err.Error() == "record not found" {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ExistsByEmail checks if a profile exists with the given email
func (r *ProfileRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	log.Printf("Checking if profile exists for email: %s", email)

	filter := base.NewFilterBuilder().
		Where("email", base.OpEqual, email).
		Build()

	profiles, err := r.BaseFilterableRepository.Find(ctx, filter)
	if err != nil {
		log.Printf("Error checking profile existence by email %s: %v", email, err)
		return false, err
	}

	exists := len(profiles) > 0
	log.Printf("Profile exists for email %s: %v", email, exists)
	return exists, nil
}