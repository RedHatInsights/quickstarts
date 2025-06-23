package resourceselector

import (
	"encoding/json"
	"fmt"

	"github.com/invopop/jsonschema"
	ollamaApi "github.com/ollama/ollama/api"
)

type ToolFunctionParameters struct {
	Type       string                  `json:"type"`
	Defs       any                     `json:"$defs,omitempty"`
	Items      any                     `json:"items,omitempty"`
	Required   []string                `json:"required"`
	Properties map[string]ToolProperty `json:"properties"`
}

type ToolProperty struct {
	Type        []string `json:"type"`
	Items       any      `json:"items,omitempty"`
	Description string   `json:"description"`
	Enum        []any    `json:"enum,omitempty"`
}

func generateOllamaToolSchema(toolName, toolDescription string, argsStruct interface{}) (ollamaApi.Tool, error) {
	reflector := jsonschema.Reflector{}
	schema := reflector.Reflect(argsStruct)

	// Convert the jsonschema.Schema object to a generic map[string]interface{}
	// This intermediate step is necessary because ollamaApi.ToolFunction.Parameters
	// expects a specific struct, not the *jsonschema.Schema type directly.
	schemaBytes, err := json.Marshal(schema)
	if err != nil {
		return ollamaApi.Tool{}, fmt.Errorf("failed to marshal generated JSON schema: %w", err)
	}

	var genericSchemaMap map[string]interface{}
	if err := json.Unmarshal(schemaBytes, &genericSchemaMap); err != nil {
		return ollamaApi.Tool{}, fmt.Errorf("failed to unmarshal JSON schema to map: %w", err)
	}

	// Now, manually map fields from the generic map to the specific ollamaApi.ToolFunction.Parameters struct
	ollamaParams := ToolFunctionParameters{} // Use the named type if available, otherwise the anonymous struct literal

	if typ, ok := genericSchemaMap["type"].(string); ok {
		ollamaParams.Type = typ
	} else {
		return ollamaApi.Tool{}, fmt.Errorf("schema missing 'type' field or not string")
	}

	if defs, ok := genericSchemaMap["$defs"]; ok {
		ollamaParams.Defs = defs
	}
	if items, ok := genericSchemaMap["items"]; ok {
		ollamaParams.Items = items
	}

	if required, ok := genericSchemaMap["required"].([]interface{}); ok {
		// required is []interface{}, convert to []string
		stringRequired := make([]string, len(required))
		for i, v := range required {
			if s, ok := v.(string); ok {
				stringRequired[i] = s
			} else {
				return ollamaApi.Tool{}, fmt.Errorf("required field contains non-string value")
			}
		}
		ollamaParams.Required = stringRequired
	}
	// Note: If 'required' is optional in your schema, this 'else' would be different.
	// For now, assume if it exists, it's []interface{}.

	if properties, ok := genericSchemaMap["properties"].(map[string]interface{}); ok {
		ollamaParams.Properties = make(map[string]ToolProperty)
		for propName, propValue := range properties {
			if propMap, ok := propValue.(map[string]interface{}); ok {
				var toolProp ToolProperty
				if typ, ok := propMap["type"].([]interface{}); ok {
					toolProp.Type = make([]string, len(typ))
					for i, v := range typ {
						if s, ok := v.(string); ok {
							toolProp.Type[i] = s
						} else {
							return ollamaApi.Tool{}, fmt.Errorf("property '%s' type field contains non-string value", propName)
						}
					}
				} else {
					return ollamaApi.Tool{}, fmt.Errorf("property '%s' type field is not an array", propName)
				}
				if desc, ok := propMap["description"].(string); ok {
					toolProp.Description = desc
				}
				if items, ok := propMap["items"]; ok {
					toolProp.Items = items
				}
				if enum, ok := propMap["enum"].([]interface{}); ok {
					toolProp.Enum = enum
				}
				ollamaParams.Properties[propName] = toolProp
			}
		}
	}

	return ollamaApi.Tool{
		Type: "function",
		Function: ollamaApi.ToolFunction{
			Name:        toolName,
			Description: toolDescription,
			Parameters: struct {
				Type       string   `json:"type"`
				Defs       any      `json:"$defs,omitempty"`
				Items      any      `json:"items,omitempty"`
				Required   []string `json:"required"`
				Properties map[string]struct {
					Type        ollamaApi.PropertyType `json:"type"`
					Items       any                    `json:"items,omitempty"`
					Description string                 `json:"description"`
					Enum        []any                  `json:"enum,omitempty"`
				} `json:"properties"`
			}{
				Type:     ollamaParams.Type,
				Defs:     ollamaParams.Defs,
				Items:    ollamaParams.Items,
				Required: ollamaParams.Required,
				Properties: func() map[string]struct {
					Type        ollamaApi.PropertyType `json:"type"`
					Items       any                    `json:"items,omitempty"`
					Description string                 `json:"description"`
					Enum        []any                  `json:"enum,omitempty"`
				} {
					props := make(map[string]struct {
						Type        ollamaApi.PropertyType `json:"type"`
						Items       any                    `json:"items,omitempty"`
						Description string                 `json:"description"`
						Enum        []any                  `json:"enum,omitempty"`
					})
					for k, v := range ollamaParams.Properties {
						// Convert []string to ollamaApi.PropertyType (which may be string or []string, adjust as needed)
						var propType ollamaApi.PropertyType
						if len(v.Type) == 1 {
							propType = ollamaApi.PropertyType([]string{v.Type[0]})
						} else {
							propType = ollamaApi.PropertyType(v.Type)
						}
						props[k] = struct {
							Type        ollamaApi.PropertyType `json:"type"`
							Items       any                    `json:"items,omitempty"`
							Description string                 `json:"description"`
							Enum        []any                  `json:"enum,omitempty"`
						}{
							Type:        propType,
							Items:       v.Items,
							Description: v.Description,
							Enum:        v.Enum,
						}
					}
					return props
				}(),
			},
		},
	}, nil
}
