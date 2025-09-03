package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Cart represents a shopping cart
type Cart struct {
	*base.BaseModel `gorm:"embedded"`
	UserID          string  `json:"user_id" gorm:"not null;uniqueIndex" validate:"required"`
	TotalItems      int     `json:"total_items" gorm:"default:0" validate:"min=0"`
	TotalPrice      float64 `json:"total_price" gorm:"default:0" validate:"min=0"`
	// Coupon fields (persisted)
	CouponCode     string     `json:"coupon_code,omitempty" gorm:"type:varchar(50)"`
	DiscountAmount float64    `json:"discount_amount" gorm:"default:0" validate:"min=0"`
	Items          []CartItem `json:"items,omitempty" gorm:"foreignKey:CartID;references:ID"`
}

// CartItem represents an item in a shopping cart
type CartItem struct {
	*base.BaseModel `gorm:"embedded"`
	CartID          string   `json:"cart_id" gorm:"not null;index" validate:"required"`
	ProductID       string   `json:"product_id" gorm:"not null;index" validate:"required"`
	Product         *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
	Quantity        int      `json:"quantity" gorm:"not null" validate:"required,min=1"`
	UnitPrice       float64  `json:"unit_price" gorm:"not null" validate:"required,min=0"`
	TotalPrice      float64  `json:"total_price" gorm:"not null" validate:"required,min=0"`
}

// CartCache represents cached cart data in Redis
type CartCache struct {
	UserID     string          `json:"user_id" validate:"required"`
	TotalItems int             `json:"total_items" validate:"min=0"`
	TotalPrice float64         `json:"total_price" validate:"min=0"`
	Items      []CartItemCache `json:"items"`
	UpdatedAt  time.Time       `json:"updated_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

// CartItemCache represents cached cart item data in Redis
type CartItemCache struct {
	ProductID  string  `json:"product_id" validate:"required"`
	Quantity   int     `json:"quantity" validate:"required,min=1"`
	UnitPrice  float64 `json:"unit_price" validate:"required,min=0"`
	TotalPrice float64 `json:"total_price" validate:"required,min=0"`
}

// CartSummary represents cart summary for quick display
type CartSummary struct {
	UserID          string    `json:"user_id" validate:"required"`
	TotalItems      int       `json:"total_items" validate:"min=0"`
	TotalPrice      float64   `json:"total_price" validate:"min=0"`
	DiscountAmount  float64   `json:"discount_amount" validate:"min=0"`
	DiscountedTotal float64   `json:"discounted_total" validate:"min=0"`
	ItemsCount      int       `json:"items_count" validate:"min=0"`
	LastUpdated     time.Time `json:"last_updated"`
	HasExpired      bool      `json:"has_expired"`
	ExpiresAt       time.Time `json:"expires_at,omitempty"`
	IsFromCache     bool      `json:"is_from_cache"`
}

// CartValidation represents cart validation results
type CartValidation struct {
	IsValid          bool                  `json:"is_valid"`
	TotalItems       int                   `json:"total_items" validate:"min=0"`
	TotalPrice       float64               `json:"total_price" validate:"min=0"`
	InvalidItems     []CartValidationItem  `json:"invalid_items,omitempty"`
	UnavailableItems []CartValidationItem  `json:"unavailable_items,omitempty"`
	PriceChanges     []CartPriceChangeItem `json:"price_changes,omitempty"`
	Errors           []string              `json:"errors,omitempty"`
}

// CartValidationItem represents validation issues with cart items
type CartValidationItem struct {
	ProductID    string `json:"product_id" validate:"required"`
	ProductName  string `json:"product_name"`
	Reason       string `json:"reason" validate:"required"`
	CurrentStock int    `json:"current_stock,omitempty" validate:"min=0"`
	RequestedQty int    `json:"requested_qty,omitempty" validate:"min=0"`
}

// CartPriceChangeItem represents price changes in cart items
type CartPriceChangeItem struct {
	ProductID   string  `json:"product_id" validate:"required"`
	ProductName string  `json:"product_name"`
	OldPrice    float64 `json:"old_price" validate:"min=0"`
	NewPrice    float64 `json:"new_price" validate:"min=0"`
	Quantity    int     `json:"quantity" validate:"min=1"`
	OldTotal    float64 `json:"old_total" validate:"min=0"`
	NewTotal    float64 `json:"new_total" validate:"min=0"`
}

// Constructors

// NewCart creates a new cart with base model
func NewCart(userID string) *Cart {
	if userID == "" {
		return nil
	}

	baseModel := base.NewBaseModel("CART", hash.Medium)
	if baseModel == nil {
		return nil
	}

	return &Cart{
		BaseModel:      baseModel,
		UserID:         userID,
		TotalItems:     0,
		TotalPrice:     0,
		CouponCode:     "",
		DiscountAmount: 0,
		Items:          []CartItem{},
	}
}

// NewCartItem creates a new cart item with base model
func NewCartItem(cartID, productID string, quantity int, unitPrice float64) *CartItem {
	if cartID == "" || productID == "" || quantity <= 0 || unitPrice < 0 {
		return nil
	}

	baseModel := base.NewBaseModel("CITM", hash.Large)
	if baseModel == nil {
		return nil
	}

	return &CartItem{
		BaseModel:  baseModel,
		CartID:     cartID,
		ProductID:  productID,
		Quantity:   quantity,
		UnitPrice:  unitPrice,
		TotalPrice: float64(quantity) * unitPrice,
	}
}

// NewCartCache creates a new cart cache instance
func NewCartCache(userID string, expiryDuration time.Duration) *CartCache {
	if userID == "" {
		return nil
	}

	now := time.Now()
	return &CartCache{
		UserID:     userID,
		TotalItems: 0,
		TotalPrice: 0,
		Items:      []CartItemCache{},
		UpdatedAt:  now,
		ExpiresAt:  now.Add(expiryDuration),
	}
}

// NewCartItemCache creates a new cart item cache instance
func NewCartItemCache(productID string, quantity int, unitPrice float64) *CartItemCache {
	if productID == "" || quantity <= 0 || unitPrice < 0 {
		return nil
	}

	return &CartItemCache{
		ProductID:  productID,
		Quantity:   quantity,
		UnitPrice:  unitPrice,
		TotalPrice: float64(quantity) * unitPrice,
	}
}

// Cart methods

// AddItem adds an item to the cart
func (c *Cart) AddItem(item CartItem) {
	if c == nil {
		return
	}

	// Check if item already exists
	for i, existingItem := range c.Items {
		if existingItem.ProductID == item.ProductID {
			c.Items[i].Quantity += item.Quantity
			c.Items[i].TotalPrice = float64(c.Items[i].Quantity) * c.Items[i].UnitPrice
			c.recalculateTotal()
			return
		}
	}

	// Add new item
	c.Items = append(c.Items, item)
	c.recalculateTotal()
}

// UpdateItem updates an existing item in the cart
func (c *Cart) UpdateItem(productID string, quantity int) bool {
	if c == nil {
		return false
	}

	for i, item := range c.Items {
		if item.ProductID == productID {
			if quantity <= 0 {
				// Remove item
				c.Items = append(c.Items[:i], c.Items[i+1:]...)
			} else {
				// Update quantity
				c.Items[i].Quantity = quantity
				c.Items[i].TotalPrice = float64(quantity) * c.Items[i].UnitPrice
			}
			c.recalculateTotal()
			return true
		}
	}
	return false
}

// RemoveItem removes an item from the cart
func (c *Cart) RemoveItem(productID string) bool {
	if c == nil {
		return false
	}

	for i, item := range c.Items {
		if item.ProductID == productID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.recalculateTotal()
			return true
		}
	}
	return false
}

// Clear removes all items from the cart
func (c *Cart) Clear() {
	if c == nil {
		return
	}

	c.Items = []CartItem{}
	c.TotalItems = 0
	c.TotalPrice = 0
}

// GetItem gets a specific item from the cart
func (c *Cart) GetItem(productID string) (*CartItem, bool) {
	if c == nil {
		return nil, false
	}

	for i, item := range c.Items {
		if item.ProductID == productID {
			return &c.Items[i], true
		}
	}
	return nil, false
}

// HasItem checks if cart contains a specific product
func (c *Cart) HasItem(productID string) bool {
	_, found := c.GetItem(productID)
	return found
}

// IsEmpty checks if the cart is empty
func (c *Cart) IsEmpty() bool {
	return c == nil || len(c.Items) == 0
}

// recalculateTotal recalculates cart totals
func (c *Cart) recalculateTotal() {
	if c == nil {
		return
	}

	totalItems := 0
	totalPrice := 0.0

	for _, item := range c.Items {
		totalItems += item.Quantity
		totalPrice += item.TotalPrice
	}

	c.TotalItems = totalItems
	c.TotalPrice = totalPrice
}

// GetTotalAmount returns the total amount for the cart
func (c *Cart) GetTotalAmount() float64 {
	if c == nil {
		return 0
	}
	return c.TotalPrice
}

// CartCache methods

// AddItem adds an item to cart cache
func (cc *CartCache) AddItem(item CartItemCache) {
	if cc == nil {
		return
	}

	// Check if item already exists
	for i, existingItem := range cc.Items {
		if existingItem.ProductID == item.ProductID {
			// Update existing item
			cc.Items[i].Quantity += item.Quantity
			cc.Items[i].TotalPrice = float64(cc.Items[i].Quantity) * cc.Items[i].UnitPrice
			cc.recalculateTotal()
			cc.UpdatedAt = time.Now()
			return
		}
	}

	// Add new item
	cc.Items = append(cc.Items, item)
	cc.recalculateTotal()
	cc.UpdatedAt = time.Now()
}

// UpdateItem updates an existing item in cart cache
func (cc *CartCache) UpdateItem(productID string, quantity int) bool {
	if cc == nil {
		return false
	}

	for i, item := range cc.Items {
		if item.ProductID == productID {
			if quantity <= 0 {
				// Remove item
				cc.Items = append(cc.Items[:i], cc.Items[i+1:]...)
			} else {
				// Update quantity
				cc.Items[i].Quantity = quantity
				cc.Items[i].TotalPrice = float64(quantity) * cc.Items[i].UnitPrice
			}
			cc.recalculateTotal()
			cc.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// RemoveItem removes an item from cart cache
func (cc *CartCache) RemoveItem(productID string) bool {
	if cc == nil {
		return false
	}

	for i, item := range cc.Items {
		if item.ProductID == productID {
			cc.Items = append(cc.Items[:i], cc.Items[i+1:]...)
			cc.recalculateTotal()
			cc.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// Clear removes all items from cart cache
func (cc *CartCache) Clear() {
	if cc == nil {
		return
	}

	cc.Items = []CartItemCache{}
	cc.TotalItems = 0
	cc.TotalPrice = 0
	cc.UpdatedAt = time.Now()
}

// IsExpired checks if cart cache has expired
func (cc *CartCache) IsExpired() bool {
	if cc == nil {
		return true
	}
	return time.Now().After(cc.ExpiresAt)
}

// GetItem gets a specific item from cart cache
func (cc *CartCache) GetItem(productID string) (CartItemCache, bool) {
	if cc == nil {
		return CartItemCache{}, false
	}

	for _, item := range cc.Items {
		if item.ProductID == productID {
			return item, true
		}
	}
	return CartItemCache{}, false
}

// HasItem checks if cart contains a specific product
func (cc *CartCache) HasItem(productID string) bool {
	_, found := cc.GetItem(productID)
	return found
}

// recalculateTotal recalculates cart cache totals
func (cc *CartCache) recalculateTotal() {
	if cc == nil {
		return
	}

	totalItems := 0
	totalPrice := 0.0

	for _, item := range cc.Items {
		totalItems += item.Quantity
		totalPrice += item.TotalPrice
	}

	cc.TotalItems = totalItems
	cc.TotalPrice = totalPrice
}

// ToSummary converts cart cache to summary
func (cc *CartCache) ToSummary() CartSummary {
	if cc == nil {
		return CartSummary{}
	}

	return CartSummary{
		UserID:          cc.UserID,
		TotalItems:      cc.TotalItems,
		TotalPrice:      cc.TotalPrice,
		DiscountAmount:  0,
		DiscountedTotal: cc.TotalPrice,
		ItemsCount:      len(cc.Items),
		LastUpdated:     cc.UpdatedAt,
		HasExpired:      cc.IsExpired(),
		ExpiresAt:       cc.ExpiresAt,
		IsFromCache:     true,
	}
}

// Table names for GORM

// TableName returns the table name for Cart
func (c *Cart) TableName() string {
	return "carts"
}

// TableName returns the table name for CartItem
func (ci *CartItem) TableName() string {
	return "cart_items"
}
