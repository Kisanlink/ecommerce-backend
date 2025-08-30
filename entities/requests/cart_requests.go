package requests

// AddToCartRequest represents the request to add an item to the cart
type AddToCartRequest struct {
	ProductID string  `json:"product_id" binding:"required"`
	Quantity  int     `json:"quantity" binding:"required,min=1"`
	UnitPrice float64 `json:"unit_price,omitempty"` // Optional, will be fetched from product if not provided
}

// UpdateCartItemRequest represents the request to update a cart item
type UpdateCartItemRequest struct {
	Quantity int `json:"quantity" binding:"required,min=0"` // 0 quantity means remove item
}

// IncrementCartItemRequest represents the request to increment cart item quantity
type IncrementCartItemRequest struct {
	Amount int `json:"amount,omitempty"` // Default to 1 if not provided
}

// DecrementCartItemRequest represents the request to decrement cart item quantity
type DecrementCartItemRequest struct {
	Amount int `json:"amount,omitempty"` // Default to 1 if not provided
}

// RemoveFromCartRequest represents the request to remove an item from the cart
type RemoveFromCartRequest struct {
	ProductID string `json:"product_id" binding:"required"`
}

// ClearCartRequest represents the request to clear the entire cart
type ClearCartRequest struct {
	UserID string `json:"user_id" binding:"required"`
}
