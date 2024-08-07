package models

import (
	"database/sql/driver"
	"errors"
	"fmt"
)

type TagType string

const (
	BundleTag      TagType = "bundle"
	ApplicationTag TagType = "application"
	ContentKind    TagType = "kind"
	TopicTag       TagType = "topic"
)

func (t *TagType) Scan(value interface{}) error {
	var tt TagType
	if value == nil {
		*t = ""
		return nil
	}
	st, ok := value.(string) // if we declare db type as ENUM gorm will scan value as []uint8
	if !ok {
		return errors.New("invalid data for tag type")
	}
	tt = TagType(st) //convert type from string to TagType

	switch tt {
	case BundleTag, ApplicationTag, ContentKind, TopicTag: //valid case
		*t = tt
		return nil
	}
	return fmt.Errorf("invalid tag type value :%s", st) //else is invalid
}

func (t TagType) Value() (driver.Value, error) {
	// only allow enum values
	switch t {
	case BundleTag, ApplicationTag, ContentKind, TopicTag:
		return string(t), nil
	}
	return nil, errors.New("invalid tag value")
}

// Tag is used for additional entity filtrations
type Tag struct {
	BaseModel
	Type        TagType      `json:"type" sql:"type:text" gorm:"not null"`
	Value       string       `json:"value" gorm:"not null;default:null"`
	Quickstarts []Quickstart `gorm:"many2many:quickstart_tags;"`
	HelpTopics  []HelpTopic  `gorm:"many2many:help_topic_tags;"`
}

type QuickstartTag struct {
	QuickstartID uint `gorm:"primaryKey"`
	TagID        uint `gorm:"primaryKey"`
}
