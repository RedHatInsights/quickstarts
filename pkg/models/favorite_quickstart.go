package models

import (
	"github.com/RedHatInsights/quickstarts/pkg/generated"
)

type FavoriteQuickstart struct {
	BaseModel
	AccountId      string `gorm:"not null;" json:"accountId"`
	QuickstartName string `gorm:"not null;" json:"quickstartName"`
	Favorite       bool   `json:"favorite"`
}

// ToAPI converts FavoriteQuickstart to generated.FavoriteQuickstart for API responses
func (fq FavoriteQuickstart) ToAPI() generated.FavoriteQuickstart {
	gen := generated.FavoriteQuickstart{}

	id := int(fq.ID)
	gen.Id = &id
	gen.AccountId = &fq.AccountId
	gen.QuickstartName = &fq.QuickstartName
	gen.Favorite = &fq.Favorite
	gen.CreatedAt = &fq.CreatedAt
	gen.UpdatedAt = &fq.UpdatedAt
	if fq.DeletedAt.Valid {
		gen.DeletedAt = &fq.DeletedAt.Time
	}

	return gen
}
