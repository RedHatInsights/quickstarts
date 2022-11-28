package main

import (
	"encoding/json"
	"errors"
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

type QuickstartMetadata struct {
	Name                  string `json:"name,omitempty"`
	ExternalDocumentation bool   `json:"externalDocumentation,omitempty"`
}

type TypeField struct {
	Text  string `json:"text,omitempty"`
	Color string `json:"color,omitempty"`
}

type LinkType struct {
	Href string `json:"href,omitempty"`
	Text string `json:"text,omitempty"`
}

type SpecStruct struct {
	Version     float32   `json:"version,omitempty"`
	Type        TypeField `json:"type,omitempty"`
	DisplayName string    `json:"displayName,omitempty"`
	Icon        string    `json:"icon,omitempty"`
	Description string    `json:"description,omitempty"`
	Link        LinkType  `json:"link,omitempty"`
}

type QuickStarts struct {
	ApiVersion string             `json:"apiVersion,omitempty"`
	Kind       string             `json:"kind,omitempty"`
	Metadata   QuickstartMetadata `json:"metadata,omitempty"`
	Spec       SpecStruct         `json:"spec,omitempty"`
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

func validateQuickStartStructure() {
	metadataFiles, err := filepath.Glob("./docs/quickstarts/**/metadata.y*")
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
			validation.Field(&metadata.Kind, validation.Required, validation.In("QuickStarts")),
		)
		handleFileErr(filePath, err)

		m := regexp.MustCompile("metadata.ya?ml$")
		quickstartsFileName := filePath

		if _, err = os.Stat(m.ReplaceAllString(quickstartsFileName, metadata.Name+".yml")); err == nil {
			quickstartsFileName = m.ReplaceAllString(quickstartsFileName, metadata.Name+".yml")
		} else {
			quickstartsFileName = m.ReplaceAllString(quickstartsFileName, metadata.Name+".yaml")
		}
		yamlfile, err = ioutil.ReadFile(quickstartsFileName)
		handleFileErr(quickstartsFileName, err)
		jsonContent, err = yaml.YAMLToJSON(yamlfile)
		handleFileErr(quickstartsFileName, err)

		var content QuickStarts
		err = json.Unmarshal(jsonContent, &content)
		handleFileErr(quickstartsFileName, err)

		err = validation.ValidateStruct(&content,
			validation.Field(&content.Kind, validation.Required, validation.In("QuickStarts")),
			validation.Field(&content.ApiVersion, validation.Required),
		)

		var spec = content.Spec
		err = validation.ValidateStruct(&spec,
			validation.Field(&spec.Version, validation.Required),
			validation.Field(&spec.DisplayName, validation.Required),
			validation.Field(&spec.Icon, validation.Required),
			validation.Field(&spec.Description, validation.Required),
		)

		var link = content.Spec.Link
		err = validation.ValidateStruct(&link,
			validation.Field(&link.Href, validation.Required),
			validation.Field(&link.Text, validation.Required),
		)

		var specType = content.Spec.Type
		err = validation.ValidateStruct(&specType,
			validation.Field(&specType.Color, validation.Required),
			validation.Field(&specType.Text, validation.Required),
		)

	}

}

func main() {
	// Validate help topics yaml files
	fmt.Println("Validating help topics")
	validateStructure()
	fmt.Println("Validating quickstarts")
	validateQuickStartStructure()
}
