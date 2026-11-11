package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/ecommerce-backend/internal/utils"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"go.uber.org/zap"
)

// SessionCartServiceInterface defines the contract for session-based cart operations
type SessionCartServiceInterface interface {
	// Session cart operations
	GetSessionCart(ctx context.Context, sessionID string) (*models.Cart, error)
	AddItemToSessionCart(ctx context.Context, sessionID, productID string, quantity int) (*models.Cart, error)
	UpdateSessionCartItem(ctx context.Context, sessionID, productID string, quantity int) (*models.Cart, error)
	RemoveSessionCartItem(ctx context.Context, sessionID, productID string) (*models.Cart, error)
	ClearSessionCart(ctx context.Context, sessionID string) error
	GetSessionCartSummary(ctx context.Context, sessionID string) (*models.CartSummary, error)

	// Coupon operations for session carts
	ApplyCouponToSessionCart(ctx context.Context, sessionID, couponCode string) (*models.Cart, float64, error)
	RemoveCouponFromSessionCart(ctx context.Context, sessionID string) (*models.Cart, error)
	ValidateCouponForSessionCart(ctx context.Context, sessionID, couponCode string) (*models.Cart, float64, error)

	// Cart migration operations
	MigrateSessionCartToUser(ctx context.Context, sessionID, userID string) (*models.Cart, error)
	MergeSessionCartWithUserCart(ctx context.Context, sessionID, userID string) (*models.Cart, error)

	// Utility operations
	GenerateSessionID() string
	ValidateSessionID(sessionID string) bool
}

// SessionCartService implements session-based cart business logic
type SessionCartService struct {
	cartRepo    repositories.CartRepositoryInterface
	productRepo repositories.ProductRepositoryInterface
	couponRepo  repositories.CouponRepositoryInterface
	logger      *zap.Logger
	// In-memory storage for session carts (in production, use Redis).
	// mu guards sessionCarts: Go maps are not safe for concurrent use, and
	// unsynchronized access across Gin goroutines caused "fatal error:
	// concurrent map read and map write" crashes (DoS). All access goes through
	// the loadCart/storeCart/deleteCart helpers below.
	mu           sync.RWMutex
	sessionCarts map[string]*models.Cart
}

// loadCart returns the in-memory cart for a session under a read lock.
func (s *SessionCartService) loadCart(sessionID string) (*models.Cart, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cart, ok := s.sessionCarts[sessionID]
	return cart, ok
}

// storeCart writes the in-memory cart for a session under a write lock.
func (s *SessionCartService) storeCart(sessionID string, cart *models.Cart) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionCarts[sessionID] = cart
}

// deleteCart removes the in-memory cart for a session under a write lock.
func (s *SessionCartService) deleteCart(sessionID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessionCarts, sessionID)
}

// NewSessionCartService creates a new session cart service
func NewSessionCartService(cartRepo repositories.CartRepositoryInterface, productRepo repositories.ProductRepositoryInterface, couponRepo repositories.CouponRepositoryInterface, logger *zap.Logger) SessionCartServiceInterface {
	return &SessionCartService{
		cartRepo:     cartRepo,
		productRepo:  productRepo,
		couponRepo:   couponRepo,
		logger:       logger,
		sessionCarts: make(map[string]*models.Cart),
	}
}

// GenerateSessionID generates a unique session ID for anonymous users
func (s *SessionCartService) GenerateSessionID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		s.logger.Error("Failed to generate session ID", zap.Error(err))
		// Fallback to timestamp-based ID
		return fmt.Sprintf("session_%d", time.Now().UnixNano())
	}
	return "session_" + hex.EncodeToString(bytes)
}

// ValidateSessionID validates if a session ID is properly formatted
func (s *SessionCartService) ValidateSessionID(sessionID string) bool {
	if sessionID == "" {
		return false
	}
	// Check if it starts with "session_" and has proper length
	return len(sessionID) > 8 && len(sessionID) < 50
}

// GetSessionCart retrieves the session cart
func (s *SessionCartService) GetSessionCart(ctx context.Context, sessionID string) (*models.Cart, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}

	// Check if session cart exists in memory
	if cart, exists := s.loadCart(sessionID); exists {
		return cart, nil
	}

	// Create new empty cart for this session
	cart := &models.Cart{
		BaseModel:      base.NewBaseModel("CART", hash.Medium),
		UserID:         sessionID, // Use sessionID as userID for session carts
		TotalItems:     0,
		TotalPrice:     0,
		CouponCode:     "",
		DiscountAmount: 0,
		Items:          []models.CartItem{},
	}

	// Override the generated ID with session ID for session carts
	cart.SetID(sessionID)

	s.storeCart(sessionID, cart)
	s.logger.Info("Created new session cart", zap.String("session_id", sessionID))

	return cart, nil
}

// AddItemToSessionCart adds an item to the session cart
func (s *SessionCartService) AddItemToSessionCart(ctx context.Context, sessionID, productID string, quantity int) (*models.Cart, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}
	if productID == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	if quantity <= 0 {
		return nil, fmt.Errorf("quantity must be positive")
	}

	// Validate product exists and get current price
	var product models.Product
	result, err := s.productRepo.GetByID(ctx, productID, &product)
	if err != nil {
		s.logger.Error("Failed to get product", zap.Error(err), zap.String("product_id", productID))
		return nil, fmt.Errorf("product not found")
	}
	if result == nil {
		return nil, fmt.Errorf("product not found")
	}

	// Check stock availability
	if product.Stock < quantity {
		return nil, fmt.Errorf("insufficient stock: requested %d, available %d", quantity, product.Stock)
	}

	// Get or create session cart
	cart, err := s.GetSessionCart(ctx, sessionID)
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
		existingItem.TotalPrice = float64(existingItem.Quantity) * existingItem.UnitPrice
	} else {
		// Create new cart item
		cartItem := models.NewCartItem(cart.ID, productID, quantity, product.Price)
		if cartItem == nil {
			return nil, fmt.Errorf("failed to create cart item")
		}
		cart.Items = append(cart.Items, *cartItem)
	}

	// Recalculate cart totals
	s.calculateCartTotals(cart)

	// Save the updated cart back to session storage
	s.storeCart(sessionID, cart)

	s.logger.Info("Item added to session cart successfully",
		zap.String("session_id", sessionID),
		zap.String("product_id", productID),
		zap.Int("quantity", quantity))

	return cart, nil
}

// UpdateSessionCartItem updates the quantity of an item in the session cart
func (s *SessionCartService) UpdateSessionCartItem(ctx context.Context, sessionID, productID string, quantity int) (*models.Cart, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}
	if productID == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	if quantity < 0 {
		return nil, fmt.Errorf("quantity cannot be negative")
	}

	// If quantity is 0, remove the item
	if quantity == 0 {
		return s.RemoveSessionCartItem(ctx, sessionID, productID)
	}

	// Get session cart
	cart, err := s.GetSessionCart(ctx, sessionID)
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

	// Validate product exists and check stock
	var product models.Product
	result, err := s.productRepo.GetByID(ctx, productID, &product)
	if err != nil {
		s.logger.Error("Failed to get product", zap.Error(err), zap.String("product_id", productID))
		return nil, fmt.Errorf("product not found")
	}
	if result == nil {
		return nil, fmt.Errorf("product not found")
	}

	if product.Stock < quantity {
		return nil, fmt.Errorf("insufficient stock: requested %d, available %d", quantity, product.Stock)
	}

	// Update quantity
	itemToUpdate.Quantity = quantity
	itemToUpdate.TotalPrice = itemToUpdate.UnitPrice * float64(quantity)

	// Recalculate cart totals
	s.calculateCartTotals(cart)

	// Save the updated cart back to session storage
	s.storeCart(sessionID, cart)

	s.logger.Info("Session cart item updated successfully",
		zap.String("session_id", sessionID),
		zap.String("product_id", productID),
		zap.Int("quantity", quantity))

	return cart, nil
}

// RemoveSessionCartItem removes an item from the session cart
func (s *SessionCartService) RemoveSessionCartItem(ctx context.Context, sessionID, productID string) (*models.Cart, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}
	if productID == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	// Get session cart
	cart, err := s.GetSessionCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Find and remove item
	newItems := make([]models.CartItem, 0, len(cart.Items))
	itemFound := false

	for i := range cart.Items {
		if cart.Items[i].ProductID == productID {
			itemFound = true
		} else {
			newItems = append(newItems, cart.Items[i])
		}
	}

	if !itemFound {
		return nil, fmt.Errorf("cart item not found")
	}

	cart.Items = newItems

	// Recalculate cart totals
	s.calculateCartTotals(cart)

	// Save the updated cart back to session storage
	s.storeCart(sessionID, cart)

	s.logger.Info("Session cart item removed successfully",
		zap.String("session_id", sessionID),
		zap.String("product_id", productID))

	return cart, nil
}

// ClearSessionCart removes all items from the session cart
func (s *SessionCartService) ClearSessionCart(ctx context.Context, sessionID string) error {
	if !s.ValidateSessionID(sessionID) {
		return fmt.Errorf("invalid session ID")
	}

	// Get session cart
	cart, err := s.GetSessionCart(ctx, sessionID)
	if err != nil {
		return err
	}

	// Clear items
	cart.Items = []models.CartItem{}
	cart.TotalItems = 0
	cart.TotalPrice = 0

	// Save the updated cart back to session storage
	s.storeCart(sessionID, cart)

	s.logger.Info("Session cart cleared successfully", zap.String("session_id", sessionID))
	return nil
}

// GetSessionCartSummary retrieves the session cart summary
func (s *SessionCartService) GetSessionCartSummary(ctx context.Context, sessionID string) (*models.CartSummary, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}

	cart, err := s.GetSessionCart(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session cart summary: %w", err)
	}

	discounted := cart.TotalPrice - cart.DiscountAmount
	if discounted < 0 {
		discounted = 0
	}
	summary := &models.CartSummary{
		UserID:          cart.UserID,
		TotalItems:      cart.TotalItems,
		TotalPrice:      cart.TotalPrice,
		DiscountAmount:  cart.DiscountAmount,
		DiscountedTotal: discounted,
		ItemsCount:      len(cart.Items),
		LastUpdated:     time.Now(),
		IsFromCache:     false,
	}

	return summary, nil
}

// MigrateSessionCartToUser migrates a session cart to a user cart (replaces user's existing cart)
func (s *SessionCartService) MigrateSessionCartToUser(ctx context.Context, sessionID, userID string) (*models.Cart, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Get session cart
	sessionCart, err := s.GetSessionCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// If session cart is empty, just return the user's existing cart
	if len(sessionCart.Items) == 0 {
		return s.cartRepo.GetUserCart(ctx, userID)
	}

	// Clear user's existing cart
	if err := s.cartRepo.ClearUserCart(ctx, userID); err != nil {
		s.logger.Warn("Failed to clear user's existing cart", zap.Error(err), zap.String("user_id", userID))
	}

	// Create new cart for user with session cart items
	userCart := models.NewCart(userID)
	if userCart == nil {
		return nil, fmt.Errorf("failed to create user cart")
	}

	// Add all session cart items to user cart
	for _, item := range sessionCart.Items {
		// Validate product still exists and get current price
		var product models.Product
		result, err := s.productRepo.GetByID(ctx, item.ProductID, &product)
		if err != nil || result == nil {
			s.logger.Warn("Product no longer exists during migration", zap.String("product_id", item.ProductID))
			continue
		}

		// Check stock availability
		if product.Stock < item.Quantity {
			s.logger.Warn("Insufficient stock during migration",
				zap.String("product_id", item.ProductID),
				zap.Int("requested", item.Quantity),
				zap.Int("available", product.Stock))
			// Adjust quantity to available stock
			item.Quantity = product.Stock
			if item.Quantity <= 0 {
				continue // Skip this item if no stock available
			}
		}

		// Update price to current price
		item.UnitPrice = product.Price
		item.TotalPrice = item.UnitPrice * float64(item.Quantity)

		// Add item to user cart using repository
		_, err = s.cartRepo.AddItemToCart(ctx, userID, item.ProductID, item.Quantity, item.UnitPrice)
		if err != nil {
			s.logger.Error("Failed to add item to user cart during migration",
				zap.Error(err),
				zap.String("product_id", item.ProductID))
			continue
		}
	}

	// Clear session cart
	s.deleteCart(sessionID)

	// Get the final user cart
	finalCart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get final user cart: %w", err)
	}

	s.logger.Info("Session cart migrated to user successfully",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.Int("items_migrated", len(sessionCart.Items)))

	return finalCart, nil
}

// MergeSessionCartWithUserCart merges session cart items with user's existing cart
func (s *SessionCartService) MergeSessionCartWithUserCart(ctx context.Context, sessionID, userID string) (*models.Cart, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	// Get session cart
	sessionCart, err := s.GetSessionCart(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// If session cart is empty, just return the user's existing cart
	if len(sessionCart.Items) == 0 {
		return s.cartRepo.GetUserCart(ctx, userID)
	}

	// Get user's existing cart
	userCart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	// Merge session cart items with user cart
	for _, sessionItem := range sessionCart.Items {
		// Validate product still exists and get current price
		var product models.Product
		result, err := s.productRepo.GetByID(ctx, sessionItem.ProductID, &product)
		if err != nil || result == nil {
			s.logger.Warn("Product no longer exists during merge", zap.String("product_id", sessionItem.ProductID))
			continue
		}

		// Check if item already exists in user cart
		var existingItem *models.CartItem
		for i := range userCart.Items {
			if userCart.Items[i].ProductID == sessionItem.ProductID {
				existingItem = &userCart.Items[i]
				break
			}
		}

		if existingItem != nil {
			// Merge quantities
			newQuantity := existingItem.Quantity + sessionItem.Quantity

			// Check stock availability
			if product.Stock < newQuantity {
				s.logger.Warn("Insufficient stock during merge",
					zap.String("product_id", sessionItem.ProductID),
					zap.Int("requested", newQuantity),
					zap.Int("available", product.Stock))
				newQuantity = product.Stock
			}

			if newQuantity > 0 {
				// Update existing item
				_, err = s.cartRepo.UpdateCartItem(ctx, userID, sessionItem.ProductID, newQuantity)
				if err != nil {
					s.logger.Error("Failed to update item in user cart during merge",
						zap.Error(err),
						zap.String("product_id", sessionItem.ProductID))
				}
			}
		} else {
			// Add new item to user cart
			quantity := sessionItem.Quantity

			// Check stock availability
			if product.Stock < quantity {
				s.logger.Warn("Insufficient stock during merge",
					zap.String("product_id", sessionItem.ProductID),
					zap.Int("requested", quantity),
					zap.Int("available", product.Stock))
				quantity = product.Stock
			}

			if quantity > 0 {
				_, err = s.cartRepo.AddItemToCart(ctx, userID, sessionItem.ProductID, quantity, product.Price)
				if err != nil {
					s.logger.Error("Failed to add item to user cart during merge",
						zap.Error(err),
						zap.String("product_id", sessionItem.ProductID))
				}
			}
		}
	}

	// Clear session cart
	s.deleteCart(sessionID)

	// Get the final user cart
	finalCart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get final user cart: %w", err)
	}

	s.logger.Info("Session cart merged with user cart successfully",
		zap.String("session_id", sessionID),
		zap.String("user_id", userID),
		zap.Int("items_merged", len(sessionCart.Items)))

	return finalCart, nil
}

// Helper method to calculate cart totals
func (s *SessionCartService) calculateCartTotals(cart *models.Cart) {
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

// ApplyCouponToSessionCart applies a coupon to a session cart
func (s *SessionCartService) ApplyCouponToSessionCart(ctx context.Context, sessionID, couponCode string) (*models.Cart, float64, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, 0, fmt.Errorf("invalid session ID")
	}

	code := strings.TrimSpace(strings.ToUpper(couponCode))

	// Get coupon with rules from database
	coupon, err := s.couponRepo.GetByCodeWithRules(ctx, code)
	if err != nil {
		s.logger.Error("Invalid coupon code",
			zap.String("code", code),
			zap.String("session_id", sessionID),
			zap.Error(err))
		return nil, 0, fmt.Errorf("invalid coupon code")
	}

	// Get session cart
	cart, err := s.GetSessionCart(ctx, sessionID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get session cart: %w", err)
	}
	if cart == nil || len(cart.Items) == 0 || cart.TotalPrice <= 0 {
		return nil, 0, fmt.Errorf("cart is empty")
	}

	// For session carts, we'll assume first-time user (simplified)
	isFirstTime := true

	// Extract product IDs from cart items
	cartProductIDs := make([]string, len(cart.Items))
	for i, item := range cart.Items {
		cartProductIDs[i] = item.ProductID
	}

	// Validate coupon with product restrictions
	if !coupon.CanUseWithItems(cart.TotalPrice, sessionID, isFirstTime, cartProductIDs) {
		s.logger.Error("Coupon cannot be used",
			zap.String("code", code),
			zap.String("session_id", sessionID),
			zap.Float64("cart_total", cart.TotalPrice),
			zap.Strings("cart_products", cartProductIDs))
		return nil, 0, fmt.Errorf("coupon cannot be used for this cart")
	}

	// Get the best rule for this cart with items
	bestRule := coupon.GetBestRuleWithItems(cart.TotalPrice, sessionID, isFirstTime, cartProductIDs)
	if bestRule == nil {
		s.logger.Error("No applicable rules found for coupon",
			zap.String("code", code),
			zap.String("session_id", sessionID),
			zap.Float64("cart_total", cart.TotalPrice),
			zap.Strings("cart_products", cartProductIDs))
		return nil, 0, fmt.Errorf("coupon cannot be used for this cart")
	}

	// Calculate discount using the best rule
	discount := bestRule.CalculateDiscount(cart.TotalPrice)
	discount = utils.RoundToTwoDecimals(discount)

	// Update cart with coupon
	cart.CouponCode = code
	cart.DiscountAmount = discount

	// Save the updated cart back to session storage
	s.storeCart(sessionID, cart)

	s.logger.Info("Coupon applied to session cart successfully",
		zap.String("code", code),
		zap.String("session_id", sessionID),
		zap.Float64("discount", discount),
		zap.String("rule_type", string(bestRule.RuleType)))

	return cart, discount, nil
}

// RemoveCouponFromSessionCart removes a coupon from a session cart
func (s *SessionCartService) RemoveCouponFromSessionCart(ctx context.Context, sessionID string) (*models.Cart, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, fmt.Errorf("invalid session ID")
	}

	cart, err := s.GetSessionCart(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get session cart: %w", err)
	}

	cart.CouponCode = ""
	cart.DiscountAmount = 0

	// Save the updated cart back to session storage
	s.storeCart(sessionID, cart)

	s.logger.Info("Coupon removed from session cart successfully",
		zap.String("session_id", sessionID))

	return cart, nil
}

// ValidateCouponForSessionCart validates a coupon for a session cart without applying it
func (s *SessionCartService) ValidateCouponForSessionCart(ctx context.Context, sessionID, couponCode string) (*models.Cart, float64, error) {
	if !s.ValidateSessionID(sessionID) {
		return nil, 0, fmt.Errorf("invalid session ID")
	}

	code := strings.TrimSpace(strings.ToUpper(couponCode))

	// Get coupon with rules from database
	coupon, err := s.couponRepo.GetByCodeWithRules(ctx, code)
	if err != nil {
		s.logger.Error("Invalid coupon code for validation",
			zap.String("code", code),
			zap.String("session_id", sessionID),
			zap.Error(err))
		return nil, 0, fmt.Errorf("invalid coupon code")
	}

	// Get session cart
	cart, err := s.GetSessionCart(ctx, sessionID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get session cart: %w", err)
	}
	if cart == nil || len(cart.Items) == 0 || cart.TotalPrice <= 0 {
		return nil, 0, fmt.Errorf("cart is empty")
	}

	// For session carts, we'll assume first-time user (simplified)
	isFirstTime := true

	// Extract product IDs from cart items
	cartProductIDs := make([]string, len(cart.Items))
	for i, item := range cart.Items {
		cartProductIDs[i] = item.ProductID
	}

	// Validate coupon with product restrictions
	if !coupon.CanUseWithItems(cart.TotalPrice, sessionID, isFirstTime, cartProductIDs) {
		s.logger.Error("Coupon cannot be used for validation",
			zap.String("code", code),
			zap.String("session_id", sessionID),
			zap.Float64("cart_total", cart.TotalPrice),
			zap.Strings("cart_products", cartProductIDs))
		return nil, 0, fmt.Errorf("coupon cannot be used for this cart")
	}

	// Get the best rule for this cart with items
	bestRule := coupon.GetBestRuleWithItems(cart.TotalPrice, sessionID, isFirstTime, cartProductIDs)
	if bestRule == nil {
		s.logger.Error("No applicable rules found for coupon validation",
			zap.String("code", code),
			zap.String("session_id", sessionID),
			zap.Float64("cart_total", cart.TotalPrice),
			zap.Strings("cart_products", cartProductIDs))
		return nil, 0, fmt.Errorf("coupon cannot be used for this cart")
	}

	// Calculate discount using the best rule
	discount := bestRule.CalculateDiscount(cart.TotalPrice)
	discount = utils.RoundToTwoDecimals(discount)

	s.logger.Info("Coupon validated for session cart successfully",
		zap.String("code", code),
		zap.String("session_id", sessionID),
		zap.Float64("discount", discount),
		zap.String("rule_type", string(bestRule.RuleType)))

	return cart, discount, nil
}
