package services

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
)

// FavoriteService provides business logic for favorites
type FavoriteService struct {
	favoriteRepo  *repositories.FavoriteRepository
	productRepo   repositories.ProductRepositoryInterface
	imageRepo     repositories.ImageRepositoryInterface
	inventoryRepo repositories.InventoryRepositoryInterface
	repoManager   *repositories.RepositoryManager
}

// NewFavoriteService creates a new favorite service
func NewFavoriteService(repoManager *repositories.RepositoryManager) *FavoriteService {
	return &FavoriteService{
		favoriteRepo:  repoManager.FavoriteRepository,
		productRepo:   repoManager.ProductRepository,
		imageRepo:     repoManager.ImageRepository,
		inventoryRepo: repoManager.InventoryRepository,
		repoManager:   repoManager,
	}
}

// AddFavorite adds a product to user's favorites
func (s *FavoriteService) AddFavorite(ctx context.Context, userID, productID string) (*models.Favorite, error) {
	// Check if product exists
	var product models.Product
	_, err := s.productRepo.GetByID(ctx, productID, &product)
	if err != nil {
		// Handle different types of product not found errors
		if err.Error() == "no matching records found" || err.Error() == "model with id "+productID+" not found" {
			return nil, fmt.Errorf("product not found")
		}
		return nil, fmt.Errorf("product not found")
	}

	// Check if favorite already exists
	exists, err := s.favoriteRepo.ExistsByUserAndProduct(ctx, userID, productID)
	if err != nil {
		fmt.Printf("DEBUG: ExistsByUserAndProduct error: %q\n", err.Error())
		return nil, fmt.Errorf("failed to check if favorite exists: %w", err)
	}
	if exists {
		return nil, fmt.Errorf("product is already in favorites")
	}

	// Create new favorite
	favorite := models.NewFavorite(userID, productID)

	// Save the favorite
	err = s.favoriteRepo.Create(ctx, favorite)
	if err != nil {
		return nil, fmt.Errorf("failed to add favorite: %w", err)
	}

	return favorite, nil
}

// GetUserFavorites retrieves all favorites for a user with product details
func (s *FavoriteService) GetUserFavorites(ctx context.Context, userID string) ([]*models.Favorite, []*models.Product, error) {
	// Get favorites
	favorites, err := s.favoriteRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get favorites: %w", err)
	}

	// Get product details for each favorite
	var products []*models.Product
	for _, favorite := range favorites {
		var product models.Product
		_, err := s.productRepo.GetByID(ctx, favorite.ProductID, &product)
		if err != nil {
			continue // Skip if product not found
		}
		products = append(products, &product)
	}

	return favorites, products, nil
}

// populateProductDetails populates product images and inventory from normalized tables
func (s *FavoriteService) populateProductDetails(ctx context.Context, product *models.Product) error {
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

// GetFavoritesWithProducts retrieves favorites with product details populated
func (s *FavoriteService) GetFavoritesWithProducts(ctx context.Context, userID string) ([]*models.Favorite, error) {
	// Use repository pattern for now - this is working and reliable
	return s.getFavoritesWithRepositoryPattern(ctx, userID)
}

// getFavoritesWithRepositoryPattern fallback method using repository pattern
func (s *FavoriteService) getFavoritesWithRepositoryPattern(ctx context.Context, userID string) ([]*models.Favorite, error) {
	// Get favorites
	favorites, err := s.favoriteRepo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get favorites: %w", err)
	}

	// Populate product details for each favorite
	for _, favorite := range favorites {
		var product models.Product
		_, err := s.productRepo.GetByID(ctx, favorite.ProductID, &product)
		if err == nil {
			// Populate images and inventory from normalized tables
			s.populateProductDetails(ctx, &product)
			favorite.Product = &product
		}
	}

	return favorites, nil
}

// RemoveFavorite removes a product from user's favorites
func (s *FavoriteService) RemoveFavorite(ctx context.Context, userID, productID string) error {
	// Check if favorite exists
	exists, err := s.favoriteRepo.ExistsByUserAndProduct(ctx, userID, productID)
	if err != nil {
		return fmt.Errorf("failed to check if favorite exists: %w", err)
	}
	if !exists {
		return fmt.Errorf("favorite not found")
	}

	// Remove the favorite
	err = s.favoriteRepo.DeleteByUserAndProduct(ctx, userID, productID)
	if err != nil {
		return fmt.Errorf("failed to remove favorite: %w", err)
	}

	return nil
}

// IsFavorite checks if a product is in user's favorites
func (s *FavoriteService) IsFavorite(ctx context.Context, userID, productID string) (bool, error) {
	return s.favoriteRepo.ExistsByUserAndProduct(ctx, userID, productID)
}
