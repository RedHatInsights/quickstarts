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

	quickstart, _, err := openapi3gen.NewSchemaRefForValue(&models.Quickstart{}, openapi3gen.SchemaCustomizer(func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		if name == "content" {
			schema.Type = "object"
		}
		if name == "deletedAt" {
			schema.Type = "string"
			schema.Format = "date-time"
			schema.OneOf = append(schema.OneOf, &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: "null",
			}}, &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: "string",
			}})
		}
		return nil
	}))
	checkErr(err)
	components.Schemas["v1.Quickstart"] = quickstart

	quickstartProgress, _, err := openapi3gen.NewSchemaRefForValue(&models.QuickstartProgress{})
	checkErr(err)
	components.Schemas["v1.QuickstartProgress"] = quickstartProgress

	favoriteQuickstart, _, err := openapi3gen.NewSchemaRefForValue(&models.FavoriteQuickstart{})
	checkErr(err)
	components.Schemas["v1.FavoriteQuickstart"] = favoriteQuickstart

	helpTopic, _, err := openapi3gen.NewSchemaRefForValue(&models.HelpTopic{}, openapi3gen.SchemaCustomizer(func(name string, t reflect.Type, tag reflect.StructTag, schema *openapi3.Schema) error {
		if name == "content" {
			schema.Type = "object"
		}
		if name == "deletedAt" {
			schema.Type = "string"
			schema.Format = "date-time"
			schema.OneOf = append(schema.OneOf, &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: "null",
			}}, &openapi3.SchemaRef{Value: &openapi3.Schema{
				Type: "string",
			}})
		}
		return nil
	}))
	checkErr(err)
	components.Schemas["v1.HelpTopic"] = helpTopic

	badRequest, _, err := openapi3gen.NewSchemaRefForValue(&models.BadRequest{})
	checkErr(err)
	components.Schemas["BadRequest"] = badRequest

	notFound, _, err := openapi3gen.NewSchemaRefForValue(&models.NotFound{})
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
