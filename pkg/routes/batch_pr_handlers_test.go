package routes

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/clients"
	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func gitServiceAdapter(t *testing.T, handler http.HandlerFunc) *ServerAdapter {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	adapter := NewServerAdapter()
	adapter.gitServiceClient = clients.NewGitService(server.URL, "psk")
	adapter.gitServiceEnabled = true
	return adapter
}

const batchBody = `{
	"metadata": {"prNumber": 42, "expectedHeadSha": "abc", "commitMessage": "docs: update links"},
	"resources": [{"name": "alpha", "files": [{"name": "alpha.yml", "content": "a: 1\n"}]}]
}`

func TestPostPullRequestBatch_Success(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/submit-batch-pr", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "psk", r.Header.Get("X-PSK-Token"))

		body, _ := io.ReadAll(r.Body)
		var got clients.GitServiceBatchRequest
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, 42, got.Metadata.PRNumber)
		assert.Equal(t, "abc", got.Metadata.ExpectedHeadSHA)
		require.Len(t, got.Resources, 1)
		assert.Equal(t, "alpha", got.Resources[0].Name)
		assert.Equal(t, "alpha.yml", got.Resources[0].Files[0].Name)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"prUrl": "https://github.com/o/r/pull/42", "prNumber": 42, "branchName": "qs-update-batch-1",
			"headSha": "def", "status": "updated", "paths": []string{"docs/quickstarts/alpha/alpha.yml"},
		})
	})

	w := httptest.NewRecorder()
	adapter.PostPullRequestBatch(w, httptest.NewRequest(http.MethodPost, "/pull-request/batch", bytes.NewBufferString(batchBody)))

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Data generated.BatchPrResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, 42, *resp.Data.PrNumber)
	assert.Equal(t, "def", *resp.Data.HeadSha)
	assert.Equal(t, []string{"docs/quickstarts/alpha/alpha.yml"}, *resp.Data.Paths)
}

func TestPostPullRequestBatch_PassesThroughActionableErrors(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusNotFound, http.StatusConflict} {
		adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(status)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "msg": "pull request #42 has new commits"})
		})

		w := httptest.NewRecorder()
		adapter.PostPullRequestBatch(w, httptest.NewRequest(http.MethodPost, "/pull-request/batch", bytes.NewBufferString(batchBody)))
		assert.Equal(t, status, w.Code)
		assert.Contains(t, w.Body.String(), "has new commits")
	}
}

func TestPostPullRequestBatch_ServerErrorIsBadGateway(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "msg": "internal detail"})
	})

	w := httptest.NewRecorder()
	adapter.PostPullRequestBatch(w, httptest.NewRequest(http.MethodPost, "/pull-request/batch", bytes.NewBufferString(batchBody)))
	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.NotContains(t, w.Body.String(), "internal detail")
}

func TestPostPullRequestBatch_Validation(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("git-service must not be called for an invalid request")
	})
	for _, body := range []string{
		`{`,
		`{"metadata": {"commitMessage": "x"}, "resources": []}`,
		`{"metadata": {}, "resources": [{"name": "a", "files": [{"name": "a.yml", "content": ""}]}]}`,
	} {
		w := httptest.NewRecorder()
		adapter.PostPullRequestBatch(w, httptest.NewRequest(http.MethodPost, "/pull-request/batch", bytes.NewBufferString(body)))
		assert.Equal(t, http.StatusBadRequest, w.Code, body)
	}
}

func TestPostPullRequestBatch_Disabled(t *testing.T) {
	adapter := NewServerAdapter()
	w := httptest.NewRecorder()
	adapter.PostPullRequestBatch(w, httptest.NewRequest(http.MethodPost, "/pull-request/batch", bytes.NewBufferString(batchBody)))
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetCreatorPrsNumberWorkspace(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/creator-prs/42/workspace", r.URL.Path)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number": 42, "title": "Update docs", "htmlUrl": "u", "branchName": "qs-update-batch-1", "headSha": "abc",
			"resources": []map[string]interface{}{{
				"name": "alpha", "path": "docs/quickstarts/alpha",
				"files": []map[string]string{{"name": "metadata.yml", "path": "docs/quickstarts/alpha/metadata.yml", "content": "name: alpha\n"}},
			}},
			"missing": []string{"gamma"},
		})
	})

	w := httptest.NewRecorder()
	adapter.GetCreatorPrsNumberWorkspace(w, httptest.NewRequest(http.MethodGet, "/creator-prs/42/workspace", nil), 42)

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Data generated.CreatorPrWorkspaceResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "abc", *resp.Data.HeadSha)
	require.Len(t, *resp.Data.Resources, 1)
	res := (*resp.Data.Resources)[0]
	assert.Equal(t, "docs/quickstarts/alpha/metadata.yml", *(*res.Files)[0].Path)
	assert.Equal(t, []string{"gamma"}, *resp.Data.Missing)
}

func TestGetCreatorPrsNumberWorkspace_TooLarge(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "msg": "too many files"})
	})

	w := httptest.NewRecorder()
	adapter.GetCreatorPrsNumberWorkspace(w, httptest.NewRequest(http.MethodGet, "/creator-prs/42/workspace", nil), 42)
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestGetCreatorPrsNumberResourcesName(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/creator-prs/42/resources/delta", r.URL.Path)
		assert.Equal(t, "abc", r.URL.Query().Get("headSha"))
		json.NewEncoder(w).Encode(map[string]interface{}{
			"headSha": "abc",
			"resource": map[string]interface{}{
				"name": "delta", "path": "docs/quickstarts/delta",
				"files": []map[string]string{{"name": "delta.yml", "path": "docs/quickstarts/delta/delta.yml", "content": "x: 1\n"}},
			},
		})
	})

	head := "abc"
	w := httptest.NewRecorder()
	adapter.GetCreatorPrsNumberResourcesName(w, httptest.NewRequest(http.MethodGet, "/creator-prs/42/resources/delta", nil), 42, "delta",
		generated.GetCreatorPrsNumberResourcesNameParams{HeadSha: &head})

	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	var resp struct {
		Data generated.CreatorPrResourceResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "delta", *resp.Data.Resource.Name)
}

func TestGetCreatorPrsNumber_MultiResourceIsConflict(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "msg": "pull request changes more than one quickstart"})
	})

	w := httptest.NewRecorder()
	adapter.GetCreatorPrsNumber(w, httptest.NewRequest(http.MethodGet, "/creator-prs/42", nil), 42)
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetRepoQuickstarts_IncludesHeadSha(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"quickstarts": []map[string]string{{"name": "alpha", "displayName": "Alpha"}},
			"headSha":     "abc",
		})
	})

	w := httptest.NewRecorder()
	adapter.GetRepoQuickstarts(w, httptest.NewRequest(http.MethodGet, "/repo-quickstarts", nil))
	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Data generated.ListRepoQuickstartsResponse `json:"data"`
	}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	require.NotNil(t, resp.Data.HeadSha)
	assert.Equal(t, "abc", *resp.Data.HeadSha)
}

func TestPostPullRequestBatch_ForwardsCreate(t *testing.T) {
	adapter := gitServiceAdapter(t, func(w http.ResponseWriter, r *http.Request) {
		var got clients.GitServiceBatchRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		require.Len(t, got.Resources, 2)
		assert.False(t, got.Resources[0].Create, "an edit stays an edit")
		assert.True(t, got.Resources[1].Create)

		json.NewEncoder(w).Encode(map[string]interface{}{"prNumber": 42, "status": "updated"})
	})
	body := `{
		"metadata": {"prNumber": 42, "expectedHeadSha": "abc", "commitMessage": "docs: add new"},
		"resources": [
			{"name": "alpha", "files": [{"name": "alpha.yml", "content": "a: 1\n"}]},
			{"name": "fresh", "create": true, "files": [{"name": "metadata.yml", "content": "name: fresh\n"}]}
		]
	}`

	w := httptest.NewRecorder()
	adapter.PostPullRequestBatch(w, httptest.NewRequest(http.MethodPost, "/pull-request/batch", bytes.NewBufferString(body)))
	assert.Equal(t, http.StatusOK, w.Code, w.Body.String())
}
