package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

type TagTemplate struct {
	Kind     string
	Value    string
	Priority *int
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
		log.Fatal(err)
	}

	helpTopicsFiles, err := filepath.Glob(path + "/docs/help-topics/**/metadata.y*")
	if err != nil {
		log.Fatal(err)
	}

	files := append(quickstartsFiles, helpTopicsFiles...)

	println(files)

	for _, file := range files {
		tagMetadata, err := readMetadata(file)
		if err != nil {
			logrus.Warningln(err.Error(), file)
		} else {
			MetadataTemplates = append(MetadataTemplates, tagMetadata)
		}
	}

	return MetadataTemplates
}

func makeQuickstartPrioritiesMap(tags []TagTemplate) (out map[string]int) {
	out = make(map[string]int)

	for _, tag := range tags {
		if tag.Kind == string(models.BundleTag) && tag.Priority != nil {
			out[tag.Value] = *tag.Priority
		}
	}

	return out
}

func quickstartMetadata(quickstartData map[string]interface{}) (map[string]interface{}, error) {
	rawMetadata, ok := quickstartData["metadata"]

	if !ok {
		return nil, fmt.Errorf("expected quickstart to contain metadata")
	}

	metadata, ok := rawMetadata.(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("expected quickstart metadata to be an object, got %v", metadata)
	}

	return metadata, nil
}

func quickstartName(metadata map[string]interface{}) (string, error) {
	rawName, ok := metadata["name"]

	if !ok {
		return "", fmt.Errorf("expected quickstart metadata to contain a name")
	}

	name, ok := rawName.(string)

	if !ok {
		return "", fmt.Errorf("expected quickstart metadata.name to be a string got %v", name)
	}

	return name, nil
}

func seedQuickstart(t MetadataTemplate, defaultTag models.Tag, priorities map[string]int) (models.Quickstart, error) {
	var newQuickstart models.Quickstart
	var originalQuickstart models.Quickstart

	yamlfile, err := ioutil.ReadFile(t.ContentPath)

	if err != nil {
		return newQuickstart, err
	}

	var quickstartData map[string]interface{}
	err = yaml.Unmarshal(yamlfile, &quickstartData)

	if err != nil {
		return newQuickstart, err
	}

	metadata, err := quickstartMetadata(quickstartData)

	if err != nil {
		return newQuickstart, err
	}

	name, err := quickstartName(metadata)

	if err != nil {
		return newQuickstart, err
	}

	if len(priorities) > 0 {
		metadata["bundle_priority"] = priorities
	}

	jsonContent, err := json.Marshal(quickstartData)

	if err != nil {
		return newQuickstart, err
	}

	r := DB.Where("name = ?", name).Find(&originalQuickstart)

	if r.Error != nil {
		// check for DB error
		return newQuickstart, r.Error
	} else if r.RowsAffected == 0 {
		// Create new quickstart
		newQuickstart.Content = jsonContent
		newQuickstart.Name = name
		DB.Create(&newQuickstart)
		err = DB.Model(&defaultTag).Association("Quickstarts").Append(&newQuickstart)
		if err != nil {
			fmt.Println("Failed creating quickstarts default tag associations", err.Error())
		}
		DB.Save(&defaultTag)
		return newQuickstart, nil
	} else {
		// Update existing quickstart
		originalQuickstart.Content = jsonContent
		// Clear all tags associations
		err := DB.Model(&originalQuickstart).Association("Tags").Clear()
		if err != nil {
			fmt.Println("Failed clearing quickstarts tags associations", err.Error())
		}
		DB.Save(&originalQuickstart)
		err = DB.Model(&defaultTag).Association("Quickstarts").Append(&originalQuickstart)
		if err != nil {
			fmt.Println("Failed creating quickstarts default tag associations", err.Error())
		}
		DB.Save(&defaultTag)
		return originalQuickstart, nil
	}
}

func seedDefaultTags() map[string]models.Tag {
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
		fmt.Println("Unable to create quickstarts kind tag!")
	}

	err = DB.Where("type = ? AND value = ?", &helpTopicKindTag.Type, &helpTopicKindTag.Value).FirstOrCreate(&helpTopicKindTag).Error
	if err != nil {
		fmt.Println("Unable to create help topic kind tag!")
	}

	DB.Save(&quickstartsKindTag)
	DB.Save(&helpTopicKindTag)

	result := make(map[string]models.Tag)
	result["quickstart"] = quickstartsKindTag
	result["helptopic"] = helpTopicKindTag

	return result
}

func seedHelpTopic(t MetadataTemplate, defaultTag models.Tag) ([]models.HelpTopic, error) {
	yamlfile, err := ioutil.ReadFile(t.ContentPath)
	returnValue := make([]models.HelpTopic, 0)
	if err != nil {
		return returnValue, err
	}

	jsonContent, err := yaml.YAMLToJSON(yamlfile)
	var d []map[string]interface{}
	if err := json.Unmarshal(jsonContent, &d); err != nil {
		return returnValue, err
	}

	for _, c := range d {
		var newHelpTopic models.HelpTopic
		var originalHelpTopic models.HelpTopic
		name := c["name"]
		r := DB.Where("name = ?", name).Find(&originalHelpTopic)

		if r.Error != nil {
			// check for DB error
			return returnValue, err
		} else if r.RowsAffected == 0 {
			// Create new help topic
			newHelpTopic.GroupName = t.Name
			newHelpTopic.Content, err = json.Marshal(c)
			if err != nil {
				return returnValue, err
			}
			newHelpTopic.Name = fmt.Sprintf("%v", name)
			DB.Create(&newHelpTopic)
			err = DB.Model(&defaultTag).Association("HelpTopics").Append(&newHelpTopic)
			if err != nil {
				fmt.Println("Failed creating help topic default tag associations", err.Error())
			}
			DB.Save(&defaultTag)
			returnValue = append(returnValue, newHelpTopic)
		} else {
			// Update existing help topic
			originalHelpTopic.Content, err = json.Marshal(c)
			originalHelpTopic.GroupName = t.Name
			if err != nil {
				return returnValue, err
			}
			// Clear all tags associations
			err := DB.Model(&originalHelpTopic).Association("Tags").Clear()
			if err != nil {
				fmt.Println("Failed clearing quickstarts tags associations", err.Error())
			}
			DB.Save(&originalHelpTopic)
			err = DB.Model(&defaultTag).Association("HelpTopics").Append(&originalHelpTopic)
			if err != nil {
				fmt.Println("Failed creating help topic default tag associations", err.Error())
			}
			DB.Save(&defaultTag)
			returnValue = append(returnValue, originalHelpTopic)
		}
	}
	return returnValue, nil
}

func clearOldContent() []models.FavoriteQuickstart {
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

	// Remove any left-over links between quickstarts and their tags.

	var staleQuickStartLinks []models.QuickstartTag
	DB.Model(&models.QuickstartTag{}).Find(&staleQuickStartLinks)

	for _, link := range staleQuickStartLinks {
		DB.Unscoped().Delete(&link)
	}

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
			logrus.Warningln("Unable to seed favorite quickstart: ", result.Error.Error(), favorite.QuickstartName)
		}
	}

	logrus.Infof("Seeded %d out of %d favorites. Ignored %d unfavorite entries. Could not find %d quickstarts (possible cause quickstart was renamed).", seedSuccess, len(favorites), ignoredFalse, len(favorites)-seedSuccess-ignoredFalse)
}

func SeedTags() {
	// clear old content pahse
	favorites := clearOldContent()
	// seeding phase
	defaultTags := seedDefaultTags()
	MetadataTemplates := findTags()

	for _, template := range MetadataTemplates {
		kind := template.Kind
		if kind == "QuickStarts" {
			var quickstart models.Quickstart
			var quickstartErr error
			var tags []models.Tag
			quickstart, quickstartErr = seedQuickstart(template, defaultTags["quickstart"], makeQuickstartPrioritiesMap(template.Tags))
			if quickstartErr != nil {
				fmt.Println("Unable to seed quickstart: ", quickstartErr.Error(), template.ContentPath)
			}
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
					fmt.Println("Error: ", r.Error.Error())
				} else if r.RowsAffected == 0 {
					DB.Create(&newTag)
					originalTag = newTag
				}

				newLink := models.QuickstartTag{QuickstartID: quickstart.ID, TagID: originalTag.ID}

				if newTag.Type == models.BundleTag {
					newLink.Priority = tag.Priority
				} else if tag.Priority != nil {
					logrus.Warningln("Unexpected priority for non-bundle tag in file", template.ContentPath)
				}

				err := DB.Create(&newLink).Error

				if err != nil {
					fmt.Println("Failed creating tags associations", err.Error())
				}

				originalTag.Quickstarts = append(originalTag.Quickstarts, quickstart)
				quickstart.Tags = append(quickstart.Tags, originalTag)

				DB.Save(&quickstart)
				DB.Save(&originalTag)
			}
		}

		if kind == "HelpTopic" {
			helpTopic, helpTopicErr := seedHelpTopic(template, defaultTags["helptopic"])
			if helpTopicErr != nil {
				fmt.Println("Unable to seed help topic: ", helpTopicErr.Error(), template.ContentPath)
			}

			for _, tag := range template.Tags {
				var newTag models.Tag
				var originalTag models.Tag
				newTag.Type = models.TagType(tag.Kind)
				newTag.Value = tag.Value

				r := DB.Preload("HelpTopics").Where("type = ? AND value = ?", models.TagType(newTag.Type), newTag.Value).Find(&originalTag)
				if r.Error != nil {
					fmt.Println("Error: ", r.Error.Error())
				} else if r.RowsAffected == 0 {
					DB.Create(&newTag)
					originalTag = newTag
				}
				// Clear all tags associations
				err := DB.Model(&originalTag).Association("HelpTopics").Clear()
				if err != nil {
					fmt.Println("Failed clearing tags associations", err.Error())
				}

				// Create tags help topic associations
				err = DB.Model(&originalTag).Association("HelpTopics").Append(&helpTopic)
				if err != nil {
					fmt.Println("Failed creating tags associations", err.Error())
				}

				DB.Save(&originalTag)
			}
		}
	}

	SeedFavorites(favorites)
}
