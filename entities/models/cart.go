package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Cart represents a shopping cart
type Cart struct {
	*base.BaseModel `gorm:"embedded"`
	UserID          string     `json:"user_id" gorm:"not null;index"`
	TotalItems      int        `json:"total_items" gorm:"default:0"`
	TotalPrice      float64    `json:"total_price" gorm:"default:0"`
	Items           []CartItem `json:"items,omitempty" gorm:"foreignKey:CartID;references:ID"`
}

// CartItem represents an item in a shopping cart
type CartItem struct {
	*base.BaseModel `gorm:"embedded"`
	CartID          string   `json:"cart_id" gorm:"not null;index"`
	ProductID       string   `json:"product_id" gorm:"not null;index"`
	Product         *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
	Quantity        int      `json:"quantity" gorm:"not null"`
	UnitPrice       float64  `json:"unit_price" gorm:"not null"`
	TotalPrice      float64  `json:"total_price" gorm:"not null"`
}

// CartCache represents cached cart data in Redis
type CartCache struct {
	UserID     string          `json:"user_id"`
	TotalItems int             `json:"total_items"`
	TotalPrice float64         `json:"total_price"`
	Items      []CartItemCache `json:"items"`
	UpdatedAt  time.Time       `json:"updated_at"`
	ExpiresAt  time.Time       `json:"expires_at"`
}

// CartItemCache represents cached cart item data in Redis
type CartItemCache struct {
	ProductID  string  `json:"product_id"`
	Quantity   int     `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
}

// CartSummary represents cart summary for quick display
type CartSummary struct {
	UserID      string    `json:"user_id"`
	TotalItems  int       `json:"total_items"`
	TotalPrice  float64   `json:"total_price"`
	ItemsCount  int       `json:"items_count"`
	LastUpdated time.Time `json:"last_updated"`
	HasExpired  bool      `json:"has_expired"`
	ExpiresAt   time.Time `json:"expires_at,omitempty"`
	IsFromCache bool      `json:"is_from_cache"`
}

// CartValidation represents cart validation results
type CartValidation struct {
	IsValid          bool                  `json:"is_valid"`
	TotalItems       int                   `json:"total_items"`
	TotalPrice       float64               `json:"total_price"`
	InvalidItems     []CartValidationItem  `json:"invalid_items,omitempty"`
	UnavailableItems []CartValidationItem  `json:"unavailable_items,omitempty"`
	PriceChanges     []CartPriceChangeItem `json:"price_changes,omitempty"`
	Errors           []string              `json:"errors,omitempty"`
}

// CartValidationItem represents validation issues with cart items
type CartValidationItem struct {
	ProductID    string `json:"product_id"`
	ProductName  string `json:"product_name"`
	Reason       string `json:"reason"`
	CurrentStock int    `json:"current_stock,omitempty"`
	RequestedQty int    `json:"requested_qty,omitempty"`
}

// CartPriceChangeItem represents price changes in cart items
type CartPriceChangeItem struct {
	ProductID   string  `json:"product_id"`
	ProductName string  `json:"product_name"`
	OldPrice    float64 `json:"old_price"`
	NewPrice    float64 `json:"new_price"`
	Quantity    int     `json:"quantity"`
	OldTotal    float64 `json:"old_total"`
	NewTotal    float64 `json:"new_total"`
}

// NewCart creates a new cart with base model
func NewCart(userID string) *Cart {
	baseModel := base.NewBaseModel("CART", hash.Medium)
	return &Cart{
		BaseModel:  baseModel,
		UserID:     userID,
		TotalItems: 0,
		TotalPrice: 0,
		Items:      []CartItem{},
	}
}

// NewCartItem creates a new cart item with base model
func NewCartItem(cartID, productID string, quantity int, unitPrice float64) *CartItem {
	baseModel := base.NewBaseModel("CITM", hash.Large)
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
func NewCartItemCache(productID string, quantity int, unitPrice float64) CartItemCache {
	return CartItemCache{
		ProductID:  productID,
		Quantity:   quantity,
		UnitPrice:  unitPrice,
		TotalPrice: float64(quantity) * unitPrice,
	}
}

// AddItem adds an item to cart cache
func (cc *CartCache) AddItem(item CartItemCache) {
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
	cc.Items = []CartItemCache{}
	cc.TotalItems = 0
	cc.TotalPrice = 0
	cc.UpdatedAt = time.Now()
}

// IsExpired checks if cart cache has expired
func (cc *CartCache) IsExpired() bool {
	return time.Now().After(cc.ExpiresAt)
}

// GetItem gets a specific item from cart cache
func (cc *CartCache) GetItem(productID string) (CartItemCache, bool) {
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

// recalculateTotal recalculates cart totals
func (cc *CartCache) recalculateTotal() {
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
	return CartSummary{
		UserID:      cc.UserID,
		TotalItems:  cc.TotalItems,
		TotalPrice:  cc.TotalPrice,
		ItemsCount:  len(cc.Items),
		LastUpdated: cc.UpdatedAt,
		HasExpired:  cc.IsExpired(),
		ExpiresAt:   cc.ExpiresAt,
		IsFromCache: true,
	}
}

// TableName returns the table name for GORM
func (c *Cart) TableName() string {
	return "carts"
}

// TableName returns the table name for GORM
func (ci *CartItem) TableName() string {
	return "cart_items"
}
