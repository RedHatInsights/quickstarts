package models

import (
	"encoding/json"
	"log"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type QuickstartProgress struct {
	gorm.Model
	QuickstartName string          `gorm:"index:progress_session,unique;default:empty" json:"quickstartName,omitempty"`
	Progress       *datatypes.JSON `json:"progress,omitempty" gorm:"type: JSONB"`
	AccountId      int             `gorm:"index:progress_session,unique;default:0" json:"accountId,omitempty"`
}

// ToAPI converts QuickstartProgress to generated.QuickstartProgress for API responses
func (qp QuickstartProgress) ToAPI() generated.QuickstartProgress {
	gen := generated.QuickstartProgress{}

	gen.QuickstartName = &qp.QuickstartName
	gen.AccountId = &qp.AccountId

	// Handle JSON progress conversion
	if qp.Progress != nil {
		var progress map[string]interface{}
		if err := json.Unmarshal(*qp.Progress, &progress); err != nil {
			log.Printf("Error unmarshalling QuickstartProgress.Progress: %v", err)
		} else {
			gen.Progress = &progress
		}
	}

	return gen
}
