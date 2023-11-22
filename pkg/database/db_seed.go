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
	Kind  string
	Value string
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

func seedFavoriteQuickstart() (models.FavoriteQuickstart, error) {

	mc := make(map[string]string)
	mc["foo"] = "bar"
	content, _ := json.Marshal(mc)

	qs := models.Quickstart{
		Name:    "fooboo",
		Content: content,
	}

	favQuickstart := models.FavoriteQuickstart{
		AccountId:      "123",
		QuickstartName: qs.Name,
		Favorite:       true,
	}

	qs.FavoriteQuickstart = append(qs.FavoriteQuickstart, favQuickstart)

	DB.Create(&qs)
	return favQuickstart, nil
}

func seedQuickstart(t MetadataTemplate, defaultTag models.Tag) (models.Quickstart, error) {
	yamlfile, err := ioutil.ReadFile(t.ContentPath)
	var newQuickstart models.Quickstart
	var originalQuickstart models.Quickstart
	if err != nil {
		return newQuickstart, err
	}

	jsonContent, err := yaml.YAMLToJSON(yamlfile)
	var data map[string]map[string]string
	json.Unmarshal(jsonContent, &data)
	name := data["metadata"]["name"]
	r := DB.Where("name = ?", name).Find(&originalQuickstart)
	if r.Error != nil {
		// check for DB error
		return newQuickstart, err
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

func clearOldContent() {
	var staleQuickstartsTags []models.Tag
	var staleTopicsTags []models.Tag

	var staleQuickstarts []models.Quickstart
	var staleHelpTopics []models.HelpTopic
	DB.Model(&models.Quickstart{}).Find(&staleQuickstarts)
	DB.Model(&models.HelpTopic{}).Find(&staleHelpTopics)

	DB.Preload("Quickstarts").Find(&staleQuickstartsTags)
	DB.Preload("HelpTopics").Find(&staleTopicsTags)

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
}

func SeedTags() {
	// clear old content pahse
	clearOldContent()
	// seeding phase
	defaultTags := seedDefaultTags()
	MetadataTemplates := findTags()

	// var favQuickstart models.FavoriteQuickstart
	var favQuickstartError error

	_, favQuickstartError = seedFavoriteQuickstart()
	if favQuickstartError != nil {
		fmt.Println("Unable to seed favoriteQuickstart: ", favQuickstartError.Error())
	}

	// DB.Save(&favQuickstart)

	for _, template := range MetadataTemplates {
		kind := template.Kind
		if kind == "QuickStarts" {
			var quickstart models.Quickstart
			var quickstartErr error
			var tags []models.Tag
			quickstart, quickstartErr = seedQuickstart(template, defaultTags["quickstart"])
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

				// Create tags quickstarts associations
				err := DB.Model(&originalTag).Association("Quickstarts").Append(&quickstart)
				if err != nil {
					fmt.Println("Failed creating tags associations", err.Error())
				}

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
}
