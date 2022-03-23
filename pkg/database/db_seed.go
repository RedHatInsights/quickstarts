package database

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"regexp"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/ghodss/yaml"
	"github.com/sirupsen/logrus"
)

type TagTemplate struct {
	Kind  string
	Value string
}

type MetadataTemplate struct {
	Kind           string        `yaml:"kind"`
	Name           string        `yaml:"name"`
	Tags           []TagTemplate `yaml:"tags"`
	QuickstartPath string
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
	m := regexp.MustCompile("metadata.yml$")
	template.QuickstartPath = m.ReplaceAllString(loc, template.Name+".yml")
	return template, nil
}

func findTags() []MetadataTemplate {
	var MetadataTemplates []MetadataTemplate
	files, err := filepath.Glob("./docs/quickstarts/**/metadata.yml")
	if err != nil {
		log.Fatal(err)
	}

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

func seedQuickstart(t MetadataTemplate, defaultTag models.Tag) (models.Quickstart, error) {
	yamlfile, err := ioutil.ReadFile(t.QuickstartPath)
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

func SeedTags() {
	defaultTags := seedDefaultTags()
	MetadataTemplates := findTags()
	for _, template := range MetadataTemplates {
		quickstart, quickstartErr := seedQuickstart(template, defaultTags["quickstart"])
		if quickstartErr != nil {
			fmt.Println("Unable to seed quickstart: ", quickstartErr.Error(), template.QuickstartPath)
		}
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
			// Clear all tags associations
			err := DB.Model(&originalTag).Association("Quickstarts").Clear()
			if err != nil {
				fmt.Println("Failed clearing tags associations", err.Error())
			}

			// Create tags quickstarts associations
			err = DB.Model(&originalTag).Association("Quickstarts").Append(&quickstart)
			if err != nil {
				fmt.Println("Failed creating tags associations", err.Error())
			}

			DB.Save(&originalTag)
		}
	}
}
