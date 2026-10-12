package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/clients"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetCreatorPrs_Success(t *testing.T) {
	mockGitService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/creator-prs", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"pullRequests": []map[string]interface{}{
				{
					"number":     11,
					"title":      "feat(quickstarts): create demo",
					"htmlUrl":    "https://github.com/org/repo/pull/11",
					"updatedAt":  "2026-09-29T12:00:00Z",
					"branchName": "qs-create-demo-1",
					"slug":       "demo",
				},
			},
		})
	}))
	defer mockGitService.Close()

	adapter := NewServerAdapter()
	adapter.gitServiceClient = clients.NewGitService(mockGitService.URL, "psk")
	adapter.gitServiceEnabled = true

	req := httptest.NewRequest(http.MethodGet, "/creator-prs", nil)
	w := httptest.NewRecorder()
	adapter.GetCreatorPrs(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]interface{})
	prs := data["pullRequests"].([]interface{})
	require.Len(t, prs, 1)
	assert.Equal(t, float64(11), prs[0].(map[string]interface{})["number"])
}

func TestGetCreatorPrs_Disabled(t *testing.T) {
	adapter := NewServerAdapter()
	adapter.gitServiceEnabled = false
	req := httptest.NewRequest(http.MethodGet, "/creator-prs", nil)
	w := httptest.NewRecorder()
	adapter.GetCreatorPrs(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCreatorPrsNumber_Success(t *testing.T) {
	mockGitService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/creator-prs/11", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":     11,
			"title":      "feat(quickstarts): create demo",
			"htmlUrl":    "https://github.com/org/repo/pull/11",
			"branchName": "qs-create-demo-1",
			"slug":       "demo",
			"files": []map[string]string{
				{"name": "metadata.yml", "content": "kind: QuickStarts\n"},
			},
		})
	}))
	defer mockGitService.Close()

	adapter := NewServerAdapter()
	adapter.gitServiceClient = clients.NewGitService(mockGitService.URL, "")
	adapter.gitServiceEnabled = true

	req := httptest.NewRequest(http.MethodGet, "/creator-prs/11", nil)
	w := httptest.NewRecorder()
	adapter.GetCreatorPrsNumber(w, req, 11)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	data := resp["data"].(map[string]interface{})
	assert.Equal(t, float64(11), data["number"])
	assert.Equal(t, "demo", data["slug"])
	files := data["files"].([]interface{})
	require.Len(t, files, 1)
}

func TestGetCreatorPrsNumber_NotFound(t *testing.T) {
	mockGitService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "msg": "creator pull request not found"})
	}))
	defer mockGitService.Close()

	adapter := NewServerAdapter()
	adapter.gitServiceClient = clients.NewGitService(mockGitService.URL, "")
	adapter.gitServiceEnabled = true

	req := httptest.NewRequest(http.MethodGet, "/creator-prs/99", nil)
	w := httptest.NewRecorder()
	adapter.GetCreatorPrsNumber(w, req, 99)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCreatorPrs_PSKHeader(t *testing.T) {
	mockGitService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "secret", r.Header.Get("X-PSK-Token"))
		json.NewEncoder(w).Encode(map[string]interface{}{"pullRequests": []interface{}{}})
	}))
	defer mockGitService.Close()

	adapter := NewServerAdapter()
	adapter.gitServiceClient = clients.NewGitService(mockGitService.URL, "secret")
	adapter.gitServiceEnabled = true

	req := httptest.NewRequest(http.MethodGet, "/creator-prs", nil)
	w := httptest.NewRecorder()
	adapter.GetCreatorPrs(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}
