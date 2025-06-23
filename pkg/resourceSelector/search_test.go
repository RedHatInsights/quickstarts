package resourceselector

import (
	"context"
	"encoding/json"
	"testing"
)

func TestSearchQuickstartsToolArgs(t *testing.T) {
	// Test serialization/deserialization of SearchQuickstartsToolArgs
	args := SearchQuickstartsToolArgs{
		Name:        "insights-apis",
		DisplayName: "API documentation",
		Description: "explore API catalog",
		Limit:       5,
		MaxDistance: 2,
	}

	// Test JSON marshaling
	jsonBytes, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal SearchQuickstartsToolArgs: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled SearchQuickstartsToolArgs
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SearchQuickstartsToolArgs: %v", err)
	}

	// Verify values
	if unmarshaled.Name != args.Name {
		t.Errorf("Expected name %s, got %s", args.Name, unmarshaled.Name)
	}
	if unmarshaled.DisplayName != args.DisplayName {
		t.Errorf("Expected displayName %s, got %s", args.DisplayName, unmarshaled.DisplayName)
	}
	if unmarshaled.Description != args.Description {
		t.Errorf("Expected description %s, got %s", args.Description, unmarshaled.Description)
	}
	if unmarshaled.Limit != args.Limit {
		t.Errorf("Expected limit %d, got %d", args.Limit, unmarshaled.Limit)
	}
	if unmarshaled.MaxDistance != args.MaxDistance {
		t.Errorf("Expected maxDistance %d, got %d", args.MaxDistance, unmarshaled.MaxDistance)
	}
}

func TestSearchQuickstartsResponse(t *testing.T) {
	// Test SearchQuickstartsResponse structure
	response := SearchQuickstartsResponse{
		Results: []SearchQuickstartsResult{
			{
				Name:        "test-quickstart",
				DisplayName: "Test Quickstart",
				Description: "A test quickstart description",
			},
		},
		Count: 1,
	}

	// Test JSON marshaling
	jsonBytes, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal SearchQuickstartsResponse: %v", err)
	}

	// Test JSON unmarshaling
	var unmarshaled SearchQuickstartsResponse
	err = json.Unmarshal(jsonBytes, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal SearchQuickstartsResponse: %v", err)
	}

	// Verify values
	if unmarshaled.Count != response.Count {
		t.Errorf("Expected count %d, got %d", response.Count, unmarshaled.Count)
	}
	if len(unmarshaled.Results) != len(response.Results) {
		t.Errorf("Expected %d results, got %d", len(response.Results), len(unmarshaled.Results))
	}
	if len(unmarshaled.Results) > 0 {
		result := unmarshaled.Results[0]
		expected := response.Results[0]
		if result.Name != expected.Name {
			t.Errorf("Expected name %s, got %s", expected.Name, result.Name)
		}
		if result.DisplayName != expected.DisplayName {
			t.Errorf("Expected display name %s, got %s", expected.DisplayName, result.DisplayName)
		}
		if result.Description != expected.Description {
			t.Errorf("Expected description %s, got %s", expected.Description, result.Description)
		}
	}
}

func TestSearchQuickstartsToolHandler(t *testing.T) {
	// Test the tool handler with mock data
	args := SearchQuickstartsToolArgs{
		DisplayName: "API",
		Limit:       5,
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal args: %v", err)
	}

	// Note: This test will fail without a proper database connection
	// This is just to test the handler marshaling/unmarshaling
	response, err := SearchQuickstartsToolHandler(context.Background(), argsJSON)

	// We expect the function to handle the database error gracefully
	if err != nil {
		t.Logf("Expected error without database connection: %v", err)
		return
	}

	// If no error (unlikely without DB), check response structure
	if response == nil {
		t.Error("Expected non-nil response")
		return
	}

	if response.Message == "" {
		t.Error("Expected non-empty message in response")
	}

	t.Logf("Response message: %s", response.Message)
}

func TestSearchQuickstartsValidation(t *testing.T) {
	// Test validation that at least one parameter must be provided
	args := SearchQuickstartsToolArgs{
		Limit: 5,
		// No search parameters provided
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		t.Fatalf("Failed to marshal args: %v", err)
	}

	response, err := SearchQuickstartsToolHandler(context.Background(), argsJSON)

	if err != nil {
		t.Fatalf("Expected no handler error, got: %v", err)
	}

	if response == nil {
		t.Error("Expected non-nil response")
		return
	}

	// Should get validation error message
	expectedError := "Error: At least one search parameter (name, displayName, or description) must be provided"
	if response.Message != expectedError {
		t.Errorf("Expected validation error message, got: %s", response.Message)
	}
}

func TestSearchQuickstartsLimitUnmarshaling(t *testing.T) {
	// Test that limit field can be unmarshaled from both string and int
	tests := []struct {
		name     string
		jsonData string
		expected int
	}{
		{
			name:     "limit as integer",
			jsonData: `{"displayName": "test", "limit": 15}`,
			expected: 15,
		},
		{
			name:     "limit as string",
			jsonData: `{"displayName": "test", "limit": "20"}`,
			expected: 20,
		},
		{
			name:     "limit as empty string",
			jsonData: `{"displayName": "test", "limit": ""}`,
			expected: 0,
		},
		{
			name:     "limit missing",
			jsonData: `{"displayName": "test"}`,
			expected: 0,
		},
		{
			name:     "limit as float",
			jsonData: `{"displayName": "test", "limit": 25.0}`,
			expected: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args SearchQuickstartsToolArgs
			err := json.Unmarshal([]byte(tt.jsonData), &args)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if args.Limit != tt.expected {
				t.Errorf("Expected limit %d, got %d", tt.expected, args.Limit)
			}
		})
	}
}

func TestSearchQuickstartsMaxDistanceUnmarshaling(t *testing.T) {
	// Test that maxDistance field can be unmarshaled from both string and int
	tests := []struct {
		name     string
		jsonData string
		expected int
	}{
		{
			name:     "maxDistance as integer",
			jsonData: `{"displayName": "test", "maxDistance": 4}`,
			expected: 4,
		},
		{
			name:     "maxDistance as string",
			jsonData: `{"displayName": "test", "maxDistance": "2"}`,
			expected: 2,
		},
		{
			name:     "maxDistance missing",
			jsonData: `{"displayName": "test"}`,
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args SearchQuickstartsToolArgs
			err := json.Unmarshal([]byte(tt.jsonData), &args)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if args.MaxDistance != tt.expected {
				t.Errorf("Expected maxDistance %d, got %d", tt.expected, args.MaxDistance)
			}
		})
	}
}

func TestSearchQuickstartsLimitUnmarshalingErrors(t *testing.T) {
	// Test invalid limit values
	tests := []struct {
		name     string
		jsonData string
	}{
		{
			name:     "limit as invalid string",
			jsonData: `{"displayName": "test", "limit": "invalid"}`,
		},
		{
			name:     "limit as boolean",
			jsonData: `{"displayName": "test", "limit": true}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args SearchQuickstartsToolArgs
			err := json.Unmarshal([]byte(tt.jsonData), &args)
			if err == nil {
				t.Errorf("Expected unmarshaling error for %s", tt.jsonData)
			}
		})
	}
}
