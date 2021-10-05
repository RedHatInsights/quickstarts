package models

import (
	"gorm.io/datatypes"
)

// Quickstart represents the quickstart json content
type Quickstart struct {
	BaseModel
	Title   string         `json:"Title,omitempty"`
	Content datatypes.JSON `gorm:"type: JSONB" json:"Content,omitempty"`
	Bundles datatypes.JSON `gorm:"type: JSONB" json:"Bundles,omitempty"`
}
