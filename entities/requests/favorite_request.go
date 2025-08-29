package requests

// AddFavoriteRequest represents the request to add a favorite
type AddFavoriteRequest struct {
	ProductID string `json:"product_id" binding:"required" validate:"required"`
}

// RemoveFavoriteRequest represents the request to remove a favorite
type RemoveFavoriteRequest struct {
	ProductID string `json:"product_id" binding:"required" validate:"required"`
}
