package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"firebase.google.com/go/v4/auth"
	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/config"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// FirebaseAuthHandler handles Firebase authentication operations
type FirebaseAuthHandler struct {
	userService     *services.UserService
	profileService  *services.ProfileService
	whatsappService *services.WhatsAppService
	otpService      *services.OTPService
}

// NewFirebaseAuthHandler creates a new Firebase auth handler
func NewFirebaseAuthHandler(
	userService *services.UserService,
	profileService *services.ProfileService,
	whatsappService *services.WhatsAppService,
	otpService *services.OTPService,
) *FirebaseAuthHandler {
	return &FirebaseAuthHandler{
		userService:     userService,
		profileService:  profileService,
		whatsappService: whatsappService,
		otpService:      otpService,
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
		h.userService.SoftDeleteUser(ctx, userID, "system")
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

	// Step 3.5: Automatically create profile for the new user (dedicated profile section)
	// This happens automatically during signup - no need for separate endpoint
	profile, profileErr := h.profileService.CreateAutoProfile(ctx, userID, req.Email)
	if profileErr != nil {
		// Log error but don't fail signup - profile can be created later
		// Only fail if it's not a "already exists" error
		if !strings.Contains(profileErr.Error(), "profile already exists") {
			fmt.Printf("⚠️ Warning: Failed to create profile during signup: %v\n", profileErr)
			// Continue with signup - profile creation is not critical
		} else {
			fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
		}
	} else {
		fmt.Printf("✅ Profile successfully created in NeonDB during signup: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
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
		// Use VerifyIDTokenAndCheckRevoked with clock skew tolerance
		// Skip strict verification for login since we just got the token from Firebase REST API
		// The token is fresh and valid - clock skew can cause false negatives
		token, err := client.VerifyIDToken(ctx, signInResp.IdToken)
		if err != nil {
			// Check if it's a clock skew error - if so, proceed anyway since token is fresh from Firebase
			errStr := err.Error()
			if strings.Contains(errStr, "future timestamp") || strings.Contains(errStr, "issued at") {
				// Clock skew detected - token is valid but local clock is behind
				// Since we just got this token from Firebase REST API, it's definitely valid
				// Log warning and continue with user lookup by email
				fmt.Printf("⚠️ Clock skew detected during login token verification: %v\n", err)
				fmt.Printf("⚠️ Proceeding with login since token was just issued by Firebase\n")

				// Get user by email instead of relying on token
				userByEmailResponse, _ := h.userService.GetUserByEmail(ctx, signInResp.Email)
				if userByEmailResponse != nil && userByEmailResponse.Success {
					localUser = userByEmailResponse.Data.(map[string]interface{})
					userID, _ := localUser["id"].(string)

					// Ensure profile exists
					profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
					if profile == nil || profileErr != nil {
						h.profileService.CreateAutoProfile(ctx, userID, signInResp.Email)
					}
				}

				// Return success with tokens
				userData := gin.H{
					"uid":   signInResp.LocalId,
					"email": signInResp.Email,
					"name":  signInResp.DisplayName,
				}
				if localUser != nil {
					userData["id"] = localUser["id"]
					userData["is_verified"] = localUser["is_verified"]
					userData["is_active"] = localUser["is_active"]
					// Add role for frontend routing
					if role, ok := localUser["role"].(string); ok && role != "" {
						userData["role"] = role
					} else {
						userData["role"] = "user"
					}
				} else {
					userData["role"] = "user"
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
				return
			}
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
			userID, _ := localUser["id"].(string)

			// Check if user has a profile, create if not
			profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
			if profile == nil || profileErr != nil {
				// Profile doesn't exist, create it
				profile, createErr := h.profileService.CreateAutoProfile(ctx, userID, userRecord.Email)
				if createErr != nil {
					if !strings.Contains(createErr.Error(), "profile already exists") {
						fmt.Printf("⚠️ Warning: Failed to create profile during login: %v\n", createErr)
					} else {
						fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
					}
				} else {
					fmt.Printf("✅ Profile successfully created during login: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
				}
			}
		} else {
			// SECURITY (#5 account takeover): only link/resolve a local account by email when the
			// Firebase identity's email is verified. An unverified email/password account must never
			// be able to claim an existing local record by email.
			if !userRecord.EmailVerified {
				utils.RespondWithError(c, http.StatusForbidden, "Email not verified",
					fmt.Errorf("verify your email before signing in with this method"))
				return
			}

			// Try to get user by email
			userByEmailResponse, _ := h.userService.GetUserByEmail(ctx, userRecord.Email)
			if userByEmailResponse != nil && userByEmailResponse.Success {
				candidate := userByEmailResponse.Data.(map[string]interface{})
				userID, _ := candidate["id"].(string)

				// Link Firebase UID to the existing user. UpdateUserFirebaseUID refuses to overwrite
				// a DIFFERENT firebase_uid already on the record, so a second identity cannot hijack
				// an account that is already linked to someone else.
				if err = h.userService.UpdateUserFirebaseUID(ctx, userID, userRecord.UID); err != nil {
					utils.RespondWithError(c, http.StatusConflict, "Account is linked to a different sign-in identity", err)
					return
				}
				localUser = candidate

				// Check if user has a profile, create if not
				profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
				if profile == nil || profileErr != nil {
					// Profile doesn't exist, create it
					profile, createErr := h.profileService.CreateAutoProfile(ctx, userID, userRecord.Email)
					if createErr != nil {
						if !strings.Contains(createErr.Error(), "profile already exists") {
							fmt.Printf("⚠️ Warning: Failed to create profile during login: %v\n", createErr)
						} else {
							fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
						}
					} else {
						fmt.Printf("✅ Profile successfully created during login: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
					}
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
			// Add role for frontend routing
			if role, ok := localUser["role"].(string); ok && role != "" {
				userData["role"] = role
			} else {
				userData["role"] = "user" // Default role
			}
		} else {
			userData["role"] = "user" // Default role if no local user
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
			"role":  "user", // Default role
		}

		if userResponse != nil && userResponse.Success {
			localUser := userResponse.Data.(map[string]interface{})
			userData["id"] = localUser["id"]
			userData["is_verified"] = localUser["is_verified"]
			userData["is_active"] = localUser["is_active"]
			userData["phone_number"] = localUser["phone_number"]
			// Add role for frontend routing
			if role, ok := localUser["role"].(string); ok && role != "" {
				userData["role"] = role
			}

			// Check if user has a profile, create if not (fallback path)
			userID, _ := localUser["id"].(string)
			profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
			if profile == nil || profileErr != nil {
				// Profile doesn't exist, create it
				profile, createErr := h.profileService.CreateAutoProfile(ctx, userID, signInResp.Email)
				if createErr != nil {
					if !strings.Contains(createErr.Error(), "profile already exists") {
						fmt.Printf("⚠️ Warning: Failed to create profile during login (fallback): %v\n", createErr)
					} else {
						fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
					}
				} else {
					fmt.Printf("✅ Profile successfully created during login (fallback): ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
				}
			}
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

// GoogleSignIn handles Google OAuth sign-in via Firebase ID token
func (h *FirebaseAuthHandler) GoogleSignIn(c *gin.Context) {
	var req GoogleOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := context.Background()

	// First, try to verify the token as a Firebase ID token
	client := config.GetFirebaseAuth()
	if client != nil {
		// Verify the Firebase ID token
		token, err := client.VerifyIDToken(ctx, req.IDToken)
		if err == nil {
			// This is a Firebase ID token from Google sign-in
			// Get the user record from Firebase
			userRecord, err := client.GetUser(ctx, token.UID)
			if err != nil {
				utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get user info from Firebase", err)
				return
			}

			// Handle the Firebase Google user
			h.handleFirebaseGoogleUser(c, userRecord, req.IDToken)
			return
		}

		// SECURITY (#19): removed the clock-skew fallback that decoded the ID token
		// payload and trusted user_id/sub WITHOUT verifying the signature — that was
		// a forged-token auth bypass. A token is accepted only if VerifyIDToken
		// succeeds above; otherwise fall through to the legitimate Google OAuth REST
		// flow below (or its error). Genuine clock skew is fixed by syncing the
		// server clock, never by trusting an unverified payload.
	}

	// Fallback: Try to use it as a Google OAuth token with Firebase REST API
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
				// Provide more specific error for invalid token
				if strings.Contains(firebaseError.Error.Errors[0].Message, "INVALID_IDP_RESPONSE") {
					errorMessage = fmt.Sprintf("Invalid Google token format. If signing in through Firebase, the token is already verified. Error: %s", firebaseError.Error.Errors[0].Message)
				} else if firebaseError.Error.Errors[0].Message == "FEDERATED_USER_ID_ALREADY_LINKED" {
					errorMessage = "This Google account is already linked to another account"
				} else {
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

	// ctx already declared at the beginning of the function
	var localUser map[string]interface{}

	// Check if this is a new Google user
	if googleResponse.IsNewUser {
		// Create user in local database
		signupReq := requests.SignupRequest{
			Email:    googleResponse.Email,
			Name:     googleResponse.DisplayName,
			Password: "", // Google users don't have passwords
		}

		// Create user in local database (will handle Google users without passwords)
		userResponse, err := h.userService.CreateGoogleUser(ctx, &signupReq, googleResponse.LocalId)
		if err != nil {
			// If user already exists, try to get them
			existingUserResponse, _ := h.userService.GetUserByEmail(ctx, googleResponse.Email)
			if existingUserResponse != nil && existingUserResponse.Success {
				localUser = existingUserResponse.Data.(map[string]interface{})
				userID, _ := localUser["id"].(string)
				// Update Firebase UID if not set
				if localUser["firebase_uid"] == "" || localUser["firebase_uid"] == nil {
					h.userService.UpdateUserFirebaseUID(ctx, userID, googleResponse.LocalId)
				}
				// Check if profile exists, create if not
				profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
				if profile == nil || profileErr != nil {
					profile, createErr := h.profileService.CreateAutoProfile(ctx, userID, googleResponse.Email)
					if createErr != nil {
						if !strings.Contains(createErr.Error(), "profile already exists") {
							fmt.Printf("⚠️ Warning: Failed to create profile for existing Google user: %v\n", createErr)
						} else {
							fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
						}
					} else {
						fmt.Printf("✅ Profile successfully created for existing Google user: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
					}
				}
			} else {
				utils.RespondWithError(c, http.StatusInternalServerError, "Failed to create user in database", err)
				return
			}
		} else {
			if userResponse.Success {
				localUser = userResponse.Data.(map[string]interface{})
				userID, _ := localUser["id"].(string)

				// Automatically create profile for the new Google user
				profile, profileErr := h.profileService.CreateAutoProfile(ctx, userID, googleResponse.Email)
				if profileErr != nil {
					if !strings.Contains(profileErr.Error(), "profile already exists") {
						fmt.Printf("⚠️ Warning: Failed to create profile during Google sign-in: %v\n", profileErr)
					}
				} else {
					fmt.Printf("✅ Profile successfully created in NeonDB during Google sign-in: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
				}
			}
		}
	} else {
		// Existing Google user - try to get from local database
		userResponse, _ := h.userService.GetUserByFirebaseUID(ctx, googleResponse.LocalId)
		if userResponse != nil && userResponse.Success {
			localUser = userResponse.Data.(map[string]interface{})
			userID, _ := localUser["id"].(string)

			// Check if profile exists, create if not
			profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
			if profile == nil || profileErr != nil {
				profile, createErr := h.profileService.CreateAutoProfile(ctx, userID, googleResponse.Email)
				if createErr != nil {
					if !strings.Contains(createErr.Error(), "profile already exists") {
						fmt.Printf("⚠️ Warning: Failed to create profile for existing Google user: %v\n", createErr)
					} else {
						fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
					}
				} else {
					fmt.Printf("✅ Profile successfully created for existing Google user: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
				}
			}
		} else {
			// Try by email
			userByEmailResponse, _ := h.userService.GetUserByEmail(ctx, googleResponse.Email)
			if userByEmailResponse != nil && userByEmailResponse.Success {
				localUser = userByEmailResponse.Data.(map[string]interface{})
				// Update Firebase UID
				userID, _ := localUser["id"].(string)
				h.userService.UpdateUserFirebaseUID(ctx, userID, googleResponse.LocalId)

				// Check if profile exists, create if not
				profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
				if profile == nil || profileErr != nil {
					profile, createErr := h.profileService.CreateAutoProfile(ctx, userID, googleResponse.Email)
					if createErr != nil {
						if !strings.Contains(createErr.Error(), "profile already exists") {
							fmt.Printf("⚠️ Warning: Failed to create profile for existing Google user: %v\n", createErr)
						} else {
							fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
						}
					} else {
						fmt.Printf("✅ Profile successfully created for existing Google user: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
					}
				}
			}
		}
	}

	// Prepare response data
	responseData := gin.H{
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
	}

	// Add local database fields if available
	if localUser != nil {
		responseData["user"].(gin.H)["id"] = localUser["id"]
		responseData["user"].(gin.H)["is_verified"] = localUser["is_verified"]
		responseData["user"].(gin.H)["is_active"] = localUser["is_active"]
		responseData["user"].(gin.H)["created_at"] = localUser["created_at"]
	}

	utils.RespondWithSuccess(c, responseData)
}

// handleFirebaseGoogleUser handles a user who signed in with Google through Firebase
func (h *FirebaseAuthHandler) handleFirebaseGoogleUser(c *gin.Context, userRecord *auth.UserRecord, idToken string) {
	ctx := context.Background()
	var localUser map[string]interface{}

	// Check if user exists in local database by Firebase UID
	userResponse, _ := h.userService.GetUserByFirebaseUID(ctx, userRecord.UID)
	if userResponse != nil && userResponse.Success {
		// User exists in local database
		localUser = userResponse.Data.(map[string]interface{})
		userID, _ := localUser["id"].(string)

		// Check if user has a profile, create if not
		profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
		if profile == nil || profileErr != nil {
			profile, createErr := h.profileService.CreateAutoProfile(ctx, userID, userRecord.Email)
			if createErr != nil {
				if !strings.Contains(createErr.Error(), "profile already exists") {
					fmt.Printf("⚠️ Warning: Failed to create profile for Google user: %v\n", createErr)
				} else {
					fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
				}
			} else {
				fmt.Printf("✅ Profile successfully created for Google user: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
			}
		}
	} else {
		// User doesn't exist, check by email
		userByEmailResponse, _ := h.userService.GetUserByEmail(ctx, userRecord.Email)
		if userByEmailResponse != nil && userByEmailResponse.Success {
			// User exists with email but not linked to Firebase UID
			localUser = userByEmailResponse.Data.(map[string]interface{})
			userID, _ := localUser["id"].(string)

			// Update Firebase UID
			h.userService.UpdateUserFirebaseUID(ctx, userID, userRecord.UID)

			// Check/create profile
			profile, profileErr := h.profileService.GetProfileByUserID(ctx, userID)
			if profile == nil || profileErr != nil {
				profile, createErr := h.profileService.CreateAutoProfile(ctx, userID, userRecord.Email)
				if createErr != nil {
					if !strings.Contains(createErr.Error(), "profile already exists") {
						fmt.Printf("⚠️ Warning: Failed to create profile for existing Google user: %v\n", createErr)
					} else {
						fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
					}
				} else {
					fmt.Printf("✅ Profile successfully created for existing Google user: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
				}
			}
		} else {
			// Create new user in local database
			signupReq := requests.SignupRequest{
				Email:    userRecord.Email,
				Name:     userRecord.DisplayName,
				Password: "", // Google users don't have passwords
			}

			// Create user in local database
			userResponse, err := h.userService.CreateGoogleUser(ctx, &signupReq, userRecord.UID)
			if err != nil {
				utils.RespondWithError(c, http.StatusInternalServerError, "Failed to create user in database", err)
				return
			}

			if userResponse.Success {
				localUser = userResponse.Data.(map[string]interface{})
				userID, _ := localUser["id"].(string)

				// Automatically create profile for the new Google user
				profile, profileErr := h.profileService.CreateAutoProfile(ctx, userID, userRecord.Email)
				if profileErr != nil {
					if !strings.Contains(profileErr.Error(), "profile already exists") {
						fmt.Printf("⚠️ Warning: Failed to create profile during Google sign-in: %v\n", profileErr)
					}
				} else {
					fmt.Printf("✅ Profile successfully created during Google sign-in: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
				}
			}
		}
	}

	// Prepare response data
	responseData := gin.H{
		"message": "Google authentication successful",
		"tokens": gin.H{
			"id_token": idToken,
		},
		"user": gin.H{
			"uid":            userRecord.UID,
			"email":          userRecord.Email,
			"display_name":   userRecord.DisplayName,
			"email_verified": userRecord.EmailVerified,
		},
	}

	// Add local database fields if available
	if localUser != nil {
		responseData["user"].(gin.H)["id"] = localUser["id"]
		responseData["user"].(gin.H)["is_verified"] = localUser["is_verified"]
		responseData["user"].(gin.H)["is_active"] = localUser["is_active"]
		responseData["user"].(gin.H)["created_at"] = localUser["created_at"]
	}

	utils.RespondWithSuccess(c, responseData)
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

// Phone Authentication structures
type FirebasePhoneRequest struct {
	PhoneNumber    string `json:"phoneNumber"`
	RecaptchaToken string `json:"recaptchaToken"`
}

type FirebasePhoneResponse struct {
	SessionInfo string `json:"sessionInfo"`
}

type FirebasePhoneVerificationRequest struct {
	SessionInfo string `json:"sessionInfo"`
	Code        string `json:"code"`
}

type FirebasePhoneVerificationResponse struct {
	Kind         string `json:"kind"`
	LocalId      string `json:"localId"`
	PhoneNumber  string `json:"phoneNumber"`
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	IsNewUser    bool   `json:"isNewUser"`
}

// SendPhoneVerificationCode sends WhatsApp verification code to phone number
// @Summary Send WhatsApp verification code
// @Description Send verification code to the provided phone number via WhatsApp
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body requests.SendPhoneCodeRequest true "Phone number for verification"
// @Success 200 {object} map[string]interface{} "Verification code sent successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request format or cooldown active"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/phone/send-code [post]
func (h *FirebaseAuthHandler) SendPhoneVerificationCode(c *gin.Context) {
	var req requests.SendPhoneCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	normalizedPhone := normalizePhoneE164(req.PhoneNumber)
	if normalizedPhone == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid phone number format. Use E.164 format (e.g. +919876543210)", nil)
		return
	}
	req.PhoneNumber = normalizedPhone

	ctx := c.Request.Context()

	// Step 1: Generate & save OTP in database
	sessionInfo, otpCode, err := h.otpService.GenerateAndSaveOTP(ctx, req.PhoneNumber)
	if err != nil {
		if errors.Is(err, services.ErrOTPCooldown) {
			utils.RespondWithError(c, http.StatusTooManyRequests, "Please wait 60 seconds before requesting another code", nil)
			return
		}
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to generate verification code", err)
		return
	}

	// Step 2: Send OTP via WhatsApp — roll back the DB record if delivery fails
	if err := h.whatsappService.SendOTPMessage(ctx, req.PhoneNumber, otpCode); err != nil {
		h.otpService.DeleteOTP(ctx, sessionInfo)
		fmt.Printf("⚠️ Failed to send WhatsApp OTP to %s: %v\n", req.PhoneNumber, err)
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to send WhatsApp message. Please check your phone number and try again.", err)
		return
	}

	utils.RespondWithSuccess(c, gin.H{
		"message":      "Verification code sent via WhatsApp successfully",
		"session_info": sessionInfo,
	})
}

// VerifyPhoneCode verifies OTP code and signs in/up the user
// @Summary Verify WhatsApp OTP code and authenticate
// @Description Verify OTP code and sign in or sign up the user
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body requests.VerifyPhoneCodeRequest true "Phone verification code"
// @Success 200 {object} map[string]interface{} "Authentication successful"
// @Failure 400 {object} map[string]interface{} "Invalid request format"
// @Failure 401 {object} map[string]interface{} "Invalid verification code"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/phone/verify-code [post]
func (h *FirebaseAuthHandler) VerifyPhoneCode(c *gin.Context) {
	var req requests.VerifyPhoneCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := c.Request.Context()

	// Normalize phone to E.164 so OTP lookup and Firebase both use the same format
	normalizedPhone := normalizePhoneE164(req.PhoneNumber)
	if normalizedPhone == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid phone number format. Use E.164 format (e.g. +919876543210)", nil)
		return
	}
	req.PhoneNumber = normalizedPhone

	// Step 1: Verify OTP with OTPService
	if err := h.otpService.VerifyOTP(ctx, req.PhoneNumber, req.Code, req.SessionInfo); err != nil {
		switch {
		case errors.Is(err, services.ErrOTPNotFound):
			utils.RespondWithError(c, http.StatusBadRequest, "Verification session not found or expired. Please request a new code", nil)
		case errors.Is(err, services.ErrOTPExpired):
			utils.RespondWithError(c, http.StatusBadRequest, "Verification code has expired. Please request a new code", nil)
		case errors.Is(err, services.ErrOTPAlreadyUsed):
			utils.RespondWithError(c, http.StatusBadRequest, "Verification code has already been used. Please request a new code", nil)
		case errors.Is(err, services.ErrOTPMaxRetries):
			utils.RespondWithError(c, http.StatusTooManyRequests, "Too many failed attempts. Please request a new code", nil)
		case errors.Is(err, services.ErrOTPInvalid):
			utils.RespondWithError(c, http.StatusUnauthorized, "Invalid verification code. Please check and try again", nil)
		default:
			utils.RespondWithError(c, http.StatusInternalServerError, "Verification failed", err)
		}
		return
	}

	// Step 2: Handle user creation/retrieval in NeonDB (local database)
	var localUser map[string]interface{}
	isNewUser := false

	userResponse, _ := h.userService.GetUserByPhoneNumber(ctx, req.PhoneNumber)
	if userResponse != nil && userResponse.Success {
		localUser = userResponse.Data.(map[string]interface{})
		userID, _ := localUser["id"].(string)

		// Check if profile exists, create if missing
		profile, profileErr := h.profileService.CreateAutoProfile(ctx, userID, "")
		if profileErr != nil {
			if !strings.Contains(profileErr.Error(), "profile already exists") {
				fmt.Printf("⚠️ Warning: Failed to create profile during phone verification for existing user: %v\n", profileErr)
			}
		} else {
			fmt.Printf("✅ Profile created for existing user: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
		}
	} else {
		// New user
		isNewUser = true
		userName := req.Name
		if userName == "" {
			userName = "User"
		}
		phoneSignupReq := requests.PhoneSignupRequest{
			PhoneNumber: req.PhoneNumber,
			Name:        userName,
		}

		userResponse, err := h.userService.CreatePhoneUser(ctx, &phoneSignupReq)
		if err != nil {
			utils.RespondWithError(c, http.StatusInternalServerError, "Failed to create user in database", err)
			return
		}

		if userResponse.Success {
			localUser = userResponse.Data.(map[string]interface{})
			userID, _ := localUser["id"].(string)

			profile, profileErr := h.profileService.CreateAutoProfile(ctx, userID, "")
			if profileErr != nil {
				if !strings.Contains(profileErr.Error(), "profile already exists") {
					fmt.Printf("⚠️ Warning: Failed to create profile during phone verification: %v\n", profileErr)
				}
			} else {
				fmt.Printf("✅ Profile created for new user: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
			}
		}
	}

	// Step 3: Get or create Firebase Auth user for phone number & mint custom token
	firebaseClient := config.GetFirebaseAuth()
	if firebaseClient == nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase not initialized", nil)
		return
	}

	var firebaseUID string
	fbUser, fbErr := firebaseClient.GetUserByPhoneNumber(ctx, req.PhoneNumber)
	if fbErr != nil {
		// Firebase user does not exist yet — create one with this phone number
		displayName := req.Name
		if displayName == "" {
			displayName = "User"
		}
		params := (&auth.UserToCreate{}).
			PhoneNumber(req.PhoneNumber).
			DisplayName(displayName)
		newFbUser, createErr := firebaseClient.CreateUser(ctx, params)
		if createErr != nil || newFbUser == nil {
			utils.RespondWithError(c, http.StatusInternalServerError, "Failed to create Firebase user", createErr)
			return
		}
		firebaseUID = newFbUser.UID
	} else if fbUser != nil {
		firebaseUID = fbUser.UID
	}

	if firebaseUID == "" {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to resolve Firebase user", nil)
		return
	}

	// Update local user's firebase_uid if not yet linked
	if localUser != nil {
		userID, _ := localUser["id"].(string)
		if localUser["firebase_uid"] == "" || localUser["firebase_uid"] == nil {
			if err := h.userService.UpdateUserFirebaseUID(ctx, userID, firebaseUID); err != nil {
				fmt.Printf("⚠️ Warning: Failed to update Firebase UID for user %s: %v\n", userID, err)
			}
			localUser["firebase_uid"] = firebaseUID
		}
	}

	// Mint Firebase custom token
	customToken, ctErr := firebaseClient.CustomToken(ctx, firebaseUID)
	if ctErr != nil || customToken == "" {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to mint authentication token", ctErr)
		return
	}

	// Exchange custom token for real id_token & refresh_token via Google Identity Toolkit
	firebaseAPIKey := os.Getenv("FIREBASE_WEB_API_KEY")
	if firebaseAPIKey == "" {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase Web API Key not configured", nil)
		return
	}

	exchangeReq := map[string]interface{}{
		"token":             customToken,
		"returnSecureToken": true,
	}
	jsonData, _ := json.Marshal(exchangeReq)
	exchangeURL := fmt.Sprintf("https://identitytoolkit.googleapis.com/v1/accounts:signInWithCustomToken?key=%s", firebaseAPIKey)
	exchangeResp, postErr := http.Post(exchangeURL, "application/json", bytes.NewBuffer(jsonData))
	if postErr != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to exchange token with Firebase", postErr)
		return
	}
	defer exchangeResp.Body.Close()
	body, _ := io.ReadAll(exchangeResp.Body)
	if exchangeResp.StatusCode != http.StatusOK {
		utils.RespondWithError(c, http.StatusInternalServerError, "Firebase token exchange failed", fmt.Errorf("status %d: %s", exchangeResp.StatusCode, string(body)))
		return
	}

	var tokenData struct {
		IdToken      string `json:"idToken"`
		RefreshToken string `json:"refreshToken"`
	}
	if err := json.Unmarshal(body, &tokenData); err != nil || tokenData.IdToken == "" {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to parse Firebase token response", err)
		return
	}
	idToken := tokenData.IdToken
	refreshToken := tokenData.RefreshToken

	// Step 4: Construct and return response

	responseData := gin.H{
		"message":       "Phone authentication successful",
		"id_token":      idToken,
		"refresh_token": refreshToken,
		"expires_in":    "3600",
		"user": gin.H{
			"uid":          firebaseUID,
			"phone_number": req.PhoneNumber,
			"is_new_user":  isNewUser,
		},
	}

	if localUser != nil {
		responseData["user"].(gin.H)["id"] = localUser["id"]
		responseData["user"].(gin.H)["name"] = localUser["name"]
		responseData["user"].(gin.H)["is_verified"] = localUser["is_verified"]
		responseData["user"].(gin.H)["is_active"] = localUser["is_active"]
		responseData["user"].(gin.H)["created_at"] = localUser["created_at"]
	}

	utils.RespondWithSuccess(c, responseData)
}

// PhoneSignup creates a new user with phone number (alternative endpoint)
// @Summary Create user with phone number
// @Description Create a new user account using phone number
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body requests.PhoneSignupRequest true "Phone signup information"
// @Success 201 {object} map[string]interface{} "User created successfully"
// @Failure 400 {object} map[string]interface{} "Invalid request or user already exists"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /auth/phone/signup [post]
func (h *FirebaseAuthHandler) PhoneSignup(c *gin.Context) {
	var req requests.PhoneSignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := context.Background()

	// Create user in local database
	userResponse, err := h.userService.CreatePhoneUser(ctx, &req)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to create user in database", err)
		return
	}

	if !userResponse.Success {
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

	// Automatically create profile for the new phone user (dedicated profile section)
	// This happens automatically during signup - no need for separate endpoint
	// Pass empty string for email since phone users don't have email - CreateAutoProfile will handle it
	profile, profileErr := h.profileService.CreateAutoProfile(ctx, userID, "")
	if profileErr != nil {
		// Log error but don't fail signup - profile can be created later
		if !strings.Contains(profileErr.Error(), "profile already exists") {
			fmt.Printf("⚠️ Warning: Failed to create profile during phone signup: %v\n", profileErr)
		} else {
			fmt.Printf("ℹ️ Profile already exists for user: %s\n", userID)
		}
	} else {
		fmt.Printf("✅ Profile successfully created in NeonDB during phone signup: ProfileID=%s, UserID=%s\n", profile.GetID(), userID)
	}

	utils.RespondWithSuccess(c, gin.H{
		"message": "Phone user created successfully. Please verify your phone number to complete registration.",
		"user": gin.H{
			"id":           userData["id"],
			"phone_number": userData["phone_number"],
			"name":         userData["name"],
			"is_verified":  userData["is_verified"],
			"is_active":    userData["is_active"],
			"created_at":   userData["created_at"],
		},
	})
}

// normalizePhoneE164 converts an Indian phone number to E.164 format (+91XXXXXXXXXX).
// Returns empty string if the input cannot be normalized.
func normalizePhoneE164(phone string) string {
	// Strip all non-digit characters except leading +
	var digits strings.Builder
	for i, r := range phone {
		if r == '+' && i == 0 {
			continue // skip leading + — we'll add it back
		}
		if r >= '0' && r <= '9' {
			digits.WriteRune(r)
		}
	}
	d := digits.String()
	switch len(d) {
	case 10:
		// Bare 10-digit Indian number — add +91
		return "+91" + d
	case 12:
		if strings.HasPrefix(d, "91") {
			return "+" + d
		}
	case 13:
		if strings.HasPrefix(d, "091") {
			return "+" + d[1:]
		}
	}
	// Already has country code or unknown format — return as-is with + prefix
	if len(d) >= 11 {
		return "+" + d
	}
	return ""
}

