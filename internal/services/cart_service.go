package services

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
	"github.com/Kisanlink/ecommerce-backend/internal/repositories"
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
	"github.com/Kisanlink/kisanlink-db/pkg/db"
	"go.uber.org/zap"
)

// CartServiceInterface defines the contract for cart service operations
type CartServiceInterface interface {
	// Cart operations
	GetUserCart(ctx context.Context, userID string) (*models.Cart, error)
	AddItemToCart(ctx context.Context, userID, productID string, quantity int) (*models.Cart, error)
	UpdateCartItem(ctx context.Context, userID, productID string, quantity int) (*models.Cart, error)
	IncrementCartItem(ctx context.Context, userID, productID string, amount int) (*models.Cart, error)
	DecrementCartItem(ctx context.Context, userID, productID string, amount int) (*models.Cart, error)
	RemoveCartItem(ctx context.Context, userID, productID string) (*models.Cart, error)
	ClearCart(ctx context.Context, userID string) error
	ClearUserCart(ctx context.Context, userID string) error // Alias for ClearCart
	GetCartSummary(ctx context.Context, userID string) (*models.CartSummary, error)
	ValidateCart(ctx context.Context, userID string) (map[string]interface{}, error)

	// Admin/maintenance operations
	CleanupDuplicateCarts(ctx context.Context) error

	// Coupons (persisted on cart)
	ApplyCoupon(ctx context.Context, userID, couponCode string) (*models.Cart, float64, error)
	RemoveCoupon(ctx context.Context, userID string) (*models.Cart, error)
}

// CartService implements cart business logic
type CartService struct {
	cartRepo    repositories.CartRepositoryInterface
	productRepo repositories.ProductRepositoryInterface
	couponRepo  repositories.CouponRepositoryInterface
	dbManager   db.DBManager
	logger      *zap.Logger
}

// NewCartService creates a new cart service
func NewCartService(cartRepo repositories.CartRepositoryInterface, productRepo repositories.ProductRepositoryInterface, couponRepo repositories.CouponRepositoryInterface, dbManager db.DBManager, logger *zap.Logger) CartServiceInterface {
	service := &CartService{
		cartRepo:    cartRepo,
		productRepo: productRepo,
		couponRepo:  couponRepo,
		dbManager:   dbManager,
		logger:      logger,
	}

	// Initialize ID generators for carts and cart items
	service.initializeIDGenerators()

	return service
}

// initializeIDGenerators initializes the ID generators for carts and cart items from database
func (s *CartService) initializeIDGenerators() {
	ctx := context.Background()

	// Initialize cart ID generator
	s.initializeCartIDGenerator(ctx)

	// Initialize cart item ID generator
	s.initializeCartItemIDGenerator(ctx)
}

// initializeCartIDGenerator initializes the cart ID generator from existing database records
func (s *CartService) initializeCartIDGenerator(ctx context.Context) {
	s.logger.Info("Initializing cart ID generator")

	// Check if cartRepo can be cast to access BaseFilterableRepository
	if cartRepoImpl, ok := s.cartRepo.(*repositories.CartRepositoryImpl); ok {
		// Query all existing cart IDs to prevent conflicts (exclude soft-deleted)
		filter := &base.Filter{
			Group: base.FilterGroup{
				Conditions: []base.FilterCondition{
					{Field: "deleted_at", Operator: base.OpIsNull, Value: nil},
				},
				Logic: base.LogicAnd,
			},
			Page:     1,
			PageSize: 10000,
		}

		carts, err := cartRepoImpl.BaseFilterableRepository.Find(ctx, filter)
		if err != nil {
			s.logger.Error("Failed to get existing carts for ID initialization", zap.Error(err))
			// Fallback to empty initialization
			hash.InitializeGlobalCountersFromDatabase("CART", []string{}, hash.Medium)
		} else {
			// Extract IDs
			existingIDs := make([]string, len(carts))
			for i, cart := range carts {
				if cart != nil {
					existingIDs[i] = cart.GetID()
				}
			}
			s.logger.Info("Found existing cart IDs", zap.Int("count", len(existingIDs)))
			hash.InitializeGlobalCountersFromDatabase("CART", existingIDs, hash.Medium)
		}
	} else {
		s.logger.Warn("Cannot cast cart repository to access base repository, using default initialization")
		hash.InitializeGlobalCountersFromDatabase("CART", []string{}, hash.Medium)
	}
	s.logger.Info("Cart ID generator initialized")
}

// initializeCartItemIDGenerator initializes the cart item ID generator from existing database records
func (s *CartService) initializeCartItemIDGenerator(ctx context.Context) {
	s.logger.Info("Initializing cart item ID generator")

	// Query all existing cart item IDs to prevent conflicts
	filter := &base.Filter{
		Group: base.FilterGroup{
			Conditions: []base.FilterCondition{},
			Logic:      base.LogicAnd,
		},
		Page:     1,
		PageSize: 10000,
	}

	var cartItems []*models.CartItem
	if err := s.dbManager.List(ctx, filter, &cartItems); err != nil {
		s.logger.Error("Failed to get existing cart items for ID initialization", zap.Error(err))
		// Fallback to empty initialization
		hash.InitializeGlobalCountersFromDatabase("CITM", []string{}, hash.Large)
	} else {
		// Extract IDs
		existingIDs := make([]string, 0, len(cartItems))
		for _, item := range cartItems {
			if item != nil && item.GetID() != "" {
				existingIDs = append(existingIDs, item.GetID())
			}
		}
		s.logger.Info("Found existing cart item IDs", zap.Int("count", len(existingIDs)))
		hash.InitializeGlobalCountersFromDatabase("CITM", existingIDs, hash.Large)
	}
	s.logger.Info("Cart item ID generator initialized")
}

// GetUserCart retrieves the user's cart
func (s *CartService) GetUserCart(ctx context.Context, userID string) (*models.Cart, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	cart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user cart", zap.Error(err), zap.String("user_id", userID))
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	return cart, nil
}

// AddItemToCart adds an item to the user's cart
func (s *CartService) AddItemToCart(ctx context.Context, userID, productID string, quantity int) (*models.Cart, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
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

	// Add item to cart
	cart, err := s.cartRepo.AddItemToCart(ctx, userID, productID, quantity, product.Price)
	if err != nil {
		s.logger.Error("Failed to add item to cart",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Int("quantity", quantity))
		return nil, fmt.Errorf("failed to add item to cart: %w", err)
	}

	s.logger.Info("Item added to cart successfully",
		zap.String("user_id", userID),
		zap.String("product_id", productID),
		zap.Int("quantity", quantity))

	return cart, nil
}

// UpdateCartItem updates the quantity of an item in the cart
func (s *CartService) UpdateCartItem(ctx context.Context, userID, productID string, quantity int) (*models.Cart, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if productID == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	if quantity < 0 {
		return nil, fmt.Errorf("quantity cannot be negative")
	}

	// If quantity is 0, remove the item
	if quantity == 0 {
		return s.RemoveCartItem(ctx, userID, productID)
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

	// Update cart item
	cart, err := s.cartRepo.UpdateCartItem(ctx, userID, productID, quantity)
	if err != nil {
		s.logger.Error("Failed to update cart item",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Int("quantity", quantity))
		return nil, fmt.Errorf("failed to update cart item: %w", err)
	}

	s.logger.Info("Cart item updated successfully",
		zap.String("user_id", userID),
		zap.String("product_id", productID),
		zap.Int("quantity", quantity))

	return cart, nil
}

// IncrementCartItem increments the quantity of an item in the cart by the specified amount
func (s *CartService) IncrementCartItem(ctx context.Context, userID, productID string, amount int) (*models.Cart, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if productID == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("increment amount must be positive")
	}

	// Get current cart to find existing item quantity
	cart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	// Find the current item quantity
	var currentQuantity int
	itemFound := false
	if cart != nil {
		for _, item := range cart.Items {
			if item.ProductID == productID {
				currentQuantity = item.Quantity
				itemFound = true
				break
			}
		}
	}

	if !itemFound {
		return nil, fmt.Errorf("item not found in cart")
	}

	newQuantity := currentQuantity + amount

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

	if product.Stock < newQuantity {
		return nil, fmt.Errorf("insufficient stock: requested %d, available %d", newQuantity, product.Stock)
	}

	// Update cart item with new quantity
	cart, err = s.cartRepo.UpdateCartItem(ctx, userID, productID, newQuantity)
	if err != nil {
		s.logger.Error("Failed to increment cart item",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Int("amount", amount))
		return nil, fmt.Errorf("failed to increment cart item: %w", err)
	}

	s.logger.Info("Cart item incremented successfully",
		zap.String("user_id", userID),
		zap.String("product_id", productID),
		zap.Int("amount", amount),
		zap.Int("new_quantity", newQuantity))

	return cart, nil
}

// DecrementCartItem decrements the quantity of an item in the cart by the specified amount
func (s *CartService) DecrementCartItem(ctx context.Context, userID, productID string, amount int) (*models.Cart, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if productID == "" {
		return nil, fmt.Errorf("product ID is required")
	}
	if amount <= 0 {
		return nil, fmt.Errorf("decrement amount must be positive")
	}

	// Get current cart to find existing item quantity
	cart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}

	// Find the current item quantity
	var currentQuantity int
	itemFound := false
	if cart != nil {
		for _, item := range cart.Items {
			if item.ProductID == productID {
				currentQuantity = item.Quantity
				itemFound = true
				break
			}
		}
	}

	if !itemFound {
		return nil, fmt.Errorf("item not found in cart")
	}

	newQuantity := currentQuantity - amount

	// If new quantity would be 0 or negative, remove the item
	if newQuantity <= 0 {
		return s.RemoveCartItem(ctx, userID, productID)
	}

	// Update cart item with new quantity
	cart, err = s.cartRepo.UpdateCartItem(ctx, userID, productID, newQuantity)
	if err != nil {
		s.logger.Error("Failed to decrement cart item",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID),
			zap.Int("amount", amount))
		return nil, fmt.Errorf("failed to decrement cart item: %w", err)
	}

	s.logger.Info("Cart item decremented successfully",
		zap.String("user_id", userID),
		zap.String("product_id", productID),
		zap.Int("amount", amount),
		zap.Int("new_quantity", newQuantity))

	return cart, nil
}

// RemoveCartItem removes an item from the cart
func (s *CartService) RemoveCartItem(ctx context.Context, userID, productID string) (*models.Cart, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	if productID == "" {
		return nil, fmt.Errorf("product ID is required")
	}

	cart, err := s.cartRepo.RemoveCartItem(ctx, userID, productID)
	if err != nil {
		s.logger.Error("Failed to remove cart item",
			zap.Error(err),
			zap.String("user_id", userID),
			zap.String("product_id", productID))
		return nil, fmt.Errorf("failed to remove cart item: %w", err)
	}

	s.logger.Info("Cart item removed successfully",
		zap.String("user_id", userID),
		zap.String("product_id", productID))

	return cart, nil
}

// ClearCart removes all items from the user's cart
func (s *CartService) ClearCart(ctx context.Context, userID string) error {
	if userID == "" {
		return fmt.Errorf("user ID is required")
	}

	if err := s.cartRepo.ClearUserCart(ctx, userID); err != nil {
		s.logger.Error("Failed to clear cart",
			zap.Error(err),
			zap.String("user_id", userID))
		return fmt.Errorf("failed to clear cart: %w", err)
	}

	s.logger.Info("Cart cleared successfully", zap.String("user_id", userID))
	return nil
}

// ClearUserCart is an alias for ClearCart for consistency
func (s *CartService) ClearUserCart(ctx context.Context, userID string) error {
	return s.ClearCart(ctx, userID)
}

// GetCartSummary retrieves the cart summary
func (s *CartService) GetCartSummary(ctx context.Context, userID string) (*models.CartSummary, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	cart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart summary: %w", err)
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
		LastUpdated:     cart.UpdatedAt,
		IsFromCache:     false, // Always from service layer
	}

	return summary, nil
}

// ValidateCart validates all items in the cart
func (s *CartService) ValidateCart(ctx context.Context, userID string) (map[string]interface{}, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}

	cart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to validate cart: %w", err)
	}

	validationIssues := []map[string]interface{}{}
	cartUpdated := false
	validItems := []models.CartItem{}

	for _, item := range cart.Items {
		// Validate product exists
		var product models.Product
		result, err := s.productRepo.GetByID(ctx, item.ProductID, &product)
		if err != nil || result == nil {
			validationIssues = append(validationIssues, map[string]interface{}{
				"product_id": item.ProductID,
				"issue":      "Product no longer exists",
				"action":     "removed_from_cart",
			})
			cartUpdated = true
			continue
		}

		// Check stock availability
		if product.Stock < item.Quantity {
			validationIssues = append(validationIssues, map[string]interface{}{
				"product_id":         item.ProductID,
				"issue":              "Insufficient stock",
				"requested_quantity": item.Quantity,
				"available_quantity": product.Stock,
				"action":             "quantity_adjusted",
			})

			// Adjust quantity to available stock
			if product.Stock > 0 {
				item.Quantity = product.Stock
				item.TotalPrice = item.UnitPrice * float64(item.Quantity)
				validItems = append(validItems, item)
			}
			cartUpdated = true
		} else {
			// Check price changes
			if product.Price != item.UnitPrice {
				validationIssues = append(validationIssues, map[string]interface{}{
					"product_id": item.ProductID,
					"issue":      "Price changed",
					"old_price":  item.UnitPrice,
					"new_price":  product.Price,
					"action":     "price_updated",
				})

				item.UnitPrice = product.Price
				item.TotalPrice = item.UnitPrice * float64(item.Quantity)
				cartUpdated = true
			}

			validItems = append(validItems, item)
		}
	}

	// Update cart if needed
	if cartUpdated {
		cart.Items = validItems
		if err := s.cartRepo.UpdateCart(ctx, cart); err != nil {
			s.logger.Error("Failed to update cart after validation", zap.Error(err))
		}
	}

	result := map[string]interface{}{
		"cart":              cart,
		"validation_issues": validationIssues,
		"cart_updated":      cartUpdated,
	}

	return result, nil
}

// --- Coupons persistence ---
func round2(f float64) float64 { return math.Round(f*100) / 100 }

func (s *CartService) ApplyCoupon(ctx context.Context, userID, couponCode string) (*models.Cart, float64, error) {
	if userID == "" {
		return nil, 0, fmt.Errorf("user ID is required")
	}

	code := strings.TrimSpace(strings.ToUpper(couponCode))

	// Get coupon with rules from database
	coupon, err := s.couponRepo.GetByCodeWithRules(ctx, code)
	if err != nil {
		s.logger.Error("Invalid coupon code",
			zap.String("code", code),
			zap.String("user_id", userID),
			zap.Error(err))
		return nil, 0, fmt.Errorf("invalid coupon code")
	}

	// Get user's cart
	cart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user cart: %w", err)
	}
	if cart == nil || len(cart.Items) == 0 || cart.TotalPrice <= 0 {
		return nil, 0, fmt.Errorf("cart is empty")
	}

	// Check if user is first time (simplified check)
	isFirstTime := true // In a real implementation, check user's order history

	// Validate coupon
	if !coupon.CanUse(cart.TotalPrice, userID, isFirstTime) {
		s.logger.Error("Coupon cannot be used",
			zap.String("code", code),
			zap.String("user_id", userID),
			zap.Float64("cart_total", cart.TotalPrice))
		return nil, 0, fmt.Errorf("coupon cannot be used for this cart")
	}

	// Get the best rule for this cart
	bestRule := coupon.GetBestRule(cart.TotalPrice, userID, isFirstTime)
	if bestRule == nil {
		s.logger.Error("No applicable rules found for coupon",
			zap.String("code", code),
			zap.String("user_id", userID),
			zap.Float64("cart_total", cart.TotalPrice))
		return nil, 0, fmt.Errorf("coupon cannot be used for this cart")
	}

	// Calculate discount using the best rule
	discount := bestRule.CalculateDiscount(cart.TotalPrice)
	discount = round2(discount)

	// Update cart with coupon
	cart.CouponCode = code
	cart.DiscountAmount = discount
	if err := s.cartRepo.UpdateCart(ctx, cart); err != nil {
		return nil, 0, fmt.Errorf("failed to update cart with coupon: %w", err)
	}

	s.logger.Info("Coupon applied successfully",
		zap.String("code", code),
		zap.String("user_id", userID),
		zap.Float64("discount", discount),
		zap.String("rule_type", string(bestRule.RuleType)))

	return cart, discount, nil
}

func (s *CartService) RemoveCoupon(ctx context.Context, userID string) (*models.Cart, error) {
	if userID == "" {
		return nil, fmt.Errorf("user ID is required")
	}
	cart, err := s.cartRepo.GetUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user cart: %w", err)
	}
	cart.CouponCode = ""
	cart.DiscountAmount = 0
	if err := s.cartRepo.UpdateCart(ctx, cart); err != nil {
		return nil, fmt.Errorf("failed to update cart: %w", err)
	}
	return cart, nil
}

// CleanupDuplicateCarts removes duplicate carts ensuring each user has only one cart
func (s *CartService) CleanupDuplicateCarts(ctx context.Context) error {
	s.logger.Info("Starting cart cleanup process to ensure one cart per user")

	if s.cartRepo == nil {
		return fmt.Errorf("cart repository is not available")
	}

	// Call the repository method to clean up duplicates
	if err := s.cartRepo.CleanupDuplicateCarts(ctx); err != nil {
		s.logger.Error("Failed to cleanup duplicate carts", zap.Error(err))
		return fmt.Errorf("cleanup failed: %w", err)
	}

	s.logger.Info("Cart cleanup completed successfully")
	return nil
}
