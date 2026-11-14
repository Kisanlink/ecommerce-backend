package services

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"golang.org/x/crypto/bcrypt"
)

// getPhoneNumberString safely extracts phone number string from pointer
func getPhoneNumberString(phoneNum *string) string {
	if phoneNum == nil {
		return ""
	}
	return *phoneNum
}

// UserService provides user-related business logic
type UserService struct {
	userRepo *repositories.UserRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo *repositories.UserRepository) *UserService {
	service := &UserService{
		userRepo: userRepo,
	}

	// Initialize the ID generator with existing user IDs
	service.initializeIDGenerator()

	return service
}

// initializeIDGenerator initializes the global ID generator with existing user IDs
func (s *UserService) initializeIDGenerator() {
	ctx := context.Background()

	// Get all existing users to extract their IDs
	filter := base.NewFilterBuilder().
		Page(1, 10000). // Get all users
		Build()

	result, err := s.userRepo.Find(ctx, filter)
	if err != nil {
		// If we can't get existing users, start from 0
		fmt.Printf("Warning: Could not initialize user ID generator: %v\n", err)
		return
	}

	// Extract all existing IDs
	existingIDs := make([]string, len(result))
	for i, user := range result {
		existingIDs[i] = user.GetID()
	}

	// Initialize the global generator with existing IDs
	hash.InitializeGlobalCountersFromDatabase("USER", existingIDs, hash.Small)
	fmt.Printf("Initialized user ID generator with %d existing user IDs\n", len(existingIDs))
}

// createUserWithTimestampID creates a new user with timestamp-based ID to avoid collisions
func (s *UserService) createUserWithTimestampID(email, name, password string) *models.User {
	// Generate timestamp-based ID to avoid collisions
	timestampID, err := hash.GenerateRandomIDWithPattern("USER", hash.Small, "timestamp")
	if err != nil {
		// Fallback to hash-based ID if timestamp fails
		hashID, hashErr := hash.GenerateRandomIDWithPattern("USER", hash.Small, "hash")
		if hashErr != nil {
			// Final fallback to default incremental
			incID, _ := hash.GenerateRandomID("USER", hash.Small)
			timestampID = incID
		} else {
			timestampID = hashID
		}
	}

	// Create base model and set the custom ID
	baseModel := base.NewBaseModel("USER", hash.Small)
	baseModel.SetID(timestampID)

	return &models.User{
		BaseModel:   baseModel,
		Email:       &email,
		Name:        name,
		Password:    password,
		PhoneNumber: nil, // nil for email-based users (no phone number)
		IsVerified:  false,
		IsActive:    true,
		Role:        "user",
	}
}

// CreateUser creates a new user with hashed password
func (s *UserService) CreateUser(ctx context.Context, req *requests.SignupRequest) (*base.BaseResponse, error) {
	// Check if user already exists
	existingUser, _ := s.userRepo.GetUserByEmail(ctx, req.Email)
	if existingUser != nil {
		return base.NewErrorResponse("User already exists",
			base.NewValidationError("email", "User with this email already exists")), nil
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user model with timestamp-based ID to avoid collisions
	user := s.createUserWithTimestampID(req.Email, req.Name, string(hashedPassword))

	// Save to NeonDB with retry logic for duplicate key errors
	maxRetries := 3
	var createErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		// Save user to NeonDB via UserRepository (which uses pgManager connected to NeonDB)
		createErr = s.userRepo.Create(ctx, user)
		if createErr == nil {
			// Successfully created user in NeonDB, break out of retry loop
			fmt.Printf("✅ User successfully saved to NeonDB: ID=%s, Email=%s\n", user.GetID(), req.Email)
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

		// Check for phone number duplicate error
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "uni_users_phone_number") {
			return base.NewErrorResponse("User already exists",
				base.NewValidationError("phone_number", "User with this phone number already exists")), nil
		}

		// Check for primary key duplicate error
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "users_pkey") {
			// Generate a new ID and try again with timestamp-based ID
			user = s.createUserWithTimestampID(req.Email, req.Name, string(hashedPassword))
			fmt.Printf("Duplicate key error detected, retrying with new timestamp-based ID (attempt %d/%d): %s\n", attempt+1, maxRetries, user.GetID())
			continue
		}

		// If it's not a duplicate key error, return the error immediately
		return nil, fmt.Errorf("failed to create user: %w", createErr)
	}

	// If we still have an error after all retries, return it
	if createErr != nil {
		return nil, fmt.Errorf("failed to create user after %d attempts: %w", maxRetries, createErr)
	}

	// Return response without password
	var emailStr string
	if user.Email != nil {
		emailStr = *user.Email
	}
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        emailStr,
		"name":         user.Name,
		"firebase_uid": user.FirebaseUID,
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
		"role":         user.Role,
		"created_at":   user.GetCreatedAt(),
	}

	return base.NewSuccessResponse("User created successfully", userData), nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(ctx context.Context, userID string) (*base.BaseResponse, error) {
	if userID == "" {
		return base.NewErrorResponse("Invalid user ID",
			base.NewValidationError("id", "User ID cannot be empty")), nil
	}

	user, err := s.userRepo.GetByID(ctx, userID, &models.User{})
	if err != nil {
		return base.NewErrorResponse("User not found",
			base.NewNotFoundError("User", userID)), nil
	}

	// Convert to response structure
	userResponse := responses.ToUserResponse(user)

	return base.NewSuccessResponse("User retrieved successfully", userResponse), nil
}

// GetUserByEmail retrieves a user by email
func (s *UserService) GetUserByEmail(ctx context.Context, email string) (*base.BaseResponse, error) {
	if email == "" {
		return base.NewErrorResponse("Invalid email",
			base.NewValidationError("email", "Email cannot be empty")), nil
	}

	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return base.NewErrorResponse("User not found",
			base.NewNotFoundError("User", email)), nil
	}

	// Return user data without password
	var emailStr string
	if user.Email != nil {
		emailStr = *user.Email
	}
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        emailStr,
		"name":         user.Name,
		"firebase_uid": user.FirebaseUID,
		"phone_number": getPhoneNumberString(user.PhoneNumber),
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
		"role":         user.Role,
		"created_at":   user.GetCreatedAt(),
		"updated_at":   user.GetUpdatedAt(),
	}

	return base.NewSuccessResponse("User retrieved successfully", userData), nil
}

// GetUserByFirebaseUID retrieves a user by Firebase UID
func (s *UserService) GetUserByFirebaseUID(ctx context.Context, firebaseUID string) (*base.BaseResponse, error) {
	if firebaseUID == "" {
		return base.NewErrorResponse("Invalid Firebase UID",
			base.NewValidationError("firebase_uid", "Firebase UID cannot be empty")), nil
	}

	user, err := s.userRepo.GetUserByFirebaseUID(ctx, firebaseUID)
	if err != nil {
		return base.NewErrorResponse("User not found",
			base.NewNotFoundError("User", firebaseUID)), nil
	}

	// Return user data without password
	var emailStr string
	if user.Email != nil {
		emailStr = *user.Email
	}
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        emailStr,
		"name":         user.Name,
		"firebase_uid": user.FirebaseUID,
		"phone_number": getPhoneNumberString(user.PhoneNumber),
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
		"role":         user.Role,
		"created_at":   user.GetCreatedAt(),
		"updated_at":   user.GetUpdatedAt(),
	}

	return base.NewSuccessResponse("User retrieved successfully", userData), nil
}

// UpdateUserFirebaseUID updates a user's Firebase UID
func (s *UserService) UpdateUserFirebaseUID(ctx context.Context, userID, firebaseUID string) error {
	user, err := s.userRepo.GetByID(ctx, userID, &models.User{})
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// SECURITY (#5 account takeover): never overwrite an existing, DIFFERENT Firebase UID. If the
	// record is already linked to another identity, refuse — re-linking must be an explicit,
	// authenticated migration, not an automatic side effect of someone signing in with the email.
	if user.FirebaseUID != nil && *user.FirebaseUID != "" && *user.FirebaseUID != firebaseUID {
		return fmt.Errorf("user %s is already linked to a different firebase identity", userID)
	}

	user.FirebaseUID = &firebaseUID
	user.SetUpdatedBy("system")

	return s.userRepo.Update(ctx, user)
}

// UpdateUser updates user information
func (s *UserService) UpdateUser(ctx context.Context, userID string, updateData map[string]interface{}) (*base.BaseResponse, error) {
	if userID == "" {
		return base.NewErrorResponse("Invalid user ID",
			base.NewValidationError("id", "User ID cannot be empty")), nil
	}

	user, err := s.userRepo.GetByID(ctx, userID, &models.User{})
	if err != nil {
		return base.NewErrorResponse("User not found",
			base.NewNotFoundError("User", userID)), nil
	}

	// Update allowed fields
	if name, ok := updateData["name"].(string); ok && name != "" {
		user.Name = name
	}
	if phoneNumber, ok := updateData["phone_number"].(string); ok && phoneNumber != "" {
		user.PhoneNumber = &phoneNumber
	} else if phoneNumber == "" {
		// Clear phone number if empty string provided
		user.PhoneNumber = nil
	}
	if isVerified, ok := updateData["is_verified"].(bool); ok {
		user.IsVerified = isVerified
	}
	if isActive, ok := updateData["is_active"].(bool); ok {
		user.IsActive = isActive
	}
	if role, ok := updateData["role"].(string); ok && role != "" {
		// Store a canonical role (trimmed, lowercase) so role checks that compare exactly — e.g. the
		// first-admin bootstrap count — agree with AdminGuard's case/space-insensitive match and a
		// non-canonical value like "Admin" can never create an uncounted admin.
		user.Role = strings.ToLower(strings.TrimSpace(role))
	}

	user.SetUpdatedBy("system")

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Return updated user data without password
	var emailStr string
	if user.Email != nil {
		emailStr = *user.Email
	}
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        emailStr,
		"name":         user.Name,
		"firebase_uid": user.FirebaseUID,
		"phone_number": getPhoneNumberString(user.PhoneNumber),
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
		"role":         user.Role,
		"created_at":   user.GetCreatedAt(),
		"updated_at":   user.GetUpdatedAt(),
	}

	return base.NewSuccessResponse("User updated successfully", userData), nil
}

// ListUsers retrieves paginated list of users
func (s *UserService) ListUsers(ctx context.Context, page, pageSize int) (*base.PaginatedResponse, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	users, err := s.userRepo.GetActiveUsers(ctx, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}

	// Convert to response format without passwords
	var userList []map[string]interface{}
	for _, user := range users {
		var emailStr string
		if user.Email != nil {
			emailStr = *user.Email
		}
		userData := map[string]interface{}{
			"id":           user.GetID(),
			"email":        emailStr,
			"name":         user.Name,
			"firebase_uid": user.FirebaseUID,
			"phone_number": getPhoneNumberString(user.PhoneNumber),
			"is_verified":  user.IsVerified,
			"is_active":    user.IsActive,
			"role":         user.Role,
			"created_at":   user.GetCreatedAt(),
			"updated_at":   user.GetUpdatedAt(),
		}
		userList = append(userList, userData)
	}

	// Get total count for pagination (exclude soft-deleted)
	// Using the custom CountActiveUsers method to avoid table identification issues
	totalCount, err := s.userRepo.CountActiveUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	pagination := base.NewPaginationInfo(page, pageSize, int(totalCount))
	return base.NewPaginatedResponse("Users retrieved successfully", userList, pagination), nil
}

// VerifyPassword verifies user password
func (s *UserService) VerifyPassword(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.userRepo.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.IsActive {
		return nil, errors.New("user account is inactive")
	}

	// Compare password
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return nil, errors.New("invalid password")
	}

	return user, nil
}

// SoftDeleteUser soft deletes a user
func (s *UserService) SoftDeleteUser(ctx context.Context, userID, deletedBy string) (*base.BaseResponse, error) {
	if userID == "" {
		return base.NewErrorResponse("Invalid user ID",
			base.NewValidationError("id", "User ID cannot be empty")), nil
	}

	err := s.userRepo.SoftDelete(ctx, userID, deletedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to delete user: %w", err)
	}

	return base.NewSuccessResponse("User deleted successfully", nil), nil
}

// CreatePhoneUser creates a new user with phone number (no email/password)
func (s *UserService) CreatePhoneUser(ctx context.Context, req *requests.PhoneSignupRequest) (*base.BaseResponse, error) {
	fmt.Printf("CreatePhoneUser called with: PhoneNumber=%s, Name=%s\n", req.PhoneNumber, req.Name)

	// Check if user already exists by phone number
	existingUser, err := s.userRepo.GetUserByPhoneNumber(ctx, req.PhoneNumber)
	if err != nil {
		// If error is "not found", that's okay - user doesn't exist yet
		if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "record not found") {
			fmt.Printf("Error checking existing user: %v\n", err)
			// Continue anyway - database constraint will catch duplicates
		}
	}
	if existingUser != nil {
		fmt.Printf("User already exists with phone number: %s\n", req.PhoneNumber)
		return base.NewErrorResponse("User already exists",
			base.NewValidationError("phone_number", "User with this phone number already exists")), nil
	}

	// Create phone user model
	user := models.NewPhoneUser(req.PhoneNumber, req.Name)
	var emailStr string
	if user.Email != nil {
		emailStr = *user.Email
	}
	phoneNumStr := getPhoneNumberString(user.PhoneNumber)
	fmt.Printf("Created user model: ID=%s, PhoneNumber=%s, Name=%s, Email=%s, Password=%s\n",
		user.GetID(), phoneNumStr, user.Name, emailStr, user.Password)

	// Save to database with retry logic for duplicate key errors
	maxRetries := 3
	var createErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		fmt.Printf("Attempting to create user in database (attempt %d/%d)...\n", attempt+1, maxRetries)
		createErr = s.userRepo.Create(ctx, user)
		if createErr == nil {
			fmt.Printf("Successfully created user in database with ID: %s\n", user.GetID())
			// Successfully created user, break out of retry loop
			break
		}

		fmt.Printf("Database creation failed (attempt %d/%d): %v\n", attempt+1, maxRetries, createErr)

		// Check if it's a duplicate key error
		errorStr := createErr.Error()

		// Check for phone number duplicate error
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "uni_users_phone_number") {
			// Phone number already exists - return user-friendly error
			fmt.Printf("Phone number already exists: %s\n", req.PhoneNumber)
			return base.NewErrorResponse("User already exists",
				base.NewValidationError("phone_number", "User with this phone number already exists")), nil
		}

		// Check for primary key duplicate error
		if strings.Contains(errorStr, "duplicate key value violates unique constraint") &&
			strings.Contains(errorStr, "users_pkey") {
			// Generate a new ID and try again
			user = models.NewPhoneUser(req.PhoneNumber, req.Name)
			fmt.Printf("Duplicate key error detected, retrying with new ID (attempt %d/%d): %s\n", attempt+1, maxRetries, user.GetID())
			continue
		}

		// If it's not a duplicate key error, return the error immediately
		fmt.Printf("Non-retryable error occurred: %v\n", createErr)
		return nil, fmt.Errorf("failed to create phone user: %w", createErr)
	}

	// If we still have an error after all retries, return it
	if createErr != nil {
		return nil, fmt.Errorf("failed to create phone user after %d attempts: %w", maxRetries, createErr)
	}

	// Return response without password
	var firebaseUID string
	if user.FirebaseUID != nil {
		firebaseUID = *user.FirebaseUID
	}

	userData := map[string]interface{}{
		"id":           user.GetID(),
		"phone_number": getPhoneNumberString(user.PhoneNumber),
		"name":         user.Name,
		"firebase_uid": firebaseUID,
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
		"role":         user.Role,
		"created_at":   user.GetCreatedAt(),
	}

	fmt.Printf("Returning success response for user: %+v\n", userData)
	return base.NewSuccessResponse("Phone user created successfully", userData), nil
}

// GetUserByPhoneNumber retrieves a user by phone number
func (s *UserService) GetUserByPhoneNumber(ctx context.Context, phoneNumber string) (*base.BaseResponse, error) {
	if phoneNumber == "" {
		return base.NewErrorResponse("Invalid phone number",
			base.NewValidationError("phone_number", "Phone number cannot be empty")), nil
	}

	user, err := s.userRepo.GetUserByPhoneNumber(ctx, phoneNumber)
	if err != nil {
		return base.NewErrorResponse("User not found",
			base.NewNotFoundError("User", phoneNumber)), nil
	}

	// Return user data without password
	var firebaseUID string
	if user.FirebaseUID != nil {
		firebaseUID = *user.FirebaseUID
	}

	var emailStr string
	if user.Email != nil {
		emailStr = *user.Email
	}

	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        emailStr,
		"name":         user.Name,
		"firebase_uid": firebaseUID,
		"phone_number": getPhoneNumberString(user.PhoneNumber),
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
		"created_at":   user.GetCreatedAt(),
		"updated_at":   user.GetUpdatedAt(),
	}

	return base.NewSuccessResponse("User retrieved successfully", userData), nil
}
