package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// UserResponse represents a user response (authentication data only)
type UserResponse struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	Name        string    `json:"name"`
	FirebaseUID string    `json:"firebase_uid,omitempty"`
	PhoneNumber string    `json:"phone_number"`
	IsVerified  bool      `json:"is_verified"`
	IsActive    bool      `json:"is_active"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// UserWithProfileResponse represents a user with their profile information
type UserWithProfileResponse struct {
	User    UserResponse    `json:"user"`
	Profile ProfileResponse `json:"profile,omitempty"`
}

// UserListResponse represents a list of users
type UserListResponse struct {
	Users      []UserResponse `json:"users"`
	Total      int64          `json:"total"`
	Page       int            `json:"page"`
	Limit      int            `json:"limit"`
	TotalPages int            `json:"total_pages"`
}

// UserStatsResponse represents user statistics
type UserStatsResponse struct {
	TotalUsers        int64 `json:"total_users"`
	ActiveUsers       int64 `json:"active_users"`
	VerifiedUsers     int64 `json:"verified_users"`
	NewUsersToday     int64 `json:"new_users_today"`
	NewUsersThisWeek  int64 `json:"new_users_this_week"`
	NewUsersThisMonth int64 `json:"new_users_this_month"`
}

// ToUserResponse converts a User model to UserResponse
func ToUserResponse(user *models.User) UserResponse {
	var firebaseUID string
	if user.FirebaseUID != nil {
		firebaseUID = *user.FirebaseUID
	}

	var emailStr string
	if user.Email != nil {
		emailStr = *user.Email
	}

	return UserResponse{
		ID:          user.ID,
		Email:       emailStr,
		Name:        user.Name,
		FirebaseUID: firebaseUID,
		PhoneNumber: user.PhoneNumber,
		IsVerified:  user.IsVerified,
		IsActive:    user.IsActive,
		Role:        user.Role,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
	}
}

// ToUserResponseList converts a slice of User models to UserResponse slice
func ToUserResponseList(users []models.User) []UserResponse {
	responses := make([]UserResponse, len(users))
	for i, user := range users {
		responses[i] = ToUserResponse(&user)
	}
	return responses
}

// ToUserWithProfileResponse converts User and Profile models to UserWithProfileResponse
func ToUserWithProfileResponse(user *models.User, profile *models.Profile) UserWithProfileResponse {
	response := UserWithProfileResponse{
		User: ToUserResponse(user),
	}

	if profile != nil {
		response.Profile = ToProfileResponse(profile)
	}

	return response
}
