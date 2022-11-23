package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ghodss/yaml"
	validation "github.com/go-ozzo/ozzo-validation"
)

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
	Version     int       `json:"version,omitempty"`
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

// // add func for validating quickstarts
// func validateQuickStartStructure() {
// 	metadataFiles, err := filepath.Glob("./docs/quickstarts/**/metadata.y*")
// 	handleErr(err)

// 	// loop through metadatafiles
// 	//		grab template file names
// 	//loop through all template files and validate structs (basically the loop below but with template instead of metadata)

// 	for _, filePath := range metadataFiles {
// 		yamlfile, err := ioutil.ReadFile(filePath)
// 		handleFileErr(filePath, err)
// 		jsonContent, err := yaml.YAMLToJSON(yamlfile)
// 		handleFileErr(filePath, err)
// 		var quickstarts QuickStarts
// 		err = json.Unmarshal(jsonContent, &quickstarts)
// 		handleFileErr(filePath, err)
// 		err = validation.ValidateStruct(&quickstarts,
// 			validation.Field(&quickstarts.Kind, validation.Required, validation.In("QuickStarts")),
// 			// validation.Field(&metadata.Spec, validation.Required, validation.)
// 		)
// 		handleFileErr(filePath, err)

// 		metadataStruct := quickstarts.Metadata
// 		err = validation.ValidateStruct(&metadataStruct,
// 			validation.Field(&metadataStruct.Name, validation.Required, validation.By(notMatch(`\s`, "name can't include whitespaces"))),
// 			validation.Field(&metadataStruct.ExternalDocumentation, validation.Required, validation.In(true, false)),
// 		)
// 		handleFileErr(filePath, err)

// 		// spec := metadata.Spec
// 		// err = validation.ValidateStruct(&spec,
// 		// 	validation.Field(&spec.))
// 	}

// }

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
	}

}
