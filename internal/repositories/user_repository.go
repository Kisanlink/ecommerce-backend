package repositories

import (
	"context"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
)

// UserRepository provides user-specific repository methods
type UserRepository struct {
	*base.BaseFilterableRepository[*models.User]
}

// NewUserRepository creates a new user repository
func NewUserRepository(dbManager interface{}) *UserRepository {
	repo := &UserRepository{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.User](),
	}
	// Set the database manager for database operations
	repo.SetDBManager(dbManager)
	return repo
}

// GetUserByEmail retrieves a user by email using filters
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*models.User, error) {
	filter := base.NewFilterBuilder().
		Where("email", base.OpEqual, email).
		Build()

	return r.FindOne(ctx, filter)
}

// GetUserByFirebaseUID retrieves a user by Firebase UID using filters
func (r *UserRepository) GetUserByFirebaseUID(ctx context.Context, firebaseUID string) (*models.User, error) {
	filter := base.NewFilterBuilder().
		Where("firebase_uid", base.OpEqual, firebaseUID).
		Build()

	return r.FindOne(ctx, filter)
}

// GetActiveUsers retrieves all active (non-deleted) users with pagination
func (r *UserRepository) GetActiveUsers(ctx context.Context, page, pageSize int) ([]*models.User, error) {
	filter := base.NewFilterBuilder().
		Where("is_deleted", base.OpEqual, false).
		Page(page, pageSize).
		Sort("created_at", "desc").
		Build()

	return r.Find(ctx, filter)
}

// SearchUsersByName searches users by first name or last name
func (r *UserRepository) SearchUsersByName(ctx context.Context, searchTerm string, page, pageSize int) ([]*models.User, error) {
	filter := base.NewFilterBuilder().
		Or(
			base.FilterCondition{Field: "first_name", Operator: base.OpContains, Value: searchTerm},
			base.FilterCondition{Field: "last_name", Operator: base.OpContains, Value: searchTerm},
		).
		Where("is_deleted", base.OpEqual, false).
		Page(page, pageSize).
		Sort("first_name", "asc").
		Build()

	return r.Find(ctx, filter)
}

// CountUsersByRole counts users by their role
func (r *UserRepository) CountUsersByRole(ctx context.Context, role string) (int64, error) {
	filter := base.NewFilterBuilder().
		Where("role", base.OpEqual, role).
		Where("is_deleted", base.OpEqual, false).
		Build()

	return r.CountWithFilter(ctx, filter)
}
