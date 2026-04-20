package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQuickstartsQuery_TagMap_UsesAllTagTypes(t *testing.T) {
	// Verify that NewQuickstartsQuery maps all 7 tag types correctly.
	// This ensures the tagMap covers every TagType constant.
	// ParseLegacyQuickstartParams reads from the URL query string, so we must
	// put values there rather than setting params struct fields directly.
	url := "/?bundle=rhel&application=rbac&product-families=openshift&use-case=deploy&content=quickstart&kind=QuickStart&topic=automation"
	req := httptest.NewRequest(http.MethodGet, url, nil)
	params := generated.GetQuickstartsParams{}
	q := NewQuickstartsQuery(req, params)

	// All 7 tag types should be present with aligned values
	require.Len(t, q.TagTypes, 7, "all 7 tag types should be populated")
	require.Len(t, q.TagValues, 7, "all 7 tag value slices should be populated")

	// Build a TagType → values map from the parallel TagTypes / TagValues slices
	tagValuesByType := make(map[models.TagType][]string, len(q.TagTypes))
	for i, tagType := range q.TagTypes {
		tagValuesByType[tagType] = q.TagValues[i]
	}

	// All 7 tag types should be present and mapped to the correct values
	expected := map[models.TagType][]string{
		models.BundleTag:       {"rhel"},
		models.ApplicationTag:  {"rbac"},
		models.ProductFamilies: {"openshift"},
		models.UseCase:         {"deploy"},
		models.ContentType:     {"quickstart"},
		models.ContentKind:     {"QuickStart"},
		models.TopicTag:        {"automation"},
	}

	assert.Equal(t, expected, tagValuesByType)
}

func TestNewQuickstartsQuery_TagMap_EmptyParams(t *testing.T) {
	// Verify no tag types or values when no filter params provided.
	params := generated.GetQuickstartsParams{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	q := NewQuickstartsQuery(req, params)

	assert.Empty(t, q.TagTypes, "no tag types with empty params")
	assert.Empty(t, q.TagValues, "no tag values with empty params")
}

func TestNewQuickstartsQuery_TagMap_SingleParam(t *testing.T) {
	// Verify only the provided tag type appears.
	req := httptest.NewRequest(http.MethodGet, "/?bundle=rhel&bundle=settings", nil)
	params := generated.GetQuickstartsParams{}
	q := NewQuickstartsQuery(req, params)

	require.Len(t, q.TagTypes, 1)
	assert.Equal(t, models.BundleTag, q.TagTypes[0])
	require.Len(t, q.TagValues, 1)
	assert.Equal(t, []string{"rhel", "settings"}, q.TagValues[0])
}

func TestNewQuickstartsQuery_TagMap_OrderFollowsGetAllTags(t *testing.T) {
	// Verify that tag types appear in the same order as GetAllTags().
	// This ensures consistent query behavior.
	req := httptest.NewRequest(http.MethodGet, "/?bundle=rhel&content=quickstart&use-case=deploy", nil)
	params := generated.GetQuickstartsParams{}
	q := NewQuickstartsQuery(req, params)

	require.Len(t, q.TagTypes, 3)
	// GetAllTags order: bundle, application, kind, topic, content, product-families, use-case
	// So with bundle, content, use-case: order should be bundle, content, use-case
	assert.Equal(t, models.BundleTag, q.TagTypes[0])
	assert.Equal(t, models.ContentType, q.TagTypes[1])
	assert.Equal(t, models.UseCase, q.TagTypes[2])
}

func TestNewQuickstartsQuery_DefaultPagination(t *testing.T) {
	params := generated.GetQuickstartsParams{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	q := NewQuickstartsQuery(req, params)

	assert.Equal(t, 50, q.Limit, "default limit should be 50")
	assert.Equal(t, 0, q.Offset, "default offset should be 0")
}

func TestSanitizeLimit(t *testing.T) {
	assert.Equal(t, 50, sanitizeLimit(0), "zero limit should default to 50")
	assert.Equal(t, 50, sanitizeLimit(-2), "negative limit below -1 should default to 50")
	assert.Equal(t, -1, sanitizeLimit(-1), "-1 should be allowed (unlimited)")
	assert.Equal(t, 10, sanitizeLimit(10), "positive limit should be kept")
}

func TestSanitizeOffset(t *testing.T) {
	assert.Equal(t, 0, sanitizeOffset(-1), "negative offset should default to 0")
	assert.Equal(t, 0, sanitizeOffset(0), "zero offset should be kept")
	assert.Equal(t, 5, sanitizeOffset(5), "positive offset should be kept")
}
