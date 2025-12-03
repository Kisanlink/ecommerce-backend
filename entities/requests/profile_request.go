package requests

import (
	"fmt"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// UpdateProfileRequest represents the request structure for updating a user profile
type UpdateProfileRequest struct {
	*base.BaseRequest `json:"-"`
	FirstName         *string `json:"first_name,omitempty" validate:"omitempty,min=2,max=50"`
	LastName          *string `json:"last_name,omitempty" validate:"omitempty,min=2,max=50"`
	Email             *string `json:"email,omitempty" validate:"omitempty,email"`
	Phone             *string `json:"phone,omitempty" validate:"omitempty,e164"`
	DateOfBirth       *string `json:"date_of_birth,omitempty" validate:"omitempty,datetime=2006-01-02"`
	Gender            *string `json:"gender,omitempty" validate:"omitempty,oneof=male female other"`

	// Address fields
	AddressLine1 *string `json:"address_line_1,omitempty" validate:"omitempty,max=255"`
	AddressLine2 *string `json:"address_line_2,omitempty" validate:"omitempty,max=255"`
	City         *string `json:"city,omitempty" validate:"omitempty,max=100"`
	State        *string `json:"state,omitempty" validate:"omitempty,max=100"`
	Country      *string `json:"country,omitempty" validate:"omitempty,max=100"`
	Pincode      *string `json:"pincode,omitempty" validate:"omitempty,max=20"`

	// Preferences
	PreferredLanguage    *string `json:"preferred_language,omitempty" validate:"omitempty,max=10"`
	NewsletterSubscribed *bool   `json:"newsletter_subscribed,omitempty"`

	// Profile picture
	ProfilePictureURL *string `json:"profile_picture_url,omitempty" validate:"omitempty,url"`
}

// ApplyToModel applies the update request to the profile model
// Only updates fields that are not nil in the request
func (r *UpdateProfileRequest) ApplyToModel(profile *models.Profile) error {
	if r.FirstName != nil {
		profile.FirstName = *r.FirstName
	}
	if r.LastName != nil {
		profile.LastName = *r.LastName
	}
	if r.Email != nil {
		// Allow users to add/update email (phone users can add email later)
		// If empty string, set to empty (don't clear existing email unless explicitly set)
		if *r.Email == "" {
			profile.Email = ""
		} else {
			profile.Email = *r.Email
		}
	}
	// Note: Phone field removed from profiles - now only in users table
	// if r.Phone != nil {
	//	profile.Phone = *r.Phone
	// }
	if r.DateOfBirth != nil {
		// If date_of_birth is empty string, set it to nil (clear the date)
		if *r.DateOfBirth == "" {
			profile.DateOfBirth = nil
		} else {
			// Parse date string to time.Time
			parsedDate, err := time.Parse("2006-01-02", *r.DateOfBirth)
			if err != nil {
				return fmt.Errorf("invalid date format for date_of_birth: expected YYYY-MM-DD, got: %s", *r.DateOfBirth)
			}
			profile.DateOfBirth = &parsedDate
		}
	}
	if r.Gender != nil {
		profile.Gender = *r.Gender
	}
	if r.AddressLine1 != nil {
		profile.AddressLine1 = *r.AddressLine1
	}
	if r.AddressLine2 != nil {
		profile.AddressLine2 = *r.AddressLine2
	}
	if r.City != nil {
		profile.City = *r.City
	}
	if r.State != nil {
		profile.State = *r.State
	}
	if r.Country != nil {
		profile.Country = *r.Country
	}
	if r.Pincode != nil {
		profile.Pincode = *r.Pincode
	}
	if r.PreferredLanguage != nil {
		profile.PreferredLanguage = *r.PreferredLanguage
	}
	if r.NewsletterSubscribed != nil {
		profile.NewsletterSubscribed = *r.NewsletterSubscribed
	}
	if r.ProfilePictureURL != nil {
		profile.ProfilePictureURL = *r.ProfilePictureURL
	}
	return nil
}
