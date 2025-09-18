package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// CartItemResponse represents a cart item in response
type CartItemResponse struct {
	ID         string           `json:"id"`
	CartID     string           `json:"cart_id"`
	ProductID  string           `json:"product_id"`
	Product    *ProductResponse `json:"product,omitempty"`
	Quantity   int              `json:"quantity"`
	UnitPrice  float64          `json:"unit_price"`
	TotalPrice float64          `json:"total_price"`
	CreatedAt  string           `json:"created_at"`
	UpdatedAt  string           `json:"updated_at"`
}

// CartResponse represents a complete cart response
type CartResponse struct {
	ID             string             `json:"id"`
	UserID         string             `json:"user_id"`
	TotalItems     int                `json:"total_items"`
	TotalPrice     float64            `json:"total_price"`
	CouponCode     string             `json:"coupon_code,omitempty"`
	DiscountAmount float64            `json:"discount_amount"`
	FinalPrice     float64            `json:"final_price"`
	Items          []CartItemResponse `json:"items"`
	ItemsCount     int                `json:"items_count"`
	IsEmpty        bool               `json:"is_empty"`
	CreatedAt      string             `json:"created_at"`
	UpdatedAt      string             `json:"updated_at"`
}

// CartSummaryResponse represents cart summary information
type CartSummaryResponse struct {
	UserID          string    `json:"user_id"`
	TotalItems      int       `json:"total_items"`
	TotalPrice      float64   `json:"total_price"`
	DiscountAmount  float64   `json:"discount_amount"`
	DiscountedTotal float64   `json:"discounted_total"`
	ItemsCount      int       `json:"items_count"`
	LastUpdated     time.Time `json:"last_updated"`
	HasExpired      bool      `json:"has_expired"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	IsFromCache     bool      `json:"is_from_cache"`
	IsEmpty         bool      `json:"is_empty"`
}

// AddToCartResponse represents response after adding item to cart
type AddToCartResponse struct {
	Cart    *CartResponse `json:"cart"`
	Message string        `json:"message"`
}

// UpdateCartResponse represents response after updating cart item
type UpdateCartResponse struct {
	Cart    *CartResponse `json:"cart"`
	Message string        `json:"message"`
}

// RemoveFromCartResponse represents response after removing item from cart
type RemoveFromCartResponse struct {
	Cart    *CartResponse `json:"cart"`
	Message string        `json:"message"`
}

// ClearCartResponse represents response after clearing cart
type ClearCartResponse struct {
	Message string `json:"message"`
}

// CartValidationResponse represents cart validation results
type CartValidationResponse struct {
	IsValid          bool                         `json:"is_valid"`
	TotalItems       int                          `json:"total_items"`
	TotalPrice       float64                      `json:"total_price"`
	InvalidItems     []CartValidationItemResponse `json:"invalid_items,omitempty"`
	UnavailableItems []CartValidationItemResponse `json:"unavailable_items,omitempty"`
	PriceChanges     []CartPriceChangeResponse    `json:"price_changes,omitempty"`
	Errors           []string                     `json:"errors,omitempty"`
}

// CartValidationItemResponse represents validation issues with cart items
type CartValidationItemResponse struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	Reason       string `json:"reason"`
	CurrentStock int    `json:"current_stock,omitempty"`
	RequestedQty int    `json:"requested_qty,omitempty"`
}

// CartPriceChangeResponse represents price changes in cart items
type CartPriceChangeResponse struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	OldPrice    float64 `json:"old_price"`
	NewPrice    float64 `json:"new_price"`
	Quantity    int     `json:"quantity"`
	OldTotal    float64 `json:"old_total"`
	NewTotal    float64 `json:"new_total"`
}

// Conversion functions from models to response structures

// NewCartItemResponse creates a CartItemResponse from CartItem model
func NewCartItemResponse(cartItem *models.CartItem) CartItemResponse {
	response := CartItemResponse{
		ID:         cartItem.ID,
		CartID:     cartItem.CartID,
		ProductID:  cartItem.ProductID,
		Quantity:   cartItem.Quantity,
		UnitPrice:  cartItem.UnitPrice,
		TotalPrice: cartItem.TotalPrice,
		CreatedAt:  cartItem.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  cartItem.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}

	// Add product information if available
	if cartItem.Product != nil {
		productResp := NewProductResponse(cartItem.Product, nil, nil)
		response.Product = &productResp
	}

	return response
}

// NewCartItemResponseWithProduct creates a CartItemResponse with product details
func NewCartItemResponseWithProduct(cartItem *models.CartItem, product *models.Product, images []*models.Image, inventory *models.Inventory) CartItemResponse {
	response := NewCartItemResponse(cartItem)

	// Add detailed product information
	if product != nil {
		productResp := NewProductResponse(product, images, inventory)
		response.Product = &productResp
	}

	return response
}

// NewCartResponse creates a CartResponse from Cart model
func NewCartResponse(cart *models.Cart) CartResponse {
	if cart == nil {
		return CartResponse{}
	}

	// Convert cart items
	items := make([]CartItemResponse, len(cart.Items))
	for i, item := range cart.Items {
		items[i] = NewCartItemResponse(&item)
	}

	// Calculate final price after discount
	finalPrice := cart.TotalPrice - cart.DiscountAmount
	if finalPrice < 0 {
		finalPrice = 0
	}

	return CartResponse{
		ID:             cart.ID,
		UserID:         cart.UserID,
		TotalItems:     cart.TotalItems,
		TotalPrice:     cart.TotalPrice,
		CouponCode:     cart.CouponCode,
		DiscountAmount: cart.DiscountAmount,
		FinalPrice:     finalPrice,
		Items:          items,
		ItemsCount:     len(cart.Items),
		IsEmpty:        len(cart.Items) == 0,
		CreatedAt:      cart.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      cart.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// NewCartResponseWithProducts creates a CartResponse with detailed product information
func NewCartResponseWithProducts(cart *models.Cart, productDetails map[string]*models.Product, imageDetails map[string][]*models.Image, inventoryDetails map[string]*models.Inventory) CartResponse {
	if cart == nil {
		return CartResponse{}
	}

	// Convert cart items with product details
	items := make([]CartItemResponse, len(cart.Items))
	for i, item := range cart.Items {
		product := productDetails[item.ProductID]
		images := imageDetails[item.ProductID]
		inventory := inventoryDetails[item.ProductID]
		items[i] = NewCartItemResponseWithProduct(&item, product, images, inventory)
	}

	// Calculate final price after discount
	finalPrice := cart.TotalPrice - cart.DiscountAmount
	if finalPrice < 0 {
		finalPrice = 0
	}

	return CartResponse{
		ID:             cart.ID,
		UserID:         cart.UserID,
		TotalItems:     cart.TotalItems,
		TotalPrice:     cart.TotalPrice,
		CouponCode:     cart.CouponCode,
		DiscountAmount: cart.DiscountAmount,
		FinalPrice:     finalPrice,
		Items:          items,
		ItemsCount:     len(cart.Items),
		IsEmpty:        len(cart.Items) == 0,
		CreatedAt:      cart.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:      cart.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}

// NewCartSummaryResponse creates a CartSummaryResponse from CartSummary model
func NewCartSummaryResponse(summary *models.CartSummary) CartSummaryResponse {
	if summary == nil {
		return CartSummaryResponse{}
	}

	return CartSummaryResponse{
		UserID:          summary.UserID,
		TotalItems:      summary.TotalItems,
		TotalPrice:      summary.TotalPrice,
		DiscountAmount:  summary.DiscountAmount,
		DiscountedTotal: summary.DiscountedTotal,
		ItemsCount:      summary.ItemsCount,
		LastUpdated:     summary.LastUpdated,
		HasExpired:      summary.HasExpired,
		ExpiresAt:       summary.ExpiresAt,
		IsFromCache:     summary.IsFromCache,
		IsEmpty:         summary.ItemsCount == 0,
	}
}

// NewCartValidationResponse creates a CartValidationResponse from CartValidation model
func NewCartValidationResponse(validation *models.CartValidation) CartValidationResponse {
	if validation == nil {
		return CartValidationResponse{}
	}

	// Convert invalid items
	invalidItems := make([]CartValidationItemResponse, len(validation.InvalidItems))
	for i, item := range validation.InvalidItems {
		invalidItems[i] = CartValidationItemResponse{
			ProductID:    item.ProductID,
			ProductName:  item.ProductName,
			Reason:       item.Reason,
			CurrentStock: item.CurrentStock,
			RequestedQty: item.RequestedQty,
		}
	}

	// Convert unavailable items
	unavailableItems := make([]CartValidationItemResponse, len(validation.UnavailableItems))
	for i, item := range validation.UnavailableItems {
		unavailableItems[i] = CartValidationItemResponse{
			ProductID:    item.ProductID,
			ProductName:  item.ProductName,
			Reason:       item.Reason,
			CurrentStock: item.CurrentStock,
			RequestedQty: item.RequestedQty,
		}
	}

	// Convert price changes
	priceChanges := make([]CartPriceChangeResponse, len(validation.PriceChanges))
	for i, change := range validation.PriceChanges {
		priceChanges[i] = CartPriceChangeResponse{
			ProductID:   change.ProductID,
			ProductName: change.ProductName,
			OldPrice:    change.OldPrice,
			NewPrice:    change.NewPrice,
			Quantity:    change.Quantity,
			OldTotal:    change.OldTotal,
			NewTotal:    change.NewTotal,
		}
	}

	return CartValidationResponse{
		IsValid:          validation.IsValid,
		TotalItems:       validation.TotalItems,
		TotalPrice:       validation.TotalPrice,
		InvalidItems:     invalidItems,
		UnavailableItems: unavailableItems,
		PriceChanges:     priceChanges,
		Errors:           validation.Errors,
	}
}

// Response constructors for different operations

// NewAddToCartResponse creates an AddToCartResponse
func NewAddToCartResponse(cart *models.Cart) AddToCartResponse {
	cartResp := NewCartResponse(cart)
	return AddToCartResponse{
		Cart:    &cartResp,
		Message: "Item added to cart successfully",
	}
}

// NewAddToCartResponseWithProducts creates an AddToCartResponse with product details
func NewAddToCartResponseWithProducts(cart *models.Cart, productDetails map[string]*models.Product, imageDetails map[string][]*models.Image, inventoryDetails map[string]*models.Inventory) AddToCartResponse {
	cartResp := NewCartResponseWithProducts(cart, productDetails, imageDetails, inventoryDetails)
	return AddToCartResponse{
		Cart:    &cartResp,
		Message: "Item added to cart successfully",
	}
}

// NewUpdateCartResponse creates an UpdateCartResponse
func NewUpdateCartResponse(cart *models.Cart) UpdateCartResponse {
	cartResp := NewCartResponse(cart)
	return UpdateCartResponse{
		Cart:    &cartResp,
		Message: "Cart updated successfully",
	}
}

// NewUpdateCartResponseWithProducts creates an UpdateCartResponse with product details
func NewUpdateCartResponseWithProducts(cart *models.Cart, productDetails map[string]*models.Product, imageDetails map[string][]*models.Image, inventoryDetails map[string]*models.Inventory) UpdateCartResponse {
	cartResp := NewCartResponseWithProducts(cart, productDetails, imageDetails, inventoryDetails)
	return UpdateCartResponse{
		Cart:    &cartResp,
		Message: "Cart updated successfully",
	}
}

// NewRemoveFromCartResponse creates a RemoveFromCartResponse
func NewRemoveFromCartResponse(cart *models.Cart) RemoveFromCartResponse {
	cartResp := NewCartResponse(cart)
	return RemoveFromCartResponse{
		Cart:    &cartResp,
		Message: "Item removed from cart successfully",
	}
}

// NewRemoveFromCartResponseWithProducts creates a RemoveFromCartResponse with product details
func NewRemoveFromCartResponseWithProducts(cart *models.Cart, productDetails map[string]*models.Product, imageDetails map[string][]*models.Image, inventoryDetails map[string]*models.Inventory) RemoveFromCartResponse {
	cartResp := NewCartResponseWithProducts(cart, productDetails, imageDetails, inventoryDetails)
	return RemoveFromCartResponse{
		Cart:    &cartResp,
		Message: "Item removed from cart successfully",
	}
}

// NewClearCartResponse creates a ClearCartResponse
func NewClearCartResponse() ClearCartResponse {
	return ClearCartResponse{
		Message: "Cart cleared successfully",
	}
}

// Helper methods for response validation and manipulation

// IsCartEmpty checks if the cart response is empty
func (cr *CartResponse) IsCartEmpty() bool {
	return cr == nil || len(cr.Items) == 0
}

// GetItemByProductID finds a cart item by product ID
func (cr *CartResponse) GetItemByProductID(productID string) (*CartItemResponse, bool) {
	if cr == nil {
		return nil, false
	}

	for i, item := range cr.Items {
		if item.ProductID == productID {
			return &cr.Items[i], true
		}
	}
	return nil, false
}

// HasItem checks if cart contains a specific product
func (cr *CartResponse) HasItem(productID string) bool {
	_, found := cr.GetItemByProductID(productID)
	return found
}

// GetTotalItemsCount returns the total number of individual items in cart
func (cr *CartResponse) GetTotalItemsCount() int {
	if cr == nil {
		return 0
	}
	return cr.TotalItems
}

// GetUniqueItemsCount returns the number of unique products in cart
func (cr *CartResponse) GetUniqueItemsCount() int {
	if cr == nil {
		return 0
	}
	return cr.ItemsCount
}

// CalculateSavings returns the total savings from discounts
func (cr *CartResponse) CalculateSavings() float64 {
	if cr == nil {
		return 0
	}
	return cr.DiscountAmount
}

// GetFinalAmount returns the final amount after all discounts
func (cr *CartResponse) GetFinalAmount() float64 {
	if cr == nil {
		return 0
	}
	return cr.FinalPrice
}
