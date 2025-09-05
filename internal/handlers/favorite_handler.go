package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
)

// FavoriteHandler handles HTTP requests related to favorites
type FavoriteHandler struct {
	favoriteService *services.FavoriteService
	authContext     *auth.AuthContext
}

// NewFavoriteHandler creates a new favorite handler
func NewFavoriteHandler(serviceManager *services.ServiceManager) *FavoriteHandler {
	return &FavoriteHandler{
		favoriteService: serviceManager.FavoriteService,
		authContext:     auth.NewAuthContext(),
	}
}

// AddFavorite handles adding a product to favorites
// @Summary Add product to favorites
// @Description Add a product to the authenticated user's favorites
// @Tags Favorites
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.AddFavoriteRequest true "Favorite request"
// @Success 201 {object} map[string]interface{} "Added to favorites"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 409 {object} map[string]interface{} "Already in favorites"
// @Router /favorites [post]
func (h *FavoriteHandler) AddFavorite(c *gin.Context) {
	var request requests.AddFavoriteRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get authenticated user ID
	userIDStr, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Add favorite
	favorite, err := h.favoriteService.AddFavorite(c.Request.Context(), userIDStr, request.ProductID)
	if err != nil {
		if err.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Product not found",
			})
			return
		}
		if err.Error() == "product is already in favorites" {
			c.JSON(http.StatusConflict, gin.H{
				"error": "Product is already in favorites",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to add favorite",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Product added to favorites successfully",
		"favorite": favorite,
	})
}

// GetFavorites handles retrieving user's favorites
// @Summary List favorites
// @Description Get the authenticated user's favorites with product details
// @Tags Favorites
// @Produce json
// @Security FirebaseAuth
// @Success 200 {object} map[string]interface{} "Favorites retrieved"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites [get]
func (h *FavoriteHandler) GetFavorites(c *gin.Context) {
	// Get authenticated user ID
	userIDStr, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Get favorites with product details
	favorites, err := h.favoriteService.GetFavoritesWithProducts(c.Request.Context(), userIDStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to get favorites",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":   "Favorites retrieved successfully",
		"favorites": favorites,
		"count":     len(favorites),
	})
}

// RemoveFavorite handles removing a product from favorites
// @Summary Remove product from favorites
// @Description Remove a product from the authenticated user's favorites
// @Tags Favorites
// @Accept json
// @Produce json
// @Security FirebaseAuth
// @Param request body requests.RemoveFavoriteRequest true "Remove favorite request"
// @Success 200 {object} map[string]interface{} "Removed from favorites"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 404 {object} map[string]interface{} "Favorite not found"
// @Router /favorites [delete]
func (h *FavoriteHandler) RemoveFavorite(c *gin.Context) {
	var request requests.RemoveFavoriteRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Get authenticated user ID
	userIDStr, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Remove favorite
	err := h.favoriteService.RemoveFavorite(c.Request.Context(), userIDStr, request.ProductID)
	if err != nil {
		if err.Error() == "favorite not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Favorite not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to remove favorite",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product removed from favorites successfully",
	})
}

// CheckFavorite handles checking if a product is in favorites
// @Summary Check favorite status
// @Description Check if a product is in the authenticated user's favorites
// @Tags Favorites
// @Produce json
// @Security FirebaseAuth
// @Param product_id query string true "Product ID"
// @Success 200 {object} map[string]interface{} "Favorite status"
// @Failure 400 {object} map[string]interface{} "Product ID is required"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites/check [get]
func (h *FavoriteHandler) CheckFavorite(c *gin.Context) {
	productID := c.Query("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Product ID is required",
		})
		return
	}

	// Get authenticated user ID
	userIDStr, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	// Check if favorite
	isFavorite, err := h.favoriteService.IsFavorite(c.Request.Context(), userIDStr, productID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to check favorite status",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_favorite": isFavorite,
		"product_id":  productID,
	})
}
