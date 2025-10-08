package models

import (
	"encoding/json"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"gorm.io/datatypes"
)

// Quickstart represents the quickstart json content
type Quickstart struct {
	BaseModel
	Name               string               `gorm:"unique;not null;default:null" json:"name"`
	Content            datatypes.JSON       `gorm:"type: JSONB" json:"content,omitempty"`
	Tags               []Tag                `gorm:"many2many:quickstart_tags;" json:"tags,omitempty"`
	FavoriteQuickstart []FavoriteQuickstart `gorm:"foreignKey:QuickstartName;references:Name" json:"favoriteQuickstart"`
}

// ToAPI converts Quickstart to generated.Quickstart for API responses
func (q Quickstart) ToAPI() generated.Quickstart {
	gen := generated.Quickstart{}

	id := int(q.ID)
	gen.Id = &id
	gen.Name = &q.Name

	// Handle JSON content conversion
	if q.Content != nil {
		var content map[string]interface{}
		if err := json.Unmarshal(q.Content, &content); err == nil {
			gen.Content = &content
		}
		// If unmarshal fails, gen.Content remains nil (which is appropriate)
	}

	gen.CreatedAt = &q.CreatedAt
	gen.UpdatedAt = &q.UpdatedAt
	if q.DeletedAt.Valid {
		gen.DeletedAt = &q.DeletedAt.Time
	}

	// Convert tags
	if len(q.Tags) > 0 {
		tags := make([]generated.Tag, len(q.Tags))
		for i, tag := range q.Tags {
			tags[i] = tag.ToAPI()
		}
		gen.Tags = &tags
	}

	// Convert favorite quickstarts
	if len(q.FavoriteQuickstart) > 0 {
		favs := make([]generated.FavoriteQuickstart, len(q.FavoriteQuickstart))
		for i, fav := range q.FavoriteQuickstart {
			favs[i] = fav.ToAPI()
		}
		gen.FavoriteQuickstart = &favs
	}

	return gen
}
