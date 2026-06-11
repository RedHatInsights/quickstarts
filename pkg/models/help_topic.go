package models

import (
	"encoding/json"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"gorm.io/datatypes"
)

// HelpTopic represents the help topic json content
type HelpTopic struct {
	BaseModel
	GroupName string         `json:"groupName"`
	Name      string         `gorm:"unique;not null;default:null" json:"name"`
	Content   datatypes.JSON `gorm:"type: JSONB" json:"content,omitempty"`
	Tags      []Tag          `gorm:"many2many:help_topic_tags;" json:"tags,omitempty"`
}

type Link struct {
	href string
	text string
}

// ToAPI converts HelpTopic to generated.HelpTopic for API responses
func (ht HelpTopic) ToAPI() generated.HelpTopic {
	gen := generated.HelpTopic{}

	id := int(ht.ID)
	gen.Id = &id
	gen.Name = &ht.Name
	gen.GroupName = &ht.GroupName

	// Handle JSON content conversion
	if ht.Content != nil {
		var content map[string]interface{}
		if err := json.Unmarshal(ht.Content, &content); err == nil {
			gen.Content = &content
		}
		// If unmarshal fails, gen.Content remains nil (which is appropriate)
	}

	gen.CreatedAt = &ht.CreatedAt
	gen.UpdatedAt = &ht.UpdatedAt
	if ht.DeletedAt.Valid {
		gen.DeletedAt = &ht.DeletedAt.Time
	}

	// Convert tags
	if len(ht.Tags) > 0 {
		tags := make([]generated.Tag, len(ht.Tags))
		for i, tag := range ht.Tags {
			tags[i] = tag.ToAPI()
		}
		gen.Tags = &tags
	}

	return gen
}
