package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"golang.org/x/crypto/bcrypt"
)

// UserService provides user-related business logic
type UserService struct {
	userRepo *repositories.UserRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo *repositories.UserRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
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

	// Create user model
	user := models.NewUser(req.Email, req.Name, string(hashedPassword))

	// Save to database
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Return response without password
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        user.Email,
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

	// Return user data without password
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        user.Email,
		"name":         user.Name,
		"firebase_uid": user.FirebaseUID,
		"phone_number": user.PhoneNumber,
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
		"role":         user.Role,
		"created_at":   user.GetCreatedAt(),
		"updated_at":   user.GetUpdatedAt(),
	}

	return base.NewSuccessResponse("User retrieved successfully", userData), nil
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
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        user.Email,
		"name":         user.Name,
		"firebase_uid": user.FirebaseUID,
		"phone_number": user.PhoneNumber,
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
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
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        user.Email,
		"name":         user.Name,
		"firebase_uid": user.FirebaseUID,
		"phone_number": user.PhoneNumber,
		"is_verified":  user.IsVerified,
		"is_active":    user.IsActive,
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

	user.FirebaseUID = firebaseUID
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
	if phoneNumber, ok := updateData["phone_number"].(string); ok {
		user.PhoneNumber = phoneNumber
	}
	if isVerified, ok := updateData["is_verified"].(bool); ok {
		user.IsVerified = isVerified
	}
	if isActive, ok := updateData["is_active"].(bool); ok {
		user.IsActive = isActive
	}
	if role, ok := updateData["role"].(string); ok && role != "" {
		user.Role = role
	}

	user.SetUpdatedBy("system")

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	// Return updated user data without password
	userData := map[string]interface{}{
		"id":           user.GetID(),
		"email":        user.Email,
		"name":         user.Name,
		"firebase_uid": user.FirebaseUID,
		"phone_number": user.PhoneNumber,
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
		userData := map[string]interface{}{
			"id":           user.GetID(),
			"email":        user.Email,
			"name":         user.Name,
			"firebase_uid": user.FirebaseUID,
			"phone_number": user.PhoneNumber,
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
