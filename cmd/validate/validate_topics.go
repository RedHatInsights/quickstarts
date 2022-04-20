package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
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

func handleErr(err error) {
	if err != nil {
		panic(err)
	}
}

func handleFileErr(filepath string, err error) {
	if err != nil {
		fmt.Println("Help topic validation error. Error occured in: ", filepath)
		handleErr(err)
	}
}

func notMatch(r string, msg string) validation.RuleFunc {
	return func(value interface{}) error {
		s := value.(string)
		matched, err := regexp.MatchString(r, s)
		if err != nil {
			return nil
		}

		if matched {
			return errors.New(msg)
		}
		return nil
	}
}

func validateStructure() {
	metadataFiles, err := filepath.Glob("./docs/help-topics/**/metadata.yaml")
	handleErr(err)
	if len(metadataFiles) > 0 {
		err = fmt.Errorf("yaml extenstions are not supported. Please use yml extenstions for: %v", metadataFiles)
		handleErr(err)
	}

	metadataFiles, err = filepath.Glob("./docs/help-topics/**/metadata.yml")
	handleErr(err)

	for _, filePath := range metadataFiles {
		yamlfile, err := ioutil.ReadFile(filePath)
		handleFileErr(filePath, err)
		jsonContent, err := yaml.YAMLToJSON(yamlfile)
		handleFileErr(filePath, err)
		var metadata TopicMetadata
		json.Unmarshal(jsonContent, &metadata)
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
	}
}

func main() {
	// Validate help topics yaml files
	fmt.Println("Validating help topics")
	validateStructure()

}
