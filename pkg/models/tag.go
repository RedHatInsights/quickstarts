package models

import (
	"database/sql/driver"
	"errors"
	"fmt"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
)

type TagType string

const (
	BundleTag       TagType = "bundle"
	ApplicationTag  TagType = "application"
	ContentKind     TagType = "kind"
	TopicTag        TagType = "topic"
	ContentType     TagType = "content"
	ProductFamilies TagType = "product-families"
	UseCase         TagType = "use-case"
)

func (t TagType) GetAllTags() []TagType {
	return []TagType{BundleTag, ApplicationTag, ContentKind, TopicTag, ContentType, ProductFamilies, UseCase}
}

func (t TagType) IsValidTag() bool {

	tags := t.GetAllTags()
	for i := range tags {
		if t == tags[i] {
			return true
		}
	}
	return false
}

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

	if tt.IsValidTag() {
		*t = tt
		return nil
	}
	return fmt.Errorf("invalid tag type value :%s", st) //else is invalid
}

func (t TagType) Value() (driver.Value, error) {
	// only allow enum values
	if t.IsValidTag() {
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

// ToAPI converts Tag to generated.Tag for API responses
func (t Tag) ToAPI() generated.Tag {
	gen := generated.Tag{}

	id := int(t.ID)
	gen.Id = &id
	typeStr := string(t.Type)
	gen.Type = &typeStr
	gen.Value = &t.Value
	gen.CreatedAt = &t.CreatedAt
	gen.UpdatedAt = &t.UpdatedAt
	if t.DeletedAt.Valid {
		gen.DeletedAt = &t.DeletedAt.Time
	}

	return gen
}
