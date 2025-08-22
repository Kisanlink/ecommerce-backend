package handlers

import (
	"context"
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// UserHandler handles user-related HTTP requests
type UserHandler struct {
	userRepo *repositories.UserRepository
}

// NewUserHandler creates a new user handler
func NewUserHandler(userRepo *repositories.UserRepository) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

func (h *UserHandler) Register(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Hash password
	user.Password = utils.HashPassword(user.Password)
	
	// Create new user with proper constructor
	newUser := models.NewUser(user.Username, user.Email, user.Password, user.Role)
	
	if err := h.userRepo.Create(context.Background(), newUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	token, _ := utils.GenerateJWT(newUser.GetID(), newUser.Role)
	c.JSON(http.StatusCreated, gin.H{"token": token, "user": newUser})
}

func (h *UserHandler) Login(c *gin.Context) {
	type LoginInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	user, err := h.userRepo.GetUserByEmail(context.Background(), input.Email)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	
	if !utils.CheckPasswordHash(input.Password, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}
	
	token, _ := utils.GenerateJWT(user.GetID(), user.Role)
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id := c.Param("id")
	
	var user models.User
	foundUser, err := h.userRepo.GetByID(context.Background(), id, &user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"user": foundUser})
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id := c.Param("id")
	
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Get existing user
	existingUser, err := h.userRepo.GetByID(context.Background(), id, &models.User{})
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	
	// Update fields
	existingUser.Username = user.Username
	existingUser.Email = user.Email
	existingUser.Role = user.Role
	existingUser.FirstName = user.FirstName
	existingUser.LastName = user.LastName
	existingUser.PhoneNumber = user.PhoneNumber
	
	if err := h.userRepo.Update(context.Background(), existingUser); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"user": existingUser})
}
