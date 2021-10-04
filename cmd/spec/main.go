package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"

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

	doc, err := openapi3.NewLoader().LoadFromData(b.Bytes())
	checkErr(err)

	jsonB, err := doc.MarshalJSON()
	checkErr(err)

	err = ioutil.WriteFile("./spec/openapi.json", jsonB, 0666)
	checkErr(err)
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
