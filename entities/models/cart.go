package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Cart represents a shopping cart
type Cart struct {
	*base.BaseModel `gorm:"embedded"`
	UserID          string  `json:"user_id" gorm:"not null;index"`
	TotalItems      int     `json:"total_items" gorm:"default:0"`
	TotalPrice      float64 `json:"total_price" gorm:"default:0"`
}

// CartItem represents an item in a shopping cart
type CartItem struct {
	*base.BaseModel `gorm:"embedded"`
	CartID          string  `json:"cart_id" gorm:"not null;index"`
	ProductID       string  `json:"product_id" gorm:"not null;index"`
	Quantity        int     `json:"quantity" gorm:"not null"`
	UnitPrice       float64 `json:"unit_price" gorm:"not null"`
	TotalPrice      float64 `json:"total_price" gorm:"not null"`
}

// NewCart creates a new cart with base model
func NewCart(userID string) *Cart {
	baseModel := base.NewBaseModel("CART", hash.Medium)
	return &Cart{
		BaseModel:  baseModel,
		UserID:     userID,
		TotalItems: 0,
		TotalPrice: 0,
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

// TableName returns the table name for GORM
func (c *Cart) TableName() string {
	return "carts"
}

// TableName returns the table name for GORM
func (ci *CartItem) TableName() string {
	return "cart_items"
}
