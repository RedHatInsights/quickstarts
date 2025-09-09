package models

import (
	"gorm.io/datatypes"
)

// Quickstart represents the quickstart json content
type Quickstart struct {
	BaseModel
	Name               string               `gorm:"unique;not null;default:null" json:"name"`
	Content            datatypes.JSON       `gorm:"type: JSONB" json:"content,omitempty"`
	Tags               []Tag                `gorm:"many2many:quickstart_tags;" json:"tags,omitempty"`
	FavoriteQuickstart []FavoriteQuickstart `gorm:"foreignKey:QuickstartName;references:Name" json:"favoriteQuickstart"`
}
