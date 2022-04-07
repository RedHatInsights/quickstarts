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
	m := regexp.MustCompile("metadata.yml$")
	template.ContentPath = m.ReplaceAllString(loc, template.Name+".yml")
	return template, nil
}

func findTags() []MetadataTemplate {
	var MetadataTemplates []MetadataTemplate
	quickstartsFiles, err := filepath.Glob("./docs/quickstarts/**/metadata.yml")
	if err != nil {
		log.Fatal(err)
	}

	helpTopicsFiles, err := filepath.Glob("./docs/help-topics/**/metadata.yml")

	if err != nil {
		log.Fatal(err)
	}

	files := append(quickstartsFiles, helpTopicsFiles...)

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

func seedHelpTopic(t MetadataTemplate, defaultTag models.Tag) (models.HelpTopic, error) {
	fmt.Println(t)
	yamlfile, err := ioutil.ReadFile(t.ContentPath)
	var newHelpTopic models.HelpTopic
	var originalHelpTopic models.HelpTopic
	if err != nil {
		return newHelpTopic, err
	}

	jsonContent, err := yaml.YAMLToJSON(yamlfile)
	var data map[string]map[string]string
	json.Unmarshal(jsonContent, &data)
	name := t.Name
	r := DB.Where("name = ?", name).Find(&originalHelpTopic)

	if r.Error != nil {
		// check for DB error
		return newHelpTopic, err
	} else if r.RowsAffected == 0 {
		// Create new help topic
		newHelpTopic.Content = jsonContent
		newHelpTopic.Name = name
		DB.Create(&newHelpTopic)
		err = DB.Model(&defaultTag).Association("HelpTopics").Append(&newHelpTopic)
		if err != nil {
			fmt.Println("Failed creating help topic default tag associations", err.Error())
		}
		DB.Save(&defaultTag)
		return newHelpTopic, nil
	} else {
		// Update existing help topic
		originalHelpTopic.Content = jsonContent
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
		return originalHelpTopic, nil
	}
}

func SeedTags() {
	defaultTags := seedDefaultTags()
	MetadataTemplates := findTags()
	for _, template := range MetadataTemplates {
		kind := template.Kind
		if kind == "QuickStarts" {
			var quickstart models.Quickstart
			var quickstartErr error
			quickstart, quickstartErr = seedQuickstart(template, defaultTags["quickstart"])
			if quickstartErr != nil {
				fmt.Println("Unable to seed quickstart: ", quickstartErr.Error(), template.ContentPath)
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
