package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QuickstartProgress struct {
	gorm.Model
	ID           uint `gorm:"primaryKey"`
	QuickstartID uint
	Quickstart   Quickstart
	Progress     datatypes.JSON `gorm:"type: JSONB"`
	AccountId    int
}
