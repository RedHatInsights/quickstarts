package resourceselector

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	ollamaApi "github.com/ollama/ollama/api"
	"github.com/sirupsen/logrus"
)

// Example of how to add a new tool to the system
// This demonstrates the pattern for adding more tools

// ReverseStringToolArgs represents arguments for the reverse string tool
type ReverseStringToolArgs struct {
	Text string `json:"text" jsonschema:"required,description=Text to reverse"`
}

// ReverseStringTool reverses a given string
func ReverseStringTool(ctx context.Context, args ReverseStringToolArgs) (*ToolResponse, error) {
	logrus.Infoln("Reverse string tool called with args:", args)

	// Reverse the string
	runes := []rune(args.Text)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	reversed := string(runes)

	message := fmt.Sprintf("Reversed '%s' to '%s'", args.Text, reversed)
	return NewToolResponse(message), nil
}

// ReverseStringToolHandler wraps ReverseStringTool to match ToolHandler signature
func ReverseStringToolHandler(ctx context.Context, args json.RawMessage) (*ToolResponse, error) {
	var reverseArgs ReverseStringToolArgs
	if err := json.Unmarshal(args, &reverseArgs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reverse string args: %w", err)
	}
	return ReverseStringTool(ctx, reverseArgs)
}

// CreateReverseStringToolSchema creates the Ollama tool schema for the reverse string tool
func CreateReverseStringToolSchema() ollamaApi.ToolFunction {
	return ollamaApi.ToolFunction{
		Name:        "reverseString",
		Description: "Reverses a given string and returns the result.",
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
			Type:     "object",
			Required: []string{"text"},
			Properties: map[string]struct {
				Type        ollamaApi.PropertyType `json:"type"`
				Items       any                    `json:"items,omitempty"`
				Description string                 `json:"description"`
				Enum        []any                  `json:"enum,omitempty"`
			}{
				"text": {
					Type:        ollamaApi.PropertyType{"string"},
					Description: "The text to reverse.",
				},
			},
		},
	}
}

// RegisterExampleTools shows how to register additional tools
// This function can be called from init() or anywhere else during startup
func RegisterExampleTools() error {
	// Create schema and register with tool registry
	reverseStringSchema := CreateReverseStringToolSchema()
	toolRegistry.RegisterTool("reverseString", ReverseStringToolHandler, reverseStringSchema)

	// Register the analyzeText tool as well
	analyzeTextSchema := CreateAnalyzeTextToolSchema()
	toolRegistry.RegisterTool("analyzeText", AnalyzeTextToolHandler, analyzeTextSchema)

	// Update ollamaTools to include the new tools
	ollamaTools = toolRegistry.GetSchemas()

	logrus.Infof("Successfully registered example tools: reverseString, analyzeText")
	return nil
}

// Example of a more complex tool with multiple parameters
type AnalyzeTextToolArgs struct {
	Text      string `json:"text" jsonschema:"required,description=Text to analyze"`
	CountType string `json:"count_type" jsonschema:"required,description=Type of count: words, characters, or lines"`
}

func AnalyzeTextTool(ctx context.Context, args AnalyzeTextToolArgs) (*ToolResponse, error) {
	logrus.Infoln("Analyze text tool called with args:", args)

	var result int
	var unit string

	switch strings.ToLower(args.CountType) {
	case "words":
		words := strings.Fields(args.Text)
		result = len(words)
		unit = "words"
	case "characters":
		result = len(args.Text)
		unit = "characters"
	case "lines":
		lines := strings.Split(args.Text, "\n")
		result = len(lines)
		unit = "lines"
	default:
		return nil, fmt.Errorf("invalid count_type: %s. Must be 'words', 'characters', or 'lines'", args.CountType)
	}

	message := fmt.Sprintf("Text analysis: %d %s", result, unit)
	return NewToolResponse(message), nil
}

func AnalyzeTextToolHandler(ctx context.Context, args json.RawMessage) (*ToolResponse, error) {
	var analyzeArgs AnalyzeTextToolArgs
	if err := json.Unmarshal(args, &analyzeArgs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal analyze text args: %w", err)
	}
	return AnalyzeTextTool(ctx, analyzeArgs)
}

func CreateAnalyzeTextToolSchema() ollamaApi.ToolFunction {
	return ollamaApi.ToolFunction{
		Name:        "analyzeText",
		Description: "Analyzes text and returns word count, character count, or line count.",
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
			Type:     "object",
			Required: []string{"text", "count_type"},
			Properties: map[string]struct {
				Type        ollamaApi.PropertyType `json:"type"`
				Items       any                    `json:"items,omitempty"`
				Description string                 `json:"description"`
				Enum        []any                  `json:"enum,omitempty"`
			}{
				"text": {
					Type:        ollamaApi.PropertyType{"string"},
					Description: "The text to analyze.",
				},
				"count_type": {
					Type:        ollamaApi.PropertyType{"string"},
					Description: "Type of analysis to perform.",
					Enum:        []any{"words", "characters", "lines"},
				},
			},
		},
	}
}
