package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFrontendFilters_CategoryIDs_AreValidTagTypes(t *testing.T) {
	// Every CategoryID in FrontendFilters must correspond to a valid TagType.
	// This test ensures CategoryID values stay coupled to TagType constants.
	for _, cat := range FrontendFilters.Categories {
		tt := TagType(cat.CategoryID)
		assert.True(t, tt.IsValidTag(),
			"CategoryID %q in category %q is not a valid TagType",
			string(cat.CategoryID), cat.CategoryName,
		)
	}
}

func TestFrontendFilters_CategoryIDs_MatchExpectedValues(t *testing.T) {
	// Verify the exact CategoryID values match the expected TagType constants.
	// This catches accidental changes to the filter structure.
	require.Len(t, FrontendFilters.Categories, 3, "expected 3 filter categories")

	expected := []struct {
		name       string
		categoryID string
	}{
		{"Product families", string(ProductFamilies)},
		{"Content type", string(ContentType)},
		{"Use case", string(UseCase)},
	}

	for i, exp := range expected {
		cat := FrontendFilters.Categories[i]
		assert.Equal(t, exp.name, cat.CategoryName, "category %d name mismatch", i)
		assert.Equal(t, exp.categoryID, string(cat.CategoryID),
			"category %d ID mismatch: expected %q, got %q",
			i, exp.categoryID, string(cat.CategoryID),
		)
	}
}

func TestFrontendFilters_JSONSerialization(t *testing.T) {
	// Ensure JSON output remains stable after any type refactoring.
	// CategoryID should serialize as the same string value regardless of Go type.
	data, err := json.Marshal(FrontendFilters)
	require.NoError(t, err)

	// Unmarshal into raw map to verify JSON string values
	var raw map[string][]map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	categories := raw["categories"]
	require.Len(t, categories, 3)
	assert.Equal(t, "product-families", categories[0]["categoryId"])
	assert.Equal(t, "content", categories[1]["categoryId"])
	assert.Equal(t, "use-case", categories[2]["categoryId"])
}

func TestFrontendFilters_JSONRoundTrip(t *testing.T) {
	// Marshal and unmarshal to verify the structure survives a round trip.
	data, err := json.Marshal(FrontendFilters)
	require.NoError(t, err)

	var parsed FilterData
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	require.Len(t, parsed.Categories, 3)
	assert.Equal(t, "Product families", parsed.Categories[0].CategoryName)
	assert.Equal(t, "Content type", parsed.Categories[1].CategoryName)
	assert.Equal(t, "Use case", parsed.Categories[2].CategoryName)

	// Verify CategoryID values survived round trip
	assert.Equal(t, string(ProductFamilies), string(parsed.Categories[0].CategoryID))
	assert.Equal(t, string(ContentType), string(parsed.Categories[1].CategoryID))
	assert.Equal(t, string(UseCase), string(parsed.Categories[2].CategoryID))
}

func TestFrontendFilters_CategoryData_NotEmpty(t *testing.T) {
	// Each category must have at least one data group with at least one item.
	for _, cat := range FrontendFilters.Categories {
		assert.NotEmpty(t, cat.CategoryData,
			"category %q has no data groups", cat.CategoryName)
		for j, group := range cat.CategoryData {
			assert.NotEmpty(t, group.Data,
				"category %q group %d has no filter items", cat.CategoryName, j)
		}
	}
}

func TestFrontendFilters_ProductFamilies_HasExpectedItems(t *testing.T) {
	// Verify the product-families category has the expected filter items.
	cat := FrontendFilters.Categories[0]
	require.Equal(t, string(ProductFamilies), string(cat.CategoryID))

	// Should have 2 groups: "Platforms" and "Console-wide services"
	require.Len(t, cat.CategoryData, 2)
	assert.Equal(t, "Platforms", cat.CategoryData[0].Group)
	assert.Equal(t, "Console-wide services", cat.CategoryData[1].Group)

	// Platforms: ansible, openshift, rhel
	assert.Len(t, cat.CategoryData[0].Data, 3)
	assert.Equal(t, "ansible", cat.CategoryData[0].Data[0].Id)
	assert.Equal(t, "openshift", cat.CategoryData[0].Data[1].Id)
	assert.Equal(t, "rhel", cat.CategoryData[0].Data[2].Id)

	// Console-wide: iam, settings, subscriptions-services
	assert.Len(t, cat.CategoryData[1].Data, 3)
	assert.Equal(t, "iam", cat.CategoryData[1].Data[0].Id)
	assert.Equal(t, "settings", cat.CategoryData[1].Data[1].Id)
	assert.Equal(t, "subscriptions-services", cat.CategoryData[1].Data[2].Id)
}

func TestFrontendFilters_ContentType_HasExpectedItems(t *testing.T) {
	cat := FrontendFilters.Categories[1]
	require.Equal(t, string(ContentType), string(cat.CategoryID))
	require.Len(t, cat.CategoryData, 1)

	items := cat.CategoryData[0].Data
	assert.Len(t, items, 4)
	assert.Equal(t, "documentation", items[0].Id)
	assert.Equal(t, "learningPath", items[1].Id)
	assert.Equal(t, "quickstart", items[2].Id)
	assert.Equal(t, "otherResource", items[3].Id)
}

func TestFrontendFilters_UseCase_HasExpectedItems(t *testing.T) {
	cat := FrontendFilters.Categories[2]
	require.Equal(t, string(UseCase), string(cat.CategoryID))
	require.Len(t, cat.CategoryData, 1)

	items := cat.CategoryData[0].Data
	assert.Len(t, items, 12)
	assert.Equal(t, "automation", items[0].Id)
	assert.Equal(t, "security", items[9].Id)
}
