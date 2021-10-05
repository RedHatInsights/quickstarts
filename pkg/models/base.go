package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel is a basic struc based on gorm.Model with added json attribues for openAPI3 generator
type BaseModel struct {
	ID        uint           `gorm:"primarykey" json:"id,omitempty"`
	CreatedAt time.Time      `json:"createdAt,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt,omitempty"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deletedAt,omitempty"`
}
