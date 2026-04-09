package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPostGitPullRequest_InvalidJSON(t *testing.T) {
	adapter := NewServerAdapter()

	req := httptest.NewRequest("POST", "/git/pull-request", strings.NewReader("not json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostGitPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid request body")
}

func TestPostGitPullRequest_MissingTitle(t *testing.T) {
	adapter := NewServerAdapter()

	body := `{"files": [{"path": "test.yaml", "content": "name: test"}]}`
	req := httptest.NewRequest("POST", "/git/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostGitPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "title is required")
}

func TestPostGitPullRequest_EmptyFiles(t *testing.T) {
	adapter := NewServerAdapter()

	body := `{"title": "Test PR", "files": []}`
	req := httptest.NewRequest("POST", "/git/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostGitPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "at least one file is required")
}

func TestPostGitPullRequest_InvalidFilePath(t *testing.T) {
	adapter := NewServerAdapter()

	body := `{"title": "Test PR", "files": [{"path": "../etc/passwd", "content": "content"}]}`
	req := httptest.NewRequest("POST", "/git/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostGitPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "path traversal not allowed")
}

func TestPostGitPullRequest_InvalidFileExtension(t *testing.T) {
	adapter := NewServerAdapter()

	body := `{"title": "Test PR", "files": [{"path": "test.exe", "content": "content"}]}`
	req := httptest.NewRequest("POST", "/git/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostGitPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "unsupported file extension")
}

func TestPostGitPullRequest_MissingGitConfig(t *testing.T) {
	adapter := NewServerAdapter()

	body := `{"title": "Test PR", "files": [{"path": "quickstarts/test.yaml", "content": "name: test"}]}`
	req := httptest.NewRequest("POST", "/git/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostGitPullRequest(w, req)

	// Should fail with 500 because GIT_REPO_URL is not configured
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	assert.Contains(t, w.Body.String(), "GIT_REPO_URL is not configured")
}

func TestPostGitPullRequest_EmptyBody(t *testing.T) {
	adapter := NewServerAdapter()

	req := httptest.NewRequest("POST", "/git/pull-request", strings.NewReader(""))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostGitPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPostGitPullRequest_EmptyFileContent(t *testing.T) {
	adapter := NewServerAdapter()

	body := `{"title": "Test PR", "files": [{"path": "test.yaml", "content": "  "}]}`
	req := httptest.NewRequest("POST", "/git/pull-request", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	adapter.PostGitPullRequest(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "file content cannot be empty")
}
