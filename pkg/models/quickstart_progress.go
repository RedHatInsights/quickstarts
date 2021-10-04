package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QuickstartProgress struct {
	gorm.Model
	QuickstartID uint           `json:"QuickstartID,omitempty"`
	Quickstart   *Quickstart    `json:"Quickstart,omitempty"`
	Progress     datatypes.JSON `json:"Progress,omitempty" gorm:"type: JSONB"`
	AccountId    int            `json:"AccountId,omitempty"`
}
