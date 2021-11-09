package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"fmt"

	"github.com/ghodss/yaml"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
)

type Swagger struct {
	Components openapi3.Components `json:"components,omitempty" yaml:"components,omitempty"`
}

// generates openapi schema
func main() {
	components := openapi3.NewComponents()
	components.Schemas = make(map[string]*openapi3.SchemaRef)

	quickstart, _, err := openapi3gen.NewSchemaRefForValue(&models.Quickstart{})
	checkErr(err)
	components.Schemas["v1.Quickstart"] = quickstart

	quickstartProgress, _, err := openapi3gen.NewSchemaRefForValue(&models.QuickstartProgress{})
	checkErr(err)
	components.Schemas["v1.QuickstartProgress"] = quickstartProgress

	swagger := Swagger{}
	swagger.Components = components
	checkErr(err)

	b := &bytes.Buffer{}
	err = json.NewEncoder(b).Encode(swagger)
	checkErr(err)

	schema, err := yaml.JSONToYAML(b.Bytes())
	checkErr(err)

	paths, err := ioutil.ReadFile("./cmd/spec/path.yaml")
	checkErr(err)

	b = &bytes.Buffer{}
	b.Write(schema)
	b.Write(paths)

	doc, err := openapi3.NewLoader().LoadFromData(b.Bytes())
	checkErr(err)

	jsonB, err := json.MarshalIndent(doc, "", "  ")
	checkErr(err)

	err = ioutil.WriteFile("./spec/openapi.json", jsonB, 0666)
	checkErr(err)
	err = ioutil.WriteFile("./spec/openapi.yaml", b.Bytes(), 0666)
	checkErr(err)

	fmt.Println("Spec was generated successfully")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
