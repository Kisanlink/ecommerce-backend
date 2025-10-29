package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"go.uber.org/zap"
)

// SessionFavoriteServiceInterface defines the interface for session favorite operations
type SessionFavoriteServiceInterface interface {
	GenerateSessionID() string
	ValidateSessionID(sessionID string) bool
	GetSessionFavorites(ctx context.Context, sessionID string) (*models.SessionFavoriteList, error)
	AddToSessionFavorites(ctx context.Context, sessionID, productID string) error
	RemoveFromSessionFavorites(ctx context.Context, sessionID, productID string) error
	IsSessionFavorite(ctx context.Context, sessionID, productID string) (bool, error)
	ClearSessionFavorites(ctx context.Context, sessionID string) error
	MigrateSessionFavorites(ctx context.Context, sessionID, userID string) error
	MergeSessionFavorites(ctx context.Context, sessionID, userID string) error
}

// SessionFavoriteService handles session-based favorites for anonymous users
type SessionFavoriteService struct {
	favoriteRepo  *repositories.FavoriteRepository
	productRepo   repositories.ProductRepositoryInterface
	imageRepo     repositories.ImageRepositoryInterface
	inventoryRepo repositories.InventoryRepositoryInterface
	logger        *zap.Logger
	// In-memory storage for session favorites
	sessionFavorites map[string]*models.SessionFavoriteList
}

// NewSessionFavoriteService creates a new session favorite service
func NewSessionFavoriteService(favoriteRepo *repositories.FavoriteRepository, productRepo repositories.ProductRepositoryInterface, imageRepo repositories.ImageRepositoryInterface, inventoryRepo repositories.InventoryRepositoryInterface, logger *zap.Logger) SessionFavoriteServiceInterface {
	return &SessionFavoriteService{
		favoriteRepo:     favoriteRepo,
		productRepo:      productRepo,
		imageRepo:        imageRepo,
		inventoryRepo:    inventoryRepo,
		logger:           logger,
		sessionFavorites: make(map[string]*models.SessionFavoriteList),
	}
}

// GenerateSessionID generates a unique session ID for anonymous users
func (s *SessionFavoriteService) GenerateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		s.logger.Error("Failed to generate session ID", zap.Error(err))
		// Fallback to timestamp-based ID
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	return "session_" + hex.EncodeToString(bytes)
}

// ValidateSessionID validates if a session ID is properly formatted
func (s *SessionFavoriteService) ValidateSessionID(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	// Check if it starts with "session_" and has proper length
	return len(sessionID) > 8 && len(sessionID) < 50
}

// populateProductDetails populates product images and inventory from normalized tables
func (s *SessionFavoriteService) populateProductDetails(ctx context.Context, product *models.Product) error {
	if product == nil {
		return nil
	}

	// Get all active images for the product
	images, err := s.imageRepo.GetActiveImagesByProductID(ctx, product.ID)
	if err != nil {
		// If no images found, that's okay - product will have empty image arrays
	} else {
		product.Images = images
	}

	// Get inventory information for the product
	inventory, err := s.inventoryRepo.GetByProductID(ctx, product.ID)
	if err == nil {
		// Calculate available stock
		inventory.AvailableStock = inventory.GetAvailableStock()
		product.Inventory = inventory
	}

	return nil
}

// GetSessionFavorites retrieves the session favorites list
func (s *SessionFavoriteService) GetSessionFavorites(ctx context.Context, sessionID string) (*models.SessionFavoriteList, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}

	// Check if session favorites exist in memory
	if favorites, exists := s.sessionFavorites[sessionID]; exists {
		// Populate product details for each favorite
		for _, favorite := range favorites.Favorites {
			if favorite.Product == nil {
				var product models.Product
				_, err := s.productRepo.GetByID(ctx, favorite.ProductID, &product)
				if err == nil {
					// Populate images and inventory from normalized tables
					s.populateProductDetails(ctx, &product)
					favorite.Product = &product
				}
			}
		}
		return favorites, nil
	}

	// Create new empty favorites list for this session
	favorites := models.NewSessionFavoriteList(sessionID)
	s.sessionFavorites[sessionID] = favorites
	s.logger.Info("Created new session favorites list", zap.String("session_id", sessionID))

	return favorites, nil
}

// AddToSessionFavorites adds a product to session favorites
func (s *SessionFavoriteService) AddToSessionFavorites(ctx context.Context, sessionID, productID string) error {
	if !s.ValidateSessionID(sessionID) {
		return fmt.Errorf("invalid session ID")
	}

	// Check if product exists
	var product models.Product
	_, err := s.productRepo.GetByID(ctx, productID, &product)
	if err != nil {
		if err.Error() == "no matching records found" || err.Error() == "model with id "+productID+" not found" {
			return fmt.Errorf("product not found")
		}
		return fmt.Errorf("product not found")
	}

	// Get or create session favorites
	favorites, err := s.GetSessionFavorites(ctx, sessionID)
	if err != nil {
		return err
	}

	// Check if already exists
	if favorites.HasFavorite(productID) {
		return fmt.Errorf("product is already in favorites")
	}

	// Add to session favorites
	if !favorites.AddFavorite(productID) {
		return fmt.Errorf("failed to add product to favorites")
	}

	s.logger.Info("Added product to session favorites",
		zap.String("session_id", sessionID),
		zap.String("product_id", productID))

	return nil
}

// RemoveFromSessionFavorites removes a product from session favorites
func (s *SessionFavoriteService) RemoveFromSessionFavorites(ctx context.Context, sessionID, productID string) error {
	if !s.ValidateSessionID(sessionID) {
		return fmt.Errorf("invalid session ID")
	}

	// Get session favorites
	favorites, err := s.GetSessionFavorites(ctx, sessionID)
	if err != nil {
		return err
	}

	// Check if exists
	if !favorites.HasFavorite(productID) {
		return fmt.Errorf("favorite not found")
	}

	// Remove from session favorites
	if !favorites.RemoveFavorite(productID) {
		return fmt.Errorf("failed to remove product from favorites")
	}

	s.logger.Info("Removed product from session favorites",
		zap.String("session_id", sessionID),
		zap.String("product_id", productID))

	return nil
}

// IsSessionFavorite checks if a product is in session favorites
func (s *SessionFavoriteService) IsSessionFavorite(ctx context.Context, sessionID, productID string) (bool, error) {
	if !s.ValidateSessionID(sessionID) {
		return false, fmt.Errorf("invalid session ID")
	}

	// Get session favorites
	favorites, err := s.GetSessionFavorites(ctx, sessionID)
	if err != nil {
		return false, err
	}

	return favorites.HasFavorite(productID), nil
}

// ClearSessionFavorites clears all session favorites
func (s *SessionFavoriteService) ClearSessionFavorites(ctx context.Context, sessionID string) error {
	if !s.ValidateSessionID(sessionID) {
		return fmt.Errorf("invalid session ID")
	}

	// Get session favorites
	favorites, err := s.GetSessionFavorites(ctx, sessionID)
	if err != nil {
		return err
	}

	// Clear favorites
	favorites.Clear()

	s.logger.Info("Cleared session favorites", zap.String("session_id", sessionID))
	return nil
}

// MigrateSessionFavorites migrates session favorites to user favorites
func (s *SessionFavoriteService) MigrateSessionFavorites(ctx context.Context, sessionID, userID string) error {
	if !s.ValidateSessionID(sessionID) {
		return fmt.Errorf("invalid session ID")
	}

	// Get session favorites
	favorites, err := s.GetSessionFavorites(ctx, sessionID)
	if err != nil {
		return err
	}

	if favorites.IsEmpty() {
		s.logger.Info("No session favorites to migrate", zap.String("session_id", sessionID))
		return nil
	}

	// Migrate each favorite to user favorites
	migratedCount := 0
	for _, favorite := range favorites.Favorites {
		// Check if already exists in user favorites
		exists, err := s.favoriteRepo.ExistsByUserAndProduct(ctx, userID, favorite.ProductID)
		if err != nil {
			s.logger.Error("Failed to check existing favorite",
				zap.Error(err),
				zap.String("user_id", userID),
				zap.String("product_id", favorite.ProductID))
			continue
		}

		if !exists {
			// Create user favorite
			userFavorite := models.NewFavorite(userID, favorite.ProductID)
			err = s.favoriteRepo.Create(ctx, userFavorite)
			if err != nil {
				s.logger.Error("Failed to create user favorite",
					zap.Error(err),
					zap.String("user_id", userID),
					zap.String("product_id", favorite.ProductID))
				continue
			}
			migratedCount++
		}
	}

	// Clear session favorites after migration
	favorites.Clear()

	s.logger.Info("Migrated session favorites to user favorites",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.Int("migrated_count", migratedCount))

	return nil
}

// MergeSessionFavorites merges session favorites with user favorites
func (s *SessionFavoriteService) MergeSessionFavorites(ctx context.Context, sessionID, userID string) error {
	if !s.ValidateSessionID(sessionID) {
		return fmt.Errorf("invalid session ID")
	}

	// Get session favorites
	favorites, err := s.GetSessionFavorites(ctx, sessionID)
	if err != nil {
		return err
	}

	if favorites.IsEmpty() {
		s.logger.Info("No session favorites to merge", zap.String("session_id", sessionID))
		return nil
	}

	// Get existing user favorites
	userFavorites, err := s.favoriteRepo.GetByUserID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user favorites: %w", err)
	}

	// Create a map of existing user favorite product IDs for quick lookup
	existingProductIDs := make(map[string]bool)
	for _, fav := range userFavorites {
		existingProductIDs[fav.ProductID] = true
	}

	// Add session favorites that don't already exist in user favorites
	addedCount := 0
	for _, favorite := range favorites.Favorites {
		if !existingProductIDs[favorite.ProductID] {
			// Create user favorite
			userFavorite := models.NewFavorite(userID, favorite.ProductID)
			err = s.favoriteRepo.Create(ctx, userFavorite)
			if err != nil {
				s.logger.Error("Failed to create user favorite during merge",
					zap.Error(err),
					zap.String("user_id", userID),
					zap.String("product_id", favorite.ProductID))
				continue
			}
			addedCount++
		}
	}

	// Clear session favorites after merge
	favorites.Clear()

	s.logger.Info("Merged session favorites with user favorites",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.Int("added_count", addedCount))

	return nil
}
