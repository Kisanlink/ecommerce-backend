package handlers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService *services.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
	}
}

// GetUserByID gets a user by ID
func (h *UserHandler) GetUserByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	ctx := context.Background()
	response, err := h.userService.GetUserByID(ctx, id)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get user", err)
		return
	}

	if !response.Success {
		utils.RespondWithError(c, http.StatusNotFound, response.Message, response.Error)
		return
	}

	utils.RespondWithSuccess(c, response.Data)
}

// GetCurrentUserProfile gets the current authenticated user's profile
func (h *UserHandler) GetCurrentUserProfile(c *gin.Context) {
	// Get user ID from Firebase middleware
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	ctx := context.Background()
	response, err := h.userService.GetUserByFirebaseUID(ctx, firebaseUID.(string))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get user profile", err)
		return
	}

	if !response.Success {
		utils.RespondWithError(c, http.StatusNotFound, response.Message, response.Error)
		return
	}

	utils.RespondWithSuccess(c, response.Data)
}

// UpdateUser updates a user by ID
func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	ctx := context.Background()
	response, err := h.userService.UpdateUser(ctx, id, updateData)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to update user", err)
		return
	}

	if !response.Success {
		utils.RespondWithError(c, http.StatusBadRequest, response.Message, response.Error)
		return
	}

	utils.RespondWithSuccess(c, response.Data)
}

// UpdateCurrentUser updates the current authenticated user's profile
func (h *UserHandler) UpdateCurrentUser(c *gin.Context) {
	// Get user ID from Firebase middleware
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	// First get the user to get their local ID
	ctx := context.Background()
	userResponse, err := h.userService.GetUserByFirebaseUID(ctx, firebaseUID.(string))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get user", err)
		return
	}

	if !userResponse.Success {
		utils.RespondWithError(c, http.StatusNotFound, "User not found", userResponse.Error)
		return
	}

	userData, ok := userResponse.Data.(map[string]interface{})
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid user data", nil)
		return
	}

	userID := userData["id"].(string)

	var updateData map[string]interface{}
	if err := c.ShouldBindJSON(&updateData); err != nil {
		utils.RespondWithError(c, http.StatusBadRequest, "Invalid request payload", err)
		return
	}

	response, err := h.userService.UpdateUser(ctx, userID, updateData)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to update profile", err)
		return
	}

	if !response.Success {
		utils.RespondWithError(c, http.StatusBadRequest, response.Message, response.Error)
		return
	}

	utils.RespondWithSuccess(c, response.Data)
}

// ListUsers lists all users with pagination
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Get pagination parameters
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("page_size", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	ctx := context.Background()
	response, err := h.userService.ListUsers(ctx, page, pageSize)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get users", err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":    response.Message,
		"data":       response.Data,
		"pagination": response.Pagination,
	})
}

// DeleteUser soft deletes a user by ID
func (h *UserHandler) DeleteUser(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		utils.RespondWithError(c, http.StatusBadRequest, "User ID is required", nil)
		return
	}

	// Get current user for audit trail
	deletedBy := "system"
	if firebaseUID, exists := c.Get("firebase_uid"); exists {
		deletedBy = firebaseUID.(string)
	}

	ctx := context.Background()
	response, err := h.userService.SoftDeleteUser(ctx, id, deletedBy)
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to delete user", err)
		return
	}

	if !response.Success {
		utils.RespondWithError(c, http.StatusBadRequest, response.Message, response.Error)
		return
	}

	utils.RespondWithSuccess(c, gin.H{"message": response.Message})
}

// DeleteCurrentUser soft deletes the current authenticated user
func (h *UserHandler) DeleteCurrentUser(c *gin.Context) {
	// Get user ID from Firebase middleware
	firebaseUID, exists := c.Get("firebase_uid")
	if !exists {
		utils.RespondWithError(c, http.StatusUnauthorized, "User not authenticated", nil)
		return
	}

	// First get the user to get their local ID
	ctx := context.Background()
	userResponse, err := h.userService.GetUserByFirebaseUID(ctx, firebaseUID.(string))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to get user", err)
		return
	}

	if !userResponse.Success {
		utils.RespondWithError(c, http.StatusNotFound, "User not found", userResponse.Error)
		return
	}

	userData, ok := userResponse.Data.(map[string]interface{})
	if !ok {
		utils.RespondWithError(c, http.StatusInternalServerError, "Invalid user data", nil)
		return
	}

	userID := userData["id"].(string)

	response, err := h.userService.SoftDeleteUser(ctx, userID, firebaseUID.(string))
	if err != nil {
		utils.RespondWithError(c, http.StatusInternalServerError, "Failed to delete account", err)
		return
	}

	if !response.Success {
		utils.RespondWithError(c, http.StatusBadRequest, response.Message, response.Error)
		return
	}

	utils.RespondWithSuccess(c, gin.H{"message": response.Message})
}
