package services

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// ProfileService handles profile-related business logic
type ProfileService struct {
	repoManager *repositories.RepositoryManager
}

// NewProfileService creates a new profile service
func NewProfileService(repoManager *repositories.RepositoryManager) *ProfileService {
	service := &ProfileService{
		repoManager: repoManager,
	}
	// Initialize ID generator with existing profile IDs to avoid conflicts
	service.initializeIDGenerator()
	return service
}

// initializeIDGenerator initializes the global ID generator with existing profile IDs
func (s *ProfileService) initializeIDGenerator() {
	ctx := context.Background()
	filter := base.NewFilterBuilder().Build()

	profiles, err := s.repoManager.ProfileRepository.Find(ctx, filter)
	if err != nil {
		log.Printf("Warning: Failed to load existing profile IDs for generator: %v", err)
		return
	}

	var existingIDs []string
	for _, profile := range profiles {
		if profile != nil {
			existingIDs = append(existingIDs, profile.GetID())
		}
	}

	log.Printf("Initialized profile ID generator with %d existing profile IDs", len(existingIDs))
}

// createProfileWithTimestampID creates a new profile with timestamp-based ID to avoid collisions
func (s *ProfileService) createProfileWithTimestampID(userID, firstName, lastName, email string) *models.Profile {
	// Generate timestamp-based ID to avoid collisions
	timestampID, err := hash.GenerateRandomIDWithPattern("PROF", hash.Small, "timestamp")
	if err != nil {
		// Fallback to hash-based ID if timestamp fails
		hashID, hashErr := hash.GenerateRandomIDWithPattern("PROF", hash.Small, "hash")
		if hashErr != nil {
			// Final fallback to default incremental
			incID, _ := hash.GenerateRandomID("PROF", hash.Small)
			timestampID = incID
		} else {
			timestampID = hashID
		}
	}

	// Create profile manually with the generated ID
	baseModel := &base.BaseModel{}
	baseModel.SetID(timestampID)
	// Set timestamps manually
	now := time.Now()
	baseModel.SetCreatedAt(now)
	baseModel.SetUpdatedAt(now)

	profile := &models.Profile{
		BaseModel:            baseModel,
		UserID:               userID,
		FirstName:            firstName,
		LastName:             lastName,
		Email:                email,
		IsActive:             true,
		IsVerified:           false,
		NewsletterSubscribed: false,
		PreferredLanguage:    "en",
		Country:              "India", // Default for KisanLink
	}

	return profile
}

// CreateAutoProfile automatically creates a profile using user information from users table
func (s *ProfileService) CreateAutoProfile(ctx context.Context, userID string, firebaseEmail string) (*models.Profile, error) {
	log.Printf("Auto-creating profile for user ID: %s", userID)

	// Check if profile already exists for this user
	exists, err := s.repoManager.ProfileRepository.ExistsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing profile: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("profile already exists for user")
	}

	// Get user information from users table
	user := &models.User{}
	_, err = s.repoManager.UserRepository.GetByID(ctx, userID, user)

	var firstName, lastName, email string

	if err != nil {
		// If user not found in users table, create profile with Firebase info
		log.Printf("User not found in users table, using Firebase data")
		firstName = "User"
		lastName = ""
		email = firebaseEmail
		if email == "" {
			email = "user@example.com" // Fallback
		}
	} else {
		// Log user data for debugging
		log.Printf("Found user in database - ID: %s, Name: %s, Email: %s, Role: %s", user.GetID(), user.Name, user.Email, user.Role)

		// Extract names from display name
		if user.Name != "" {
			names := strings.Fields(strings.TrimSpace(user.Name))
			if len(names) > 0 {
				firstName = names[0]
			}
			if len(names) > 1 {
				lastName = strings.Join(names[1:], " ")
			}
		}

		// Use defaults if empty
		if firstName == "" {
			firstName = "User"
		}

		// Use Firebase email if available, otherwise fall back to users table email
		email = firebaseEmail
		if email == "" {
			email = user.Email
		}

		// Log extracted names for debugging
		log.Printf("Extracted names - FirstName: %s, LastName: %s, Email: %s", firstName, lastName, email)
	}

	// Create profile with information using timestamp-based ID to avoid collisions
	profile := s.createProfileWithTimestampID(userID, firstName, lastName, email)

	// Set additional fields if user exists
	if err == nil {
		if user.PhoneNumber != "" {
			profile.Phone = user.PhoneNumber
		}
		profile.IsVerified = user.IsVerified
	}

	// Save to database with retry logic for duplicate key errors
	maxRetries := 3
	var createErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		createErr = s.repoManager.ProfileRepository.CreateProfile(ctx, profile)
		if createErr == nil {
			// Successfully created profile, break out of retry loop
			break
		}

		// Check if it's a duplicate key error
		errorStr := createErr.Error()
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "profiles_pkey") {
			// Generate a new ID and try again with timestamp-based ID
			profile = s.createProfileWithTimestampID(userID, firstName, lastName, email)
			// Re-set additional fields if user exists
			if err == nil {
				if user.PhoneNumber != "" {
					profile.Phone = user.PhoneNumber
				}
				profile.IsVerified = user.IsVerified
			}
			fmt.Printf("Duplicate key error detected, retrying with new timestamp-based ID (attempt %d/%d): %s\n", attempt+1, maxRetries, profile.GetID())
			continue
		}

		// If it's not a duplicate key error, return the error immediately
		return nil, fmt.Errorf("failed to create profile: %w", createErr)
	}

	// If we still have an error after all retries, return it
	if createErr != nil {
		return nil, fmt.Errorf("failed to create profile after %d attempts: %w", maxRetries, createErr)
	}

	log.Printf("Successfully auto-created profile with ID: %s", profile.GetID())
	return profile, nil
}

// GetProfileByUserID retrieves profile by user ID
func (s *ProfileService) GetProfileByUserID(ctx context.Context, userID string) (*models.Profile, error) {
	log.Printf("Getting profile for user ID: %s", userID)

	profile, err := s.repoManager.ProfileRepository.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return profile, nil
}

// UpdateProfile updates the user's profile
func (s *ProfileService) UpdateProfile(ctx context.Context, req *requests.UpdateProfileRequest) (*models.Profile, error) {
	log.Printf("Updating profile for user ID: %s", req.GetUserID())

	// Get existing profile
	profile, err := s.repoManager.ProfileRepository.GetByUserID(ctx, req.GetUserID())
	if err != nil {
		return nil, err
	}

	// Apply updates to model
	if err := req.ApplyToModel(profile); err != nil {
		return nil, fmt.Errorf("failed to apply updates to model: %w", err)
	}

	// Update profile
	if err := s.repoManager.ProfileRepository.UpdateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	log.Printf("Successfully updated profile: %s", profile.GetID())
	return profile, nil
}

// UpdateProfilePictureWithS3 updates profile with S3 image details
func (s *ProfileService) UpdateProfilePictureWithS3(ctx context.Context, userID, imageURL, s3Key string) (*models.Profile, error) {
	log.Printf("Updating profile picture for user ID: %s", userID)

	// Get existing profile
	profile, err := s.GetProfileByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing profile: %w", err)
	}

	// Update S3 fields
	profile.ProfilePictureURL = imageURL
	profile.ProfilePictureS3Key = s3Key

	// Save updated profile
	if err := s.repoManager.ProfileRepository.UpdateProfile(ctx, profile); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	log.Printf("Successfully updated profile picture: %s", profile.GetID())
	return profile, nil
}
