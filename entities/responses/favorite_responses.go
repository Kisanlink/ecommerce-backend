package responses

import (
	"time"

	"github.com/Kisanlink/ecommerce-backend/entities/models"
)

// FavoriteResponse represents a lightweight favorite response for frontend
type FavoriteResponse struct {
	ProductID string `json:"product_id"`
	Product   struct {
		ID              string  `json:"id"`
		Name            string  `json:"name"`
		Price           float64 `json:"price"`
		ActualPrice     float64 `json:"actual_price"`
		DiscountPercent int     `json:"discount_percent"`
		Rating          float64 `json:"rating"`
		PrimaryImage    string  `json:"primary_image"`
		StockStatus     string  `json:"stock_status"`
		IsAvailable     bool    `json:"is_available"`
	} `json:"product"`
}

// SessionFavoriteResponse represents a lightweight session favorite response
type SessionFavoriteResponse struct {
	SessionID   string              `json:"session_id"`
	Favorites   []*FavoriteResponse `json:"favorites"`
	TotalCount  int                 `json:"total_count"`
	LastUpdated time.Time           `json:"last_updated"`
}

// NewFavoriteResponse creates a new favorite response from a favorite model
func NewFavoriteResponse(favorite *models.Favorite) *FavoriteResponse {
	if favorite == nil || favorite.Product == nil {
		return nil
	}

	response := &FavoriteResponse{
		ProductID: favorite.ProductID,
	}

	// Populate product data
	response.Product.ID = favorite.Product.ID
	response.Product.Name = favorite.Product.Name
	response.Product.Price = favorite.Product.Price
	response.Product.ActualPrice = favorite.Product.ActualPrice
	response.Product.DiscountPercent = favorite.Product.DiscountPercent
	response.Product.Rating = favorite.Product.Rating

	// Set primary image
	if len(favorite.Product.Images) > 0 {
		// Find primary image or use first image
		for _, img := range favorite.Product.Images {
			if img.IsPrimary {
				response.Product.PrimaryImage = img.ImageURL
				break
			}
		}
		// If no primary image found, use first image
		if response.Product.PrimaryImage == "" {
			response.Product.PrimaryImage = favorite.Product.Images[0].ImageURL
		}
	}

	// Set stock status and availability
	if favorite.Product.Inventory != nil {
		response.Product.StockStatus = string(favorite.Product.Inventory.StockStatus)
		response.Product.IsAvailable = favorite.Product.Inventory.CurrentStock > 0
	} else {
		response.Product.StockStatus = "Unknown"
		response.Product.IsAvailable = false
	}

	return response
}

// NewSessionFavoriteResponse creates a new session favorite response
func NewSessionFavoriteResponse(sessionFavorites *models.SessionFavoriteList) *SessionFavoriteResponse {
	if sessionFavorites == nil {
		return &SessionFavoriteResponse{
			Favorites:   []*FavoriteResponse{},
			TotalCount:  0,
			LastUpdated: time.Now(),
		}
	}

	var favoriteResponses []*FavoriteResponse
	for _, fav := range sessionFavorites.Favorites {
		if response := NewFavoriteResponse(&models.Favorite{
			ProductID: fav.ProductID,
			Product:   fav.Product,
		}); response != nil {
			favoriteResponses = append(favoriteResponses, response)
		}
	}

	return &SessionFavoriteResponse{
		SessionID:   sessionFavorites.SessionID,
		Favorites:   favoriteResponses,
		TotalCount:  len(favoriteResponses),
		LastUpdated: sessionFavorites.LastUpdated,
	}
}

// NewFavoritesListResponse creates a response for a list of favorites
func NewFavoritesListResponse(favorites []*models.Favorite) []*FavoriteResponse {
	var responses []*FavoriteResponse
	for _, fav := range favorites {
		if response := NewFavoriteResponse(fav); response != nil {
			responses = append(responses, response)
		}
	}
	return responses
}
