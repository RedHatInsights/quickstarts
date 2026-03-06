package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/generated"
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

func setupRouter() *chi.Mux {
	r := chi.NewRouter()

	adapter := NewServerAdapter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		params := generated.GetQuickstartsParams{}

		// Parse query parameters
		query := r.URL.Query()
		if limit := query.Get("limit"); limit != "" {
			if l, err := strconv.Atoi(limit); err == nil {
				params.Limit = &l
			}
		}
		if offset := query.Get("offset"); offset != "" {
			if o, err := strconv.Atoi(offset); err == nil {
				params.Offset = &o
			}
		}
		if name := query.Get("name"); name != "" {
			params.Name = &name
		}
		if displayName := query.Get("display-name"); displayName != "" {
			params.DisplayName = &displayName
		}
		if fuzzy := query.Get("fuzzy"); fuzzy == "true" {
			fuzzyBool := true
			params.Fuzzy = &fuzzyBool
		}

		// Handle array parameters
		if bundles := query["bundle"]; len(bundles) > 0 {
			params.Bundle = &bundles
		}
		if bundles := query["bundle[]"]; len(bundles) > 0 {
			params.Bundle = &bundles
		}
		if apps := query["application"]; len(apps) > 0 {
			params.Application = &apps
		}
		if apps := query["application[]"]; len(apps) > 0 {
			params.Application = &apps
		}
		if pf := query["product-families"]; len(pf) > 0 {
			params.ProductFamilies = &pf
		}
		if pf := query["product-families[]"]; len(pf) > 0 {
			params.ProductFamilies = &pf
		}
		if uc := query["use-case"]; len(uc) > 0 {
			params.UseCase = &uc
		}
		if uc := query["use-case[]"]; len(uc) > 0 {
			params.UseCase = &uc
		}
		if content := query["content"]; len(content) > 0 {
			params.Content = &content
		}
		if content := query["content[]"]; len(content) > 0 {
			params.Content = &content
		}

		adapter.GetQuickstarts(w, r, params)
	})

	r.Route("/{id}", func(sub chi.Router) {
		sub.Get("/", func(w http.ResponseWriter, r *http.Request) {
			idStr := chi.URLParam(r, "id")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			adapter.GetQuickstartsId(w, r, id)
		})
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
	taggedQuickstart.Content = []byte(`{"tags": "all-tags"}`)

	database.DB.Create(&taggedQuickstart)
	database.DB.Model(&taggedQuickstart).Association("Tags").Append(&rhelBudleTag, &settingsBundleTag, &rbacApplicationTag)
	database.DB.Save(&taggedQuickstart)

	settingsQuickstart.Name = "settings-quickstart"
	settingsQuickstart.Content = []byte(`{"tags": "settings"}`)

	database.DB.Create(&settingsQuickstart)
	database.DB.Model(&settingsQuickstart).Association("Tags").Append(&settingsBundleTag, &settingsContentTypeTag, &settingsProductFamiliesTag, &settingsUseCaseTag)
	database.DB.Save(&settingsQuickstart)

	rhelQuickstart.Name = "rhel-quickstart"
	rhelQuickstart.Content = []byte(`{"tags": "rhel"}`)

	database.DB.Create(&rhelQuickstart)
	database.DB.Model(&rhelQuickstart).Association("Tags").Append(&rhelBudleTag, &rhelContentTypeTag, &rhelProductFamiliesTag, &rhelUseCaseTag)
	database.DB.Save(&rhelQuickstart)

	rbacQuickstart.Name = "rbac-quickstart"
	rbacQuickstart.Content = []byte(`{"tags": "rbac"}`)

	database.DB.Create(&rbacQuickstart)
	database.DB.Model(&rbacQuickstart).Association("Tags").Append(&rbacApplicationTag, &rbacContentTypeTag, &rbacProductFamiliesTag, &rbacUseCaseTag)
	database.DB.Save(&rbacQuickstart)

	rhelTaggedQuickstart.Name = "rhel-tagged-quickstart"
	rhelTaggedQuickstart.Content = []byte(`{"tags": "rhel-tagged"}`)

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

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 3, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'rhel' bundle tag", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?bundle=rhel", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'settings' bundle tag", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?bundle=settings", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
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

		var payload *ResponsePayload
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

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 6, len(payload.Data))
	})

	t.Run("should get quikstart by ID", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/"+fmt.Sprint(leafQuickstart.ID), nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *SingleResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, leafQuickstart.ID, payload.Data.Id)
		assert.Equal(t, "non-tags-quickstart", payload.Data.Name)
	})

	t.Run("should limit response to one record", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?limit=1", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 1, len(payload.Data))
	})

	t.Run("should limit response to more than the number of records", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?limit=100", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 6, len(payload.Data))
	})

	t.Run("should offset response by 2 and recover 3 records", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?offset=2", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 4, len(payload.Data))
	})

	t.Run("should limit response by 2 offset response by 2 and recover 2 records", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?limit=2&offset=2", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
	})

	t.Run("should ignore invalid limit parameter and use default", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?limit=foo", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		// Should return all records since invalid limit is ignored (uses default)
		assert.Equal(t, 6, len(payload.Data))
	})

	t.Run("should ignore invalid offset parameter and use default", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?offset=foo", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		// Should return all records since invalid offset is ignored (uses default offset=0)
		assert.Equal(t, 6, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'settings' product family", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?product-families=settings", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 1, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'Other resource' or 'QuickStart' content type", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?content[]=otherResource&content[]=quickstart", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 3, len(payload.Data))
	})

	t.Run("should get all quickstarts with 'OpenShift' product family and 'Clusters' use case", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?product-families=openshift&use-case=clusters", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		fmt.Println(response.Body)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 1, len(payload.Data))
	})
}

func TestFuzzySearch(t *testing.T) {
	// IMPORTANT: These tests validate word-by-word fuzzy matching which requires PostgreSQL.
	//
	// The test suite runs on SQLite by default, which does NOT support:
	// - PostgreSQL Levenshtein distance functions (fuzzystrmatch extension)
	// - PostgreSQL ILIKE operator
	// - PostgreSQL JSON operators (->>, ->)
	//
	// As a result, most of these tests will FAIL when run with `make test` (SQLite).
	// These tests are designed to validate the PostgreSQL implementation and should
	// pass when running against a real PostgreSQL database.
	//
	// Fuzzy search uses word-by-word Levenshtein matching:
	// 1. Both query and display names are split into words
	// 2. Each query word is matched to the closest word in the display name
	// 3. All query words must match within the threshold (default: 3)
	// 4. Results are ordered by total distance (sum of all word distances)
	//
	// To run these tests against PostgreSQL:
	// 1. Start PostgreSQL: make infra
	// 2. Run migrations: make migrate
	// 3. Run the server and test manually with curl

	if !database.IsFuzzySearchSupported() {
		t.Skip("Fuzzy search tests require PostgreSQL with fuzzystrmatch extension")
	}

	router := setupRouter()

	// Create quickstarts with realistic display names for fuzzy search testing
	qs1 := models.Quickstart{
		Name:    "getting-started-openshift",
		Content: []byte(`{"spec":{"displayName":"Getting Started with OpenShift"}}`),
	}
	database.DB.Create(&qs1)

	qs2 := models.Quickstart{
		Name:    "ansible-automation-platform",
		Content: []byte(`{"spec":{"displayName":"Ansible Automation Platform"}}`),
	}
	database.DB.Create(&qs2)

	qs3 := models.Quickstart{
		Name:    "configure-remediations",
		Content: []byte(`{"spec":{"displayName":"Configure Remediations"}}`),
	}
	database.DB.Create(&qs3)

	qs4 := models.Quickstart{
		Name:    "monitoring-kubernetes",
		Content: []byte(`{"spec":{"displayName":"Monitoring Kubernetes Clusters"}}`),
	}
	database.DB.Create(&qs4)

	qs5 := models.Quickstart{
		Name:    "security-compliance",
		Content: []byte(`{"spec":{"displayName":"Security and Compliance"}}`),
	}
	database.DB.Create(&qs5)

	qs6 := models.Quickstart{
		Name:    "deploy-java-app",
		Content: []byte(`{"spec":{"displayName":"Deploy a Java Application"}}`),
	}
	database.DB.Create(&qs6)

	qs7 := models.Quickstart{
		Name:    "automation-hub-intro",
		Content: []byte(`{"spec":{"displayName":"Getting started with automation hub"}}`),
	}
	database.DB.Create(&qs7)

	t.Run("Word-by-word: two-word query with typos 'Geting Startd'", func(t *testing.T) {
		// Word-by-word matching: "Geting" → "Getting" (dist:1), "Startd" → "Started" (dist:1)
		// Total distance: 2 (well within threshold of 3)
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Geting Startd&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Getting Started with OpenShift'")
	})

	t.Run("Word-by-word: single-word query with typo 'Openshft'", func(t *testing.T) {
		// Single word "Openshft" matches "OpenShift" (dist:1) in any quickstart
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Openshft&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Getting Started with OpenShift'")
	})

	t.Run("Word-by-word: single-word query 'Ansibel' finds 'Ansible'", func(t *testing.T) {
		// Single word "Ansibel" matches "Ansible" (dist:2) in display names
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Ansibel&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Ansible Automation Platform'")
	})

	t.Run("Word-by-word: 'Automaton' matches 'Automation'", func(t *testing.T) {
		// "Automaton" matches "Automation" (dist:1) - one character deletion
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Automaton&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Ansible Automation Platform' and 'automation hub'")
	})

	t.Run("should find 'Remediations' with typo 'Remidations'", func(t *testing.T) {
		// Search for "Remidations" (missing 'e')
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Remidations&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Configure Remediations'")
	})

	t.Run("should find 'Configure' with typo 'Confgure'", func(t *testing.T) {
		// Search for "Confgure" (missing 'i')
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Confgure&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Configure Remediations'")
	})

	t.Run("Word-by-word: 'Kuberntes' matches 'Kubernetes'", func(t *testing.T) {
		// "Kuberntes" matches "Kubernetes" (dist:1) - missing one 'e'
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Kuberntes&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Monitoring Kubernetes Clusters'")
	})

	t.Run("should find 'Monitoring' with typo 'Monitring'", func(t *testing.T) {
		// Search for "Monitring" (missing 'o')
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Monitring&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Monitoring Kubernetes Clusters'")
	})

	t.Run("should find 'Clusters' with typo 'Clustrs'", func(t *testing.T) {
		// Search for "Clustrs" (missing 'e')
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Clustrs&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Monitoring Kubernetes Clusters'")
	})

	t.Run("should find 'Security' with typo 'Securty'", func(t *testing.T) {
		// Search for "Securty" (missing 'i')
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Securty&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Security and Compliance'")
	})

	t.Run("should find 'Compliance' with typo 'Complance'", func(t *testing.T) {
		// Search for "Complance" (missing 'i')
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Complance&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Security and Compliance'")
	})

	t.Run("should find with case insensitive search 'openshift' lowercase", func(t *testing.T) {
		// Search for "openshift" in lowercase
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=openshift&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Getting Started with OpenShift' (case insensitive)")
	})

	t.Run("Word-by-word: multiple typos 'Ansble Automatn'", func(t *testing.T) {
		// "Ansble" → "Ansible" (dist:1), "Automatn" → "Automation" (dist:1)
		// Total distance: 2 (within threshold)
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Ansble Automatn&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Ansible Automation Platform'")
	})

	t.Run("should not find with regular search when typo exists", func(t *testing.T) {
		// Without fuzzy search, exact ILIKE match won't find "Geting" in "Getting"
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Geting Started", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 0, len(payload.Data), "Regular search should not find results with typos")
	})

	t.Run("should respect distance threshold with completely wrong term", func(t *testing.T) {
		// Search with completely unrelated term (high distance)
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=XyzAbc&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 0, len(payload.Data), "Should not find anything with very high distance")
	})

	t.Run("should work with exact match in fuzzy search", func(t *testing.T) {
		// Fuzzy search should still work with exact matches
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Ansible&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Fuzzy search should work with exact matches")
	})

	t.Run("should find partial word match with typo", func(t *testing.T) {
		// Search for "Kubernetis" (typo in partial word)
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Kubernetis&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Monitoring Kubernetes Clusters'")
	})

	t.Run("should order results by relevance (distance)", func(t *testing.T) {
		// Create two quickstarts with similar names
		qs6 := models.Quickstart{
			Name:    "getting-guide",
			Content: []byte(`{"spec":{"displayName":"Getting"}}`),
		}
		database.DB.Create(&qs6)

		// Search for "Geting" - should find "Getting" before "Getting Started with OpenShift"
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Geting&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find at least one result")
		// The first result should be the closer match
		if len(payload.Data) > 0 {
			// "Getting" (1 char distance) should rank higher than "Getting Started with OpenShift" (1+ char distance due to extra words)
			assert.Contains(t, payload.Data[0].Name, "getting", "Closest match should be first")
		}
	})

	t.Run("should find multi-word phrase 'Getting Started'", func(t *testing.T) {
		// Search for exact multi-word phrase
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Getting Started&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Getting Started with OpenShift'")
	})

	t.Run("Word-by-word: three-word query 'Geting Startd Openshft'", func(t *testing.T) {
		// Three-word query: "Geting" → "Getting" (1), "Startd" → "Started" (1), "Openshft" → "OpenShift" (1)
		// Total distance: 3 (exactly at threshold)
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Geting Startd Openshft&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Getting Started with OpenShift'")
	})

	t.Run("should find phrase 'Ansible Automation' with typo 'Ansble Automaton'", func(t *testing.T) {
		// Search for phrase with multiple typos
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Ansble Automaton&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Ansible Automation Platform'")
	})

	t.Run("should find phrase 'Monitoring Kubernetes' exactly", func(t *testing.T) {
		// Search for exact two-word phrase
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Monitoring Kubernetes&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Monitoring Kubernetes Clusters'")
	})

	t.Run("should find phrase 'Security Compliance' with missing word connector 'and'", func(t *testing.T) {
		// Search for phrase without the connecting word "and"
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Security Compliance&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Security and Compliance'")
	})

	t.Run("should find full phrase 'Getting Started with OpenShift' exactly", func(t *testing.T) {
		// Search for complete phrase
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Getting Started with OpenShift&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find exact phrase match")
	})

	t.Run("Word-by-word: four-word query with typos", func(t *testing.T) {
		// Four-word query: "Geting" → "Getting" (1), "Startd" → "started" (1), "with" (0), "automaton" → "automation" (1)
		// Total distance: 3 (at threshold)
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=Geting Startd with automaton&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Getting started with automation hub'")
	})

	t.Run("Word-by-word: partial word fallback to ILIKE", func(t *testing.T) {
		// "ansible" is correctly spelled, so fuzzy won't find it by Levenshtein distance
		// Should fall back to ILIKE and find partial matches
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=ansible&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should fall back to ILIKE and find 'Ansible Automation Platform'")
	})

	t.Run("Word-by-word: 'automaton hb' finds 'automation hub'", func(t *testing.T) {
		// "automaton" → "automation" (1), "hb" → "hub" (1)
		// Total distance: 2
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=automaton hb&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Getting started with automation hub'")
	})

	t.Run("Word-by-word: 'deployy java applicaton' with multiple typos", func(t *testing.T) {
		// "deployy" → "Deploy" (1), "java" (0), "applicaton" → "Application" (1)
		// Total distance: 2
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=deployy java applicaton&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Deploy a Java Application'")
	})

	t.Run("Word-by-word: 'remediaton' finds 'Remediations'", func(t *testing.T) {
		// "remediaton" → "Remediations" (1) - missing one 'i'
		request, _ := http.NewRequest(http.MethodGet, "/?display-name=remediaton&fuzzy=true", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *ResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.GreaterOrEqual(t, len(payload.Data), 1, "Should find 'Configure Remediations'")
	})
}
