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

	// Admin/maintenance operations
	CleanupDuplicateCarts(ctx context.Context) error
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
		return nil, nil // Return nil instead of error when Redis is not available
	}

	key := r.getCartCacheKey(userID)
	data, err := r.redisManager.Get(key)
	if err != nil {
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "nil") {
			return nil, nil // Cache miss
		}
		r.logger.Warn("Failed to get cart from cache", zap.Error(err), zap.String("user_id", userID))
		return nil, nil // Don't fail on cache errors
	}

	var cartCache models.CartCache
	if err := json.Unmarshal([]byte(data), &cartCache); err != nil {
		r.logger.Error("Failed to unmarshal cart cache", zap.Error(err), zap.String("user_id", userID))
		// Clear corrupted cache
		r.DeleteCartFromCache(ctx, userID)
		return nil, nil
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
		r.logger.Error("Failed to marshal cart cache", zap.Error(err))
		return nil // Don't fail on cache errors
	}

	if err := r.redisManager.Set(key, data, r.cacheExpiry); err != nil {
		r.logger.Error("Failed to save cart cache", zap.Error(err))
		return nil // Don't fail on cache errors
	}

	return nil
}

func (r *CartRepositoryImpl) DeleteCartFromCache(ctx context.Context, userID string) error {
	if r.redisManager == nil {
		return nil
	}

	key := r.getCartCacheKey(userID)
	if err := r.redisManager.Del(key); err != nil {
		r.logger.Warn("Failed to delete cart cache", zap.Error(err))
	}
	return nil
}

// Database Operations
func (r *CartRepositoryImpl) GetUserCart(ctx context.Context, userID string) (*models.Cart, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID cannot be empty")
	}

	// Always get from database for consistency
	dbCart, err := r.getUserCartFromDB(ctx, userID)
	if err != nil {
		r.logger.Error("Failed to get cart from database", zap.Error(err), zap.String("user_id", userID))
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
		return nil, fmt.Errorf("failed to query carts: %w", err)
	}

	var cart *models.Cart
	if len(carts) == 0 {
		// Create new cart - this user has no cart yet
		cart = r.createNewCartForUser(userID)
		if err := r.CreateCart(ctx, cart); err != nil {
			return nil, fmt.Errorf("failed to create new cart: %w", err)
		}
		r.logger.Info("Created new cart for user", zap.String("user_id", userID), zap.String("cart_id", cart.ID))
	} else if len(carts) == 1 {
		// Perfect: user has exactly one cart
		cart = carts[0]
		// Validate that the cart has a valid ID
		if cart.ID == "" {
			r.logger.Error("Cart retrieved from database has empty ID", zap.String("user_id", userID))
			// Create a new cart instead of trying to fix the broken one
			cart = r.createNewCartForUser(userID)
			if err := r.CreateCart(ctx, cart); err != nil {
				return nil, fmt.Errorf("failed to create new cart after fixing invalid cart: %w", err)
			}
		}
	} else {
		// ERROR: User has multiple carts - this should not happen!
		r.logger.Error("User has multiple carts - fixing by keeping the most recent one",
			zap.String("user_id", userID),
			zap.Int("cart_count", len(carts)))

		// Find the most recently updated cart
		var mostRecentCart *models.Cart
		for _, c := range carts {
			if mostRecentCart == nil || c.UpdatedAt.After(mostRecentCart.UpdatedAt) {
				mostRecentCart = c
			}
		}
		cart = mostRecentCart

		// Delete the other carts to maintain one cart per user rule
		go func() {
			for _, c := range carts {
				if c.ID != cart.ID {
					r.logger.Info("Deleting duplicate cart",
						zap.String("user_id", userID),
						zap.String("cart_id", c.ID))
					if err := r.DeleteCart(context.Background(), c.ID, userID); err != nil {
						r.logger.Error("Failed to delete duplicate cart",
							zap.Error(err),
							zap.String("cart_id", c.ID))
					}
				}
			}
		}()
	}

	// Load cart items only if cart has valid ID
	if cart.ID != "" {
		if err := r.loadCartItems(ctx, cart); err != nil {
			r.logger.Error("Failed to load cart items", zap.Error(err), zap.String("cart_id", cart.ID))
			// Continue without items rather than failing
			cart.Items = []models.CartItem{}
		}
	}

	// Ensure totals are calculated
	r.calculateCartTotals(cart)

	return cart, nil
}

func (r *CartRepositoryImpl) createNewCartForUser(userID string) *models.Cart {
	cart := models.NewCart(userID)
	return cart
}

func (r *CartRepositoryImpl) loadCartItems(ctx context.Context, cart *models.Cart) error {
	if cart.ID == "" {
		return fmt.Errorf("cart ID is empty")
	}

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
		return fmt.Errorf("failed to load cart items: %w", err)
	}

	// Convert to slice of structs
	cart.Items = make([]models.CartItem, len(cartItems))
	for i, item := range cartItems {
		if item != nil {
			cart.Items[i] = *item
		}
	}

	return nil
}

func (r *CartRepositoryImpl) CreateCart(ctx context.Context, cart *models.Cart) error {
	if cart == nil {
		return fmt.Errorf("cart cannot be nil")
	}

	if err := r.dbManager.Create(ctx, cart); err != nil {
		return fmt.Errorf("failed to create cart: %w", err)
	}

	r.logger.Info("Cart created successfully", zap.String("cart_id", cart.ID), zap.String("user_id", cart.UserID))
	return nil
}

func (r *CartRepositoryImpl) UpdateCart(ctx context.Context, cart *models.Cart) error {
	if cart == nil {
		return fmt.Errorf("cart cannot be nil")
	}

	// Update cart totals
	r.calculateCartTotals(cart)

	if err := r.dbManager.Update(ctx, cart); err != nil {
		return fmt.Errorf("failed to update cart: %w", err)
	}

	// Update cache asynchronously
	go func() {
		if cartCache := r.convertCartToCache(cart); cartCache != nil {
			r.SaveCartToCache(context.Background(), cartCache)
		}
	}()

	return nil
}

func (r *CartRepositoryImpl) DeleteCart(ctx context.Context, cartID, userID string) error {
	if cartID == "" || userID == "" {
		return fmt.Errorf("cart ID and user ID cannot be empty")
	}

	cart := &models.Cart{}
	if err := r.dbManager.SoftDelete(ctx, cartID, cart, userID); err != nil {
		return fmt.Errorf("failed to delete cart: %w", err)
	}

	// Clear cache
	go func() {
		r.DeleteCartFromCache(context.Background(), userID)
	}()

	return nil
}

// Cart Item Operations
func (r *CartRepositoryImpl) AddItemToCart(ctx context.Context, userID, productID string, quantity int, unitPrice float64) (*models.Cart, error) {
	if userID == "" || productID == "" {
		return nil, fmt.Errorf("user ID and product ID cannot be empty")
	}

	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	if unitPrice < 0 {
		return nil, fmt.Errorf("unit price cannot be negative")
	}

	// Acquire lock to prevent concurrent modifications
	if err := r.acquireLock(ctx, userID); err != nil {
		r.logger.Warn("Failed to acquire cart lock", zap.Error(err), zap.String("user_id", userID))
		// Continue without lock if Redis is not available
	} else {
		defer r.releaseLock(ctx, userID)
	}

	cart, err := r.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart: %w", err)
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
			return nil, fmt.Errorf("failed to update existing cart item: %w", err)
		}
	} else {
		// Create new cart item
		cartItem := models.NewCartItem(cart.ID, productID, quantity, unitPrice)
		if cartItem == nil {
			return nil, fmt.Errorf("failed to create cart item")
		}

		if err := r.dbManager.Create(ctx, cartItem); err != nil {
			return nil, fmt.Errorf("failed to create cart item: %w", err)
		}

		cart.Items = append(cart.Items, *cartItem)
	}

	// Update cart totals and save
	if err := r.UpdateCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to update cart: %w", err)
	}

	return cart, nil
}

func (r *CartRepositoryImpl) UpdateCartItem(ctx context.Context, userID, productID string, quantity int) (*models.Cart, error) {
	if userID == "" || productID == "" {
		return nil, fmt.Errorf("user ID and product ID cannot be empty")
	}

	if err := r.acquireLock(ctx, userID); err != nil {
		r.logger.Warn("Failed to acquire cart lock", zap.Error(err))
	} else {
		defer r.releaseLock(ctx, userID)
	}

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
		return nil, fmt.Errorf("failed to update cart item: %w", err)
	}

	// Update cart
	if err := r.UpdateCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (r *CartRepositoryImpl) RemoveCartItem(ctx context.Context, userID, productID string) (*models.Cart, error) {
	if userID == "" || productID == "" {
		return nil, fmt.Errorf("user ID and product ID cannot be empty")
	}

	if err := r.acquireLock(ctx, userID); err != nil {
		r.logger.Warn("Failed to acquire cart lock", zap.Error(err))
	} else {
		defer r.releaseLock(ctx, userID)
	}

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

	// Hard delete from database (cart items don't need to be preserved)
	cartItemToDelete := &models.CartItem{}
	if err := r.dbManager.Delete(ctx, itemToRemove.ID, cartItemToDelete); err != nil {
		r.logger.Error("Failed to delete cart item", zap.Error(err))
		return nil, fmt.Errorf("failed to delete cart item: %w", err)
	}

	cart.Items = newItems

	// Update cart
	if err := r.UpdateCart(ctx, cart); err != nil {
		return nil, err
	}

	return cart, nil
}

func (r *CartRepositoryImpl) ClearUserCart(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID cannot be empty")
	}

	if err := r.acquireLock(ctx, userID); err != nil {
		r.logger.Warn("Failed to acquire cart lock", zap.Error(err))
	} else {
		defer r.releaseLock(ctx, userID)
	}

	cart, err := r.GetUserCart(ctx, userID)
	if err != nil {
		return err
	}

	// Hard delete all cart items (cart items don't need to be preserved)
	for _, item := range cart.Items {
		cartItemToDelete := &models.CartItem{}
		if err := r.dbManager.Delete(ctx, item.ID, cartItemToDelete); err != nil {
			r.logger.Error("Failed to delete cart item", zap.Error(err), zap.String("item_id", item.ID))
			return fmt.Errorf("failed to delete cart item %s: %w", item.ID, err)
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
	if cart == nil {
		return
	}

	cart.TotalItems = 0
	cart.TotalPrice = 0

	for _, item := range cart.Items {
		cart.TotalItems += item.Quantity
		cart.TotalPrice += item.TotalPrice
	}

	// Clamp discount to non-negative and not exceeding total
	if cart.DiscountAmount < 0 {
		cart.DiscountAmount = 0
	}
	if cart.DiscountAmount > cart.TotalPrice {
		cart.DiscountAmount = cart.TotalPrice
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
		if item.Quantity <= 0 {
			issues = append(issues, fmt.Sprintf("Item %s has invalid quantity", item.ProductID))
		}
		if item.UnitPrice <= 0 {
			issues = append(issues, fmt.Sprintf("Item %s has invalid price", item.ProductID))
		}
	}

	return issues, nil
}

// CleanupDuplicateCarts removes duplicate carts for all users, keeping only the most recent one
func (r *CartRepositoryImpl) CleanupDuplicateCarts(ctx context.Context) error {
	r.logger.Info("Starting cleanup of duplicate carts")

	// Get all carts grouped by user_id
	var allCarts []*models.Cart
	if err := r.dbManager.List(ctx, &base.Filter{}, &allCarts); err != nil {
		return fmt.Errorf("failed to fetch all carts: %w", err)
	}

	// Group carts by user_id
	userCarts := make(map[string][]*models.Cart)
	for _, cart := range allCarts {
		userCarts[cart.UserID] = append(userCarts[cart.UserID], cart)
	}

	duplicateCount := 0
	cleanedUsers := 0

	// Process each user's carts
	for userID, carts := range userCarts {
		if len(carts) > 1 {
			// User has duplicate carts - keep the most recent one
			cleanedUsers++
			duplicateCount += len(carts) - 1

			r.logger.Warn("Found user with multiple carts",
				zap.String("user_id", userID),
				zap.Int("cart_count", len(carts)))

			// Find the most recently updated cart
			var mostRecentCart *models.Cart
			for _, cart := range carts {
				if mostRecentCart == nil || cart.UpdatedAt.After(mostRecentCart.UpdatedAt) {
					mostRecentCart = cart
				}
			}

			// Delete all other carts
			for _, cart := range carts {
				if cart.ID != mostRecentCart.ID {
					r.logger.Info("Deleting duplicate cart",
						zap.String("user_id", userID),
						zap.String("cart_id", cart.ID),
						zap.String("keeping_cart_id", mostRecentCart.ID))

					if err := r.DeleteCart(ctx, cart.ID, userID); err != nil {
						r.logger.Error("Failed to delete duplicate cart",
							zap.Error(err),
							zap.String("user_id", userID),
							zap.String("cart_id", cart.ID))
						// Continue with other carts rather than failing
					}
				}
			}
		}
	}

	r.logger.Info("Duplicate cart cleanup completed",
		zap.Int("users_cleaned", cleanedUsers),
		zap.Int("duplicate_carts_removed", duplicateCount),
		zap.Int("total_users_checked", len(userCarts)))

	return nil
}
