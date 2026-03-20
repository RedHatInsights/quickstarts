package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/ghodss/yaml"
)

type TagTemplate struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type MetadataTemplate struct {
	Kind        string        `yaml:"kind"`
	Name        string        `yaml:"name"`
	Tags        []TagTemplate `yaml:"tags"`
	ContentPath string
}

func readMetadata(loc string) (MetadataTemplate, error) {
	yamlfile, err := ioutil.ReadFile(loc)
	var template MetadataTemplate
	if err != nil {
		return template, err
	}

	err = yaml.Unmarshal(yamlfile, &template)
	if err != nil {
		return template, err
	}
	m := regexp.MustCompile("metadata.ya?ml$")
	if _, err := os.Stat(m.ReplaceAllString(loc, template.Name+".yml")); err == nil {
		template.ContentPath = m.ReplaceAllString(loc, template.Name+".yml")
	} else {
		template.ContentPath = m.ReplaceAllString(loc, template.Name+".yaml")
	}

	return template, nil
}

func findTags() []MetadataTemplate {
	var MetadataTemplates []MetadataTemplate
	path, err := os.Getwd()
	path = strings.TrimRight(path, "pkg")
	quickstartsFiles, err := filepath.Glob(path + "/docs/quickstarts/**/metadata.y*")
	if err != nil {
		slog.Error("Failed to find quickstarts metadata files", "error", err)
		quickstartsFiles = []string{}
	}

	helpTopicsFiles, err := filepath.Glob(path + "/docs/help-topics/**/metadata.y*")
	if err != nil {
		slog.Error("Failed to find help topics metadata files", "error", err)
		helpTopicsFiles = []string{}
	}

	files := append(quickstartsFiles, helpTopicsFiles...)

	slog.Info("Found metadata files to process", "total", len(files), "quickstarts", len(quickstartsFiles), "help_topics", len(helpTopicsFiles))

	for _, file := range files {
		tagMetadata, err := readMetadata(file)
		if err != nil {
			slog.Warn("Failed to read metadata", "file", file, "error", err)
		} else {
			MetadataTemplates = append(MetadataTemplates, tagMetadata)
		}
	}

	slog.Info("Successfully parsed metadata templates", "count", len(MetadataTemplates))
	return MetadataTemplates
}

func addTags(t MetadataTemplate) ([]byte, error) {
	yamlfile, err := ioutil.ReadFile(t.ContentPath)
	if err != nil {
		return []byte{}, err
	}

	jsonContent, err := yaml.YAMLToJSON(yamlfile)
	if err != nil {
		return []byte{}, err
	}

	var data map[string]map[string]interface{}
	json.Unmarshal(jsonContent, &data)
	data["metadata"]["tags"] = t.Tags

	jsonContent, err = json.Marshal(data)

	return jsonContent, err
}

func seedQuickstart(t MetadataTemplate, defaultTag models.Tag) (models.Quickstart, error) {
	var newQuickstart models.Quickstart
	var originalQuickstart models.Quickstart

	jsonContent, err := addTags(t)
	if err != nil {
		slog.Error("Failed to add tags for quickstart", "path", t.ContentPath, "error", err)
		return newQuickstart, err
	}
	var data map[string]map[string]string
	json.Unmarshal(jsonContent, &data)
	name := data["metadata"]["name"]
	r := DB.Where("name = ?", name).Find(&originalQuickstart)
	if r.Error != nil {
		// check for DB error
		slog.Error("Database error while checking for existing quickstart", "name", name, "error", r.Error)
		return newQuickstart, r.Error
	} else if r.RowsAffected == 0 {
		// Create new quickstart
		newQuickstart.Content = jsonContent
		newQuickstart.Name = name
		DB.Create(&newQuickstart)
		err = DB.Model(&defaultTag).Association("Quickstarts").Append(&newQuickstart)
		if err != nil {
			slog.Error("Failed creating quickstarts default tag associations", "name", name, "error", err)
		}
		DB.Save(&defaultTag)
		slog.Info("Created new quickstart", "name", name)
		return newQuickstart, nil
	} else {
		// Update existing quickstart
		originalQuickstart.Content = jsonContent
		// Clear all tags associations
		err := DB.Model(&originalQuickstart).Association("Tags").Clear()
		if err != nil {
			slog.Error("Failed clearing tags associations for quickstart", "name", name, "error", err)
		}
		DB.Save(&originalQuickstart)
		err = DB.Model(&defaultTag).Association("Quickstarts").Append(&originalQuickstart)
		if err != nil {
			slog.Error("Failed creating quickstarts default tag associations", "name", name, "error", err)
		}
		DB.Save(&defaultTag)
		slog.Info("Updated existing quickstart", "name", name)
		return originalQuickstart, nil
	}
}

func seedDefaultTags() map[string]models.Tag {
	slog.Info("Seeding default tags...")
	quickstartsKindTag := models.Tag{
		Type:  models.ContentKind,
		Value: "quickstart",
	}
	helpTopicKindTag := models.Tag{
		Type:  models.ContentKind,
		Value: "helptopic",
	}
	err := DB.Where("type = ? AND value = ?", &quickstartsKindTag.Type, &quickstartsKindTag.Value).FirstOrCreate(&quickstartsKindTag).Error
	if err != nil {
		slog.Error("Unable to create quickstarts kind tag", "error", err)
	}

	err = DB.Where("type = ? AND value = ?", &helpTopicKindTag.Type, &helpTopicKindTag.Value).FirstOrCreate(&helpTopicKindTag).Error
	if err != nil {
		slog.Error("Unable to create help topic kind tag", "error", err)
	}

	DB.Save(&quickstartsKindTag)
	DB.Save(&helpTopicKindTag)

	result := make(map[string]models.Tag)
	result["quickstart"] = quickstartsKindTag
	result["helptopic"] = helpTopicKindTag

	slog.Info("Default tags seeded successfully")
	return result
}

func seedHelpTopic(t MetadataTemplate, defaultTag models.Tag) ([]models.HelpTopic, error) {
	yamlfile, err := ioutil.ReadFile(t.ContentPath)
	returnValue := make([]models.HelpTopic, 0)
	if err != nil {
		slog.Error("Failed to read help topic file", "path", t.ContentPath, "error", err)
		return returnValue, err
	}

	jsonContent, err := yaml.YAMLToJSON(yamlfile)
	if err != nil {
		slog.Error("Failed to convert YAML to JSON", "path", t.ContentPath, "error", err)
		return returnValue, err
	}
	var d []map[string]interface{}
	if err := json.Unmarshal(jsonContent, &d); err != nil {
		slog.Error("Failed to unmarshal JSON", "path", t.ContentPath, "error", err)
		return returnValue, err
	}

	for _, c := range d {
		var newHelpTopic models.HelpTopic
		var originalHelpTopic models.HelpTopic
		name := c["name"]
		r := DB.Where("name = ?", name).Find(&originalHelpTopic)

		if r.Error != nil {
			// check for DB error
			slog.Error("Database error while checking for existing help topic", "name", name, "error", r.Error)
			return returnValue, r.Error
		} else if r.RowsAffected == 0 {
			// Create new help topic
			newHelpTopic.GroupName = t.Name
			newHelpTopic.Content, err = json.Marshal(c)
			if err != nil {
				slog.Error("Failed to marshal content for help topic", "name", name, "error", err)
				return returnValue, err
			}
			newHelpTopic.Name = fmt.Sprintf("%v", name)
			DB.Create(&newHelpTopic)
			err = DB.Model(&defaultTag).Association("HelpTopics").Append(&newHelpTopic)
			if err != nil {
				slog.Error("Failed creating help topic default tag associations", "name", name, "error", err)
			}
			DB.Save(&defaultTag)
			slog.Info("Created new help topic", "name", name, "group", t.Name)
			returnValue = append(returnValue, newHelpTopic)
		} else {
			// Update existing help topic
			originalHelpTopic.Content, err = json.Marshal(c)
			originalHelpTopic.GroupName = t.Name
			if err != nil {
				slog.Error("Failed to marshal content for help topic", "name", name, "error", err)
				return returnValue, err
			}
			// Clear all tags associations
			err := DB.Model(&originalHelpTopic).Association("Tags").Clear()
			if err != nil {
				slog.Error("Failed clearing tags associations for help topic", "name", name, "error", err)
			}
			DB.Save(&originalHelpTopic)
			err = DB.Model(&defaultTag).Association("HelpTopics").Append(&originalHelpTopic)
			if err != nil {
				slog.Error("Failed creating help topic default tag associations", "name", name, "error", err)
			}
			DB.Save(&defaultTag)
			slog.Info("Updated existing help topic", "name", name, "group", t.Name)
			returnValue = append(returnValue, originalHelpTopic)
		}
	}
	return returnValue, nil
}

func clearOldContent() []models.FavoriteQuickstart {
	slog.Info("Clearing old content...")
	var favorites []models.FavoriteQuickstart
	var staleQuickstartsTags []models.Tag
	var staleTopicsTags []models.Tag

	var staleQuickstarts []models.Quickstart
	var staleHelpTopics []models.HelpTopic
	DB.Model(&models.FavoriteQuickstart{}).Find(&favorites)

	DB.Model(&models.Quickstart{}).Find(&staleQuickstarts)
	DB.Model(&models.HelpTopic{}).Find(&staleHelpTopics)

	DB.Preload("Quickstarts").Find(&staleQuickstartsTags)
	DB.Preload("HelpTopics").Find(&staleTopicsTags)

	for _, favorite := range favorites {
		DB.Model(&favorite).Association("Quickstart").Clear()
		DB.Unscoped().Delete(&favorite)
	}

	for _, tag := range append(staleQuickstartsTags, staleTopicsTags...) {
		DB.Model(&tag).Association("Quickstarts").Clear()
		DB.Model(&tag).Association("HelpTopics").Clear()
		DB.Unscoped().Delete(&tag)
	}

	for _, q := range staleQuickstarts {
		DB.Model(&q).Association("Tags").Clear()
		DB.Unscoped().Delete(&q)
	}

	for _, h := range staleHelpTopics {
		DB.Model(&h).Association("Tags").Clear()
		DB.Unscoped().Delete(&h)
	}

	slog.Info("Cleared old content",
		"favorites", len(favorites),
		"quickstarts", len(staleQuickstarts),
		"help_topics", len(staleHelpTopics),
		"tags", len(staleQuickstartsTags)+len(staleTopicsTags))
	return favorites
}

func SeedFavorites(favorites []models.FavoriteQuickstart) {
	seedSuccess := 0
	ignoredFalse := 0
	for _, favorite := range favorites {
		var quickstart models.Quickstart
		result := DB.Where("name = ?", favorite.QuickstartName).First(&quickstart)
		if result.Error == nil && result.RowsAffected != 0 && favorite.Favorite {
			DB.Create(&favorite)
			seedSuccess++
		} else if !favorite.Favorite {
			ignoredFalse++
		} else {
			slog.Warn("Unable to seed favorite quickstart", "name", favorite.QuickstartName, "error", result.Error)
		}
	}

	slog.Info("Seeded favorites",
		"success", seedSuccess,
		"total", len(favorites),
		"ignored_unfavorited", ignoredFalse,
		"not_found", len(favorites)-seedSuccess-ignoredFalse)
}

func SeedTags() {
	slog.Info("Starting database seeding process...")

	// clear old content phase
	favorites := clearOldContent()
	// seeding phase
	defaultTags := seedDefaultTags()
	MetadataTemplates := findTags()

	quickstartCount := 0
	quickstartErrorCount := 0
	helpTopicCount := 0
	helpTopicErrorCount := 0

	slog.Info("Processing templates...", "count", len(MetadataTemplates))

	for _, template := range MetadataTemplates {
		kind := template.Kind
		if kind == "QuickStarts" {
			var quickstart models.Quickstart
			var quickstartErr error
			var tags []models.Tag
			quickstart, quickstartErr = seedQuickstart(template, defaultTags["quickstart"])
			if quickstartErr != nil {
				slog.Error("Unable to seed quickstart", "path", template.ContentPath, "error", quickstartErr)
				quickstartErrorCount++
				continue
			}
			quickstartCount++

			// Clear all tags associations
			quickstart.Tags = tags
			DB.Save(&quickstart)

			for _, tag := range template.Tags {
				var newTag models.Tag
				var originalTag models.Tag
				newTag.Type = models.TagType(tag.Kind)
				newTag.Value = tag.Value

				r := DB.Preload("Quickstarts").Where("type = ? AND value = ?", models.TagType(newTag.Type), newTag.Value).Find(&originalTag)
				if r.Error != nil {
					slog.Error("Database error while finding tag", "type", tag.Kind, "value", tag.Value, "error", r.Error)
				} else if r.RowsAffected == 0 {
					DB.Create(&newTag)
					originalTag = newTag
				}

				// Create tags quickstarts associations
				err := DB.Model(&originalTag).Association("Quickstarts").Append(&quickstart)
				if err != nil {
					slog.Error("Failed creating tag association for quickstart", "quickstart", quickstart.Name, "tag_type", tag.Kind, "tag_value", tag.Value, "error", err)
				}

				quickstart.Tags = append(quickstart.Tags, originalTag)

				DB.Save(&quickstart)
				DB.Save(&originalTag)
			}
		}

		if kind == "HelpTopic" {
			helpTopic, helpTopicErr := seedHelpTopic(template, defaultTags["helptopic"])
			if helpTopicErr != nil {
				slog.Error("Unable to seed help topic", "path", template.ContentPath, "error", helpTopicErr)
				helpTopicErrorCount++
				continue
			}
			helpTopicCount += len(helpTopic)

			for _, tag := range template.Tags {
				var newTag models.Tag
				var originalTag models.Tag
				newTag.Type = models.TagType(tag.Kind)
				newTag.Value = tag.Value

				r := DB.Preload("HelpTopics").Where("type = ? AND value = ?", models.TagType(newTag.Type), newTag.Value).Find(&originalTag)
				if r.Error != nil {
					slog.Error("Database error while finding tag", "type", tag.Kind, "value", tag.Value, "error", r.Error)
				} else if r.RowsAffected == 0 {
					DB.Create(&newTag)
					originalTag = newTag
				}
				// Clear all tags associations
				err := DB.Model(&originalTag).Association("HelpTopics").Clear()
				if err != nil {
					slog.Error("Failed clearing help topic tag associations", "tag_type", tag.Kind, "tag_value", tag.Value, "error", err)
				}

				// Create tags help topic associations
				err = DB.Model(&originalTag).Association("HelpTopics").Append(&helpTopic)
				if err != nil {
					slog.Error("Failed creating tag association for help topics", "tag_type", tag.Kind, "tag_value", tag.Value, "error", err)
				}

				DB.Save(&originalTag)
			}
		}
	}

	slog.Info("Content seeding summary",
		"quickstarts", quickstartCount,
		"quickstart_errors", quickstartErrorCount,
		"help_topics", helpTopicCount,
		"help_topic_errors", helpTopicErrorCount)

	SeedFavorites(favorites)
	slog.Info("Database seeding completed")
}
