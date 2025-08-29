package models

import (
	"github.com/Kisanlink/kisanlink-db/pkg/base"
	"github.com/Kisanlink/kisanlink-db/pkg/core/hash"
)

// Favorite represents a user's favorite product
type Favorite struct {
	*base.BaseModel `gorm:"embedded"`
	UserID          string   `json:"user_id" gorm:"not null;index"`
	ProductID       string   `json:"product_id" gorm:"not null;index"`
	Product         *Product `json:"product,omitempty" gorm:"foreignKey:ProductID;references:ID"`
}

// NewFavorite creates a new favorite with base model
func NewFavorite(userID, productID string) *Favorite {
	baseModel := base.NewBaseModel("FAV", hash.Medium)
	return &Favorite{
		BaseModel: baseModel,
		UserID:    userID,
		ProductID: productID,
	}
}

// TableName returns the table name for GORM
func (f *Favorite) TableName() string {
	return "favorites"
}
