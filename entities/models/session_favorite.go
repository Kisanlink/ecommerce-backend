package models

import (
	"time"

	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// SessionFavorite represents a favorite item for anonymous users
type SessionFavorite struct {
	*base.BaseModel `gorm:"embedded"`
	SessionID       string    `json:"session_id" gorm:"not null;index"`
	ProductID       string    `json:"product_id" gorm:"not null;index"`
	Product         *Product  `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
	AddedAt         time.Time `json:"added_at"`
}

// SessionFavoriteList represents a collection of session favorites
type SessionFavoriteList struct {
	SessionID   string             `json:"session_id"`
	Favorites   []*SessionFavorite `json:"favorites"`
	TotalCount  int                `json:"total_count"`
	LastUpdated time.Time          `json:"last_updated"`
}

// NewSessionFavorite creates a new session favorite
func NewSessionFavorite(sessionID, productID string) *SessionFavorite {
	baseModel := base.NewBaseModel("SFAV", hash.Medium)
	return &SessionFavorite{
		BaseModel: baseModel,
		SessionID: sessionID,
		ProductID: productID,
		AddedAt:   time.Now(),
	}
}

// NewSessionFavoriteList creates a new session favorite list
func NewSessionFavoriteList(sessionID string) *SessionFavoriteList {
	return &SessionFavoriteList{
		SessionID:   sessionID,
		Favorites:   []*SessionFavorite{},
		TotalCount:  0,
		LastUpdated: time.Now(),
	}
}

// AddFavorite adds a product to the session favorites list
func (sfl *SessionFavoriteList) AddFavorite(productID string) bool {
	if sfl == nil {
		return false
	}

	// Check if already exists
	for _, fav := range sfl.Favorites {
		if fav.ProductID == productID {
			return false // Already exists
		}
	}

	// Add new favorite
	favorite := NewSessionFavorite(sfl.SessionID, productID)
	sfl.Favorites = append(sfl.Favorites, favorite)
	sfl.TotalCount = len(sfl.Favorites)
	sfl.LastUpdated = time.Now()
	return true
}

// RemoveFavorite removes a product from the session favorites list
func (sfl *SessionFavoriteList) RemoveFavorite(productID string) bool {
	if sfl == nil {
		return false
	}

	for i, fav := range sfl.Favorites {
		if fav.ProductID == productID {
			sfl.Favorites = append(sfl.Favorites[:i], sfl.Favorites[i+1:]...)
			sfl.TotalCount = len(sfl.Favorites)
			sfl.LastUpdated = time.Now()
			return true
		}
	}
	return false
}

// HasFavorite checks if a product is in the session favorites
func (sfl *SessionFavoriteList) HasFavorite(productID string) bool {
	if sfl == nil {
		return false
	}

	for _, fav := range sfl.Favorites {
		if fav.ProductID == productID {
			return true
		}
	}
	return false
}

// Clear removes all favorites from the session
func (sfl *SessionFavoriteList) Clear() {
	if sfl == nil {
		return
	}

	sfl.Favorites = []*SessionFavorite{}
	sfl.TotalCount = 0
	sfl.LastUpdated = time.Now()
}

// IsEmpty checks if the session favorites list is empty
func (sfl *SessionFavoriteList) IsEmpty() bool {
	return sfl == nil || len(sfl.Favorites) == 0
}

// GetFavoriteIDs returns a slice of product IDs
func (sfl *SessionFavoriteList) GetFavoriteIDs() []string {
	if sfl == nil {
		return []string{}
	}

	ids := make([]string, len(sfl.Favorites))
	for i, fav := range sfl.Favorites {
		ids[i] = fav.ProductID
	}
	return ids
}

// TableName returns the table name for GORM
func (sf *SessionFavorite) TableName() string {
	return "session_favorites"
}
