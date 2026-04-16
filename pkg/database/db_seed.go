package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/ghodss/yaml"
	"gorm.io/gorm"
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
		log.Fatal(err)
	}

	helpTopicsFiles, err := filepath.Glob(path + "/docs/help-topics/**/metadata.y*")
	if err != nil {
		slog.Error("Failed to find help topics metadata files", "error", err)
		log.Fatal(err)
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

func seedQuickstart(tx *gorm.DB, t MetadataTemplate, defaultTag models.Tag) (models.Quickstart, error) {
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
	r := tx.Where("name = ?", name).Find(&originalQuickstart)
	if r.Error != nil {
		// check for DB error
		slog.Error("Database error while checking for existing quickstart", "name", name, "error", r.Error)
		return newQuickstart, r.Error
	} else if r.RowsAffected == 0 {
		// Create new quickstart
		newQuickstart.Content = jsonContent
		newQuickstart.Name = name
		tx.Create(&newQuickstart)
		err = tx.Model(&defaultTag).Association("Quickstarts").Append(&newQuickstart)
		if err != nil {
			slog.Error("Failed creating quickstarts default tag associations", "name", name, "error", err)
		}
		tx.Save(&defaultTag)
		slog.Info("Created new quickstart", "name", name)
		return newQuickstart, nil
	} else {
		// Update existing quickstart
		originalQuickstart.Content = jsonContent
		// Clear all tags associations
		err := tx.Model(&originalQuickstart).Association("Tags").Clear()
		if err != nil {
			slog.Error("Failed clearing tags associations for quickstart", "name", name, "error", err)
		}
		tx.Save(&originalQuickstart)
		err = tx.Model(&defaultTag).Association("Quickstarts").Append(&originalQuickstart)
		if err != nil {
			slog.Error("Failed creating quickstarts default tag associations", "name", name, "error", err)
		}
		tx.Save(&defaultTag)
		slog.Info("Updated existing quickstart", "name", name)
		return originalQuickstart, nil
	}
}

func seedDefaultTags(tx *gorm.DB) map[string]models.Tag {
	slog.Info("Seeding default tags...")
	quickstartsKindTag := models.Tag{
		Type:  models.ContentKind,
		Value: "quickstart",
	}
	helpTopicKindTag := models.Tag{
		Type:  models.ContentKind,
		Value: "helptopic",
	}
	err := tx.Where("type = ? AND value = ?", &quickstartsKindTag.Type, &quickstartsKindTag.Value).FirstOrCreate(&quickstartsKindTag).Error
	if err != nil {
		slog.Error("Unable to create quickstarts kind tag", "error", err)
	}

	err = tx.Where("type = ? AND value = ?", &helpTopicKindTag.Type, &helpTopicKindTag.Value).FirstOrCreate(&helpTopicKindTag).Error
	if err != nil {
		slog.Error("Unable to create help topic kind tag", "error", err)
	}

	tx.Save(&quickstartsKindTag)
	tx.Save(&helpTopicKindTag)

	result := make(map[string]models.Tag)
	result["quickstart"] = quickstartsKindTag
	result["helptopic"] = helpTopicKindTag

	slog.Info("Default tags seeded successfully")
	return result
}

func seedHelpTopic(tx *gorm.DB, t MetadataTemplate, defaultTag models.Tag) ([]models.HelpTopic, error) {
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
		r := tx.Where("name = ?", name).Find(&originalHelpTopic)

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
			tx.Create(&newHelpTopic)
			err = tx.Model(&defaultTag).Association("HelpTopics").Append(&newHelpTopic)
			if err != nil {
				slog.Error("Failed creating help topic default tag associations", "name", name, "error", err)
			}
			tx.Save(&defaultTag)
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
			err := tx.Model(&originalHelpTopic).Association("Tags").Clear()
			if err != nil {
				slog.Error("Failed clearing tags associations for help topic", "name", name, "error", err)
			}
			tx.Save(&originalHelpTopic)
			err = tx.Model(&defaultTag).Association("HelpTopics").Append(&originalHelpTopic)
			if err != nil {
				slog.Error("Failed creating help topic default tag associations", "name", name, "error", err)
			}
			tx.Save(&defaultTag)
			slog.Info("Updated existing help topic", "name", name, "group", t.Name)
			returnValue = append(returnValue, originalHelpTopic)
		}
	}
	return returnValue, nil
}

func clearOldContent(tx *gorm.DB) []models.FavoriteQuickstart {
	slog.Info("Clearing old content...")
	var favorites []models.FavoriteQuickstart
	var staleQuickstartsTags []models.Tag
	var staleTopicsTags []models.Tag

	var staleQuickstarts []models.Quickstart
	var staleHelpTopics []models.HelpTopic
	tx.Model(&models.FavoriteQuickstart{}).Find(&favorites)

	tx.Model(&models.Quickstart{}).Find(&staleQuickstarts)
	tx.Model(&models.HelpTopic{}).Find(&staleHelpTopics)

	tx.Preload("Quickstarts").Find(&staleQuickstartsTags)
	tx.Preload("HelpTopics").Find(&staleTopicsTags)

	for _, favorite := range favorites {
		tx.Model(&favorite).Association("Quickstart").Clear()
		tx.Unscoped().Delete(&favorite)
	}

	for _, tag := range append(staleQuickstartsTags, staleTopicsTags...) {
		tx.Model(&tag).Association("Quickstarts").Clear()
		tx.Model(&tag).Association("HelpTopics").Clear()
		tx.Unscoped().Delete(&tag)
	}

	for _, q := range staleQuickstarts {
		tx.Model(&q).Association("Tags").Clear()
		tx.Unscoped().Delete(&q)
	}

	for _, h := range staleHelpTopics {
		tx.Model(&h).Association("Tags").Clear()
		tx.Unscoped().Delete(&h)
	}

	slog.Info("Cleared old content",
		"favorites", len(favorites),
		"quickstarts", len(staleQuickstarts),
		"help_topics", len(staleHelpTopics),
		"tags", len(staleQuickstartsTags)+len(staleTopicsTags))
	return favorites
}

func seedFavorites(tx *gorm.DB, favorites []models.FavoriteQuickstart) {
	seedSuccess := 0
	ignoredFalse := 0
	for _, favorite := range favorites {
		var quickstart models.Quickstart
		result := tx.Where("name = ?", favorite.QuickstartName).First(&quickstart)
		if result.Error == nil && result.RowsAffected != 0 && favorite.Favorite {
			tx.Create(&favorite)
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

// findOrCreateTag looks up a tag by type and value, creating it if it doesn't
// exist. The preload parameter specifies which association to preload
// ("Quickstarts" or "HelpTopics").
func findOrCreateTag(tx *gorm.DB, preload string, kind models.TagType, value string) (models.Tag, error) {
	var tag models.Tag

	r := tx.Preload(preload).
		Where("type = ? AND value = ?", kind, value).
		Find(&tag)

	if r.Error != nil {
		return tag, r.Error
	}
	if r.RowsAffected == 0 {
		tag.Type = kind
		tag.Value = value
		if err := tx.Create(&tag).Error; err != nil {
			return tag, err
		}
	}
	return tag, nil
}

// seedAdvisoryLockID is the fixed lock ID used with pg_advisory_xact_lock to
// serialize concurrent database seeding across pods. The value is arbitrary
// but must remain constant across all deployments.
const seedAdvisoryLockID = 42

// acquireAdvisoryLockIfSupported attempts to acquire a PostgreSQL advisory lock
// scoped to the current transaction. On non-PostgreSQL databases (e.g. SQLite
// in tests) this is a no-op.
func acquireAdvisoryLockIfSupported(tx *gorm.DB) {
	if tx.Dialector.Name() != "postgres" {
		return
	}
	if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", seedAdvisoryLockID).Error; err != nil {
		slog.Warn("Failed to acquire advisory lock, proceeding without concurrency protection", "error", err)
	}
}

func SeedTags() {
	slog.Info("Starting database seeding process...")

	// Pre-compute metadata templates outside the transaction since this
	// only reads YAML files from disk and does not touch the database.
	MetadataTemplates := findTags()

	err := DB.Transaction(func(tx *gorm.DB) error {
		acquireAdvisoryLockIfSupported(tx)

		// clear old content phase
		favorites := clearOldContent(tx)
		// seeding phase
		defaultTags := seedDefaultTags(tx)

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
				quickstart, quickstartErr = seedQuickstart(tx, template, defaultTags["quickstart"])
				if quickstartErr != nil {
					slog.Error("Unable to seed quickstart", "path", template.ContentPath, "error", quickstartErr)
					quickstartErrorCount++
					continue
				}
				quickstartCount++

				// Clear all tags associations
				quickstart.Tags = tags
				tx.Save(&quickstart)

				for _, tagTemplate := range template.Tags {
					foundTag, err := findOrCreateTag(tx, "Quickstarts",
						models.TagType(tagTemplate.Kind), tagTemplate.Value)
					if err != nil {
						slog.Error("Database error while finding tag",
							"type", tagTemplate.Kind, "value", tagTemplate.Value, "error", err)
						continue
					}

					if err := tx.Model(&foundTag).Association("Quickstarts").Append(&quickstart); err != nil {
						slog.Error("Failed creating tag association for quickstart",
							"quickstart", quickstart.Name, "tag_type", tagTemplate.Kind,
							"tag_value", tagTemplate.Value, "error", err)
						continue
					}

					quickstart.Tags = append(quickstart.Tags, foundTag)
					tx.Save(&quickstart)
					tx.Save(&foundTag)
				}
			}

			if kind == "HelpTopic" {
				helpTopic, helpTopicErr := seedHelpTopic(tx, template, defaultTags["helptopic"])
				if helpTopicErr != nil {
					slog.Error("Unable to seed help topic", "path", template.ContentPath, "error", helpTopicErr)
					helpTopicErrorCount++
					continue
				}
				helpTopicCount += len(helpTopic)

				for _, tagTemplate := range template.Tags {
					foundTag, err := findOrCreateTag(tx, "HelpTopics",
						models.TagType(tagTemplate.Kind), tagTemplate.Value)
					if err != nil {
						slog.Error("Database error while finding tag",
							"type", tagTemplate.Kind, "value", tagTemplate.Value, "error", err)
						continue
					}

					if err := tx.Model(&foundTag).Association("HelpTopics").Clear(); err != nil {
						slog.Error("Failed clearing help topic tag associations",
							"tag_type", tagTemplate.Kind, "tag_value", tagTemplate.Value, "error", err)
						continue
					}

					if err := tx.Model(&foundTag).Association("HelpTopics").Append(&helpTopic); err != nil {
						slog.Error("Failed creating tag association for help topics",
							"tag_type", tagTemplate.Kind, "tag_value", tagTemplate.Value, "error", err)
						continue
					}

					tx.Save(&foundTag)
				}
			}
		}

		slog.Info("Content seeding summary",
			"quickstarts", quickstartCount,
			"quickstart_errors", quickstartErrorCount,
			"help_topics", helpTopicCount,
			"help_topic_errors", helpTopicErrorCount)

		seedFavorites(tx, favorites)
		return nil
	})

	if err != nil {
		slog.Error("Database seeding transaction failed", "error", err)
		return
	}

	slog.Info("Database seeding completed successfully")
}
