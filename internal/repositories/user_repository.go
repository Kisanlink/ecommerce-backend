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

// GetByFirebaseUID is an alias to GetUserByFirebaseUID for guard usage
func (r *UserRepository) GetByFirebaseUID(ctx context.Context, firebaseUID string) (*models.User, error) {
	return r.GetUserByFirebaseUID(ctx, firebaseUID)
}

// GetActiveUsers retrieves all active (non-deleted) users with pagination
func (r *UserRepository) GetActiveUsers(ctx context.Context, page, pageSize int) ([]*models.User, error) {
	filter := base.NewFilterBuilder().
		WhereNull("deleted_at").
		Page(page, pageSize).
		Sort("created_at", "desc").
		Build()

	return r.Find(ctx, filter)
}

// SearchUsersByName searches users by name
func (r *UserRepository) SearchUsersByName(ctx context.Context, searchTerm string, page, pageSize int) ([]*models.User, error) {
	filter := base.NewFilterBuilder().
		Where("name", base.OpContains, searchTerm).
		WhereNull("deleted_at").
		Page(page, pageSize).
		Sort("name", "asc").
		Build()

	return r.Find(ctx, filter)
}

// CountUsersByStatus counts users by their active status
func (r *UserRepository) CountUsersByStatus(ctx context.Context, isActive bool) (int64, error) {
	filter := base.NewFilterBuilder().
		Where("is_active", base.OpEqual, isActive).
		WhereNull("deleted_at").
		Build()

	return r.CountWithFilter(ctx, filter)
}

// CountActiveUsers counts all active (non-deleted) users
func (r *UserRepository) CountActiveUsers(ctx context.Context) (int64, error) {
	// Since Count methods have table identification issues in kisanlink-db,
	// we'll retrieve all records and count them (not ideal for large datasets)
	filter := base.NewFilterBuilder().
		WhereNull("deleted_at").
		Build()

	users, err := r.Find(ctx, filter)
	if err != nil {
		return 0, err
	}

	return int64(len(users)), nil
}
