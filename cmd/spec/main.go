package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"reflect"

	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3gen"
	"github.com/ghodss/yaml"
)

type Swagger struct {
	Components openapi3.Components `json:"components,omitempty" yaml:"components,omitempty"`
}

// generates openapi schema
func main() {
	components := openapi3.NewComponents()
	components.Schemas = make(map[string]*openapi3.SchemaRef)

	quickstart, err := openapi3gen.NewSchemaRefForValue(&models.Quickstart{}, components.Schemas, openapi3gen.SchemaCustomizer(func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		if name == "content" {
			schema.Type = &openapi3.Types{"object"}
		}
		if name == "deletedAt" {
			schema.Type = &openapi3.Types{"string"}
			schema.Format = "date-time"
			schema.OneOf = append(schema.OneOf, &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: &openapi3.Types{"null"},
			}}, &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
			}})
		}
		return nil
	}))
	checkErr(err)
	components.Schemas["v1.Quickstart"] = quickstart

	quickstartProgress, err := openapi3gen.NewSchemaRefForValue(&models.QuickstartProgress{}, components.Schemas)
	checkErr(err)
	components.Schemas["v1.QuickstartProgress"] = quickstartProgress

	favoriteQuickstart, err := openapi3gen.NewSchemaRefForValue(&models.FavoriteQuickstart{}, components.Schemas)
	checkErr(err)
	components.Schemas["v1.FavoriteQuickstart"] = favoriteQuickstart

	helpTopic, err := openapi3gen.NewSchemaRefForValue(&models.HelpTopic{}, components.Schemas, openapi3gen.SchemaCustomizer(func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		if name == "content" {
			schema.Type = &openapi3.Types{"object"}
		}
		if name == "deletedAt" {
			schema.Type = &openapi3.Types{"string"}
			schema.Format = "date-time"
			schema.OneOf = append(schema.OneOf, &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: &openapi3.Types{"null"},
			}}, &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: &openapi3.Types{"string"},
			}})
		}
		return nil
	}))
	checkErr(err)
	components.Schemas["v1.HelpTopic"] = helpTopic

	badRequest, err := openapi3gen.NewSchemaRefForValue(&models.BadRequest{}, components.Schemas)
	checkErr(err)
	components.Schemas["BadRequest"] = badRequest

	notFound, err := openapi3gen.NewSchemaRefForValue(&models.NotFound{}, components.Schemas)
	checkErr(err)
	components.Schemas["NotFound"] = notFound

	swagger := Swagger{}
	swagger.Components = components
	checkErr(err)

	b := &bytes.Buffer{}
	err = json.NewEncoder(b).Encode(swagger)
	checkErr(err)

	parameters, err := ioutil.ReadFile("./cmd/spec/parameters.yaml")
	checkErr(err)

	schema, err := yaml.JSONToYAML(b.Bytes())
	checkErr(err)

	schema = append(schema, parameters...)
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
