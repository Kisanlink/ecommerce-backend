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
