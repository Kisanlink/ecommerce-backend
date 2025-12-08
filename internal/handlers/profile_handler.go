package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// ProfileHandler handles profile-related HTTP requests
type ProfileHandler struct {
	profileService *services.ProfileService
	userService    *services.UserService
	r2Service      *services.R2Service
	logger         *zap.Logger
}

// NewProfileHandler creates a new profile handler
func NewProfileHandler(serviceManager *services.ServiceManager) *ProfileHandler {
	logger, _ := zap.NewProduction()
	return &ProfileHandler{
		profileService: serviceManager.ProfileService,
		userService:    serviceManager.UserService,
		r2Service:      serviceManager.R2Service,
		logger:         logger,
	}
}

// getUserID extracts and validates user ID from context
func (h *ProfileHandler) getUserID(c *gin.Context) (string, error) {
	// Try firebase_uid first (set by FirebaseAuthMiddleware)
	firebaseUIDValue, exists := c.Get("firebase_uid")
	if exists {
		firebaseUID, ok := firebaseUIDValue.(string)
		if ok && firebaseUID != "" {
			// Look up the actual user in database by Firebase UID
			userResponse, err := h.userService.GetUserByFirebaseUID(c.Request.Context(), firebaseUID)
			if err != nil {
				return "", fmt.Errorf("user not found in database")
			}
			if userResponse != nil && userResponse.Success {
				userData := userResponse.Data.(map[string]interface{})
				if userID, exists := userData["id"].(string); exists {
					return userID, nil
				}
			}
			return "", fmt.Errorf("user not found in database")
		}
	}

	// Fallback to user_id (set by AuthMiddleware for non-Firebase auth)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		return "", fmt.Errorf("user not authenticated")
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
	requestID := c.GetString("request_id")
	h.logger.Info("Auto profile creation requested",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()))

	userID, err := h.getUserID(c)
	if err != nil {
		h.logger.Error("User authentication failed for auto profile creation",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("error_type", "authentication_error"))
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

	h.logger.Info("Creating auto profile for authenticated user",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("firebase_email", firebaseEmail))

	profile, err := h.profileService.CreateAutoProfile(c.Request.Context(), userID, firebaseEmail)
	if err != nil {
		if err.Error() == "profile already exists for user" {
			h.logger.Warn("Profile creation failed - profile already exists",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("error_type", "profile_exists_error"))
			c.JSON(http.StatusConflict, gin.H{"error": "Profile already exists"})
			return
		}

		h.logger.Error("Failed to create auto profile",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("firebase_email", firebaseEmail),
			zap.Error(err),
			zap.String("error_type", "profile_creation_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create profile", "details": err.Error()})
		return
	}

	h.logger.Info("Auto profile created successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("profile_id", profile.ID),
		zap.String("firebase_email", firebaseEmail))

	// Convert to response structure
	profileResponse := responses.ToProfileResponse(profile)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Profile created successfully",
		"data":    profileResponse,
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
	requestID := c.GetString("request_id")
	h.logger.Info("Profile retrieval requested",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()))

	userID, err := h.getUserID(c)
	if err != nil {
		h.logger.Error("User authentication failed for profile retrieval",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("error_type", "authentication_error"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("Retrieving profile for authenticated user",
		zap.String("request_id", requestID),
		zap.String("user_id", userID))

	profile, err := h.profileService.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		if err.Error() == "record not found" {
			h.logger.Warn("Profile not found for user, attempting to create one",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("error_type", "profile_not_found"))

			// Try to get user email for profile creation
			var userEmail string
			firebaseUIDValue, exists := c.Get("firebase_uid")
			if exists {
				firebaseUID, ok := firebaseUIDValue.(string)
				if ok && firebaseUID != "" {
					// Get user details from database
					userResponse, _ := h.userService.GetUserByFirebaseUID(c.Request.Context(), firebaseUID)
					if userResponse != nil && userResponse.Success {
						userData := userResponse.Data.(map[string]interface{})
						if email, exists := userData["email"].(string); exists {
							userEmail = email
						}
					}
				}
			}

			// Attempt to create profile automatically
			profile, err = h.profileService.CreateAutoProfile(c.Request.Context(), userID, userEmail)
			if err != nil {
				h.logger.Error("Failed to auto-create profile",
					zap.String("request_id", requestID),
					zap.String("user_id", userID),
					zap.Error(err))
				c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found and could not be created"})
				return
			}

			h.logger.Info("Profile auto-created successfully",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("profile_id", profile.ID))
		} else {
			h.logger.Error("Failed to retrieve profile",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.Error(err),
				zap.String("error_type", "profile_retrieval_error"))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile", "details": err.Error()})
			return
		}
	}

	h.logger.Info("Profile retrieved successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("profile_id", profile.ID),
		zap.Bool("has_profile_picture", profile.ProfilePictureR2Key != ""),
		zap.Bool("has_full_name", profile.FirstName != "" && profile.LastName != ""))

	// Convert to response structure
	profileResponse := responses.ToProfileResponse(profile)

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile retrieved successfully",
		"data":    profileResponse,
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
	requestID := c.GetString("request_id")
	h.logger.Info("Profile update requested",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()))

	userID, err := h.getUserID(c)
	if err != nil {
		h.logger.Error("User authentication failed for profile update",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("error_type", "authentication_error"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	req := &requests.UpdateProfileRequest{}
	if err := c.ShouldBindJSON(req); err != nil {
		h.logger.Error("Invalid profile update request data",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err),
			zap.String("error_type", "validation_error"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data", "details": err.Error()})
		return
	}

	// Initialize BaseRequest and set user ID
	req.BaseRequest = &base.BaseRequest{}
	req.BaseRequest.SetUserID(userID)

	// Log update details (be careful not to log sensitive data)
	h.logger.Info("Processing profile update",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.Bool("has_first_name", req.FirstName != nil && *req.FirstName != ""),
		zap.Bool("has_last_name", req.LastName != nil && *req.LastName != ""),
		zap.Bool("has_phone", req.Phone != nil && *req.Phone != ""),
		zap.Bool("has_address_line1", req.AddressLine1 != nil && *req.AddressLine1 != ""))

	profile, err := h.profileService.UpdateProfile(c.Request.Context(), req)
	if err != nil {
		if err.Error() == "record not found" {
			h.logger.Warn("Profile update failed - profile not found",
				zap.String("request_id", requestID),
				zap.String("user_id", userID),
				zap.String("error_type", "profile_not_found"))
			c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found"})
			return
		}

		h.logger.Error("Failed to update profile",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err),
			zap.String("error_type", "profile_update_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile", "details": err.Error()})
		return
	}

	h.logger.Info("Profile updated successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("profile_id", profile.ID))

	// Convert to response structure
	profileResponse := responses.ToProfileResponse(profile)

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"data":    profileResponse,
	})
}

// UploadProfileImage uploads a profile image file to R2
// @Summary Upload profile image
// @Description Upload a profile image file to R2 and update the profile
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
	requestID := c.GetString("request_id")
	h.logger.Info("Profile image upload requested",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()))

	userID, err := h.getUserID(c)
	if err != nil {
		h.logger.Error("User authentication failed for profile image upload",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("error_type", "authentication_error"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Get the uploaded file
	file, header, err := c.Request.FormFile("image")
	if err != nil {
		h.logger.Error("No image file provided in request",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err),
			zap.String("error_type", "file_upload_error"))
		c.JSON(http.StatusBadRequest, gin.H{"error": "No image file provided", "details": err.Error()})
		return
	}
	defer file.Close()

	// Log file details for debugging
	h.logger.Info("Profile image file received",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("filename", header.Filename),
		zap.Int64("file_size", header.Size),
		zap.String("content_type", header.Header.Get("Content-Type")))

	// Check if R2 service is available
	if h.r2Service == nil || !h.r2Service.IsConfigured() {
		h.logger.Error("R2 service not configured for profile image upload",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("error_type", "r2_service_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service not available"})
		return
	}

	// Upload image to R2
	h.logger.Info("Uploading profile image to R2",
		zap.String("request_id", requestID),
		zap.String("user_id", userID))

	r2Key, err := h.r2Service.UploadProfileImage(c.Request.Context(), userID, file, header)
	if err != nil {
		h.logger.Error("Failed to upload profile image to R2",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("filename", header.Filename),
			zap.Error(err),
			zap.String("error_type", "r2_upload_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upload image", "details": err.Error()})
		return
	}

	h.logger.Info("Profile image uploaded to R2 successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("r2_key", r2Key))

	// Get public URL for the uploaded image
	h.logger.Info("Getting public URL for uploaded image",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("r2_key", r2Key))

	imageURL, err := h.r2Service.GetProfileImageURL(c.Request.Context(), r2Key, 24*time.Hour)
	if err != nil {
		h.logger.Error("Failed to get public URL for profile image",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("r2_key", r2Key),
			zap.Error(err),
			zap.String("error_type", "url_generation_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate image URL", "details": err.Error()})
		return
	}

	// Update profile with R2 key and URL (using S3 field names for backward compatibility)
	h.logger.Info("Updating profile with uploaded image details",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("r2_key", r2Key))

	profile, err := h.profileService.UpdateProfilePictureWithS3(c.Request.Context(), userID, imageURL, r2Key)
	if err != nil {
		h.logger.Error("Failed to update profile with image details",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("r2_key", r2Key),
			zap.Error(err),
			zap.String("error_type", "profile_update_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile with image", "details": err.Error()})
		return
	}

	h.logger.Info("Profile image upload completed successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("profile_id", profile.ID),
		zap.String("r2_key", r2Key),
		zap.String("filename", header.Filename),
		zap.Int64("file_size", header.Size))

	// Convert to response structure
	profileResponse := responses.ToProfileResponse(profile)

	c.JSON(http.StatusOK, gin.H{
		"message":   "Profile image uploaded successfully",
		"data":      profileResponse,
		"image_url": imageURL,
		"r2_key":    r2Key,
	})
}

// GetProfileImageURL generates a public URL for the profile image
// @Summary Get profile image URL
// @Description Generate a public URL for the profile image (R2 URLs don't expire)
// @Tags Profile
// @Produce json
// @Security FirebaseAuth
// @Param expires_in_hours query int false "Expiry in hours (ignored for R2)" default(1)
// @Success 200 {object} map[string]interface{} "Image URL generated"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "No profile image"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /profile/image/url [get]
func (h *ProfileHandler) GetProfileImageURL(c *gin.Context) {
	requestID := c.GetString("request_id")
	h.logger.Info("Profile image URL generation requested",
		zap.String("request_id", requestID),
		zap.String("method", c.Request.Method),
		zap.String("path", c.Request.URL.Path),
		zap.String("client_ip", c.ClientIP()))

	userID, err := h.getUserID(c)
	if err != nil {
		h.logger.Error("User authentication failed for profile image URL generation",
			zap.String("request_id", requestID),
			zap.Error(err),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")),
			zap.String("error_type", "authentication_error"))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	// Parse expiry duration from query parameter (kept for API compatibility, but R2 URLs don't expire)
	expiryHours := 1
	if expiryParam := c.Query("expires_in_hours"); expiryParam != "" {
		if hours, parseErr := strconv.Atoi(expiryParam); parseErr == nil && hours > 0 {
			expiryHours = hours
		}
	}

	h.logger.Info("Retrieving profile for image URL generation",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.Int("expiry_hours", expiryHours))

	// Get user's profile
	profile, err := h.profileService.GetProfileByUserID(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to retrieve profile for image URL generation",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.Error(err),
			zap.String("error_type", "profile_retrieval_error"))
		c.JSON(http.StatusNotFound, gin.H{"error": "Profile not found", "details": err.Error()})
		return
	}

	if profile.ProfilePictureR2Key == "" {
		h.logger.Warn("No profile image found for URL generation",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("profile_id", profile.ID),
			zap.String("error_type", "no_profile_image"))
		c.JSON(http.StatusNotFound, gin.H{"error": "No profile image found"})
		return
	}

	// Check if R2 service is available
	if h.r2Service == nil || !h.r2Service.IsConfigured() {
		h.logger.Error("R2 service not configured for profile image URL generation",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("error_type", "r2_service_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Storage service not available"})
		return
	}

	h.logger.Info("Generating public URL for profile image",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("profile_id", profile.ID),
		zap.String("r2_key", profile.ProfilePictureR2Key))

	// Generate public URL (R2 public URLs don't expire)
	imageURL, err := h.r2Service.GetProfileImageURL(
		c.Request.Context(),
		profile.ProfilePictureR2Key,
		time.Duration(expiryHours)*time.Hour,
	)
	if err != nil {
		h.logger.Error("Failed to generate public URL for profile image",
			zap.String("request_id", requestID),
			zap.String("user_id", userID),
			zap.String("r2_key", profile.ProfilePictureR2Key),
			zap.Error(err),
			zap.String("error_type", "url_generation_error"))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate image URL", "details": err.Error()})
		return
	}

	h.logger.Info("Profile image URL generated successfully",
		zap.String("request_id", requestID),
		zap.String("user_id", userID),
		zap.String("profile_id", profile.ID),
		zap.String("r2_key", profile.ProfilePictureR2Key))

	c.JSON(http.StatusOK, gin.H{
		"image_url":  imageURL,
		"expires_in": "never (R2 public URL)",
		"r2_key":     profile.ProfilePictureR2Key,
	})
}
