package handlers

import (
	"net/http"

	"github.com/Kisanlink/ecommerce-backend/entities/requests"
	"github.com/Kisanlink/ecommerce-backend/entities/responses"
	"github.com/Kisanlink/ecommerce-backend/internal/middleware/auth"
	"github.com/Kisanlink/ecommerce-backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// SessionFavoriteHandler handles HTTP requests related to session-based favorites
type SessionFavoriteHandler struct {
	favoriteService        *services.FavoriteService
	sessionFavoriteService services.SessionFavoriteServiceInterface
	authContext            *auth.AuthContext
	logger                 *zap.Logger
}

// NewSessionFavoriteHandler creates a new session favorite handler
func NewSessionFavoriteHandler(favoriteService *services.FavoriteService, sessionFavoriteService services.SessionFavoriteServiceInterface, logger *zap.Logger) *SessionFavoriteHandler {
	return &SessionFavoriteHandler{
		favoriteService:        favoriteService,
		sessionFavoriteService: sessionFavoriteService,
		authContext:            auth.NewAuthContext(),
		logger:                 logger,
	}
}

// getFavoriteIdentifier returns either userID (if authenticated) or sessionID (if anonymous)
func (h *SessionFavoriteHandler) getFavoriteIdentifier(c *gin.Context) (string, bool, error) {
	// Try to get authenticated user ID first
	if userID, authenticated := h.authContext.GetAuthenticatedUserID(c); authenticated {
		h.logger.Info("User is authenticated for favorites", zap.String("user_id", userID))
		return userID, true, nil
	}

	h.logger.Info("User is not authenticated, using session-based favorites")

	// If not authenticated, try to get session ID from header or query param
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		sessionID = c.Query("session_id")
	}

	// If no session ID provided, generate a new one
	if sessionID == "" {
		sessionID = h.sessionFavoriteService.GenerateSessionID()
		h.logger.Info("Generated new session ID for anonymous user", zap.String("session_id", sessionID))
	}

	// Validate session ID
	if !h.sessionFavoriteService.ValidateSessionID(sessionID) {
		h.logger.Error("Invalid session ID", zap.String("session_id", sessionID))
		return "", false, nil
	}

	return sessionID, false, nil
}

// GetFavorites retrieves the user's favorites (authenticated or session-based)
// @Summary Get favorites
// @Description Get the user's favorites (works for both authenticated and anonymous users)
// @Tags Favorites
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param session_id query string false "Session ID for anonymous users (alternative to header)"
// @Success 200 {object} map[string]interface{} "Favorites retrieved successfully"
// @Failure 400 {object} map[string]interface{} "Invalid session ID"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites [get]
func (h *SessionFavoriteHandler) GetFavorites(c *gin.Context) {
	favoriteIdentifier, isAuthenticated, err := h.getFavoriteIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	var favorites interface{}
	var favoritesErr error

	if isAuthenticated {
		// Use regular favorite service for authenticated users
		userFavorites, err := h.favoriteService.GetFavoritesWithProducts(c.Request.Context(), favoriteIdentifier)
		if err != nil {
			favoritesErr = err
		} else {
			// Convert to lightweight response
			favorites = responses.NewFavoritesListResponse(userFavorites)
		}
	} else {
		// Use session favorite service for anonymous users
		sessionFavorites, err := h.sessionFavoriteService.GetSessionFavorites(c.Request.Context(), favoriteIdentifier)
		if err != nil {
			favoritesErr = err
		} else {
			// Convert to lightweight response
			favorites = responses.NewSessionFavoriteResponse(sessionFavorites)
		}
	}

	if favoritesErr != nil {
		h.logger.Error("Failed to get favorites", zap.Error(favoritesErr), zap.String("identifier", favoriteIdentifier))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to retrieve favorites",
		})
		return
	}

	response := gin.H{
		"success":          true,
		"favorites":        favorites,
		"is_authenticated": isAuthenticated,
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["session_id"] = favoriteIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// AddFavorite adds a product to the user's favorites (authenticated or session-based)
// @Summary Add product to favorites
// @Description Add a product to the user's favorites (works for both authenticated and anonymous users)
// @Tags Favorites
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param session_id query string false "Session ID for anonymous users (alternative to header)"
// @Param request body requests.AddFavoriteRequest true "Add favorite request"
// @Success 201 {object} map[string]interface{} "Added to favorites"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Product not found"
// @Failure 409 {object} map[string]interface{} "Already in favorites"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites [post]
func (h *SessionFavoriteHandler) AddFavorite(c *gin.Context) {
	var request requests.AddFavoriteRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	favoriteIdentifier, isAuthenticated, err := h.getFavoriteIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	var addErr error

	if isAuthenticated {
		// Use regular favorite service for authenticated users
		_, addErr = h.favoriteService.AddFavorite(c.Request.Context(), favoriteIdentifier, request.ProductID)
	} else {
		// Use session favorite service for anonymous users
		addErr = h.sessionFavoriteService.AddToSessionFavorites(c.Request.Context(), favoriteIdentifier, request.ProductID)
	}

	if addErr != nil {
		if addErr.Error() == "product not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Product not found",
			})
			return
		}
		if addErr.Error() == "product is already in favorites" {
			c.JSON(http.StatusConflict, gin.H{
				"success": false,
				"error":   "Product is already in favorites",
			})
			return
		}
		h.logger.Error("Failed to add favorite", zap.Error(addErr), zap.String("identifier", favoriteIdentifier))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to add favorite",
			"details": addErr.Error(),
		})
		return
	}

	response := gin.H{
		"success":          true,
		"message":          "Product added to favorites successfully",
		"is_authenticated": isAuthenticated,
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["session_id"] = favoriteIdentifier
	}

	c.JSON(http.StatusCreated, response)
}

// RemoveFavorite removes a product from the user's favorites (authenticated or session-based)
// @Summary Remove product from favorites
// @Description Remove a product from the user's favorites (works for both authenticated and anonymous users)
// @Tags Favorites
// @Accept json
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param session_id query string false "Session ID for anonymous users (alternative to header)"
// @Param request body requests.RemoveFavoriteRequest true "Remove favorite request"
// @Success 200 {object} map[string]interface{} "Removed from favorites"
// @Failure 400 {object} map[string]interface{} "Invalid request"
// @Failure 404 {object} map[string]interface{} "Favorite not found"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites [delete]
func (h *SessionFavoriteHandler) RemoveFavorite(c *gin.Context) {
	var request requests.RemoveFavoriteRequest

	// Bind and validate request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	favoriteIdentifier, isAuthenticated, err := h.getFavoriteIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	var removeErr error

	if isAuthenticated {
		// Use regular favorite service for authenticated users
		removeErr = h.favoriteService.RemoveFavorite(c.Request.Context(), favoriteIdentifier, request.ProductID)
	} else {
		// Use session favorite service for anonymous users
		removeErr = h.sessionFavoriteService.RemoveFromSessionFavorites(c.Request.Context(), favoriteIdentifier, request.ProductID)
	}

	if removeErr != nil {
		if removeErr.Error() == "favorite not found" {
			c.JSON(http.StatusNotFound, gin.H{
				"success": false,
				"error":   "Favorite not found",
			})
			return
		}
		h.logger.Error("Failed to remove favorite", zap.Error(removeErr), zap.String("identifier", favoriteIdentifier))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to remove favorite",
			"details": removeErr.Error(),
		})
		return
	}

	response := gin.H{
		"success":          true,
		"message":          "Product removed from favorites successfully",
		"is_authenticated": isAuthenticated,
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["session_id"] = favoriteIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// CheckFavorite checks if a product is in the user's favorites (authenticated or session-based)
// @Summary Check favorite status
// @Description Check if a product is in the user's favorites (works for both authenticated and anonymous users)
// @Tags Favorites
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param session_id query string false "Session ID for anonymous users (alternative to header)"
// @Param product_id query string true "Product ID"
// @Success 200 {object} map[string]interface{} "Favorite status"
// @Failure 400 {object} map[string]interface{} "Product ID is required"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites/check [get]
func (h *SessionFavoriteHandler) CheckFavorite(c *gin.Context) {
	productID := c.Query("product_id")
	if productID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Product ID is required",
		})
		return
	}

	favoriteIdentifier, isAuthenticated, err := h.getFavoriteIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	var isFavorite bool
	var checkErr error

	if isAuthenticated {
		// Use regular favorite service for authenticated users
		isFavorite, checkErr = h.favoriteService.IsFavorite(c.Request.Context(), favoriteIdentifier, productID)
	} else {
		// Use session favorite service for anonymous users
		isFavorite, checkErr = h.sessionFavoriteService.IsSessionFavorite(c.Request.Context(), favoriteIdentifier, productID)
	}

	if checkErr != nil {
		h.logger.Error("Failed to check favorite status", zap.Error(checkErr), zap.String("identifier", favoriteIdentifier))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to check favorite status",
			"details": checkErr.Error(),
		})
		return
	}

	response := gin.H{
		"success":          true,
		"is_favorite":      isFavorite,
		"product_id":       productID,
		"is_authenticated": isAuthenticated,
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["session_id"] = favoriteIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// ClearFavorites clears all favorites for the user (authenticated or session-based)
// @Summary Clear all favorites
// @Description Clear all favorites for the user (works for both authenticated and anonymous users)
// @Tags Favorites
// @Produce json
// @Param X-Session-ID header string false "Session ID for anonymous users"
// @Param session_id query string false "Session ID for anonymous users (alternative to header)"
// @Success 200 {object} map[string]interface{} "Favorites cleared"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites/clear [delete]
func (h *SessionFavoriteHandler) ClearFavorites(c *gin.Context) {
	favoriteIdentifier, isAuthenticated, err := h.getFavoriteIdentifier(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	var clearErr error

	if isAuthenticated {
		// For authenticated users, we need to get all favorites and remove them one by one
		// since there's no bulk clear method in the regular favorite service
		favorites, err := h.favoriteService.GetFavoritesWithProducts(c.Request.Context(), favoriteIdentifier)
		if err != nil {
			h.logger.Error("Failed to get favorites for clearing", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "Failed to clear favorites",
			})
			return
		}

		for _, favorite := range favorites {
			removeErr := h.favoriteService.RemoveFavorite(c.Request.Context(), favoriteIdentifier, favorite.ProductID)
			if removeErr != nil {
				h.logger.Error("Failed to remove favorite during clear", zap.Error(removeErr))
			}
		}
	} else {
		// Use session favorite service for anonymous users
		clearErr = h.sessionFavoriteService.ClearSessionFavorites(c.Request.Context(), favoriteIdentifier)
	}

	if clearErr != nil {
		h.logger.Error("Failed to clear favorites", zap.Error(clearErr), zap.String("identifier", favoriteIdentifier))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to clear favorites",
			"details": clearErr.Error(),
		})
		return
	}

	response := gin.H{
		"success":          true,
		"message":          "Favorites cleared successfully",
		"is_authenticated": isAuthenticated,
	}

	// Add session ID to response for anonymous users
	if !isAuthenticated {
		response["session_id"] = favoriteIdentifier
	}

	c.JSON(http.StatusOK, response)
}

// MigrateFavorites migrates session favorites to user favorites (requires authentication)
// @Summary Migrate session favorites
// @Description Migrate session favorites to user favorites after login
// @Tags Favorites
// @Produce json
// @Security FirebaseAuth
// @Param X-Session-ID header string true "Session ID to migrate"
// @Success 200 {object} map[string]interface{} "Favorites migrated"
// @Failure 400 {object} map[string]interface{} "Invalid session ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites/migrate [post]
func (h *SessionFavoriteHandler) MigrateFavorites(c *gin.Context) {
	// Get authenticated user ID
	userID, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get session ID from header
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID is required",
		})
		return
	}

	// Validate session ID
	if !h.sessionFavoriteService.ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	// Migrate session favorites to user favorites
	err := h.sessionFavoriteService.MigrateSessionFavorites(c.Request.Context(), sessionID, userID)
	if err != nil {
		h.logger.Error("Failed to migrate session favorites", zap.Error(err), zap.String("user_id", userID), zap.String("session_id", sessionID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to migrate favorites",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Favorites migrated successfully",
		"user_id": userID,
	})
}

// MergeFavorites merges session favorites with user favorites (requires authentication)
// @Summary Merge session favorites
// @Description Merge session favorites with user favorites after login
// @Tags Favorites
// @Produce json
// @Security FirebaseAuth
// @Param X-Session-ID header string true "Session ID to merge"
// @Success 200 {object} map[string]interface{} "Favorites merged"
// @Failure 400 {object} map[string]interface{} "Invalid session ID"
// @Failure 401 {object} map[string]interface{} "User not authenticated"
// @Failure 500 {object} map[string]interface{} "Internal server error"
// @Router /favorites/merge [post]
func (h *SessionFavoriteHandler) MergeFavorites(c *gin.Context) {
	// Get authenticated user ID
	userID, authenticated := h.authContext.GetAuthenticatedUserID(c)
	if !authenticated {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"error":   "User not authenticated",
		})
		return
	}

	// Get session ID from header
	sessionID := c.GetHeader("X-Session-ID")
	if sessionID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Session ID is required",
		})
		return
	}

	// Validate session ID
	if !h.sessionFavoriteService.ValidateSessionID(sessionID) {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error":   "Invalid session ID",
		})
		return
	}

	// Merge session favorites with user favorites
	err := h.sessionFavoriteService.MergeSessionFavorites(c.Request.Context(), sessionID, userID)
	if err != nil {
		h.logger.Error("Failed to merge session favorites", zap.Error(err), zap.String("user_id", userID), zap.String("session_id", sessionID))
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "Failed to merge favorites",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Favorites merged successfully",
		"user_id": userID,
	})
}
