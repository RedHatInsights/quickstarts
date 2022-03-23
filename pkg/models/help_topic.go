package models

import (
	"gorm.io/datatypes"
)

// HelpTopic represents the help topic json content
type HelpTopic struct {
	BaseModel
	Name    string         `gorm:"unique;not null;default:null" json:"name"`
	Content datatypes.JSON `gorm:"type: JSONB" json:"content,omitempty"`
	Tags    []Tag          `gorm:"many2many:help_topic_tags;"`
}
