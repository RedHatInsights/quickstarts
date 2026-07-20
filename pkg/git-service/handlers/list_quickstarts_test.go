package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListQuickstarts_Success(t *testing.T) {
	repo := &mockRepoManager{
		directories: []string{"getting-started", "cost-management"},
		fileContents: map[string]string{
			"/docs/quickstarts/getting-started/getting-started.yml": "spec:\n  displayName: Getting Started\n",
			"/docs/quickstarts/cost-management/cost-management.yml": "spec:\n  displayName: Cost Management\n",
		},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-quickstarts", nil)
	rec := httptest.NewRecorder()

	handler.ListQuickstarts(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListQuickstartsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Len(t, resp.Quickstarts, 2)
	assert.Equal(t, "getting-started", resp.Quickstarts[0].Name)
	assert.Equal(t, "Getting Started", resp.Quickstarts[0].DisplayName)
	assert.Equal(t, "cost-management", resp.Quickstarts[1].Name)
	assert.Equal(t, "Cost Management", resp.Quickstarts[1].DisplayName)
}

func TestListQuickstarts_FallsBackToYamlExtension(t *testing.T) {
	repo := &mockRepoManager{
		directories: []string{"my-quickstart"},
		fileContents: map[string]string{
			"/docs/quickstarts/my-quickstart/my-quickstart.yaml": "spec:\n  displayName: My Quickstart\n",
		},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-quickstarts", nil)
	rec := httptest.NewRecorder()

	handler.ListQuickstarts(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListQuickstartsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "My Quickstart", resp.Quickstarts[0].DisplayName)
}

func TestListQuickstarts_FallsBackToDirName(t *testing.T) {
	repo := &mockRepoManager{
		directories: []string{"no-yaml-here"},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-quickstarts", nil)
	rec := httptest.NewRecorder()

	handler.ListQuickstarts(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListQuickstartsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "no-yaml-here", resp.Quickstarts[0].DisplayName)
}

func TestListQuickstarts_PullLatestError(t *testing.T) {
	repo := &mockRepoManager{pullLatestErr: fmt.Errorf("network error")}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-quickstarts", nil)
	rec := httptest.NewRecorder()

	handler.ListQuickstarts(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to pull latest changes")
}

func TestListQuickstarts_ListDirectoriesError(t *testing.T) {
	repo := &mockRepoManager{listDirsErr: fmt.Errorf("permission denied")}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-quickstarts", nil)
	rec := httptest.NewRecorder()

	handler.ListQuickstarts(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to list quickstarts")
}

func TestListQuickstarts_EmptyRepo(t *testing.T) {
	repo := &mockRepoManager{
		directories: []string{},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/list-quickstarts", nil)
	rec := httptest.NewRecorder()

	handler.ListQuickstarts(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp ListQuickstartsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Empty(t, resp.Quickstarts)
}

func TestGetQuickstartContent_Success(t *testing.T) {
	repo := &mockRepoManager{
		files: []string{"metadata.yaml", "getting-started.yml"},
		fileContents: map[string]string{
			"/docs/quickstarts/getting-started/metadata.yaml":      "kind: QuickStarts\n",
			"/docs/quickstarts/getting-started/getting-started.yml": "spec:\n  displayName: Getting Started\n",
		},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	r := chi.NewRouter()
	r.Get("/api/v1/quickstart-content/{name}", handler.GetQuickstartContent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quickstart-content/getting-started", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp QuickstartContentResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "getting-started", resp.Name)
	assert.Len(t, resp.Files, 2)
	assert.Equal(t, "metadata.yaml", resp.Files[0].Name)
	assert.Equal(t, "getting-started.yml", resp.Files[1].Name)
}

func TestGetQuickstartContent_NotFound(t *testing.T) {
	repo := &mockRepoManager{
		listFilesErr: fmt.Errorf("no such directory"),
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	r := chi.NewRouter()
	r.Get("/api/v1/quickstart-content/{name}", handler.GetQuickstartContent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quickstart-content/nonexistent", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
	assert.Contains(t, rec.Body.String(), "quickstart not found")
}

func TestGetQuickstartContent_PathTraversal(t *testing.T) {
	handler := NewHandler(&mockRepoManager{}, &mockGitHubClient{}, "", "/docs/quickstarts/")

	r := chi.NewRouter()
	r.Get("/api/v1/quickstart-content/{name}", handler.GetQuickstartContent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quickstart-content/foo..bar", nil)
	rec := httptest.NewRecorder()

	// chi decodes URL params, so test with a name containing ".." as a path segment
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("name", "../../etc")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	handler.GetQuickstartContent(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Contains(t, rec.Body.String(), "name contains invalid path segment")
}

func TestGetQuickstartContent_PullLatestError(t *testing.T) {
	repo := &mockRepoManager{pullLatestErr: fmt.Errorf("network error")}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	r := chi.NewRouter()
	r.Get("/api/v1/quickstart-content/{name}", handler.GetQuickstartContent)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/quickstart-content/test", nil)
	rec := httptest.NewRecorder()

	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to pull latest changes")
}

func TestExtractDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		yaml     string
		expected string
	}{
		{"valid", "spec:\n  displayName: My Quickstart\n", "My Quickstart"},
		{"missing", "spec:\n  description: test\n", ""},
		{"invalid yaml", "not: [valid: yaml", ""},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, extractDisplayName(tc.yaml))
		})
	}
}
