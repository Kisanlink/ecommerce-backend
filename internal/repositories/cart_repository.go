package repositories

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/database"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// CartRepositoryInterface defines the contract for cart repository operations
type CartRepositoryInterface interface {
	// Core cart operations
	GetUserCart(ctx context.Context, userID string) (*models.Cart, error)
	CreateCart(ctx context.Context, cart *models.Cart) error
	UpdateCart(ctx context.Context, cart *models.Cart) error
	DeleteCart(ctx context.Context, cartID, userID string) error

	// Cart item operations
	AddItemToCart(ctx context.Context, userID, productID string, quantity int, unitPrice float64) (*models.Cart, error)
	UpdateCartItem(ctx context.Context, userID, productID string, quantity int) (*models.Cart, error)
	RemoveCartItem(ctx context.Context, userID, productID string) (*models.Cart, error)
	ClearUserCart(ctx context.Context, userID string) error

	// Cache operations
	GetCartFromCache(ctx context.Context, userID string) (*models.CartCache, error)
	SaveCartToCache(ctx context.Context, cartCache *models.CartCache) error
	DeleteCartFromCache(ctx context.Context, userID string) error

	// Validation
	ValidateCartItems(ctx context.Context, userID string) ([]string, error) // Returns list of issues
}

// CartRepositoryImpl implements CartRepositoryInterface
type CartRepositoryImpl struct {
	*base.BaseFilterableRepository[*models.Cart]
	dbManager    db.DBManager
	redisManager *database.RedisManager
	logger       *zap.Logger
	cacheExpiry  time.Duration
}

// NewCartRepository creates a new cart repository with PostgreSQL and Redis support
func NewCartRepository(dbManager db.DBManager, redisManager *database.RedisManager, logger *zap.Logger) CartRepositoryInterface {
	repo := &CartRepositoryImpl{
		BaseFilterableRepository: base.NewBaseFilterableRepository[*models.Cart](),
		dbManager:                dbManager,
		redisManager:             redisManager,
		logger:                   logger,
		cacheExpiry:              24 * time.Hour, // 24 hours cache
	}
	repo.SetDBManager(dbManager)
	return repo
}

// Redis key generation
func (r *CartRepositoryImpl) getCartCacheKey(userID string) string {
	return fmt.Sprintf("user_cart:%s", userID)
}

func (r *CartRepositoryImpl) getCartLockKey(userID string) string {
	return fmt.Sprintf("cart_lock:%s", userID)
}

// Cache Operations
func (r *CartRepositoryImpl) GetCartFromCache(ctx context.Context, userID string) (*models.CartCache, error) {
	if r.redisManager == nil {
		return nil, fmt.Errorf("redis not available")
	}

	key := r.getCartCacheKey(userID)
	data, err := r.redisManager.Get(key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "nil") {
			return nil, nil // Cache miss
		}
		return nil, err
	}

	var cartCache models.CartCache
	if err := json.Unmarshal([]byte(data), &cartCache); err != nil {
		r.logger.Error("Failed to unmarshal cart cache", zap.Error(err), zap.String("user_id", userID))
		return nil, err
	}

	// Check if cache has expired
	if cartCache.IsExpired() {
		r.DeleteCartFromCache(ctx, userID)
		return nil, nil
	}

	return &cartCache, nil
}

func (r *CartRepositoryImpl) SaveCartToCache(ctx context.Context, cartCache *models.CartCache) error {
	if r.redisManager == nil {
		return nil // Skip caching if Redis not available
	}

	key := r.getCartCacheKey(cartCache.UserID)
	cartCache.UpdatedAt = time.Now()
	cartCache.ExpiresAt = time.Now().Add(r.cacheExpiry)

	data, err := json.Marshal(cartCache)
	if err != nil {
		return err
	}

	return r.redisManager.Set(key, data, r.cacheExpiry)
}

func (r *CartRepositoryImpl) DeleteCartFromCache(ctx context.Context, userID string) error {
	if r.redisManager == nil {
		return nil
	}

	key := r.getCartCacheKey(userID)
	return r.redisManager.Del(key)
}

// Database Operations
func (r *CartRepositoryImpl) GetUserCart(ctx context.Context, userID string) (*models.Cart, error) {
	// Always get from database for consistency and proper ID generation
	// Cache is used for read performance optimization, but DB is source of truth
	dbCart, err := r.getUserCartFromDB(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Update cache asynchronously for future reads
	go func() {
		if cartCache := r.convertCartToCache(dbCart); cartCache != nil {
			r.SaveCartToCache(context.Background(), cartCache)
		}
	}()

	return dbCart, nil
}

func (r *CartRepositoryImpl) getUserCartFromDB(ctx context.Context, userID string) (*models.Cart, error) {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "user_id", Operator: base.OpEqual, Value: userID},
			},
			Logic: base.LogicAnd,
		},
	}

	var carts []*models.Cart
	if err := r.dbManager.List(ctx, filter, &carts); err != nil {
		return nil, err
	}

	if len(carts) == 0 {
		// Create new cart
		cart := r.createNewCartForUser(userID)
		if err := r.CreateCart(ctx, cart); err != nil {
			return nil, err
		}
		return cart, nil
	}

	cart := carts[0]

	// Load cart items
	if err := r.loadCartItems(ctx, cart); err != nil {
		r.logger.Error("Failed to load cart items", zap.Error(err))
		// Continue without items rather than failing
	}

	// Update cache
	go func() {
		if cartCache := r.convertCartToCache(cart); cartCache != nil {
			r.SaveCartToCache(context.Background(), cartCache)
		}
	}()

	return cart, nil
}

func (r *CartRepositoryImpl) createNewCartForUser(userID string) *models.Cart {
	cart := models.NewCart(userID) // Use the proper constructor from the model
	return cart
}

func (r *CartRepositoryImpl) loadCartItems(ctx context.Context, cart *models.Cart) error {
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{
				{Field: "cart_id", Operator: base.OpEqual, Value: cart.ID},
			},
			Logic: base.LogicAnd,
		},
	}

	var cartItems []*models.CartItem
	if err := r.dbManager.List(ctx, filter, &cartItems); err != nil {
		return err
	}

	// Convert to slice of structs
	cart.Items = make([]models.CartItem, len(cartItems))
	for i, item := range cartItems {
		cart.Items[i] = *item
	}

	return nil
}

func (r *CartRepositoryImpl) CreateCart(ctx context.Context, cart *models.Cart) error {
	return r.dbManager.Create(ctx, cart)
}

func (r *CartRepositoryImpl) UpdateCart(ctx context.Context, cart *models.Cart) error {
	// Update cart totals
	r.calculateCartTotals(cart)

	if err := r.dbManager.Update(ctx, cart); err != nil {
		return err
	}

	// Update cache
	go func() {
		if cartCache := r.convertCartToCache(cart); cartCache != nil {
			r.SaveCartToCache(context.Background(), cartCache)
		}
	}()

	return nil
}

func (r *CartRepositoryImpl) DeleteCart(ctx context.Context, cartID, userID string) error {
	cart := &models.Cart{}
	if err := r.dbManager.SoftDelete(ctx, cartID, cart, userID); err != nil {
		return err
	}

	// Clear cache
	go func() {
		r.DeleteCartFromCache(context.Background(), userID)
	}()

	return nil
}

// Cart Item Operations
func (r *CartRepositoryImpl) AddItemToCart(ctx context.Context, userID, productID string, quantity int, unitPrice float64) (*models.Cart, error) {
	// Acquire lock to prevent concurrent modifications
	if err := r.acquireLock(ctx, userID); err != nil {
		return nil, err
	}
	defer r.releaseLock(ctx, userID)

	cart, err := r.GetUserCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Check if item already exists
	var existingItem *models.CartItem
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			existingItem = &cart.Items[i]
			break
		}
	}

	if existingItem != nil {
		// Update existing item
		existingItem.Quantity += quantity
		existingItem.TotalPrice = existingItem.UnitPrice * float64(existingItem.Quantity)

		if err := r.dbManager.Update(ctx, existingItem); err != nil {
			return nil, err
		}
	} else {
		// Create new cart item
		cartItem := models.NewCartItem(cart.ID, productID, quantity, unitPrice)

		if err := r.dbManager.Create(ctx, cartItem); err != nil {
			return nil, err
		}

		cart.Items = append(cart.Items, *cartItem)
	}

	// Update cart totals and save
	if err := r.UpdateCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (r *CartRepositoryImpl) UpdateCartItem(ctx context.Context, userID, productID string, quantity int) (*models.Cart, error) {
	if err := r.acquireLock(ctx, userID); err != nil {
		return nil, err
	}
	defer r.releaseLock(ctx, userID)

	cart, err := r.GetUserCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Find item to update
	var itemToUpdate *models.CartItem
	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			itemToUpdate = &cart.Items[i]
			break
		}
	}

	if itemToUpdate == nil {
		return nil, fmt.Errorf("cart item not found")
	}

	if quantity <= 0 {
		// Remove item
		return r.RemoveCartItem(ctx, userID, productID)
	}

	// Update quantity
	itemToUpdate.Quantity = quantity
	itemToUpdate.TotalPrice = itemToUpdate.UnitPrice * float64(quantity)

	if err := r.dbManager.Update(ctx, itemToUpdate); err != nil {
		return nil, err
	}

	// Update cart
	if err := r.UpdateCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (r *CartRepositoryImpl) RemoveCartItem(ctx context.Context, userID, productID string) (*models.Cart, error) {
	if err := r.acquireLock(ctx, userID); err != nil {
		return nil, err
	}
	defer r.releaseLock(ctx, userID)

	cart, err := r.GetUserCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Find and remove item
	var itemToRemove *models.CartItem
	newItems := make([]models.CartItem, 0, len(cart.Items))

	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			itemToRemove = &cart.Items[i]
		} else {
			newItems = append(newItems, cart.Items[i])
		}
	}

	if itemToRemove == nil {
		return nil, fmt.Errorf("cart item not found")
	}

	// Soft delete from database
	if err := r.dbManager.SoftDelete(ctx, itemToRemove.ID, itemToRemove, userID); err != nil {
		return nil, err
	}

	cart.Items = newItems

	// Update cart
	if err := r.UpdateCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (r *CartRepositoryImpl) ClearUserCart(ctx context.Context, userID string) error {
	if err := r.acquireLock(ctx, userID); err != nil {
		return err
	}
	defer r.releaseLock(ctx, userID)

	cart, err := r.GetUserCart(ctx, userID)
	if err != nil {
		return err
	}

	// Soft delete all cart items
	for _, item := range cart.Items {
		if err := r.dbManager.SoftDelete(ctx, item.ID, &item, userID); err != nil {
			r.logger.Error("Failed to delete cart item", zap.Error(err), zap.String("item_id", item.ID))
		}
	}

	// Clear items and update cart
	cart.Items = []models.CartItem{}
	if err := r.UpdateCart(ctx, cart); err != nil {
		return err
	}

	return nil
}

// Helper methods
func (r *CartRepositoryImpl) calculateCartTotals(cart *models.Cart) {
	cart.TotalItems = 0
	cart.TotalPrice = 0

	for _, item := range cart.Items {
		cart.TotalItems += item.Quantity
		cart.TotalPrice += item.TotalPrice
	}
}

func (r *CartRepositoryImpl) convertCartToCache(cart *models.Cart) *models.CartCache {
	if cart == nil {
		return nil
	}

	cartCache := &models.CartCache{
		UserID:     cart.UserID,
		TotalItems: cart.TotalItems,
		TotalPrice: cart.TotalPrice,
		Items:      make([]models.CartItemCache, len(cart.Items)),
		UpdatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(r.cacheExpiry),
	}

	for i, item := range cart.Items {
		cartCache.Items[i] = models.CartItemCache{
			ProductID:  item.ProductID,
			Quantity:   item.Quantity,
			UnitPrice:  item.UnitPrice,
			TotalPrice: item.TotalPrice,
		}
	}

	return cartCache
}

// Locking mechanisms for concurrent access
func (r *CartRepositoryImpl) acquireLock(ctx context.Context, userID string) error {
	if r.redisManager == nil {
		return nil // Skip locking if Redis not available
	}

	lockKey := r.getCartLockKey(userID)
	acquired, err := r.redisManager.SetNX(lockKey, "locked", 30*time.Second)
	if err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("cart is locked for user %s", userID)
	}
	return nil
}

func (r *CartRepositoryImpl) releaseLock(ctx context.Context, userID string) {
	if r.redisManager == nil {
		return
	}

	lockKey := r.getCartLockKey(userID)
	if err := r.redisManager.Del(lockKey); err != nil {
		r.logger.Error("Failed to release cart lock", zap.Error(err), zap.String("user_id", userID))
	}
}

func (r *CartRepositoryImpl) ValidateCartItems(ctx context.Context, userID string) ([]string, error) {
	cart, err := r.GetUserCart(ctx, userID)
	if err != nil {
		return nil, err
	}

	var issues []string

	for _, item := range cart.Items {
		// This would require product validation
		// For now, just return basic validation
		if item.Quantity <= 0 {
			issues = append(issues, fmt.Sprintf("Item %s has invalid quantity", item.ProductID))
		}
		if item.UnitPrice <= 0 {
			issues = append(issues, fmt.Sprintf("Item %s has invalid price", item.ProductID))
		}
	}

	return issues, nil
}
