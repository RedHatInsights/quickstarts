package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QuickstartProgress struct {
	gorm.Model
	Quickstart uint           `json:"quickstart,omitempty"`
	Progress   datatypes.JSON `json:"progress,omitempty" gorm:"type: JSONB"`
	AccountId  int            `json:"accountId,omitempty"`
}
