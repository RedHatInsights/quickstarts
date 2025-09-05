package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
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

type SeedingResult struct {
	QuickstartsProcessed int
	QuickstartsCreated   int
	QuickstartsUpdated   int
	HelpTopicsProcessed  int
	HelpTopicsCreated    int
	HelpTopicsUpdated    int
	TagsCreated          int
	FavoritesRestored    int
	Errors               []error
}

func readMetadata(loc string) (MetadataTemplate, error) {
	logrus.Debugf("Reading metadata from: %s", loc)
	yamlfile, err := ioutil.ReadFile(loc)
	var template MetadataTemplate
	if err != nil {
		logrus.Errorf("Failed to read metadata file %s: %v", loc, err)
		return template, fmt.Errorf("failed to read metadata file %s: %w", loc, err)
	}

	err = yaml.Unmarshal(yamlfile, &template)
	if err != nil {
		logrus.Errorf("Failed to unmarshal YAML from %s: %v", loc, err)
		return template, fmt.Errorf("failed to unmarshal YAML from %s: %w", loc, err)
	}
	m := regexp.MustCompile("metadata.ya?ml$")
	if _, err := os.Stat(m.ReplaceAllString(loc, template.Name+".yml")); err == nil {
		template.ContentPath = m.ReplaceAllString(loc, template.Name+".yml")
	} else {
		template.ContentPath = m.ReplaceAllString(loc, template.Name+".yaml")
	}

	logrus.Debugf("Successfully read metadata for %s (kind: %s, content path: %s)", template.Name, template.Kind, template.ContentPath)
	return template, nil
}

func findTags() ([]MetadataTemplate, error) {
	logrus.Info("Starting metadata discovery process")
	var MetadataTemplates []MetadataTemplate
	path, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	path = strings.TrimRight(path, "pkg")
	logrus.Debugf("Using base path: %s", path)

	quickstartsPattern := path + "/docs/quickstarts/**/metadata.y*"
	logrus.Debugf("Searching for quickstarts with pattern: %s", quickstartsPattern)
	quickstartsFiles, err := filepath.Glob(quickstartsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob quickstarts files with pattern %s: %w", quickstartsPattern, err)
	}
	logrus.Infof("Found %d quickstart metadata files", len(quickstartsFiles))

	helpTopicsPattern := path + "/docs/help-topics/**/metadata.y*"
	logrus.Debugf("Searching for help topics with pattern: %s", helpTopicsPattern)
	helpTopicsFiles, err := filepath.Glob(helpTopicsPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob help topics files with pattern %s: %w", helpTopicsPattern, err)
	}
	logrus.Infof("Found %d help topic metadata files", len(helpTopicsFiles))

	files := append(quickstartsFiles, helpTopicsFiles...)
	logrus.Infof("Processing %d total metadata files", len(files))

	successCount := 0
	errorCount := 0
	for _, file := range files {
		tagMetadata, err := readMetadata(file)
		if err != nil {
			logrus.Errorf("Failed to read metadata from %s: %v", file, err)
			errorCount++
			continue
		}
		MetadataTemplates = append(MetadataTemplates, tagMetadata)
		successCount++
	}

	logrus.Infof("Metadata discovery complete: %d successful, %d errors", successCount, errorCount)
	if errorCount > 0 {
		return MetadataTemplates, fmt.Errorf("encountered %d errors during metadata discovery", errorCount)
	}

	return MetadataTemplates, nil
}

func addTags(t MetadataTemplate) ([]byte, error) {
	logrus.Debugf("Adding tags to content at: %s", t.ContentPath)
	yamlfile, err := ioutil.ReadFile(t.ContentPath)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to read content file %s: %w", t.ContentPath, err)
	}

	jsonContent, err := yaml.YAMLToJSON(yamlfile)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to convert YAML to JSON for %s: %w", t.ContentPath, err)
	}

	// Parse as generic interface{} first to handle different file structures
	var data interface{}
	if err := json.Unmarshal(jsonContent, &data); err != nil {
		return []byte{}, fmt.Errorf("failed to unmarshal JSON content for %s: %w", t.ContentPath, err)
	}
	
	// Convert to map[string]interface{} for manipulation
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return []byte{}, fmt.Errorf("content file %s does not have expected top-level object structure", t.ContentPath)
	}
	
	// Ensure metadata exists and is a map
	if dataMap["metadata"] == nil {
		dataMap["metadata"] = make(map[string]interface{})
	}
	
	metadata, ok := dataMap["metadata"].(map[string]interface{})
	if !ok {
		dataMap["metadata"] = make(map[string]interface{})
		metadata = dataMap["metadata"].(map[string]interface{})
	}
	
	// Add tags to metadata
	metadata["tags"] = t.Tags

	jsonContent, err = json.Marshal(dataMap)
	if err != nil {
		return []byte{}, fmt.Errorf("failed to marshal JSON content for %s: %w", t.ContentPath, err)
	}

	logrus.Debugf("Successfully added %d tags to %s", len(t.Tags), t.Name)
	return jsonContent, nil
}

func seedQuickstart(t MetadataTemplate, defaultTag models.Tag) (models.Quickstart, bool, error) {
	logrus.Debugf("Seeding quickstart: %s", t.Name)
	var newQuickstart models.Quickstart
	var originalQuickstart models.Quickstart

	jsonContent, err := addTags(t)
	if err != nil {
		return newQuickstart, false, fmt.Errorf("failed to add tags for quickstart %s: %w", t.Name, err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(jsonContent, &data); err != nil {
		return newQuickstart, false, fmt.Errorf("failed to unmarshal content for quickstart %s: %w", t.Name, err)
	}
	
	// Extract name from metadata
	metadata, ok := data["metadata"].(map[string]interface{})
	if !ok {
		return newQuickstart, false, fmt.Errorf("metadata section not found or invalid in quickstart %s", t.ContentPath)
	}
	
	nameInterface, exists := metadata["name"]
	if !exists {
		return newQuickstart, false, fmt.Errorf("quickstart name not found in metadata for %s", t.ContentPath)
	}
	
	name, ok := nameInterface.(string)
	if !ok {
		return newQuickstart, false, fmt.Errorf("quickstart name is not a string in metadata for %s", t.ContentPath)
	}
	
	if name == "" {
		return newQuickstart, false, fmt.Errorf("quickstart name is empty in metadata for %s", t.ContentPath)
	}

	r := DB.Where("name = ?", name).Find(&originalQuickstart)
	if r.Error != nil {
		return newQuickstart, false, fmt.Errorf("database error while looking up quickstart %s: %w", name, r.Error)
	}

	if r.RowsAffected == 0 {
		// Create new quickstart
		logrus.Infof("Creating new quickstart: %s", name)
		newQuickstart.Content = jsonContent
		newQuickstart.Name = name
		
		if err := DB.Create(&newQuickstart).Error; err != nil {
			return newQuickstart, false, fmt.Errorf("failed to create quickstart %s: %w", name, err)
		}

		if err := DB.Model(&defaultTag).Association("Quickstarts").Append(&newQuickstart); err != nil {
			return newQuickstart, false, fmt.Errorf("failed to create default tag association for quickstart %s: %w", name, err)
		}

		if err := DB.Save(&defaultTag).Error; err != nil {
			return newQuickstart, false, fmt.Errorf("failed to save default tag for quickstart %s: %w", name, err)
		}
		
		logrus.Infof("Successfully created quickstart: %s", name)
		return newQuickstart, true, nil
	} else {
		// Update existing quickstart
		logrus.Infof("Updating existing quickstart: %s", name)
		originalQuickstart.Content = jsonContent
		
		// Clear all tags associations
		if err := DB.Model(&originalQuickstart).Association("Tags").Clear(); err != nil {
			return originalQuickstart, false, fmt.Errorf("failed to clear tag associations for quickstart %s: %w", name, err)
		}

		if err := DB.Save(&originalQuickstart).Error; err != nil {
			return originalQuickstart, false, fmt.Errorf("failed to save updated quickstart %s: %w", name, err)
		}

		if err := DB.Model(&defaultTag).Association("Quickstarts").Append(&originalQuickstart); err != nil {
			return originalQuickstart, false, fmt.Errorf("failed to create default tag association for updated quickstart %s: %w", name, err)
		}

		if err := DB.Save(&defaultTag).Error; err != nil {
			return originalQuickstart, false, fmt.Errorf("failed to save default tag for updated quickstart %s: %w", name, err)
		}
		
		logrus.Infof("Successfully updated quickstart: %s", name)
		return originalQuickstart, false, nil
	}
}

func seedDefaultTags() (map[string]models.Tag, error) {
	logrus.Info("Seeding default tags")
	quickstartsKindTag := models.Tag{
		Type:  models.ContentKind,
		Value: "quickstart",
	}
	helpTopicKindTag := models.Tag{
		Type:  models.ContentKind,
		Value: "helptopic",
	}

	logrus.Debugf("Creating/finding quickstart kind tag: %s", quickstartsKindTag.Value)
	err := DB.Where("type = ? AND value = ?", &quickstartsKindTag.Type, &quickstartsKindTag.Value).FirstOrCreate(&quickstartsKindTag).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create quickstarts kind tag: %w", err)
	}

	logrus.Debugf("Creating/finding help topic kind tag: %s", helpTopicKindTag.Value)
	err = DB.Where("type = ? AND value = ?", &helpTopicKindTag.Type, &helpTopicKindTag.Value).FirstOrCreate(&helpTopicKindTag).Error
	if err != nil {
		return nil, fmt.Errorf("failed to create help topic kind tag: %w", err)
	}

	if err := DB.Save(&quickstartsKindTag).Error; err != nil {
		return nil, fmt.Errorf("failed to save quickstarts kind tag: %w", err)
	}

	if err := DB.Save(&helpTopicKindTag).Error; err != nil {
		return nil, fmt.Errorf("failed to save help topic kind tag: %w", err)
	}

	result := make(map[string]models.Tag)
	result["quickstart"] = quickstartsKindTag
	result["helptopic"] = helpTopicKindTag

	logrus.Info("Successfully seeded default tags")
	return result, nil
}

func seedHelpTopic(t MetadataTemplate, defaultTag models.Tag) ([]models.HelpTopic, int, error) {
	logrus.Debugf("Seeding help topics from: %s", t.Name)
	returnValue := make([]models.HelpTopic, 0)
	createdCount := 0

	yamlfile, err := ioutil.ReadFile(t.ContentPath)
	if err != nil {
		return returnValue, 0, fmt.Errorf("failed to read help topic content file %s: %w", t.ContentPath, err)
	}

	jsonContent, err := yaml.YAMLToJSON(yamlfile)
	if err != nil {
		return returnValue, 0, fmt.Errorf("failed to convert YAML to JSON for help topic %s: %w", t.ContentPath, err)
	}

	var d []map[string]interface{}
	if err := json.Unmarshal(jsonContent, &d); err != nil {
		return returnValue, 0, fmt.Errorf("failed to unmarshal help topic content %s: %w", t.ContentPath, err)
	}

	logrus.Infof("Processing %d help topics from group: %s", len(d), t.Name)

	for i, c := range d {
		var newHelpTopic models.HelpTopic
		var originalHelpTopic models.HelpTopic
		name := c["name"]
		if name == nil || fmt.Sprintf("%v", name) == "" {
			logrus.Errorf("Help topic %d in group %s has no name, skipping", i, t.Name)
			continue
		}

		nameStr := fmt.Sprintf("%v", name)
		logrus.Debugf("Processing help topic: %s", nameStr)

		r := DB.Where("name = ?", nameStr).Find(&originalHelpTopic)
		if r.Error != nil {
			return returnValue, createdCount, fmt.Errorf("database error while looking up help topic %s: %w", nameStr, r.Error)
		}

		if r.RowsAffected == 0 {
			// Create new help topic
			logrus.Infof("Creating new help topic: %s", nameStr)
			newHelpTopic.GroupName = t.Name
			newHelpTopic.Content, err = json.Marshal(c)
			if err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to marshal content for help topic %s: %w", nameStr, err)
			}
			newHelpTopic.Name = nameStr

			if err := DB.Create(&newHelpTopic).Error; err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to create help topic %s: %w", nameStr, err)
			}

			if err := DB.Model(&defaultTag).Association("HelpTopics").Append(&newHelpTopic); err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to create default tag association for help topic %s: %w", nameStr, err)
			}

			if err := DB.Save(&defaultTag).Error; err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to save default tag for help topic %s: %w", nameStr, err)
			}

			returnValue = append(returnValue, newHelpTopic)
			createdCount++
		} else {
			// Update existing help topic
			logrus.Infof("Updating existing help topic: %s", nameStr)
			originalHelpTopic.Content, err = json.Marshal(c)
			if err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to marshal content for help topic %s: %w", nameStr, err)
			}
			originalHelpTopic.GroupName = t.Name

			// Clear all tags associations
			if err := DB.Model(&originalHelpTopic).Association("Tags").Clear(); err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to clear tag associations for help topic %s: %w", nameStr, err)
			}

			if err := DB.Save(&originalHelpTopic).Error; err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to save updated help topic %s: %w", nameStr, err)
			}

			if err := DB.Model(&defaultTag).Association("HelpTopics").Append(&originalHelpTopic); err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to create default tag association for updated help topic %s: %w", nameStr, err)
			}

			if err := DB.Save(&defaultTag).Error; err != nil {
				return returnValue, createdCount, fmt.Errorf("failed to save default tag for updated help topic %s: %w", nameStr, err)
			}

			returnValue = append(returnValue, originalHelpTopic)
		}
	}

	logrus.Infof("Successfully processed %d help topics from group %s (%d created, %d updated)", len(returnValue), t.Name, createdCount, len(returnValue)-createdCount)
	return returnValue, createdCount, nil
}

func clearOldContent() ([]models.FavoriteQuickstart, error) {
	logrus.Info("Starting cleanup of old content")
	var favorites []models.FavoriteQuickstart
	var staleQuickstartsTags []models.Tag
	var staleTopicsTags []models.Tag
	var staleQuickstarts []models.Quickstart
	var staleHelpTopics []models.HelpTopic

	// Get all existing favorites to preserve
	if err := DB.Model(&models.FavoriteQuickstart{}).Find(&favorites).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing favorites: %w", err)
	}
	logrus.Infof("Found %d existing favorites to preserve", len(favorites))

	// Get all existing content to clear
	if err := DB.Model(&models.Quickstart{}).Find(&staleQuickstarts).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing quickstarts: %w", err)
	}
	logrus.Infof("Found %d existing quickstarts to clear", len(staleQuickstarts))

	if err := DB.Model(&models.HelpTopic{}).Find(&staleHelpTopics).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing help topics: %w", err)
	}
	logrus.Infof("Found %d existing help topics to clear", len(staleHelpTopics))

	if err := DB.Preload("Quickstarts").Find(&staleQuickstartsTags).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing quickstart tags: %w", err)
	}

	if err := DB.Preload("HelpTopics").Find(&staleTopicsTags).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing help topic tags: %w", err)
	}
	logrus.Infof("Found %d total existing tags to clear", len(staleQuickstartsTags)+len(staleTopicsTags))

	// Clear favorites
	logrus.Debug("Clearing favorite associations and records")
	for _, favorite := range favorites {
		if err := DB.Model(&favorite).Association("Quickstart").Clear(); err != nil {
			return nil, fmt.Errorf("failed to clear favorite quickstart association for favorite %d: %w", favorite.ID, err)
		}
		if err := DB.Unscoped().Delete(&favorite).Error; err != nil {
			return nil, fmt.Errorf("failed to delete favorite %d: %w", favorite.ID, err)
		}
	}

	// Clear all tag associations and delete tags
	logrus.Debug("Clearing tag associations and records")
	allTags := append(staleQuickstartsTags, staleTopicsTags...)
	for _, tag := range allTags {
		if err := DB.Model(&tag).Association("Quickstarts").Clear(); err != nil {
			return nil, fmt.Errorf("failed to clear quickstart associations for tag %d: %w", tag.ID, err)
		}
		if err := DB.Model(&tag).Association("HelpTopics").Clear(); err != nil {
			return nil, fmt.Errorf("failed to clear help topic associations for tag %d: %w", tag.ID, err)
		}
		if err := DB.Unscoped().Delete(&tag).Error; err != nil {
			return nil, fmt.Errorf("failed to delete tag %d: %w", tag.ID, err)
		}
	}

	// Clear quickstart associations and delete quickstarts
	logrus.Debug("Clearing quickstart associations and records")
	for _, q := range staleQuickstarts {
		if err := DB.Model(&q).Association("Tags").Clear(); err != nil {
			return nil, fmt.Errorf("failed to clear tag associations for quickstart %s: %w", q.Name, err)
		}
		if err := DB.Unscoped().Delete(&q).Error; err != nil {
			return nil, fmt.Errorf("failed to delete quickstart %s: %w", q.Name, err)
		}
	}

	// Clear help topic associations and delete help topics
	logrus.Debug("Clearing help topic associations and records")
	for _, h := range staleHelpTopics {
		if err := DB.Model(&h).Association("Tags").Clear(); err != nil {
			return nil, fmt.Errorf("failed to clear tag associations for help topic %s: %w", h.Name, err)
		}
		if err := DB.Unscoped().Delete(&h).Error; err != nil {
			return nil, fmt.Errorf("failed to delete help topic %s: %w", h.Name, err)
		}
	}

	logrus.Info("Successfully cleared all old content")
	return favorites, nil
}

func SeedFavorites(favorites []models.FavoriteQuickstart) error {
	logrus.Infof("Starting to restore %d favorite quickstarts", len(favorites))
	seedSuccess := 0
	ignoredFalse := 0
	notFound := 0

	for _, favorite := range favorites {
		var quickstart models.Quickstart
		result := DB.Where("name = ?", favorite.QuickstartName).First(&quickstart)
		
		if result.Error == nil && result.RowsAffected != 0 && favorite.Favorite {
			if err := DB.Create(&favorite).Error; err != nil {
				return fmt.Errorf("failed to create favorite for quickstart %s: %w", favorite.QuickstartName, err)
			}
			logrus.Debugf("Restored favorite for quickstart: %s", favorite.QuickstartName)
			seedSuccess++
		} else if !favorite.Favorite {
			logrus.Debugf("Skipping unfavorite entry for: %s", favorite.QuickstartName)
			ignoredFalse++
		} else {
			logrus.Warningf("Unable to restore favorite quickstart %s: quickstart not found (may have been renamed)", favorite.QuickstartName)
			notFound++
		}
	}

	logrus.Infof("Favorites restoration complete: %d restored, %d ignored (unfavorite), %d not found", seedSuccess, ignoredFalse, notFound)
	return nil
}

func SeedTags() error {
	logrus.Info("=== STARTING DATABASE SEEDING PROCESS ===")
	
	result := &SeedingResult{}
	
	// Phase 1: Clear old content and preserve favorites
	logrus.Info("Phase 1: Clearing old content")
	favorites, err := clearOldContent()
	if err != nil {
		logrus.Errorf("Failed to clear old content: %v", err)
		return fmt.Errorf("seeding failed during cleanup phase: %w", err)
	}

	// Phase 2: Seed default tags
	logrus.Info("Phase 2: Creating default tags")
	defaultTags, err := seedDefaultTags()
	if err != nil {
		logrus.Errorf("Failed to seed default tags: %v", err)
		return fmt.Errorf("seeding failed during default tags creation: %w", err)
	}

	// Phase 3: Discover metadata files
	logrus.Info("Phase 3: Discovering content metadata")
	metadataTemplates, err := findTags()
	if err != nil {
		logrus.Errorf("Failed to discover metadata files: %v", err)
		return fmt.Errorf("seeding failed during metadata discovery: %w", err)
	}

	// Phase 4: Process quickstarts and help topics
	logrus.Info("Phase 4: Processing content and tags")
	for _, template := range metadataTemplates {
		kind := template.Kind
		
		if kind == "QuickStarts" {
			logrus.Debugf("Processing QuickStart template: %s", template.Name)
			quickstart, isNew, err := seedQuickstart(template, defaultTags["quickstart"])
			if err != nil {
				errMsg := fmt.Sprintf("failed to seed quickstart %s: %v", template.Name, err)
				logrus.Error(errMsg)
				result.Errors = append(result.Errors, fmt.Errorf(errMsg))
				continue
			}
			
			result.QuickstartsProcessed++
			if isNew {
				result.QuickstartsCreated++
			} else {
				result.QuickstartsUpdated++
			}

			// Clear existing tag associations
			var tags []models.Tag
			quickstart.Tags = tags
			if err := DB.Save(&quickstart).Error; err != nil {
				errMsg := fmt.Sprintf("failed to clear tag associations for quickstart %s: %v", quickstart.Name, err)
				logrus.Error(errMsg)
				result.Errors = append(result.Errors, fmt.Errorf(errMsg))
				continue
			}

			// Process tags for this quickstart
			for _, tag := range template.Tags {
				var newTag models.Tag
				var originalTag models.Tag
				newTag.Type = models.TagType(tag.Kind)
				newTag.Value = tag.Value

				logrus.Debugf("Processing tag: %s=%s for quickstart %s", tag.Kind, tag.Value, quickstart.Name)

				r := DB.Preload("Quickstarts").Where("type = ? AND value = ?", models.TagType(newTag.Type), newTag.Value).Find(&originalTag)
				if r.Error != nil {
					errMsg := fmt.Sprintf("failed to lookup tag %s=%s: %v", tag.Kind, tag.Value, r.Error)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}
				
				if r.RowsAffected == 0 {
					// Create new tag
					if err := DB.Create(&newTag).Error; err != nil {
						errMsg := fmt.Sprintf("failed to create tag %s=%s: %v", tag.Kind, tag.Value, err)
						logrus.Error(errMsg)
						result.Errors = append(result.Errors, fmt.Errorf(errMsg))
						continue
					}
					originalTag = newTag
					result.TagsCreated++
					logrus.Debugf("Created new tag: %s=%s", tag.Kind, tag.Value)
				}

				// Create tag-quickstart association
				if err := DB.Model(&originalTag).Association("Quickstarts").Append(&quickstart); err != nil {
					errMsg := fmt.Sprintf("failed to create tag association %s=%s for quickstart %s: %v", tag.Kind, tag.Value, quickstart.Name, err)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}

				quickstart.Tags = append(quickstart.Tags, originalTag)

				if err := DB.Save(&quickstart).Error; err != nil {
					errMsg := fmt.Sprintf("failed to save quickstart %s after adding tag: %v", quickstart.Name, err)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}

				if err := DB.Save(&originalTag).Error; err != nil {
					errMsg := fmt.Sprintf("failed to save tag %s=%s: %v", tag.Kind, tag.Value, err)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}
			}
		}

		if kind == "HelpTopic" {
			logrus.Debugf("Processing HelpTopic template: %s", template.Name)
			helpTopics, createdCount, err := seedHelpTopic(template, defaultTags["helptopic"])
			if err != nil {
				errMsg := fmt.Sprintf("failed to seed help topic %s: %v", template.Name, err)
				logrus.Error(errMsg)
				result.Errors = append(result.Errors, fmt.Errorf(errMsg))
				continue
			}

			result.HelpTopicsProcessed += len(helpTopics)
			result.HelpTopicsCreated += createdCount
			result.HelpTopicsUpdated += len(helpTopics) - createdCount

			// Process tags for help topics
			for _, tag := range template.Tags {
				var newTag models.Tag
				var originalTag models.Tag
				newTag.Type = models.TagType(tag.Kind)
				newTag.Value = tag.Value

				logrus.Debugf("Processing tag: %s=%s for help topic group %s", tag.Kind, tag.Value, template.Name)

				r := DB.Preload("HelpTopics").Where("type = ? AND value = ?", models.TagType(newTag.Type), newTag.Value).Find(&originalTag)
				if r.Error != nil {
					errMsg := fmt.Sprintf("failed to lookup tag %s=%s: %v", tag.Kind, tag.Value, r.Error)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}
				
				if r.RowsAffected == 0 {
					// Create new tag
					if err := DB.Create(&newTag).Error; err != nil {
						errMsg := fmt.Sprintf("failed to create tag %s=%s: %v", tag.Kind, tag.Value, err)
						logrus.Error(errMsg)
						result.Errors = append(result.Errors, fmt.Errorf(errMsg))
						continue
					}
					originalTag = newTag
					result.TagsCreated++
					logrus.Debugf("Created new tag: %s=%s", tag.Kind, tag.Value)
				}

				// Clear existing help topic associations for this tag
				if err := DB.Model(&originalTag).Association("HelpTopics").Clear(); err != nil {
					errMsg := fmt.Sprintf("failed to clear help topic associations for tag %s=%s: %v", tag.Kind, tag.Value, err)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}

				// Create tag-help topic associations
				if err := DB.Model(&originalTag).Association("HelpTopics").Append(&helpTopics); err != nil {
					errMsg := fmt.Sprintf("failed to create tag associations %s=%s for help topics: %v", tag.Kind, tag.Value, err)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}

				if err := DB.Save(&originalTag).Error; err != nil {
					errMsg := fmt.Sprintf("failed to save tag %s=%s: %v", tag.Kind, tag.Value, err)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}
			}
		}
	}

	// Phase 5: Restore favorites
	logrus.Info("Phase 5: Restoring favorite quickstarts")
	if err := SeedFavorites(favorites); err != nil {
		errMsg := fmt.Sprintf("failed to restore favorites: %v", err)
		logrus.Error(errMsg)
		result.Errors = append(result.Errors, fmt.Errorf(errMsg))
	} else {
		result.FavoritesRestored = len(favorites)
	}

	// Final summary
	logrus.Info("=== SEEDING PROCESS SUMMARY ===")
	logrus.Infof("✓ Quickstarts: %d processed (%d created, %d updated)", result.QuickstartsProcessed, result.QuickstartsCreated, result.QuickstartsUpdated)
	logrus.Infof("✓ Help Topics: %d processed (%d created, %d updated)", result.HelpTopicsProcessed, result.HelpTopicsCreated, result.HelpTopicsUpdated)
	logrus.Infof("✓ Tags Created: %d", result.TagsCreated)
	logrus.Infof("✓ Favorites Restored: %d", result.FavoritesRestored)

	if len(result.Errors) > 0 {
		logrus.Errorf("✗ SEEDING COMPLETED WITH %d ERRORS:", len(result.Errors))
		for i, err := range result.Errors {
			logrus.Errorf("  Error %d: %v", i+1, err)
		}
		return fmt.Errorf("database seeding completed with %d errors - see logs for details", len(result.Errors))
	}

	logrus.Info("✓ SEEDING COMPLETED SUCCESSFULLY")
	return nil
}
