package models

import (
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

type ContentJson struct {
	content string `json:"content"; string`
	links   []Link `json:"links"`
	name    string `json:"name"`
	tags    []Tag  `json:"tags"`
	title   string `json:"title"`
}

type Link struct {
	href string
	text string
}
