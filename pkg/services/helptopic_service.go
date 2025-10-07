package services

import (
	"fmt"
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
)

// HelpTopicService handles business logic for help topics
type HelpTopicService struct{}

// HelpTopicFilter holds any combination of name‐ and tag‐based filters.
type HelpTopicFilter struct {
	Names []string
	Tags  map[models.TagType][]string
}

// NewHelpTopicService creates a new help topic service
func NewHelpTopicService() *HelpTopicService {
	return &HelpTopicService{}
}

// FindByFilter runs one query, joining in exactly as many tag‐filters as you need.
func (s *HelpTopicService) FindByFilter(f HelpTopicFilter) ([]models.HelpTopic, error) {
	db := database.DB.Model(&models.HelpTopic{})

	// name filter
	if len(f.Names) > 0 {
		db = db.Where("help_topics.name IN ?", f.Names)
	}

	// dynamic joins for each tag type through help_topic_tags junction table
	for tagType, values := range f.Tags {
		if len(values) == 0 {
			continue
		}
		alias := fmt.Sprintf("t_%s", strings.ToLower(string(tagType)))
		junctionAlias := fmt.Sprintf("htt_%s", strings.ToLower(string(tagType)))
		db = db.
			Joins(
				fmt.Sprintf(
					"JOIN help_topic_tags %s ON %s.help_topic_id = help_topics.id",
					junctionAlias, junctionAlias,
				),
			).
			Joins(
				fmt.Sprintf(
					"JOIN tags %s ON %s.tag_id = %s.id AND %s.type = ? AND %s.value IN ?",
					alias, junctionAlias, alias, alias, alias,
				),
				tagType, values,
			)
	}

	var result []models.HelpTopic
	return result, db.Find(&result).Error
}

// FindByName finds a help topic by name
func (s *HelpTopicService) FindByName(name string) (models.HelpTopic, error) {
	var helpTopic models.HelpTopic
	err := database.DB.Where("name = ?", name).First(&helpTopic).Error
	return helpTopic, err
}


// FindWithFilters finds help topics with bundle, application, and name filters
func (s *HelpTopicService) FindWithFilters(
	bundleQueries, applicationQueries, nameQueries []string,
) ([]models.HelpTopic, error) {
	tags := make(map[models.TagType][]string)
	if len(bundleQueries) > 0 {
		tags[models.BundleTag] = bundleQueries
	}
	if len(applicationQueries) > 0 {
		tags[models.ApplicationTag] = applicationQueries
	}

	return s.FindByFilter(HelpTopicFilter{
		Names: nameQueries,
		Tags:  tags,
	})
}
