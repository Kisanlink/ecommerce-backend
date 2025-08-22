package models

type Recipe struct {
	ID          uint   `json:"id" gorm:"primaryKey"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Ingredients string `json:"ingredients"`
	Steps       string `json:"steps"`
	ImageURL    string `json:"image_url"`
}
