package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/gin-gonic/gin"
)

// ProfileHandler handles profile-related HTTP requests
type ProfileHandler struct {
	profileService *services.ProfileService
	s3Service      *services.S3Service
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(serviceManager *services.ServiceManager) *ProfileHandler {
	return &ProfileHandler{
		profileService: serviceManager.ProfileService,
		s3Service:      serviceManager.S3Service,
	}
}

// getUserID extracts and validates user ID from context
func (h *ProfileHandler) getUserID(c *gin.Context) (string, error) {
	// Try firebase_uid first (set by FirebaseAuthMiddleware)
	userIDValue, exists := c.Get("firebase_uid")
	if !exists {
		// Fallback to user_id (set by AuthMiddleware)
		userIDValue, exists = c.Get("user_id")
		if !exists {
			return "", fmt.Errorf("user not authenticated")
		}
	}

	userID, ok := userIDValue.(string)
	if !ok || userID == "" {
		return "", fmt.Errorf("invalid user ID format")
	}

	return userID, nil
}

// CreateAutoProfile automatically creates a profile using user information
// @Summary Create profile (auto)
// @Description Create a profile for the authenticated user using Firebase info
// @Tags Profile
// @Produce json
// @Security FirebaseAuth
// @Success 201 {object} map[string]interface{} "Profile created"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 409 {object} map[string]interface{} "Profile already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /profile/auto [post]
func (h *ProfileHandler) CreateAutoProfile(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Get Firebase email from context if available
	firebaseEmail := ""
	if emailValue, exists := c.Get("firebase_email"); exists {
		if email, ok := emailValue.(string); ok {
			firebaseEmail = email
		}
	}

	profile, err := h.profileService.CreateAutoProfile(c.Request.Context(), userID, firebaseEmail)
	if err != nil {
		if err.Error() == "profile already exists for user" {
			c.JSON(http.StatusConflict, gin.H{"error": "Profile already exists"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create profile", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Profile created successfully",
		"data":    profile,
	})
}

// GetProfile retrieves the user's profile
// @Summary Get profile
// @Description Get the authenticated user's profile
// @Tags Profile
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Profile retrieved"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Profile not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /profile [get]
func (h *ProfileHandler) GetProfile(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	profile, err := h.profileService.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile retrieved successfully",
		"data":    profile,
	})
}

// UpdateProfile updates the user's profile
// @Summary Update profile
// @Description Update profile fields for the authenticated user
// @Tags Profile
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.UpdateProfileRequest true "Profile update"
// @Success 200 {object} map[string]interface{} "Profile updated"
// @Failure 400 {object} map[string]interface{} "Invalid request data"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Profile not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /profile [put]
func (h *ProfileHandler) UpdateProfile(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	req := &requests.UpdateProfileRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Initialize BaseRequest and set user ID
	req.BaseRequest = &base.BaseRequest{}
	req.BaseRequest.SetUserID(userID)

	profile, err := h.profileService.UpdateProfile(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"data":    profile,
	})
}

// UploadProfileImage uploads a profile image file to S3
// @Summary Upload profile image
// @Description Upload a profile image file to S3 and update the profile
// @Tags Profile
// @Accept multipart/form-data
// @Produce json
// @Security FirebaseAuth
// @Param image formData file true "Profile image file"
// @Success 200 {object} map[string]interface{} "Image uploaded"
// @Failure 400 {object} map[string]interface{} "No image or bad request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /profile/image [post]
func (h *ProfileHandler) UploadProfileImage(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided", "details": err.Error()})
		return
	}
	defer file.Close()

	// Upload image to S3
	s3Key, err := h.s3Service.UploadProfileImage(c.Request.Context(), userID, file, header)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image", "details": err.Error()})
		return
	}

	// Generate presigned URL for immediate access
	imageURL, err := h.s3Service.GetProfileImageURL(c.Request.Context(), s3Key, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate image URL", "details": err.Error()})
		return
	}

	// Update profile with S3 key and URL
	profile, err := h.profileService.UpdateProfilePictureWithS3(c.Request.Context(), userID, imageURL, s3Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile with image", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Profile image uploaded successfully",
		"data":      profile,
		"image_url": imageURL,
		"s3_key":    s3Key,
	})
}

// GetProfileImageURL generates a presigned URL for the profile image
// @Summary Get profile image URL
// @Description Generate a presigned URL for the profile image
// @Tags Profile
// @Produce json
// @Security FirebaseAuth
// @Param expires_in_hours query int false "Expiry in hours" default(1)
// @Success 200 {object} map[string]interface{} "Image URL generated"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "No profile image"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /profile/image/url [get]
func (h *ProfileHandler) GetProfileImageURL(c *gin.Context) {
	userID, err := h.getUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Get user's profile
	profile, err := h.profileService.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found", "details": err.Error()})
		return
	}

	if profile.ProfilePictureS3Key == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "No profile image found"})
		return
	}

	// Parse expiry duration from query parameter (default 1 hour)
	expiryHours := 1
	if expiryParam := c.Query("expires_in_hours"); expiryParam != "" {
		if hours, parseErr := strconv.Atoi(expiryParam); parseErr == nil && hours > 0 {
			expiryHours = hours
		}
	}

	// Generate presigned URL
	imageURL, err := h.s3Service.GetProfileImageURL(
		c.Request.Context(),
		profile.ProfilePictureS3Key,
		time.Duration(expiryHours)*time.Hour,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate image URL", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"image_url":  imageURL,
		"expires_in": fmt.Sprintf("%d hours", expiryHours),
		"s3_key":     profile.ProfilePictureS3Key,
	})
}
