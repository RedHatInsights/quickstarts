package database

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestContentStructureVerification(t *testing.T) {
	// Test what AddTagsToContent actually produces
	fileHelper := NewFileHelper("test")
	
	// Test with the hcs-getting-started quickstart
	contentPath := "../docs/quickstarts/hcs-getting-started/hcs-getting-started.yml"
	tags := []interface{}{
		map[string]interface{}{"kind": "bundle", "value": "subscriptions"},
	}
	
	result, err := fileHelper.AddTagsToContent(contentPath, tags)
	assert.NoError(t, err)
	
	// Parse the result to see what structure we get
	var parsed map[string]interface{}
	err = json.Unmarshal(result, &parsed)
	assert.NoError(t, err)
	
	t.Logf("Keys in parsed content: %v", getKeys(parsed))
	t.Logf("Has apiVersion: %v", parsed["apiVersion"])
	t.Logf("Has kind: %v", parsed["kind"])
	t.Logf("Has metadata: %v", parsed["metadata"] != nil)
	t.Logf("Has spec: %v", parsed["spec"] != nil)
	
	// Check if spec contains the actual quickstart data
	if spec, ok := parsed["spec"].(map[string]interface{}); ok {
		t.Logf("Spec keys: %v", getKeys(spec))
		t.Logf("Has displayName: %v", spec["displayName"])
		t.Logf("Has description: %v", spec["description"])
	}
	
	// Verify we have the essential fields
	assert.NotNil(t, parsed["apiVersion"], "Should have apiVersion")
	assert.NotNil(t, parsed["kind"], "Should have kind")
	assert.NotNil(t, parsed["metadata"], "Should have metadata")
	assert.NotNil(t, parsed["spec"], "Should have spec")
}

func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}