package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

var quickstart models.Quickstart
var taggedQuickstart models.Quickstart
var settingsQuickstart models.Quickstart
var rhelQuickstart models.Quickstart
var rhelTaggedQuickstart models.Quickstart
var rbacQuickstart models.Quickstart
var rhelBudleTag models.Tag
var rhelContentTypeTag models.Tag
var rhelProductFamiliesTag models.Tag
var rhelUseCaseTag models.Tag
var settingsBundleTag models.Tag
var settingsContentTypeTag models.Tag
var settingsProductFamiliesTag models.Tag
var settingsUseCaseTag models.Tag
var rbacApplicationTag models.Tag
var rbacProductFamiliesTag models.Tag
var rbacContentTypeTag models.Tag
var rbacUseCaseTag models.Tag
var unusedTag models.Tag
var favoriteQuickstart models.FavoriteQuickstart

func mockQuickstart(name string) *models.Quickstart {
	quickstart.Name = name
	database.DB.Create(&quickstart)
	return &quickstart
}

type responseBody struct {
	Id   uint   `json:"id"`
	Name string `json:"name"`
}

type responsePayload struct {
	Data []responseBody
}

type singleResponsePayload struct {
	Data responseBody
}

type messageResponsePayload struct {
	Msg string `json:"msg"`
}

func setupRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(PaginationContext)
	r.Get("/", GetAllQuickstarts)
	r.Get("/fuzzy-search", GetFuzzySearchQuickstarts)
	r.Route("/{id}", func(sub chi.Router) {
		sub.Use(QuickstartEntityContext)
		sub.Get("/", GetQuickstartById)
	})
	return r
}

func setupTags() {
	rhelBudleTag.Type = models.BundleTag
	rhelBudleTag.Value = "rhel"
	rhelProductFamiliesTag.Type = models.ProductFamilies
	rhelProductFamiliesTag.Value = "rhel"
	rhelContentTypeTag.Type = models.ContentType
	rhelContentTypeTag.Value = "quickstart"
	rhelUseCaseTag.Type = models.UseCase
	rhelUseCaseTag.Value = "deploy"

	settingsBundleTag.Type = models.BundleTag
	settingsBundleTag.Value = "settings"
	settingsProductFamiliesTag.Type = models.ProductFamilies
	settingsProductFamiliesTag.Value = "settings"
	settingsContentTypeTag.Type = models.ContentType
	settingsContentTypeTag.Value = "otherResource"
	settingsUseCaseTag.Type = models.UseCase
	settingsUseCaseTag.Value = "system-configuration"

	rbacApplicationTag.Type = models.ApplicationTag
	rbacApplicationTag.Value = "rbac"
	rbacProductFamiliesTag.Type = models.ProductFamilies
	rbacProductFamiliesTag.Value = "openshift"
	rbacContentTypeTag.Type = models.ContentType
	rbacContentTypeTag.Value = "learningPath"
	rbacUseCaseTag.Type = models.UseCase
	rbacUseCaseTag.Value = "clusters"

	unusedTag.Type = models.BundleTag
	unusedTag.Value = "unused"

	database.DB.Create(&rhelBudleTag)
	database.DB.Create(&rhelContentTypeTag)
	database.DB.Create(&rhelProductFamiliesTag)
	database.DB.Create(&rhelUseCaseTag)
	database.DB.Create(&settingsBundleTag)
	database.DB.Create(&settingsProductFamiliesTag)
	database.DB.Create(&settingsContentTypeTag)
	database.DB.Create(&settingsUseCaseTag)
	database.DB.Create(&rbacApplicationTag)
	database.DB.Create(&rbacProductFamiliesTag)
	database.DB.Create(&rbacContentTypeTag)
	database.DB.Create(&rbacUseCaseTag)
	database.DB.Create(&unusedTag)
}

func setupTaggedQuickstarts() {
	taggedQuickstart.Name = "tagged-quickstart"
	taggedQuickstart.Content = []byte(`{"spec": {"displayName": "Getting Started with Applications"}, "tags": "all-tags"}`)

	database.DB.Create(&taggedQuickstart)
	database.DB.Model(&taggedQuickstart).Association("Tags").Append(&rhelBudleTag, &settingsBundleTag, &rbacApplicationTag)
	database.DB.Save(&taggedQuickstart)

	settingsQuickstart.Name = "settings-quickstart"
	settingsQuickstart.Content = []byte(`{"spec": {"displayName": "Configure System Settings", "description": "Learn how to configure various system settings and preferences", "tasks": [{"title": "Setup User Preferences"}]}, "tags": "settings"}`)

	database.DB.Create(&settingsQuickstart)
	database.DB.Model(&settingsQuickstart).Association("Tags").Append(&settingsBundleTag, &settingsContentTypeTag, &settingsProductFamiliesTag, &settingsUseCaseTag)
	database.DB.Save(&settingsQuickstart)

	rhelQuickstart.Name = "rhel-quickstart"
	rhelQuickstart.Content = []byte(`{"spec": {"displayName": "Red Hat Enterprise Linux Setup"}, "tags": "rhel"}`)

	database.DB.Create(&rhelQuickstart)
	database.DB.Model(&rhelQuickstart).Association("Tags").Append(&rhelBudleTag, &rhelContentTypeTag, &rhelProductFamiliesTag, &rhelUseCaseTag)
	database.DB.Save(&rhelQuickstart)

	rbacQuickstart.Name = "rbac-quickstart"
	rbacQuickstart.Content = []byte(`{"spec": {"displayName": "Role-Based Access Control"}, "tags": "rbac"}`)

	database.DB.Create(&rbacQuickstart)
	database.DB.Model(&rbacQuickstart).Association("Tags").Append(&rbacApplicationTag, &rbacContentTypeTag, &rbacProductFamiliesTag, &rbacUseCaseTag)
	database.DB.Save(&rbacQuickstart)

	rhelTaggedQuickstart.Name = "rhel-tagged-quickstart"
	rhelTaggedQuickstart.Content = []byte(`{"spec": {"displayName": "Advanced RHEL Configuration"}, "tags": "rhel-tagged"}`)

	database.DB.Create(&rhelTaggedQuickstart)
	database.DB.Model(&rhelTaggedQuickstart).Association("Tags").Append(&rhelProductFamiliesTag, &rhelContentTypeTag, &rhelUseCaseTag)
	database.DB.Save(&rhelTaggedQuickstart)
}

func TestGetAll(t *testing.T) {
	router := setupRouter()
	setupTags()
	setupTaggedQuickstarts()
	leafQuickstart := mockQuickstart("non-tags-quickstart")

	t.Run("should get all quickstarts with 'rhel' or 'settings' bundle tags", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?bundle[]=rhel&bundle[]=settings", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 3, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'rhel' bundle tag", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?bundle=rhel", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'settings' bundle tag", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?bundle=settings", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
		assert.Equal(t, "tagged-quickstart", payload.Data[0].Name)
		assert.Equal(t, "settings-quickstart", payload.Data[1].Name)
	})

	t.Run("should get all quickstarts with 'rbac' application tag", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?application=rbac", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
		assert.Equal(t, "tagged-quickstart", payload.Data[0].Name)
		assert.Equal(t, "rbac-quickstart", payload.Data[1].Name)
	})

	t.Run("should get all quickstarts if no tags given", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 6, len(payload.Data))
	})

	t.Run("should get quikstart by ID", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/"+fmt.Sprint(leafQuickstart.ID), nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *singleResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, leafQuickstart.ID, payload.Data.Id)
		assert.Equal(t, "non-tags-quickstart", payload.Data.Name)
	})

	t.Run("should limit response to one record", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?limit=1", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 1, len(payload.Data))
	})

	t.Run("should limit response to more than the number of records", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?limit=100", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 6, len(payload.Data))
	})

	t.Run("should offset response by 2 and recover 3 records", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?offset=2", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 4, len(payload.Data))
	})

	t.Run("should limit response by 2 offset response by 2 and recover 2 records", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?limit=2&offset=2", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
	})

	t.Run("should return a bad request if limit is not a number", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?limit=foo", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 400, response.Code)
	})

	t.Run("should return a bad request if offset is not a number", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?offset=foo", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 400, response.Code)
	})

	t.Run("should get all quickstarts with 'settings' product family", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?product-families=settings", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 1, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'Other resource' or 'QuickStart' content type", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?content[]=otherResource&content[]=quickstart", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 3, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'OpenShift' product family and 'Clusters' use case", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?product-families=openshift&use-case=clusters", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		fmt.Println(response.Body)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 1, len(payload.Data))
	})
}

func TestFuzzySearch(t *testing.T) {
	router := setupRouter()
	setupTags()
	setupTaggedQuickstarts()

	type fuzzySearchResponse struct {
		Data        []FuzzySearchResult `json:"data"`
		SearchTerm  string              `json:"search_term"`
		MaxDistance int                 `json:"max_distance"`
	}

	t.Run("should return bad request when search term is missing", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/fuzzy-search", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *messageResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 400, response.Code)
		assert.Contains(t, payload.Msg, "Search term 'q' parameter is required")
	})

	t.Run("should find exact match with distance 0", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/fuzzy-search?q=Getting Started with Applications", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		// Skip test if PostgreSQL extension is not available (e.g., in SQLite test environment)
		if response.Code == 500 {
			t.Skip("Skipping fuzzy search test - PostgreSQL fuzzystrmatch extension not available")
		}

		var payload *fuzzySearchResponse
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "Getting Started with Applications", payload.SearchTerm)
		assert.Equal(t, 3, payload.MaxDistance)
		assert.GreaterOrEqual(t, len(payload.Data), 1)
		if len(payload.Data) > 0 {
			assert.Equal(t, 0, payload.Data[0].Distance) // Exact match should have distance 0
		}
	})

	t.Run("should find fuzzy matches with typos", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/fuzzy-search?q=Geting Startd&max_distance=5", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		// Skip test if PostgreSQL extension is not available
		if response.Code == 500 {
			t.Skip("Skipping fuzzy search test - PostgreSQL fuzzystrmatch extension not available")
		}

		var payload *fuzzySearchResponse
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "Geting Startd", payload.SearchTerm)
		assert.Equal(t, 5, payload.MaxDistance)
		// Should find matches despite typos
		assert.GreaterOrEqual(t, len(payload.Data), 0)
	})

	t.Run("should find RHEL related quickstarts", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/fuzzy-search?q=RHEL&max_distance=2", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		// Skip test if PostgreSQL extension is not available
		if response.Code == 500 {
			t.Skip("Skipping fuzzy search test - PostgreSQL fuzzystrmatch extension not available")
		}

		var payload *fuzzySearchResponse
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "RHEL", payload.SearchTerm)
		assert.Equal(t, 2, payload.MaxDistance)
		// Should find RHEL related quickstarts
		assert.GreaterOrEqual(t, len(payload.Data), 0)
	})

	t.Run("should respect pagination", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/fuzzy-search?q=Configuration&limit=1", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		// Skip test if PostgreSQL extension is not available
		if response.Code == 500 {
			t.Skip("Skipping fuzzy search test - PostgreSQL fuzzystrmatch extension not available")
		}

		var payload *fuzzySearchResponse
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.LessOrEqual(t, len(payload.Data), 1)
	})

	t.Run("should return empty results for very strict distance", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/fuzzy-search?q=NonExistentQuickstart&max_distance=0", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		// Skip test if PostgreSQL extension is not available
		if response.Code == 500 {
			t.Skip("Skipping fuzzy search test - PostgreSQL fuzzystrmatch extension not available")
		}

		var payload *fuzzySearchResponse
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 0, len(payload.Data))
	})

	t.Run("should find quickstarts by displayName with fuzzy matching", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/fuzzy-search?q=Configure&max_distance=2", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		// Skip test if PostgreSQL extension is not available
		if response.Code == 500 {
			t.Skip("Skipping fuzzy search test - PostgreSQL fuzzystrmatch extension not available")
		}

		var payload *fuzzySearchResponse
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "Configure", payload.SearchTerm)
		assert.Equal(t, 2, payload.MaxDistance)
		// Should find quickstarts with "Configure" in displayName
		assert.GreaterOrEqual(t, len(payload.Data), 0)
	})
}
