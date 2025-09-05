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
// @Summary Get user by ID
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{} "User retrieved"
// @Failure 400 {object} map[string]interface{} "User ID is required"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/{id} [get]
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
// @Summary Get current user profile
// @Tags Users
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Profile retrieved"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/me [get]
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
// @Summary Update user
// @Tags Users
// @Accept json
// @Produce json
// @Param id path string true "User ID"
// @Param request body object true "Fields to update"
// @Success 200 {object} map[string]interface{} "User updated"
// @Failure 400 {object} map[string]interface{} "Invalid request or user ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/{id} [put]
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
// @Summary Update current user
// @Tags Users
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body object true "Fields to update"
// @Success 200 {object} map[string]interface{} "Profile updated"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/me [put]
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
// @Summary List users
// @Tags Users
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Items per page" default(20)
// @Success 200 {object} map[string]interface{} "Users listed"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users [get]
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
// @Summary Delete user
// @Tags Users
// @Produce json
// @Param id path string true "User ID"
// @Success 200 {object} map[string]interface{} "User deleted"
// @Failure 400 {object} map[string]interface{} "User ID is required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /users/{id} [delete]
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
// @Summary Delete current user
// @Tags Users
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Account deleted"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "User not found"
// @Router /users/me [delete]
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
