package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/clients"
	"github.com/stretchr/testify/assert"
)

func TestPostPullRequest_Success(t *testing.T) {
	mockGitService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/submit-pr", r.URL.Path)
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var reqBody map[string]interface{}
		json.NewDecoder(r.Body).Decode(&reqBody)
		files := reqBody["files"].([]interface{})
		assert.Len(t, files, 1)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"prUrl":      "https://github.com/org/repo/pull/42",
			"branchName": "quickstart/test-123",
			"commitSha":  "abc123def456",
			"status":     "created",
		})
	}))
	defer mockGitService.Close()

	adapter := NewServerAdapter()
	adapter.gitServiceClient = clients.NewGitService(mockGitService.URL, "")
	adapter.gitServiceEnabled = true

	body := `{
		"files": [{"name": "metadata.yaml", "content": "test content"}],
		"metadata": {
			"branchName": "quickstart/test-123",
			"commitMessage": "Add test quickstart",
			"prTitle": "Create test quickstart"
		}
	}`

	req := httptest.NewRequest("POST", "/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostPullRequest(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	data, ok := resp["data"].(map[string]interface{})
	assert.True(t, ok, "response should have data field")
	assert.Equal(t, "https://github.com/org/repo/pull/42", data["prUrl"])
	assert.Equal(t, "quickstart/test-123", data["branchName"])
	assert.Equal(t, "abc123def456", data["commitSha"])
	assert.Equal(t, "created", data["status"])
}

func TestPostPullRequest_InvalidJSON(t *testing.T) {
	adapter := NewServerAdapter()
	adapter.gitServiceEnabled = true

	req := httptest.NewRequest("POST", "/pull-request", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestPostPullRequest_EmptyFiles(t *testing.T) {
	adapter := NewServerAdapter()
	adapter.gitServiceEnabled = true

	body := `{
		"files": [],
		"metadata": {
			"branchName": "test",
			"commitMessage": "msg",
			"prTitle": "title"
		}
	}`

	req := httptest.NewRequest("POST", "/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "files are required")
}

func TestPostPullRequest_MissingMetadataFields(t *testing.T) {
	adapter := NewServerAdapter()
	adapter.gitServiceEnabled = true

	body := `{
		"files": [{"name": "test.yaml", "content": "content"}],
		"metadata": {
			"branchName": "test"
		}
	}`

	req := httptest.NewRequest("POST", "/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "branchName, commitMessage, and prTitle are required")
}

func TestPostPullRequest_GitServiceError(t *testing.T) {
	mockGitService := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "error",
			"msg":    "failed to push branch",
		})
	}))
	defer mockGitService.Close()

	adapter := NewServerAdapter()
	adapter.gitServiceClient = clients.NewGitService(mockGitService.URL, "")
	adapter.gitServiceEnabled = true

	body := `{
		"files": [{"name": "test.yaml", "content": "content"}],
		"metadata": {
			"branchName": "test",
			"commitMessage": "msg",
			"prTitle": "title"
		}
	}`

	req := httptest.NewRequest("POST", "/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostPullRequest(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
	assert.Contains(t, w.Body.String(), "failed to push branch")
}

func TestPostPullRequest_GitServiceUnreachable(t *testing.T) {
	adapter := NewServerAdapter()
	adapter.gitServiceClient = clients.NewGitService("http://localhost:1", "")
	adapter.gitServiceEnabled = true

	body := `{
		"files": [{"name": "test.yaml", "content": "content"}],
		"metadata": {
			"branchName": "test",
			"commitMessage": "msg",
			"prTitle": "title"
		}
	}`

	req := httptest.NewRequest("POST", "/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostPullRequest(w, req)

	assert.Equal(t, http.StatusBadGateway, w.Code)
}
