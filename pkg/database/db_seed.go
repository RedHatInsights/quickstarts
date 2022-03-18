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

func seedQuickstart(t MetadataTemplate) (models.Quickstart, error) {
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
		return newQuickstart, nil
	} else {
		// Update existing quickstart
		originalQuickstart.Content = jsonContent
		DB.Save(&originalQuickstart)
		return originalQuickstart, nil
	}
}

func SeedTags() {
	MetadataTemplates := findTags()
	for _, template := range MetadataTemplates {
		quickstart, quickstartErr := seedQuickstart(template)
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
