package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel is a basic struc based on gorm.Model with added json attribues for openAPI3 generator
type BaseModel struct {
	gorm.Model
	ID        uint           `gorm:"primarykey" json:"ID,omitempty"`
	CreatedAt time.Time      `json:"CreatedAt,omitempty"`
	UpdatedAt time.Time      `json:"UpdatedAt,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"DeletedAt,omitempty"`
}
