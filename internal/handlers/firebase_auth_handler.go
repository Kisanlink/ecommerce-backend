package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"firebase.google.com/go/v4/auth"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// FirebaseAuthHandler handles Firebase authentication operations
type FirebaseAuthHandler struct {
	userService *services.UserService
}

// NewFirebaseAuthHandler creates a new Firebase auth handler
func NewFirebaseAuthHandler(userService *services.UserService) *FirebaseAuthHandler {
	return &FirebaseAuthHandler{
		userService: userService,
	}
}

// Firebase Auth REST API structures
type FirebaseSignInRequest struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type FirebaseSignInResponse struct {
	Kind         string `json:"kind"`
	LocalId      string `json:"localId"`
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	Registered   bool   `json:"registered"`
}

type FirebaseErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Errors  []struct {
			Message string `json:"message"`
			Domain  string `json:"domain"`
			Reason  string `json:"reason"`
		} `json:"errors"`
	} `json:"error"`
}

// Signup creates a new user in Firebase and stores their details in local database
// @Summary Create a new user account
// @Description Register a new user with email and password, creating accounts in both local database and Firebase
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body requests.SignupRequest true "User signup information"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request or user already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/signup [post]
func (h *FirebaseAuthHandler) Signup(c *gin.Context) {
	var req requests.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := context.Background()

	// Step 1: Create user in local database first
	userResponse, err := h.userService.CreateUser(ctx, &req)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to create user in database", err)
		return
	}
	if userResponse.Success == false {
		utils.RespondWithError(c, http.StatusBadRequest, userResponse.Message, userResponse.Error)
		return
	}

	// Extract user data from response
	userData, ok := userResponse.Data.(map[string]interface{})
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid user data format", nil)
		return
	}
	userID := userData["id"].(string)

	// Step 2: Create user in Firebase
	client := config.GetFirebaseAuth()
	if client == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase not initialized", nil)
		return
	}

	params := (&auth.UserToCreate{}).
		Email(req.Email).
		Password(req.Password).
		DisplayName(req.Name).
		EmailVerified(false)

	userRecord, err := client.CreateUser(ctx, params)
	if err != nil {
		// If Firebase creation fails, we should remove the user from local database
		// For now, we'll just log this error and continue
		utils.RespondWithError(c, http.StatusBadRequest, "Failed to create user in Firebase", err)
		return
	}

	// Step 3: Update local user record with Firebase UID
	err = h.userService.UpdateUserFirebaseUID(ctx, userID, userRecord.UID)
	if err != nil {
		// Firebase user was created but local update failed
		// In production, you might want to queue this for retry
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to link Firebase account", err)
		return
	}

	// Step 4: Automatically sign in the user after signup to get real Firebase tokens
	loginReq := requests.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	// Get Firebase Web API Key from environment
	firebaseAPIKey := os.Getenv("FIREBASE_WEB_API_KEY")
	if firebaseAPIKey == "" {
		// Return success with user info but without tokens
		utils.RespondWithSuccess(c, gin.H{
			"message": "User created successfully",
			"user": gin.H{
				"id":          userID,
				"uid":         userRecord.UID,
				"email":       userRecord.Email,
				"name":        userRecord.DisplayName,
				"is_verified": userData["is_verified"],
				"is_active":   userData["is_active"],
				"created_at":  userData["created_at"],
			},
			"note": "Please login to get authentication tokens",
		})
		return
	}

	// Step 5: Sign in the newly created user to get tokens
	signInReq := FirebaseSignInRequest{
		Email:             loginReq.Email,
		Password:          loginReq.Password,
		ReturnSecureToken: true,
	}

	jsonData, err := json.Marshal(signInReq)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to prepare sign-in request", err)
		return
	}

	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", firebaseAPIKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		// Return success for user creation even if auto-login fails
		utils.RespondWithSuccess(c, gin.H{
			"message": "User created successfully. Please login to get authentication tokens",
			"user": gin.H{
				"id":          userID,
				"uid":         userRecord.UID,
				"email":       userRecord.Email,
				"name":        userRecord.DisplayName,
				"is_verified": userData["is_verified"],
				"is_active":   userData["is_active"],
				"created_at":  userData["created_at"],
			},
		})
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.RespondWithSuccess(c, gin.H{
			"message": "User created successfully. Please login to get authentication tokens",
			"user": gin.H{
				"id":          userID,
				"uid":         userRecord.UID,
				"email":       userRecord.Email,
				"name":        userRecord.DisplayName,
				"is_verified": userData["is_verified"],
				"is_active":   userData["is_active"],
				"created_at":  userData["created_at"],
			},
		})
		return
	}

	if resp.StatusCode == http.StatusOK {
		var signInResp FirebaseSignInResponse
		if err := json.Unmarshal(body, &signInResp); err == nil {
			// Return successful signup with immediate login tokens
			utils.RespondWithSuccess(c, gin.H{
				"message": "User created and signed in successfully",
				"user": gin.H{
					"id":          userID,
					"uid":         userRecord.UID,
					"email":       userRecord.Email,
					"name":        userRecord.DisplayName,
					"is_verified": userData["is_verified"],
					"is_active":   userData["is_active"],
					"created_at":  userData["created_at"],
				},
				"tokens": gin.H{
					"id_token":      signInResp.IdToken,
					"refresh_token": signInResp.RefreshToken,
					"expires_in":    signInResp.ExpiresIn,
				},
			})
			return
		}
	}

	// Fallback: return user info without tokens
	utils.RespondWithSuccess(c, gin.H{
		"message": "User created successfully. Please login to get authentication tokens",
		"user": gin.H{
			"id":          userID,
			"uid":         userRecord.UID,
			"email":       userRecord.Email,
			"name":        userRecord.DisplayName,
			"is_verified": userData["is_verified"],
			"is_active":   userData["is_active"],
			"created_at":  userData["created_at"],
		},
	})
}

// Login authenticates a user using Firebase Auth REST API and returns proper Firebase tokens
// @Summary Authenticate user
// @Description Login with email and password to get Firebase authentication tokens
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body requests.LoginRequest true "User login credentials"
// @Success 200 {object} map[string]interface{} "Login successful"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Invalid credentials"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/login [post]
func (h *FirebaseAuthHandler) Login(c *gin.Context) {
	var req requests.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := context.Background()

	// Get Firebase Web API Key from environment
	firebaseAPIKey := os.Getenv("FIREBASE_WEB_API_KEY")
	if firebaseAPIKey == "" {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase Web API Key not configured", nil)
		return
	}

	// Prepare Firebase Auth REST API request
	signInReq := FirebaseSignInRequest{
		Email:             req.Email,
		Password:          req.Password,
		ReturnSecureToken: true,
	}

	jsonData, err := json.Marshal(signInReq)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to prepare authentication request", err)
		return
	}

	// Call Firebase Auth REST API
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s", firebaseAPIKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Authentication service unavailable", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to read authentication response", err)
		return
	}

	// Handle authentication failure
	if resp.StatusCode != http.StatusOK {
		var firebaseError FirebaseErrorResponse
		if err := json.Unmarshal(body, &firebaseError); err == nil {
			// Map Firebase error messages to user-friendly messages
			message := "Invalid email or password"
			switch firebaseError.Error.Message {
			case "EMAIL_NOT_FOUND":
				message = "User not found"
			case "INVALID_PASSWORD":
				message = "Invalid password"
			case "USER_DISABLED":
				message = "Account has been disabled"
			case "TOO_MANY_ATTEMPTS_TRY_LATER":
				message = "Too many failed attempts. Please try again later"
			}
			utils.RespondWithError(c, http.StatusUnauthorized, message, nil)
		} else {
			utils.RespondWithError(c, http.StatusUnauthorized, "Authentication failed", nil)
		}
		return
	}

	// Parse successful authentication response
	var signInResp FirebaseSignInResponse
	if err := json.Unmarshal(body, &signInResp); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to parse authentication response", err)
		return
	}

	// Optionally verify the ID token with Firebase Admin SDK for additional security
	client := config.GetFirebaseAuth()
	var localUser map[string]interface{}

	if client != nil {
		token, err := client.VerifyIDToken(ctx, signInResp.IdToken)
		if err != nil {
			utils.RespondWithError(c, http.StatusUnauthorized, "Token verification failed", err)
			return
		}

		// Get additional user information from Firebase Admin SDK
		userRecord, err := client.GetUser(ctx, token.UID)
		if err != nil {
			utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get user info", err)
			return
		}

		// Try to get user from local database by Firebase UID
		userResponse, _ := h.userService.GetUserByFirebaseUID(ctx, userRecord.UID)
		if userResponse != nil && userResponse.Success {
			// User exists in local database
			localUser = userResponse.Data.(map[string]interface{})
		} else {
			// Try to get user by email
			userByEmailResponse, _ := h.userService.GetUserByEmail(ctx, userRecord.Email)
			if userByEmailResponse != nil && userByEmailResponse.Success {
				// User exists but may not have Firebase UID linked
				localUser = userByEmailResponse.Data.(map[string]interface{})
				userID := localUser["id"].(string)

				// Link Firebase UID to existing user
				err = h.userService.UpdateUserFirebaseUID(ctx, userID, userRecord.UID)
				if err != nil {
					// Log error but continue with login
					fmt.Printf("Failed to update Firebase UID for user %s: %v\n", userID, err)
				}
			}
		}

		// Prepare user data for response
		userData := gin.H{
			"uid":            userRecord.UID,
			"email":          userRecord.Email,
			"name":           userRecord.DisplayName,
			"email_verified": userRecord.EmailVerified,
			"created_at":     userRecord.UserMetadata.CreationTimestamp,
			"last_login":     userRecord.UserMetadata.LastLogInTimestamp,
		}

		// Add local database fields if available
		if localUser != nil {
			userData["id"] = localUser["id"]
			userData["is_verified"] = localUser["is_verified"]
			userData["is_active"] = localUser["is_active"]
			userData["phone_number"] = localUser["phone_number"]
		}

		// Return successful authentication response with real Firebase tokens
		utils.RespondWithSuccess(c, gin.H{
			"message": "Login successful",
			"user":    userData,
			"tokens": gin.H{
				"id_token":      signInResp.IdToken,
				"refresh_token": signInResp.RefreshToken,
				"expires_in":    signInResp.ExpiresIn,
			},
		})
	} else {
		// Fallback response if Admin SDK is not available
		// Try to get local user data
		userResponse, _ := h.userService.GetUserByEmail(ctx, signInResp.Email)
		userData := gin.H{
			"uid":   signInResp.LocalId,
			"email": signInResp.Email,
			"name":  signInResp.DisplayName,
		}

		if userResponse != nil && userResponse.Success {
			localUser := userResponse.Data.(map[string]interface{})
			userData["id"] = localUser["id"]
			userData["is_verified"] = localUser["is_verified"]
			userData["is_active"] = localUser["is_active"]
			userData["phone_number"] = localUser["phone_number"]
		}

		utils.RespondWithSuccess(c, gin.H{
			"message": "Login successful",
			"user":    userData,
			"tokens": gin.H{
				"id_token":      signInResp.IdToken,
				"refresh_token": signInResp.RefreshToken,
				"expires_in":    signInResp.ExpiresIn,
			},
		})
	}
}

// VerifyToken verifies a Firebase ID token
// @Summary Verify Firebase ID token
// @Description Verify the validity of a Firebase ID token and return user information
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body requests.VerifyTokenRequest true "Firebase ID token to verify"
// @Success 200 {object} map[string]interface{} "Token verified successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Invalid or expired token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/verify-token [post]
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
// @Summary Get user profile
// @Description Get the current authenticated user's profile information
// @Tags Authentication
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "User profile retrieved successfully"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/profile [get]
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

// Logout revokes the user's refresh tokens (server-side) and returns success.
// Client should delete stored tokens locally. This does NOT delete the account.
func (h *FirebaseAuthHandler) Logout(c *gin.Context) {
	uid, exists := c.Get("firebase_uid")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	ctx := context.Background()
	client := config.GetFirebaseAuth()
	if client != nil {
		if err := client.RevokeRefreshTokens(ctx, uid.(string)); err != nil {
			// If revocation fails, still instruct client to clear tokens; but report the server-side issue
			utils.RespondWithError(c, http.StatusInternalServerError, "Failed to revoke refresh tokens", err)
			return
		}
	}

	utils.RespondWithSuccess(c, gin.H{
		"message": "Logged out successfully",
	})
}

// RefreshToken refreshes a Firebase refresh token to get a new ID token
// @Summary Refresh authentication token
// @Description Refresh a Firebase refresh token to get a new ID token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body object{refresh_token string} true "Refresh token"
// @Success 200 {object} map[string]interface{} "Token refreshed successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Invalid or expired refresh token"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/refresh-token [post]
func (h *FirebaseAuthHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// Get Firebase Web API Key from environment
	firebaseAPIKey := os.Getenv("FIREBASE_WEB_API_KEY")
	if firebaseAPIKey == "" {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase Web API Key not configured", nil)
		return
	}

	// Prepare refresh token request
	refreshReq := map[string]string{
		"grant_type":    "refresh_token",
		"refresh_token": req.RefreshToken,
	}

	jsonData, err := json.Marshal(refreshReq)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to prepare refresh request", err)
		return
	}

	// Call Firebase Auth REST API for token refresh
	url := fmt.Sprintf("https://securetoken.googleapis.com/v1/token?key=%s", firebaseAPIKey)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Token refresh service unavailable", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to read refresh response", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		utils.RespondWithError(c, http.StatusUnauthorized, "Invalid or expired refresh token", nil)
		return
	}

	var refreshResp map[string]interface{}
	if err := json.Unmarshal(body, &refreshResp); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to parse refresh response", err)
		return
	}

	utils.RespondWithSuccess(c, gin.H{
		"message": "Token refreshed successfully",
		"tokens": gin.H{
			"id_token":      refreshResp["id_token"],
			"refresh_token": refreshResp["refresh_token"],
			"expires_in":    refreshResp["expires_in"],
		},
	})
}

// Google OAuth structures
type GoogleOAuthRequest struct {
	IDToken           string `json:"id_token" binding:"required"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type GoogleOAuthResponse struct {
	Kind         string `json:"kind"`
	LocalId      string `json:"localId"`
	Email        string `json:"email"`
	DisplayName  string `json:"displayName"`
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	IsNewUser    bool   `json:"isNewUser"`
}

// GoogleSignIn handles Google OAuth sign-in
func (h *FirebaseAuthHandler) GoogleSignIn(c *gin.Context) {
	var req GoogleOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	// Get Firebase Web API Key from environment
	firebaseAPIKey := os.Getenv("FIREBASE_WEB_API_KEY")
	if firebaseAPIKey == "" {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase Web API Key not configured", nil)
		return
	}

	// Call Firebase Auth REST API for Google OAuth
	url := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithIdp?key=%s", firebaseAPIKey)

	// Create the IdP (Identity Provider) request
	idpRequest := map[string]interface{}{
		"postBody":            fmt.Sprintf("id_token=%s&providerId=google.com", req.IDToken),
		"requestUri":          "http://localhost:8080",
		"returnIdpCredential": true,
		"returnSecureToken":   true,
	}

	idpJsonData, err := json.Marshal(idpRequest)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to prepare IdP request", err)
		return
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(idpJsonData))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to authenticate with Google", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to read Google authentication response", err)
		return
	}

	// Handle Firebase API errors
	if resp.StatusCode != http.StatusOK {
		var firebaseError FirebaseErrorResponse
		if err := json.Unmarshal(body, &firebaseError); err == nil {
			errorMessage := "Google authentication failed"
			if len(firebaseError.Error.Errors) > 0 {
				switch firebaseError.Error.Errors[0].Message {
				case "INVALID_IDP_RESPONSE":
					errorMessage = "Invalid Google token"
				case "FEDERATED_USER_ID_ALREADY_LINKED":
					errorMessage = "This Google account is already linked to another account"
				default:
					errorMessage = firebaseError.Error.Errors[0].Message
				}
			}
			utils.RespondWithError(c, http.StatusUnauthorized, errorMessage, nil)
			return
		}
		utils.RespondWithError(c, http.StatusInternalServerError, "Google authentication failed", nil)
		return
	}

	// Parse successful response
	var googleResponse GoogleOAuthResponse
	if err := json.Unmarshal(body, &googleResponse); err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to parse Google authentication response", err)
		return
	}

	// Return successful authentication response
	c.JSON(http.StatusOK, gin.H{
		"message":       "Google authentication successful",
		"id_token":      googleResponse.IdToken,
		"refresh_token": googleResponse.RefreshToken,
		"expires_in":    googleResponse.ExpiresIn,
		"user": gin.H{
			"uid":          googleResponse.LocalId,
			"email":        googleResponse.Email,
			"display_name": googleResponse.DisplayName,
		},
		"is_new_user": googleResponse.IsNewUser,
	})
}

// GetGoogleAuthURL returns the Google OAuth URL for client-side authentication
func (h *FirebaseAuthHandler) GetGoogleAuthURL(c *gin.Context) {
	// This endpoint provides information for client-side Google OAuth
	c.JSON(http.StatusOK, gin.H{
		"message": "For Google Sign-In, use Firebase SDK on the client side",
		"instructions": gin.H{
			"step1": "Use Google Sign-In SDK on your frontend to get Google ID token",
			"step2": "Send the Google ID token to POST /api/v1/auth/google endpoint",
			"step3": "Receive Firebase ID token and refresh token in response",
		},
		"google_oauth_setup": gin.H{
			"client_setup": "Configure Google OAuth in Firebase Console under Authentication > Sign-in method",
			"frontend_sdk": "Use Firebase SDK or Google Sign-In SDK in your frontend",
		},
	})
}
