package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// CreateGoogleUser creates a new user from Google OAuth (no password required)
func (s *UserService) CreateGoogleUser(ctx context.Context, req *requests.SignupRequest, firebaseUID string) (*base.BaseResponse, error) {
	fmt.Printf("CreateGoogleUser called with: Email=%s, Name=%s, FirebaseUID=%s\n", req.Email, req.Name, firebaseUID)

	// Check if user already exists by email
	existingUser, _ := s.userRepo.GetUserByEmail(ctx, req.Email)
	if existingUser != nil {
		fmt.Printf("User already exists with email: %s\n", req.Email)
		return base.NewErrorResponse("User already exists",
			base.NewValidationError("email", "User with this email already exists")), nil
	}

	// Check if user already exists by Firebase UID
	existingUserByUID, _ := s.userRepo.GetUserByFirebaseUID(ctx, firebaseUID)
	if existingUserByUID != nil {
		fmt.Printf("User already exists with Firebase UID: %s\n", firebaseUID)
		return base.NewErrorResponse("User already exists",
			base.NewValidationError("firebase_uid", "User with this Google account already exists")), nil
	}

	// Create user model with timestamp-based ID to avoid collisions
	// Google users don't have passwords, so we set an empty password
	user := s.createUserWithTimestampID(req.Email, req.Name, "")

	// Set Firebase UID for Google user
	user.FirebaseUID = &firebaseUID

	// Google users are considered verified
	user.IsVerified = true

	// Save to NeonDB with retry logic for duplicate key errors
	maxRetries := 3
	var createErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Save user to NeonDB via UserRepository
		createErr = s.userRepo.Create(ctx, user)
		if createErr == nil {
			// Successfully created user in NeonDB
			fmt.Printf("✅ Google user successfully saved to NeonDB: ID=%s, Email=%s\n", user.GetID(), req.Email)
			break
		}

		// Check if it's a duplicate key error
		errorStr := createErr.Error()

		// Check for email duplicate error
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "users_email_key") {
			return base.NewErrorResponse("User already exists",
				base.NewValidationError("email", "User with this email already exists")), nil
		}

		// Check for Firebase UID duplicate error
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "users_firebase_uid_key") {
			return base.NewErrorResponse("User already exists",
				base.NewValidationError("firebase_uid", "User with this Google account already exists")), nil
		}

		// Check for primary key duplicate error
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "users_pkey") {
			// Generate a new ID and try again with timestamp-based ID
			user = s.createUserWithTimestampID(req.Email, req.Name, "")
			user.FirebaseUID = &firebaseUID
			user.IsVerified = true
			fmt.Printf("Duplicate key error detected, retrying with new timestamp-based ID (attempt %d/%d): %s\n", attempt+1, maxRetries, user.GetID())
			continue
		}

		// If it's not a duplicate key error, return the error immediately
		return nil, fmt.Errorf("failed to create Google user: %w", createErr)
	}

	// If we still have an error after all retries, return it
	if createErr != nil {
		return nil, fmt.Errorf("failed to create Google user after %d attempts: %w", maxRetries, createErr)
	}

	// Return response with user data (without password)
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        req.Email,
		"name":         user.Name,
		"firebase_uid": firebaseUID,
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
		"created_at":   user.GetCreatedAt(),
	}

	fmt.Printf("Returning success response for Google user: %+v\n", userData)
	return base.NewSuccessResponse("Google user created successfully", userData), nil
}