package models

type FavoriteQuickstart struct {
	BaseModel
	AccountId      string `gorm:"not null;" json:"accountId"`
	QuickstartName string `gorm:"not null;" json:"quickstartName"`
	Favorite       bool   `json:"favorite"`
}
