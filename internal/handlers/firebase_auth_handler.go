package handlers

import (
	"context"
	"net/http"

	"firebase.google.com/go/v4/auth"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// FirebaseAuthHandler handles Firebase authentication operations
type FirebaseAuthHandler struct{}

// NewFirebaseAuthHandler creates a new Firebase auth handler
func NewFirebaseAuthHandler() *FirebaseAuthHandler {
	return &FirebaseAuthHandler{}
}

// Signup creates a new user in Firebase
func (h *FirebaseAuthHandler) Signup(c *gin.Context) {
	var req requests.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := context.Background()
	client := config.GetFirebaseAuth()
	if client == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase not initialized", nil)
		return
	}

	// Create user in Firebase
	params := (&auth.UserToCreate{}).
		Email(req.Email).
		Password(req.Password).
		DisplayName(req.Name).
		EmailVerified(false)

	userRecord, err := client.CreateUser(ctx, params)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Failed to create user", err)
		return
	}

	// Generate custom token for immediate login
	token, err := client.CustomToken(ctx, userRecord.UID)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	utils.RespondWithSuccess(c, gin.H{
		"message": "User created successfully",
		"user": gin.H{
			"uid":   userRecord.UID,
			"email": userRecord.Email,
			"name":  userRecord.DisplayName,
		},
		"custom_token": token,
	})
}

// VerifyToken verifies a Firebase ID token
func (h *FirebaseAuthHandler) VerifyToken(c *gin.Context) {
	var req requests.VerifyTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := context.Background()
	client := config.GetFirebaseAuth()
	if client == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase not initialized", nil)
		return
	}

	// Verify the ID token
	token, err := client.VerifyIDToken(ctx, req.IDToken)
	if err != nil {
		utils.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired token", err)
		return
	}

	// Get user record
	userRecord, err := client.GetUser(ctx, token.UID)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get user info", err)
		return
	}

	utils.RespondWithSuccess(c, gin.H{
		"message": "Token verified successfully",
		"user": gin.H{
			"uid":            userRecord.UID,
			"email":          userRecord.Email,
			"name":           userRecord.DisplayName,
			"email_verified": userRecord.EmailVerified,
		},
		"token_info": gin.H{
			"issued_at": token.IssuedAt,
			"expires":   token.Expires,
		},
	})
}

// GetProfile gets the current user's profile
func (h *FirebaseAuthHandler) GetProfile(c *gin.Context) {
	uid, exists := c.Get("firebase_uid")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	ctx := context.Background()
	client := config.GetFirebaseAuth()
	if client == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase not initialized", nil)
		return
	}

	userRecord, err := client.GetUser(ctx, uid.(string))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get user info", err)
		return
	}

	utils.RespondWithSuccess(c, gin.H{
		"user": gin.H{
			"uid":            userRecord.UID,
			"email":          userRecord.Email,
			"name":           userRecord.DisplayName,
			"email_verified": userRecord.EmailVerified,
			"created_at":     userRecord.UserMetadata.CreationTimestamp,
			"last_login":     userRecord.UserMetadata.LastLogInTimestamp,
		},
	})
}

// UpdateProfile updates the current user's profile
func (h *FirebaseAuthHandler) UpdateProfile(c *gin.Context) {
	uid, exists := c.Get("firebase_uid")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	var updateData struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	if err := c.ShouldBindJSON(&updateData); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := context.Background()
	client := config.GetFirebaseAuth()
	if client == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase not initialized", nil)
		return
	}

	params := &auth.UserToUpdate{}
	if updateData.Name != "" {
		params = params.DisplayName(updateData.Name)
	}
	if updateData.Email != "" {
		params = params.Email(updateData.Email)
	}

	userRecord, err := client.UpdateUser(ctx, uid.(string), params)
	if err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Failed to update user", err)
		return
	}

	utils.RespondWithSuccess(c, gin.H{
		"message": "Profile updated successfully",
		"user": gin.H{
			"uid":   userRecord.UID,
			"email": userRecord.Email,
			"name":  userRecord.DisplayName,
		},
	})
}

// DeleteAccount deletes the current user's account
func (h *FirebaseAuthHandler) DeleteAccount(c *gin.Context) {
	uid, exists := c.Get("firebase_uid")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	ctx := context.Background()
	client := config.GetFirebaseAuth()
	if client == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase not initialized", nil)
		return
	}

	err := client.DeleteUser(ctx, uid.(string))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to delete user", err)
		return
	}

	utils.RespondWithSuccess(c, gin.H{
		"message": "Account deleted successfully",
	})
}
