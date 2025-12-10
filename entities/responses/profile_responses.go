package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// ProfileResponse represents a profile response (personal information only)
type ProfileResponse struct {
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	FirstName   string     `json:"first_name"`
	LastName    string     `json:"last_name"`
	FullName    string     `json:"full_name"`
	DateOfBirth *time.Time `json:"date_of_birth,omitempty"`
	Gender      string     `json:"gender"`

	// Address Information
	AddressLine1      string `json:"address_line_1"`
	AddressLine2      string `json:"address_line_2"`
	City              string `json:"city"`
	State             string `json:"state"`
	Country           string `json:"country"`
	Pincode           string `json:"pincode"`
	FullAddress       string `json:"full_address"`
	IsAddressComplete bool   `json:"is_address_complete"`

	// Profile Image
	ProfilePictureURL   string `json:"profile_picture_url"`
	ProfilePictureR2Key string `json:"profile_picture_r2_key"`
	Bio                 string `json:"bio"`

	// Preferences
	NewsletterSubscribed bool   `json:"newsletter_subscribed"`
	PreferredLanguage    string `json:"preferred_language"`

	// Account Status
	IsActive   bool `json:"is_active"`
	IsVerified bool `json:"is_verified"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProfileListResponse represents a list of profiles
type ProfileListResponse struct {
	Profiles   []ProfileResponse `json:"profiles"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
}

// ProfileUpdateResponse represents a profile update response
type ProfileUpdateResponse struct {
	Profile   ProfileResponse `json:"profile"`
	Message   string          `json:"message"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ProfileStatsResponse represents profile statistics
type ProfileStatsResponse struct {
	TotalProfiles       int64 `json:"total_profiles"`
	ActiveProfiles      int64 `json:"active_profiles"`
	VerifiedProfiles    int64 `json:"verified_profiles"`
	CompleteProfiles    int64 `json:"complete_profiles"`
	ProfilesWithAddress int64 `json:"profiles_with_address"`
	ProfilesWithPicture int64 `json:"profiles_with_picture"`
}

// ToProfileResponse converts a Profile model to ProfileResponse
func ToProfileResponse(profile *models.Profile) ProfileResponse {
	return ProfileResponse{
		ID:                   profile.GetID(),
		UserID:               profile.UserID,
		FirstName:            profile.FirstName,
		LastName:             profile.LastName,
		FullName:             profile.GetFullName(),
		DateOfBirth:          profile.DateOfBirth,
		Gender:               profile.Gender,
		AddressLine1:         profile.AddressLine1,
		AddressLine2:         profile.AddressLine2,
		City:                 profile.City,
		State:                profile.State,
		Country:              profile.Country,
		Pincode:              profile.Pincode,
		FullAddress:          profile.GetFullAddress(),
		IsAddressComplete:    profile.IsAddressComplete(),
		ProfilePictureURL:    profile.ProfilePictureURL,
		ProfilePictureR2Key:  profile.ProfilePictureR2Key,
		Bio:                  profile.Bio,
		NewsletterSubscribed: profile.NewsletterSubscribed,
		PreferredLanguage:    profile.PreferredLanguage,
		IsActive:             profile.IsActive,
		IsVerified:           profile.IsVerified,
		CreatedAt:            profile.GetCreatedAt(),
		UpdatedAt:            profile.GetUpdatedAt(),
	}
}

// ToProfileResponseList converts a slice of Profile models to ProfileResponse slice
func ToProfileResponseList(profiles []models.Profile) []ProfileResponse {
	responses := make([]ProfileResponse, len(profiles))
	for i, profile := range profiles {
		responses[i] = ToProfileResponse(&profile)
	}
	return responses
}

// ToProfileUpdateResponse creates a profile update response
func ToProfileUpdateResponse(profile *models.Profile, message string) ProfileUpdateResponse {
	return ProfileUpdateResponse{
		Profile:   ToProfileResponse(profile),
		Message:   message,
		UpdatedAt: profile.GetUpdatedAt(),
	}
}
