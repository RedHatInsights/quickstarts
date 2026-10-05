package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCreatorPRs_Success(t *testing.T) {
	updated := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	gh := &mockGitHubClient{
		listPRs: []ghclient.CreatorPR{
			{
				Number:     11,
				Title:      "feat(quickstarts): create demo",
				HTMLURL:    "https://github.com/org/repo/pull/11",
				UpdatedAt:  updated,
				BranchName: "qs-create-demo-1",
				Slug:       "demo",
			},
		},
	}
	repo := &mockRepoManager{}
	handler := NewHandler(repo, gh, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/creator-prs", nil)
	rec := httptest.NewRecorder()

	handler.ListCreatorPRs(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp ListCreatorPRsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.PullRequests, 1)
	assert.Equal(t, 11, resp.PullRequests[0].Number)
	assert.Equal(t, "qs-create-demo-1", resp.PullRequests[0].BranchName)
	assert.Equal(t, "demo", resp.PullRequests[0].Slug)
	assert.Equal(t, 0, repo.pullLatestCount)
}

func TestListCreatorPRs_GitHubError(t *testing.T) {
	gh := &mockGitHubClient{listPRsErr: fmt.Errorf("github down")}
	handler := NewHandler(&mockRepoManager{}, gh, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/creator-prs", nil)
	rec := httptest.NewRecorder()
	handler.ListCreatorPRs(rec, req)

	assert.Equal(t, http.StatusBadGateway, rec.Code)
}

func TestGetCreatorPR_Success(t *testing.T) {
	gh := &mockGitHubClient{
		getPR: &ghclient.CreatorPR{
			Number:     11,
			Title:      "feat(quickstarts): create demo",
			HTMLURL:    "https://github.com/org/repo/pull/11",
			BranchName: "qs-create-demo-1",
			Slug:       "demo",
		},
		prFilesSlug: "demo",
		prFiles: []ghclient.File{
			{Name: "metadata.yml", Content: "kind: QuickStarts\n"},
			{Name: "demo.yml", Content: "spec:\n  displayName: Demo\n"},
		},
	}
	handler := NewHandler(&mockRepoManager{}, gh, "", "/docs/quickstarts/")

	r := chi.NewRouter()
	r.Get("/api/v1/creator-prs/{number}", handler.GetCreatorPR)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/creator-prs/11", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp CreatorPRContentResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, 11, resp.Number)
	assert.Equal(t, "demo", resp.Slug)
	assert.Equal(t, "qs-create-demo-1", resp.BranchName)
	require.Len(t, resp.Files, 2)
	assert.Equal(t, "metadata.yml", resp.Files[0].Name)
}

func TestGetCreatorPR_NotFound(t *testing.T) {
	gh := &mockGitHubClient{getPRErr: ghclient.ErrNotFound}
	handler := NewHandler(&mockRepoManager{}, gh, "", "/docs/quickstarts/")

	r := chi.NewRouter()
	r.Get("/api/v1/creator-prs/{number}", handler.GetCreatorPR)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/creator-prs/99", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestGetCreatorPR_InvalidNumber(t *testing.T) {
	handler := NewHandler(&mockRepoManager{}, &mockGitHubClient{}, "", "/docs/quickstarts/")

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("number", "0")
	req := httptest.NewRequest(http.MethodGet, "/api/v1/creator-prs/0", nil)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))
	rec := httptest.NewRecorder()

	handler.GetCreatorPR(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
