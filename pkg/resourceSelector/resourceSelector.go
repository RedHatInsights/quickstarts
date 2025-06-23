package resourceselector

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	ollamaApi "github.com/ollama/ollama/api"
	"github.com/sirupsen/logrus"
)

type HelloToolArgs struct {
	Name string `json:"name" jsonschema:"required,description=Name of the person to greet"`
}

type GreetUserArgs struct {
	Name string `json:"name"`
}

// GreetUserResponse represents the response from the GreetUser tool.
// This should be the same as defined previously.
type GreetUserResponse struct {
	Message string `json:"message"`
}

// ToolResponse represents the response from a tool execution
type ToolResponse struct {
	Message string `json:"message"`
}

// NewToolResponse creates a new tool response
func NewToolResponse(message string) *ToolResponse {
	return &ToolResponse{Message: message}
}

// ToolHandler represents a generic tool function that can be called
type ToolHandler func(ctx context.Context, args json.RawMessage) (*ToolResponse, error)

// ToolRegistry holds tool functions and their schemas
type ToolRegistry struct {
	handlers map[string]ToolHandler
	schemas  map[string]ollamaApi.ToolFunction
}

type PropertyType []string

type ToolFunction struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  struct {
		Type       string   `json:"type"`
		Defs       any      `json:"$defs,omitempty"`
		Items      any      `json:"items,omitempty"`
		Required   []string `json:"required"`
		Properties map[string]struct {
			Type        PropertyType `json:"type"`
			Items       any          `json:"items,omitempty"`
			Description string       `json:"description"`
			Enum        []any        `json:"enum,omitempty"`
		} `json:"properties"`
	} `json:"parameters"`
}

var (
	ollamaClient *ollamaApi.Client
	ollamaTools  []ollamaApi.Tool
	toolRegistry *ToolRegistry
)

// NewToolRegistry creates a new tool registry
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		handlers: make(map[string]ToolHandler),
		schemas:  make(map[string]ollamaApi.ToolFunction),
	}
}

// RegisterTool registers a tool function with its schema
func (tr *ToolRegistry) RegisterTool(name string, handler ToolHandler, schema ollamaApi.ToolFunction) {
	tr.handlers[name] = handler
	tr.schemas[name] = schema
}

// GetHandler returns the handler for a given tool name
func (tr *ToolRegistry) GetHandler(name string) (ToolHandler, bool) {
	handler, exists := tr.handlers[name]
	return handler, exists
}

// GetSchemas returns all tool schemas as a slice for Ollama
func (tr *ToolRegistry) GetSchemas() []ollamaApi.Tool {
	tools := make([]ollamaApi.Tool, 0, len(tr.schemas))
	for _, schema := range tr.schemas {
		tools = append(tools, ollamaApi.Tool{
			Type:     "function",
			Function: schema,
		})
	}
	return tools
}

// GetToolNames returns all registered tool names
func (tr *ToolRegistry) GetToolNames() []string {
	names := make([]string, 0, len(tr.handlers))
	for name := range tr.handlers {
		names = append(names, name)
	}
	return names
}

func GreetUserTool(ctx context.Context, args HelloToolArgs) (*ToolResponse, error) {
	logrus.Infoln("Hello tool called with args:", args)
	message := fmt.Sprintf("Hello, %s!", args.Name)
	return NewToolResponse(message), nil
}

// GreetUserToolHandler wraps GreetUserTool to match ToolHandler signature
func GreetUserToolHandler(ctx context.Context, args json.RawMessage) (*ToolResponse, error) {
	var greetArgs HelloToolArgs
	if err := json.Unmarshal(args, &greetArgs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal greet user args: %w", err)
	}
	return GreetUserTool(ctx, greetArgs)
}

// AddNumbersToolArgs represents arguments for the add numbers tool
type AddNumbersToolArgs struct {
	A float64 `json:"a" jsonschema:"required,description=First number to add"`
	B float64 `json:"b" jsonschema:"required,description=Second number to add"`
}

// SearchQuickstartsToolArgs represents arguments for the search quickstarts tool
type SearchQuickstartsToolArgs struct {
	Name        string `json:"name" jsonschema:"description=Search by quickstart name (exact match)"`
	DisplayName string `json:"displayName" jsonschema:"description=Search by display name (fuzzy match)"`
	Description string `json:"description" jsonschema:"description=Search by description (fuzzy match)"`
	Limit       int    `json:"limit" jsonschema:"description=Maximum number of results to return (default 10)"`
	MaxDistance int    `json:"maxDistance" jsonschema:"description=Maximum Levenshtein distance for fuzzy matching (default 7)"`
}

// UnmarshalJSON provides custom JSON unmarshaling to handle string/int conversion for limit
func (s *SearchQuickstartsToolArgs) UnmarshalJSON(data []byte) error {
	// Define a temporary struct with the same fields but limit as interface{}
	type TempArgs struct {
		Name        string      `json:"name"`
		DisplayName string      `json:"displayName"`
		Description string      `json:"description"`
		Limit       interface{} `json:"limit"`
		MaxDistance interface{} `json:"maxDistance"`
	}

	var temp TempArgs
	if err := json.Unmarshal(data, &temp); err != nil {
		return err
	}

	// Copy the string fields
	s.Name = temp.Name
	s.DisplayName = temp.DisplayName
	s.Description = temp.Description

	// Handle the limit field - could be string, int, or nil
	if err := s.parseIntField(temp.Limit, &s.Limit, "limit"); err != nil {
		return err
	}

	// Handle the maxDistance field - could be string, int, or nil
	if err := s.parseIntField(temp.MaxDistance, &s.MaxDistance, "maxDistance"); err != nil {
		return err
	}

	return nil
}

// parseIntField handles parsing of integer fields that might come as strings
func (s *SearchQuickstartsToolArgs) parseIntField(value interface{}, target *int, fieldName string) error {
	switch v := value.(type) {
	case string:
		if v == "" {
			*target = 0
		} else {
			// Try to parse string as int
			if parsed, err := strconv.Atoi(v); err == nil {
				*target = parsed
			} else {
				return fmt.Errorf("invalid %s value: %s", fieldName, v)
			}
		}
	case float64:
		*target = int(v)
	case int:
		*target = v
	case nil:
		*target = 0
	default:
		return fmt.Errorf("%s must be a number or string, got: %T", fieldName, v)
	}
	return nil
}

// SearchQuickstartsResult represents a single quickstart search result
type SearchQuickstartsResult struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
}

// SearchQuickstartsResponse represents the response from searching quickstarts
type SearchQuickstartsResponse struct {
	Results []SearchQuickstartsResult `json:"results"`
	Count   int                       `json:"count"`
}

// AddNumbersTool adds two numbers together
func AddNumbersTool(ctx context.Context, args AddNumbersToolArgs) (*ToolResponse, error) {
	logrus.Infoln("Add numbers tool called with args:", args)
	result := args.A + args.B
	message := fmt.Sprintf("%.2f + %.2f = %.2f", args.A, args.B, result)
	return NewToolResponse(message), nil
}

// AddNumbersToolHandler wraps AddNumbersTool to match ToolHandler signature
func AddNumbersToolHandler(ctx context.Context, args json.RawMessage) (*ToolResponse, error) {
	var addArgs AddNumbersToolArgs
	if err := json.Unmarshal(args, &addArgs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal add numbers args: %w", err)
	}
	return AddNumbersTool(ctx, addArgs)
}

// buildFuzzyPartialCondition creates a SQL condition to check if any words in the query
// have fuzzy matches within the target field
func buildFuzzyPartialCondition(fieldName string, words []string, maxDistance int) string {
	if len(words) == 0 {
		return "FALSE"
	}

	var conditions []string
	for _, word := range words {
		// For each word, check if there's a fuzzy match within the target field
		// We'll use array_to_string with string_to_array to split target into words
		condition := fmt.Sprintf(`EXISTS (
			SELECT 1 FROM unnest(string_to_array(LOWER(%s), ' ')) AS target_word 
			WHERE levenshtein(target_word, '%s') <= %d
		)`, fieldName, strings.ReplaceAll(word, "'", "''"), maxDistance)
		conditions = append(conditions, condition)
	}

	// Require at least half of the words to have fuzzy matches (minimum 1)
	minMatches := max(1, len(words)/2)
	if len(words) == 2 {
		minMatches = 1 // For 2 words, require at least 1 match
	}

	// Count how many conditions are true and require minimum matches
	countCondition := fmt.Sprintf(`(
		(CASE WHEN %s THEN 1 ELSE 0 END) >= %d
	)`, strings.Join(conditions, " THEN 1 ELSE 0 END) + (CASE WHEN "), minMatches)

	return countCondition
}

// buildFuzzyPartialScore creates a SQL expression to score fuzzy partial matches
func buildFuzzyPartialScore(fieldName string, words []string, maxDistance int) string {
	if len(words) == 0 {
		return "0"
	}

	// Calculate average minimum distance for matched words
	var scoreComponents []string
	for _, word := range words {
		component := fmt.Sprintf(`(
			SELECT COALESCE(MIN(levenshtein(target_word, '%s')), %d) 
			FROM unnest(string_to_array(LOWER(%s), ' ')) AS target_word 
			WHERE levenshtein(target_word, '%s') <= %d
		)`,
			strings.ReplaceAll(word, "'", "''"), maxDistance+1,
			fieldName,
			strings.ReplaceAll(word, "'", "''"), maxDistance)
		scoreComponents = append(scoreComponents, component)
	}

	// Return average distance of matched words
	return fmt.Sprintf("((%s) / %d)", strings.Join(scoreComponents, " + "), len(words))
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SearchQuickstartsTool searches for quickstarts by name or description
func SearchQuickstartsTool(ctx context.Context, args SearchQuickstartsToolArgs) (*ToolResponse, error) {
	logrus.Infoln("Search quickstarts tool called with args:", args)

	// Validate that at least one search parameter is provided
	if args.Name == "" && args.DisplayName == "" && args.Description == "" {
		return NewToolResponse("Error: At least one search parameter (name, displayName, or description) must be provided"), nil
	}

	// Check if database connection is available
	if database.DB == nil {
		logrus.Warnln("Database connection is not available")
		return NewToolResponse("Error: Database connection not available"), nil
	}

	// Set defaults
	limit := args.Limit
	if limit <= 0 || limit > 50 {
		limit = 10
	}

	maxDistance := args.MaxDistance
	if maxDistance <= 0 {
		maxDistance = 7 // Increased from 3 to 5 for better typo tolerance
	}

	// Build query with both fuzzy and partial matching
	var results []struct {
		models.Quickstart
		Score int `gorm:"column:score"`
	}

	// Build word arrays for fuzzy partial matching
	var displayNameWords, descriptionWords, nameWords []string
	if args.DisplayName != "" {
		displayNameWords = strings.Fields(strings.ToLower(args.DisplayName))
	}
	if args.Description != "" {
		descriptionWords = strings.Fields(strings.ToLower(args.Description))
	}
	if args.Name != "" {
		nameWords = strings.Fields(strings.ToLower(args.Name))
	}

	// Build the scoring subquery with fuzzy partial matching (no full fuzzy matching)
	scoreQuery := database.DB.Model(&models.Quickstart{}).
		Select(`quickstarts.*, 
			CASE 
				-- Exact matches get best score
				WHEN ? != '' AND name = ? THEN 0
				WHEN ? != '' AND LOWER(content->'spec'->>'displayName') = LOWER(?) THEN 0
				WHEN ? != '' AND LOWER(content->'spec'->>'description') = LOWER(?) THEN 0
				
				-- Exact partial matches (substring) - score based on match coverage
				WHEN ? != '' AND LOWER(content->'spec'->>'displayName') LIKE LOWER(?) THEN 
					(LENGTH(?) * 100) / LENGTH(content->'spec'->>'displayName')
				WHEN ? != '' AND LOWER(content->'spec'->>'description') LIKE LOWER(?) THEN 
					(LENGTH(?) * 150) / LENGTH(content->'spec'->>'description')
				WHEN ? != '' AND LOWER(name) LIKE LOWER(?) THEN 
					(LENGTH(?) * 120) / LENGTH(name)
				
				-- Fuzzy partial matches - check if words in query have fuzzy matches in target
				WHEN ? != '' AND (`+buildFuzzyPartialCondition("content->'spec'->>'displayName'", displayNameWords, maxDistance)+`) THEN 
					100 + (`+buildFuzzyPartialScore("content->'spec'->>'displayName'", displayNameWords, maxDistance)+`)
				WHEN ? != '' AND (`+buildFuzzyPartialCondition("content->'spec'->>'description'", descriptionWords, maxDistance)+`) THEN 
					150 + (`+buildFuzzyPartialScore("content->'spec'->>'description'", descriptionWords, maxDistance)+`)
				WHEN ? != '' AND (`+buildFuzzyPartialCondition("name", nameWords, maxDistance)+`) THEN 
					120 + (`+buildFuzzyPartialScore("name", nameWords, maxDistance)+`)
				
				ELSE 999  -- No match
			END as score`,
			// Exact matches
			args.Name, args.Name,
			args.DisplayName, args.DisplayName,
			args.Description, args.Description,

			// Exact partial matches (substring)
			args.DisplayName, "%"+args.DisplayName+"%", args.DisplayName,
			args.Description, "%"+args.Description+"%", args.Description,
			args.Name, "%"+args.Name+"%", args.Name,

			// Fuzzy partial matches
			args.DisplayName,
			args.Description,
			args.Name)

	// Build WHERE clause for filtering (include exact, partial, and fuzzy partial conditions)
	var whereConditions []string
	var whereValues []interface{}

	if args.Name != "" {
		fuzzyPartialCondition := buildFuzzyPartialCondition("name", nameWords, maxDistance)
		whereConditions = append(whereConditions,
			fmt.Sprintf("(name = ? OR LOWER(name) LIKE LOWER(?) OR %s)", fuzzyPartialCondition))
		whereValues = append(whereValues, args.Name, "%"+args.Name+"%")
	}

	if args.DisplayName != "" {
		fuzzyPartialCondition := buildFuzzyPartialCondition("content->'spec'->>'displayName'", displayNameWords, maxDistance)
		whereConditions = append(whereConditions,
			fmt.Sprintf("(LOWER(content->'spec'->>'displayName') = LOWER(?) OR LOWER(content->'spec'->>'displayName') LIKE LOWER(?) OR %s)", fuzzyPartialCondition))
		whereValues = append(whereValues, args.DisplayName, "%"+args.DisplayName+"%")
	}

	if args.Description != "" {
		fuzzyPartialCondition := buildFuzzyPartialCondition("content->'spec'->>'description'", descriptionWords, maxDistance)
		whereConditions = append(whereConditions,
			fmt.Sprintf("(LOWER(content->'spec'->>'description') = LOWER(?) OR LOWER(content->'spec'->>'description') LIKE LOWER(?) OR %s)", fuzzyPartialCondition))
		whereValues = append(whereValues, args.Description, "%"+args.Description+"%")
	}

	if len(whereConditions) > 0 {
		whereClause := "(" + strings.Join(whereConditions, " OR ") + ")"
		scoreQuery = scoreQuery.Where(whereClause, whereValues...)
	}

	// Execute query using a subquery to properly filter by score
	err := database.DB.
		Table("(?) as scored_quickstarts", scoreQuery).
		Where("score < 999").
		Order("score ASC").
		Limit(1). // Only get the best match
		Find(&results).Error

	if err != nil {
		// If Levenshtein function is not available, fallback to ILIKE
		if strings.Contains(err.Error(), "levenshtein") {
			logrus.Warnln("Levenshtein function not available, falling back to ILIKE")
			return fallbackToILIKE(ctx, args, 1) // Also limit fallback to 1 result
		}
		logrus.Errorf("Database query failed: %v", err)
		return NewToolResponse(fmt.Sprintf("Error searching quickstarts: %v", err)), nil
	}

	// Check if we found any results
	if len(results) == 0 {
		// Build search description for no results
		var searchParams []string
		if args.Name != "" {
			searchParams = append(searchParams, fmt.Sprintf("name='%s'", args.Name))
		}
		if args.DisplayName != "" {
			searchParams = append(searchParams, fmt.Sprintf("displayName='%s'", args.DisplayName))
		}
		if args.Description != "" {
			searchParams = append(searchParams, fmt.Sprintf("description='%s'", args.Description))
		}
		searchDescription := strings.Join(searchParams, ", ")

		message := fmt.Sprintf("No quickstarts found matching (%s) with Levenshtein distance ≤%d", searchDescription, maxDistance)
		return NewToolResponse(message), nil
	}

	// Get the best match (first result)
	bestMatch := results[0]

	// Create the complete quickstart result with full object
	searchResult := struct {
		Score       int               `json:"score"`
		MatchType   string            `json:"matchType"`
		Quickstart  models.Quickstart `json:"quickstart"`
		DisplayName string            `json:"displayName"`
		Description string            `json:"description"`
	}{
		Score:      bestMatch.Score,
		Quickstart: bestMatch.Quickstart,
	}

	// Determine match type based on score
	switch {
	case bestMatch.Score == 0:
		searchResult.MatchType = "exact_match"
	case bestMatch.Score < 100: // Exact partial matches (coverage-based scoring)
		searchResult.MatchType = "partial_match"
	case bestMatch.Score >= 100 && bestMatch.Score < 200: // Fuzzy partial matches
		searchResult.MatchType = "fuzzy_partial_match"
	default:
		searchResult.MatchType = "no_match"
	}

	// Extract displayName and description from JSONB content
	var content map[string]interface{}
	if err := json.Unmarshal(bestMatch.Content, &content); err == nil {
		if spec, ok := content["spec"].(map[string]interface{}); ok {
			if displayName, ok := spec["displayName"].(string); ok {
				searchResult.DisplayName = displayName
			}
			if description, ok := spec["description"].(string); ok {
				searchResult.Description = description
			}
		}
	}

	// Create structured response
	response := struct {
		Tool         string      `json:"tool"`
		SearchParams interface{} `json:"searchParams"`
		Result       interface{} `json:"result"`
		Fallback     bool        `json:"fallback"`
	}{
		Tool: "searchQuickstarts",
		SearchParams: struct {
			Name        string `json:"name,omitempty"`
			DisplayName string `json:"displayName,omitempty"`
			Description string `json:"description,omitempty"`
			MaxDistance int    `json:"maxDistance"`
		}{
			Name:        args.Name,
			DisplayName: args.DisplayName,
			Description: args.Description,
			MaxDistance: maxDistance,
		},
		Result:   searchResult,
		Fallback: false,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return NewToolResponse(fmt.Sprintf("Error formatting results: %v", err)), nil
	}

	return NewToolResponse(string(responseBytes)), nil
}

// fallbackToILIKE provides fallback functionality when Levenshtein is not available
func fallbackToILIKE(ctx context.Context, args SearchQuickstartsToolArgs, limit int) (*ToolResponse, error) {
	logrus.Infoln("Using ILIKE fallback for search")

	var quickstarts []models.Quickstart
	query := database.DB.Model(&models.Quickstart{})

	var conditions []string
	var values []interface{}

	if args.Name != "" {
		conditions = append(conditions, "name ILIKE ?")
		values = append(values, "%"+args.Name+"%")
	}

	if args.DisplayName != "" {
		conditions = append(conditions, "content->'spec'->>'displayName' ILIKE ?")
		values = append(values, "%"+args.DisplayName+"%")
	}

	if args.Description != "" {
		conditions = append(conditions, "content->'spec'->>'description' ILIKE ?")
		values = append(values, "%"+args.Description+"%")
	}

	if len(conditions) > 0 {
		whereClause := strings.Join(conditions, " OR ")
		query = query.Where(whereClause, values...)
	}

	err := query.Limit(limit).Find(&quickstarts).Error
	if err != nil {
		return NewToolResponse(fmt.Sprintf("Error searching quickstarts: %v", err)), nil
	}

	// Check if we found any results
	if len(quickstarts) == 0 {
		// Build search description for no results
		var searchParams []string
		if args.Name != "" {
			searchParams = append(searchParams, fmt.Sprintf("name='%s'", args.Name))
		}
		if args.DisplayName != "" {
			searchParams = append(searchParams, fmt.Sprintf("displayName='%s'", args.DisplayName))
		}
		if args.Description != "" {
			searchParams = append(searchParams, fmt.Sprintf("description='%s'", args.Description))
		}
		searchDescription := strings.Join(searchParams, ", ")

		message := fmt.Sprintf("No quickstarts found matching (%s) using ILIKE fallback", searchDescription)
		return NewToolResponse(message), nil
	}

	// Get the first result as the best match (ILIKE doesn't have scoring)
	bestMatch := quickstarts[0]

	// Create the complete quickstart result
	searchResult := struct {
		Score       int               `json:"score"`
		MatchType   string            `json:"matchType"`
		Quickstart  models.Quickstart `json:"quickstart"`
		DisplayName string            `json:"displayName"`
		Description string            `json:"description"`
	}{
		Score:      0, // ILIKE fallback doesn't have real scoring
		MatchType:  "pattern_match",
		Quickstart: bestMatch,
	}

	// Extract displayName and description from JSONB content
	var content map[string]interface{}
	if err := json.Unmarshal(bestMatch.Content, &content); err == nil {
		if spec, ok := content["spec"].(map[string]interface{}); ok {
			if displayName, ok := spec["displayName"].(string); ok {
				searchResult.DisplayName = displayName
			}
			if description, ok := spec["description"].(string); ok {
				searchResult.Description = description
			}
		}
	}

	// Create structured response
	response := struct {
		Tool         string      `json:"tool"`
		SearchParams interface{} `json:"searchParams"`
		Result       interface{} `json:"result"`
		Fallback     bool        `json:"fallback"`
	}{
		Tool: "searchQuickstarts",
		SearchParams: struct {
			Name        string `json:"name,omitempty"`
			DisplayName string `json:"displayName,omitempty"`
			Description string `json:"description,omitempty"`
			MaxDistance int    `json:"maxDistance"`
		}{
			Name:        args.Name,
			DisplayName: args.DisplayName,
			Description: args.Description,
			MaxDistance: 0, // Not applicable for ILIKE
		},
		Result:   searchResult,
		Fallback: true,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		return NewToolResponse(fmt.Sprintf("Error formatting results: %v", err)), nil
	}

	return NewToolResponse(string(responseBytes)), nil
}

// SearchQuickstartsToolHandler wraps SearchQuickstartsTool to match ToolHandler signature
func SearchQuickstartsToolHandler(ctx context.Context, args json.RawMessage) (*ToolResponse, error) {
	var searchArgs SearchQuickstartsToolArgs
	if err := json.Unmarshal(args, &searchArgs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search quickstarts args: %w", err)
	}
	return SearchQuickstartsTool(ctx, searchArgs)
}

func init() {
	// Initialize tool registry
	toolRegistry = NewToolRegistry()

	// Create the greetUser tool schema for Ollama
	greetToolSchema := ollamaApi.ToolFunction{
		Name:        "greetUser", // The name of your tool function
		Description: "Greets a user by their name.",
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
			Required: []string{"name"},
			Properties: map[string]struct {
				Type        ollamaApi.PropertyType `json:"type"`
				Items       any                    `json:"items,omitempty"`
				Description string                 `json:"description"`
				Enum        []any                  `json:"enum,omitempty"`
			}{
				"name": {
					Type:        ollamaApi.PropertyType{"string"},
					Description: "The name of the user to greet.",
				},
			},
		},
	}

	// Register the greetUser tool with our tool registry
	toolRegistry.RegisterTool("greetUser", GreetUserToolHandler, greetToolSchema)

	// Create the addNumbers tool schema for Ollama
	addNumbersToolSchema := ollamaApi.ToolFunction{
		Name:        "addNumbers",
		Description: "Adds two numbers together and returns the result.",
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
			Required: []string{"a", "b"},
			Properties: map[string]struct {
				Type        ollamaApi.PropertyType `json:"type"`
				Items       any                    `json:"items,omitempty"`
				Description string                 `json:"description"`
				Enum        []any                  `json:"enum,omitempty"`
			}{
				"a": {
					Type:        ollamaApi.PropertyType{"number"},
					Description: "The first number to add.",
				},
				"b": {
					Type:        ollamaApi.PropertyType{"number"},
					Description: "The second number to add.",
				},
			},
		},
	}

	// Register the addNumbers tool with our tool registry
	toolRegistry.RegisterTool("addNumbers", AddNumbersToolHandler, addNumbersToolSchema)

	// Create the searchQuickstarts tool schema for Ollama
	searchQuickstartsToolSchema := ollamaApi.ToolFunction{
		Name:        "searchQuickstarts",
		Description: "Search for quickstarts by name, displayName, or description in the database. Supports 3 matching strategies: 1) Exact matches, 2) Exact partial matches (substring), 3) Fuzzy partial matches (word-level typo tolerance). Query 'Create RHAL images' will match 'Create RHEL images for deployment' via fuzzy partial matching.",
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
			Required: []string{}, // No required fields since we validate at least one is provided
			Properties: map[string]struct {
				Type        ollamaApi.PropertyType `json:"type"`
				Items       any                    `json:"items,omitempty"`
				Description string                 `json:"description"`
				Enum        []any                  `json:"enum,omitempty"`
			}{
				"name": {
					Type:        ollamaApi.PropertyType{"string"},
					Description: "Search by exact quickstart name (e.g., 'insights-getting-started').",
				},
				"displayName": {
					Type:        ollamaApi.PropertyType{"string"},
					Description: "Search by display name with partial and fuzzy matching. Supports typo tolerance: 'Create RHAL images' matches 'Create RHEL images for deployment'. Use for titles and display names.",
				},
				"description": {
					Type:        ollamaApi.PropertyType{"string"},
					Description: "Search by description content with partial and fuzzy matching.",
				},
				"limit": {
					Type:        ollamaApi.PropertyType{"integer"},
					Description: "Maximum number of results to return (default 10, max 50).",
				},
				"maxDistance": {
					Type:        ollamaApi.PropertyType{"integer"},
					Description: "Maximum Levenshtein distance for fuzzy matching (default 7).",
				},
			},
		},
	}

	// Register the searchQuickstarts tool with our tool registry
	toolRegistry.RegisterTool("searchQuickstarts", SearchQuickstartsToolHandler, searchQuickstartsToolSchema)

	// Get all tools for Ollama
	ollamaTools = toolRegistry.GetSchemas()

	// Log registered tools
	registeredTools := toolRegistry.GetToolNames()
	logrus.Infof("Registered %d tools: %v", len(registeredTools), registeredTools)

	ollamaHost := "http://127.0.0.1:11434"
	baseURL, err := url.Parse(ollamaHost)
	if err != nil {
		logrus.Fatalf("Failed to parse Ollama host URL: %v", err)
	}
	httpClient := &http.Client{}

	ollamaClient = ollamaApi.NewClient(baseURL, httpClient)
}

type UserLLMQuery struct {
	Query string `json:"query"`
}

func generateToolCall(ctx context.Context, query string) (*ollamaApi.ToolCall, string, error) {
	stream := false
	req := ollamaApi.ChatRequest{
		Model: "llama3.1",
		Messages: []ollamaApi.Message{
			{
				Role:    "user",
				Content: query,
			},
		},
		Tools:  ollamaTools,
		Stream: &stream,
	}

	var finalResponse ollamaApi.ChatResponse
	err := ollamaClient.Chat(ctx, &req, func(resp ollamaApi.ChatResponse) error {
		finalResponse = resp // Capture the single, complete response
		return nil           // Return nil to indicate successful processing of this chunk (the only chunk)
	})

	if err != nil {
		// This error will contain any network errors or API errors from Ollama
		return nil, "", fmt.Errorf("Ollama chat API call failed: %w", err)
	}

	logrus.Debugln("Starting chat with Ollama client for query:", query)
	if len(finalResponse.Message.ToolCalls) > 0 {
		// Assuming only one tool call for this simple example
		return &finalResponse.Message.ToolCalls[0], "", nil
	}

	return nil, finalResponse.Message.Content, nil
}

func HandleMCPUserQuery(w http.ResponseWriter, r *http.Request) {
	var userQuery UserLLMQuery
	err := json.NewDecoder(r.Body).Decode(&userQuery)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		logrus.Errorln("Failed to decode user query:", err)
		return
	}

	if userQuery.Query == "" {
		http.Error(w, "Query field cannot be empty", http.StatusBadRequest)
		logrus.Errorln("Received empty query string.")
		return
	}

	logrus.Debugln("Received user query:", userQuery.Query)

	toolCall, messageContent, err := generateToolCall(r.Context(), userQuery.Query)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate tool call: %v", err), http.StatusInternalServerError)
		logrus.Errorln("Error generating tool call:", err)
		return
	}

	if toolCall == nil {
		response := struct {
			Status  string `json:"status"`
			Query   string `json:"query"`
			Message string `json:"message"`
		}{
			Status:  "no_tool_call",
			Query:   userQuery.Query,
			Message: fmt.Sprintf("No tool call generated for query: \"%s\". Message content: \"%s\"", userQuery.Query, messageContent),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
		return
	}

	logrus.Debugln("Generated tool call:", toolCall)

	toolCallBytes, err := json.Marshal(toolCall.Function.Arguments)
	if err != nil {
		logrus.Errorf("Failed to marshal tool call arguments from Ollama: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	logrus.Debugf("Executing tool %s with arguments: %s", toolCall.Function.Name, string(toolCallBytes))

	// Get the tool handler from the registry
	toolHandler, exists := toolRegistry.GetHandler(toolCall.Function.Name)
	if !exists {
		logrus.Errorf("Tool not found: %s", toolCall.Function.Name)
		http.Error(w, fmt.Sprintf("Tool not found: %s", toolCall.Function.Name), http.StatusBadRequest)
		return
	}

	// Call the tool function directly using the handler
	toolResponse, err := toolHandler(r.Context(), toolCallBytes)
	if err != nil {
		logrus.Errorf("Error during tool execution: %v", err)
		http.Error(w, fmt.Sprintf("Error executing tool: %v", err), http.StatusInternalServerError)
		return
	}

	logrus.Debugf("Received tool response: %+v", toolResponse)

	// Parse the tool response as JSON if possible
	var toolData interface{}
	if err := json.Unmarshal([]byte(toolResponse.Message), &toolData); err != nil {
		// If it's not JSON, treat as plain text
		toolData = toolResponse.Message
	}

	// Create structured response
	response := struct {
		Status     string `json:"status"`
		Query      string `json:"query"`
		ToolResult struct {
			Tool string      `json:"tool"`
			Data interface{} `json:"data"`
		} `json:"tool_result"`
	}{
		Status: "success",
		Query:  userQuery.Query,
		ToolResult: struct {
			Tool string      `json:"tool"`
			Data interface{} `json:"data"`
		}{
			Tool: toolCall.Function.Name,
			Data: toolData,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
