package models

import (
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QuickstartProgress struct {
	gorm.Model
	QuickstartName string          `gorm:"index:progress_session,unique;default:empty" json:"quickstartName,omitempty"`
	Progress       *datatypes.JSON `json:"progress,omitempty" gorm:"type: JSONB"`
	AccountId      int             `gorm:"index:progress_session,unique;default:0" json:"accountId,omitempty"`
}
