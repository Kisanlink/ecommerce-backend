package repositories

import (
	"context"
	"fmt"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
)

// FavoriteRepository handles database operations for favorites
type FavoriteRepository struct {
	*base.BaseFilterableRepository[*models.Favorite]
	dbManager db.DBManager
}

// NewFavoriteRepository creates a new favorite repository
func NewFavoriteRepository(dbManager db.DBManager) *FavoriteRepository {
	repo := &FavoriteRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Favorite](),
		dbManager:                dbManager,
	}
	repo.SetDBManager(dbManager)
	return repo
}

// GetByUserID retrieves all favorites for a user
func (r *FavoriteRepository) GetByUserID(ctx context.Context, userID string) ([]*models.Favorite, error) {
	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		WhereNull("deleted_at").
		Sort("created_at", "desc").
		Build()

	return r.Find(ctx, filter)
}

// GetByUserAndProduct finds a specific favorite by user and product
func (r *FavoriteRepository) GetByUserAndProduct(ctx context.Context, userID, productID string) (*models.Favorite, error) {
	filter := base.NewFilterBuilder().
		Where("user_id", base.OpEqual, userID).
		Where("product_id", base.OpEqual, productID).
		WhereNull("deleted_at").
		Build()

	return r.FindOne(ctx, filter)
}

// ExistsByUserAndProduct checks if a favorite exists for a user and product
func (r *FavoriteRepository) ExistsByUserAndProduct(ctx context.Context, userID, productID string) (bool, error) {
	favorite, err := r.GetByUserAndProduct(ctx, userID, productID)
	if err != nil {
		// If no matching records found, it means the favorite doesn't exist (which is not an error)
		if err.Error() == "no matching records found" {
			return false, nil
		}
		return false, err
	}
	return favorite != nil, nil
}

// DeleteByUserAndProduct soft deletes a favorite
func (r *FavoriteRepository) DeleteByUserAndProduct(ctx context.Context, userID, productID string) error {
	// Find the favorite first
	favorite, err := r.GetByUserAndProduct(ctx, userID, productID)
	if err != nil {
		// If no matching records found, it means the favorite doesn't exist
		if err.Error() == "no matching records found" {
			return fmt.Errorf("favorite not found")
		}
		return fmt.Errorf("failed to find favorite: %w", err)
	}
	if favorite == nil {
		return fmt.Errorf("favorite not found")
	}

	// Use the dbManager's SoftDelete method directly
	return r.dbManager.SoftDelete(ctx, favorite.ID, favorite, "system")
}
