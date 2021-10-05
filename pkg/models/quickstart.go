package models

import (
	"gorm.io/datatypes"
)

// Quickstart represents the quickstart json content
type Quickstart struct {
	BaseModel
	Title   string         `json:"title,omitempty"`
	Content datatypes.JSON `gorm:"type: JSONB" json:"content,omitempty"`
	Bundles datatypes.JSON `gorm:"type: JSONB" json:"bundles,omitempty"`
}
