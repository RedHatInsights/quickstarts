package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ghodss/yaml"
	validation "github.com/go-ozzo/ozzo-validation"
)

type TopicTag struct {
	Kind  string `json:"kind,omitempty"`
	Value string `json:"value,omitempty"`
}

type TopicMetadata struct {
	Kind string     `json:"kind,omitempty"`
	Name string     `json:"name,omitempty"`
	Tags []TopicTag `json:"tags,omitempty"`
}

type TopicContent struct {
	Name    string   `json:"name,omitempty"`
	Content string   `json:"content,omitempty"`
	Title   string   `json:"title,omitempty"`
	Tags    []string `json:"tags,omitempty"`
}

func validateStructure() {
	metadataFiles, err := filepath.Glob("./docs/help-topics/**/metadata.y*")
	handleErr(err)

	for _, filePath := range metadataFiles {
		yamlfile, err := ioutil.ReadFile(filePath)
		handleFileErr(filePath, err)
		jsonContent, err := yaml.YAMLToJSON(yamlfile)
		handleFileErr(filePath, err)
		var metadata TopicMetadata
		err = json.Unmarshal(jsonContent, &metadata)
		handleFileErr(filePath, err)
		err = validation.ValidateStruct(&metadata,
			validation.Field(&metadata.Kind, validation.Required, validation.In("HelpTopic")),
			validation.Field(&metadata.Name, validation.Required, validation.By(notMatch(`\s`, "name can't include whitespaces"))),
			validation.Field(&metadata.Tags, validation.Each(validation.Required)),
		)
		handleFileErr(filePath, err)

		for _, tag := range metadata.Tags {
			err = validation.ValidateStruct(&tag,
				validation.Field(&tag.Kind, validation.Required, validation.In("bundle", "application")),
				validation.Field(&tag.Value, validation.Required),
			)
			handleFileErr(filePath, err)
		}

		// validate topic file existance
		m := regexp.MustCompile("metadata.ya?ml$")
		topicFileName := filePath
		if _, err := os.Stat(m.ReplaceAllString(topicFileName, metadata.Name+".yml")); err == nil {
			topicFileName = m.ReplaceAllString(topicFileName, metadata.Name+".yml")
		} else {
			topicFileName = m.ReplaceAllString(topicFileName, metadata.Name+".yaml")
		}
		yamlfile, err = ioutil.ReadFile(topicFileName)
		handleFileErr(topicFileName, err)
		jsonContent, err = yaml.YAMLToJSON(yamlfile)
		handleFileErr(topicFileName, err)
		var content []TopicContent
		err = json.Unmarshal(jsonContent, &content)
		handleFileErr(topicFileName, err)

		for _, c := range content {
			err = validation.ValidateStruct(&c,
				validation.Field(&c.Name, validation.Required, validation.By(notMatch(`\s`, "name can't include whitespaces"))),
				validation.Field(&c.Content, validation.Required),
				validation.Field(&c.Title, validation.Required),
				validation.Field(&c.Tags, validation.Each(validation.Required)),
			)
			handleFileErr(topicFileName, err)
		}
	}
}

func main() {
	// Validate help topics yaml files
	fmt.Println("Validating help topics")
	validateStructure()
	fmt.Println("Validating quickstarts")
	validateQuickStartStructure()
}
