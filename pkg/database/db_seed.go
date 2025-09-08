package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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
	fileHelper := NewFileHelper("metadata-reader")
	var template MetadataTemplate
	
	if err := fileHelper.ReadYAMLFile(loc, &template); err != nil {
		return template, err
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
	fileHelper := NewFileHelper("metadata-discovery")
	var MetadataTemplates []MetadataTemplate
	
	path, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	path = strings.TrimRight(path, "pkg")
	logrus.Debugf("Using base path: %s", path)

	quickstartsPattern := path + "/docs/quickstarts/**/metadata.y*"
	quickstartsFiles, err := fileHelper.GlobFiles(quickstartsPattern)
	if err != nil {
		return nil, err
	}

	helpTopicsPattern := path + "/docs/help-topics/**/metadata.y*"
	helpTopicsFiles, err := fileHelper.GlobFiles(helpTopicsPattern)
	if err != nil {
		return nil, err
	}

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
	fileHelper := NewFileHelper("content-tagger")
	return fileHelper.AddTagsToContent(t.ContentPath, t.Tags)
}

func seedQuickstart(t MetadataTemplate, defaultTag models.Tag) (models.Quickstart, bool, error) {
	dbHelper := NewDBHelper(DB, "quickstart-seeder")
	fileHelper := NewFileHelper("quickstart-seeder")
	var newQuickstart models.Quickstart
	var originalQuickstart models.Quickstart

	jsonContent, err := addTags(t)
	if err != nil {
		return newQuickstart, false, fmt.Errorf("failed to add tags for quickstart %s: %w", t.Name, err)
	}

	// Extract name from metadata using helper
	name, err := fileHelper.ExtractStringFromMetadata(jsonContent, "name", t.ContentPath)
	if err != nil {
		return newQuickstart, false, err
	}

	r := DB.Where("name = ?", name).Find(&originalQuickstart)
	if r.Error != nil {
		return newQuickstart, false, fmt.Errorf("database error while looking up quickstart %s: %w", name, r.Error)
	}

	if r.RowsAffected == 0 {
		// Create new quickstart
		newQuickstart.Content = jsonContent
		newQuickstart.Name = name
		
		if err := dbHelper.Create(&newQuickstart, "quickstart", name); err != nil {
			return newQuickstart, false, err
		}

		if err := dbHelper.AppendAssociation(&defaultTag, "Quickstarts", &newQuickstart, "default tag", "quickstart"); err != nil {
			return newQuickstart, false, err
		}

		if err := dbHelper.Update(&defaultTag, "default tag", "quickstart"); err != nil {
			return newQuickstart, false, err
		}
		
		return newQuickstart, true, nil
	} else {
		// Update existing quickstart
		originalQuickstart.Content = jsonContent
		
		// Clear all tags associations
		if err := dbHelper.ClearAssociation(&originalQuickstart, "Tags", "quickstart", name); err != nil {
			return originalQuickstart, false, err
		}

		if err := dbHelper.Update(&originalQuickstart, "quickstart", name); err != nil {
			return originalQuickstart, false, err
		}

		if err := dbHelper.AppendAssociation(&defaultTag, "Quickstarts", &originalQuickstart, "default tag", "quickstart"); err != nil {
			return originalQuickstart, false, err
		}

		if err := dbHelper.Update(&defaultTag, "default tag", "quickstart"); err != nil {
			return originalQuickstart, false, err
		}
		
		return originalQuickstart, false, nil
	}
}

func seedDefaultTags() (map[string]models.Tag, error) {
	dbHelper := NewDBHelper(DB, "default-tag-seeder")
	
	quickstartsKindTag := models.Tag{
		Type:  models.ContentKind,
		Value: "quickstart",
	}
	helpTopicKindTag := models.Tag{
		Type:  models.ContentKind,
		Value: "helptopic",
	}

	_, err := dbHelper.FindOrCreate(&quickstartsKindTag, 
		map[string]interface{}{"type": quickstartsKindTag.Type, "value": quickstartsKindTag.Value}, 
		"quickstart kind tag", quickstartsKindTag.Value)
	if err != nil {
		return nil, err
	}

	_, err = dbHelper.FindOrCreate(&helpTopicKindTag, 
		map[string]interface{}{"type": helpTopicKindTag.Type, "value": helpTopicKindTag.Value}, 
		"help topic kind tag", helpTopicKindTag.Value)
	if err != nil {
		return nil, err
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
	dbHelper := NewDBHelper(DB, "content-cleaner")
	var favorites []models.FavoriteQuickstart
	var staleQuickstartsTags []models.Tag
	var staleTopicsTags []models.Tag
	var staleQuickstarts []models.Quickstart
	var staleHelpTopics []models.HelpTopic

	// Get all existing content to preserve/clear
	if err := DB.Model(&models.FavoriteQuickstart{}).Find(&favorites).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch existing favorites: %w", err)
	}
	logrus.Infof("Found %d existing favorites to preserve", len(favorites))

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

	// Clear favorites using batch processing
	favoriteItems := make([]interface{}, len(favorites))
	for i, f := range favorites {
		favoriteItems[i] = f
	}
	_, errors := dbHelper.ProcessBatch(favoriteItems, func(item interface{}) error {
		favorite := item.(models.FavoriteQuickstart)
		if err := dbHelper.ClearAssociation(&favorite, "Quickstart", "favorite", favorite.ID); err != nil {
			return err
		}
		return dbHelper.Delete(&favorite, "favorite", favorite.ID)
	}, "favorites cleanup")
	if len(errors) > 0 {
		return nil, fmt.Errorf("errors during favorites cleanup: %v", errors[0])
	}

	// Clear tags using batch processing
	allTags := append(staleQuickstartsTags, staleTopicsTags...)
	tagItems := make([]interface{}, len(allTags))
	for i, t := range allTags {
		tagItems[i] = t
	}
	_, errors = dbHelper.ProcessBatch(tagItems, func(item interface{}) error {
		tag := item.(models.Tag)
		if err := dbHelper.ClearAssociation(&tag, "Quickstarts", "tag", tag.ID); err != nil {
			return err
		}
		if err := dbHelper.ClearAssociation(&tag, "HelpTopics", "tag", tag.ID); err != nil {
			return err
		}
		return dbHelper.Delete(&tag, "tag", tag.ID)
	}, "tags cleanup")
	if len(errors) > 0 {
		return nil, fmt.Errorf("errors during tags cleanup: %v", errors[0])
	}

	// Clear quickstarts using batch processing
	quickstartItems := make([]interface{}, len(staleQuickstarts))
	for i, q := range staleQuickstarts {
		quickstartItems[i] = q
	}
	_, errors = dbHelper.ProcessBatch(quickstartItems, func(item interface{}) error {
		quickstart := item.(models.Quickstart)
		if err := dbHelper.ClearAssociation(&quickstart, "Tags", "quickstart", quickstart.Name); err != nil {
			return err
		}
		return dbHelper.Delete(&quickstart, "quickstart", quickstart.Name)
	}, "quickstarts cleanup")
	if len(errors) > 0 {
		return nil, fmt.Errorf("errors during quickstarts cleanup: %v", errors[0])
	}

	// Clear help topics using batch processing
	helpTopicItems := make([]interface{}, len(staleHelpTopics))
	for i, h := range staleHelpTopics {
		helpTopicItems[i] = h
	}
	_, errors = dbHelper.ProcessBatch(helpTopicItems, func(item interface{}) error {
		helpTopic := item.(models.HelpTopic)
		if err := dbHelper.ClearAssociation(&helpTopic, "Tags", "help topic", helpTopic.Name); err != nil {
			return err
		}
		return dbHelper.Delete(&helpTopic, "help topic", helpTopic.Name)
	}, "help topics cleanup")
	if len(errors) > 0 {
		return nil, fmt.Errorf("errors during help topics cleanup: %v", errors[0])
	}

	logrus.Info("Successfully cleared all old content")
	return favorites, nil
}

func SeedFavorites(favorites []models.FavoriteQuickstart) error {
	dbHelper := NewDBHelper(DB, "favorites-restorer")
	logrus.Infof("Starting to restore %d favorite quickstarts", len(favorites))
	
	favoriteItems := make([]interface{}, len(favorites))
	for i, f := range favorites {
		favoriteItems[i] = f
	}
	
	seedSuccess := 0
	ignoredFalse := 0
	notFound := 0

	_, errors := dbHelper.ProcessBatch(favoriteItems, func(item interface{}) error {
		favorite := item.(models.FavoriteQuickstart)
		var quickstart models.Quickstart
		result := DB.Where("name = ?", favorite.QuickstartName).First(&quickstart)
		
		if result.Error == nil && result.RowsAffected != 0 && favorite.Favorite {
			if err := dbHelper.Create(&favorite, "favorite", favorite.QuickstartName); err != nil {
				return err
			}
			seedSuccess++
			return nil
		} else if !favorite.Favorite {
			logrus.Debugf("Skipping unfavorite entry for: %s", favorite.QuickstartName)
			ignoredFalse++
			return nil
		} else {
			logrus.Warningf("Unable to restore favorite quickstart %s: quickstart not found (may have been renamed)", favorite.QuickstartName)
			notFound++
			return nil
		}
	}, "favorites restoration")

	logrus.Infof("Favorites restoration complete: %d restored, %d ignored (unfavorite), %d not found, %d errors", seedSuccess, ignoredFalse, notFound, len(errors))
	if len(errors) > 0 {
		return fmt.Errorf("errors during favorites restoration: %v", errors[0])
	}
	return nil
}

// processQuickstartPhase handles quickstart processing for the seeding operation
func processQuickstartPhase(template MetadataTemplate, defaultTags map[string]models.Tag, result *SeedingResult) error {
	logrus.Debugf("Processing QuickStart template: %s", template.Name)
	quickstart, isNew, err := seedQuickstart(template, defaultTags["quickstart"])
	if err != nil {
		errMsg := fmt.Sprintf("failed to seed quickstart %s: %v", template.Name, err)
		logrus.Error(errMsg)
		result.Errors = append(result.Errors, fmt.Errorf(errMsg))
		return fmt.Errorf(errMsg)
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
		return fmt.Errorf(errMsg)
	}

	// Process tags for this quickstart
	return processQuickstartTags(template, quickstart, result)
}

// processQuickstartTags handles tag processing for a quickstart
func processQuickstartTags(template MetadataTemplate, quickstart models.Quickstart, result *SeedingResult) error {
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
	return nil
}

// processHelpTopicPhase handles help topic processing for the seeding operation
func processHelpTopicPhase(template MetadataTemplate, defaultTags map[string]models.Tag, result *SeedingResult) error {
	logrus.Debugf("Processing HelpTopic template: %s", template.Name)
	helpTopics, createdCount, err := seedHelpTopic(template, defaultTags["helptopic"])
	if err != nil {
		errMsg := fmt.Sprintf("failed to seed help topic %s: %v", template.Name, err)
		logrus.Error(errMsg)
		result.Errors = append(result.Errors, fmt.Errorf(errMsg))
		return fmt.Errorf(errMsg)
	}

	result.HelpTopicsProcessed += len(helpTopics)
	result.HelpTopicsCreated += createdCount
	result.HelpTopicsUpdated += len(helpTopics) - createdCount

	// Process tags for help topics
	return processHelpTopicTags(template, helpTopics, result)
}

// processHelpTopicTags handles tag processing for help topics
func processHelpTopicTags(template MetadataTemplate, helpTopics []models.HelpTopic, result *SeedingResult) error {
	for _, helpTopic := range helpTopics {
		for _, tag := range template.Tags {
			var newTag models.Tag
			var originalTag models.Tag
			newTag.Type = models.TagType(tag.Kind)
			newTag.Value = tag.Value

			r := DB.Preload("HelpTopics").Where("type = ? AND value = ?", models.TagType(newTag.Type), newTag.Value).Find(&originalTag)
			if r.Error != nil {
				errMsg := fmt.Sprintf("failed to lookup tag %s=%s for help topic: %v", tag.Kind, tag.Value, r.Error)
				logrus.Error(errMsg)
				result.Errors = append(result.Errors, fmt.Errorf(errMsg))
				continue
			}
			
			if r.RowsAffected == 0 {
				// Create new tag
				if err := DB.Create(&newTag).Error; err != nil {
					errMsg := fmt.Sprintf("failed to create tag %s=%s for help topic: %v", tag.Kind, tag.Value, err)
					logrus.Error(errMsg)
					result.Errors = append(result.Errors, fmt.Errorf(errMsg))
					continue
				}
				originalTag = newTag
				result.TagsCreated++
			}

			// Create tag-help topic association
			if err := DB.Model(&originalTag).Association("HelpTopics").Append(&helpTopic); err != nil {
				errMsg := fmt.Sprintf("failed to create tag association %s=%s for help topic %s: %v", tag.Kind, tag.Value, helpTopic.Name, err)
				logrus.Error(errMsg)
				result.Errors = append(result.Errors, fmt.Errorf(errMsg))
				continue
			}

			helpTopic.Tags = append(helpTopic.Tags, originalTag)

			if err := DB.Save(&helpTopic).Error; err != nil {
				errMsg := fmt.Sprintf("failed to save help topic %s after adding tag: %v", helpTopic.Name, err)
				logrus.Error(errMsg)
				result.Errors = append(result.Errors, fmt.Errorf(errMsg))
				continue
			}

			if err := DB.Save(&originalTag).Error; err != nil {
				errMsg := fmt.Sprintf("failed to save tag %s=%s for help topic: %v", tag.Kind, tag.Value, err)
				logrus.Error(errMsg)
				result.Errors = append(result.Errors, fmt.Errorf(errMsg))
				continue
			}
		}
	}
	return nil
}

// logSeedingResults logs the final results of the seeding process
func logSeedingResults(result *SeedingResult) {
	logrus.Infof("=== SEEDING PROCESS COMPLETE ===")
	logrus.Infof("Quickstarts: %d processed (%d created, %d updated)", 
		result.QuickstartsProcessed, result.QuickstartsCreated, result.QuickstartsUpdated)
	logrus.Infof("Help Topics: %d processed (%d created, %d updated)", 
		result.HelpTopicsProcessed, result.HelpTopicsCreated, result.HelpTopicsUpdated)
	logrus.Infof("Tags: %d created", result.TagsCreated)
	logrus.Infof("Favorites: %d restored", result.FavoritesRestored)
	
	if len(result.Errors) > 0 {
		logrus.Errorf("Completed with %d errors", len(result.Errors))
		for i, err := range result.Errors {
			logrus.Errorf("Error %d: %v", i+1, err)
		}
	} else {
		logrus.Info("Completed successfully with no errors")
	}
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
