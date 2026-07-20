package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testFiles() []GitServiceFile {
	return []GitServiceFile{
		{Name: "metadata.yaml", Content: "name: test"},
	}
}

func testMetadata() GitServiceMetadata {
	return GitServiceMetadata{
		BranchName:    "test-branch",
		CommitMessage: "test commit",
		PRTitle:       "Test PR",
		PRBody:        "Test body",
		UserEmail:     "test@test.com",
	}
}

func TestSubmitPR_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/submit-pr", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req gitServiceRequest
		json.NewDecoder(r.Body).Decode(&req)
		assert.Len(t, req.Files, 1)
		assert.Equal(t, "test-branch", req.Metadata.BranchName)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GitServiceResponse{
			PRURL:      "https://github.com/owner/repo/pull/1",
			BranchName: "test-branch",
			CommitSHA:  "abc123",
			Status:     "created",
		})
	}))
	defer server.Close()

	client := NewGitService(server.URL, "")
	resp, err := client.SubmitPR(context.Background(), testFiles(), testMetadata())
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/owner/repo/pull/1", resp.PRURL)
	assert.Equal(t, "test-branch", resp.BranchName)
	assert.Equal(t, "abc123", resp.CommitSHA)
	assert.Equal(t, "created", resp.Status)
}

func TestSubmitPR_PSKHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "my-secret-token", r.Header.Get("X-PSK-Token"))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GitServiceResponse{Status: "created"})
	}))
	defer server.Close()

	client := NewGitService(server.URL, "my-secret-token")
	_, err := client.SubmitPR(context.Background(), testFiles(), testMetadata())
	require.NoError(t, err)
}

func TestSubmitPR_ErrorResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(gitServiceError{
			Status: "error",
			Msg:    "clone failed",
		})
	}))
	defer server.Close()

	client := NewGitService(server.URL, "")
	_, err := client.SubmitPR(context.Background(), testFiles(), testMetadata())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "clone failed")
	assert.Contains(t, err.Error(), "500")
}

func TestSubmitPR_Unreachable(t *testing.T) {
	client := NewGitService("http://localhost:1", "")
	_, err := client.SubmitPR(context.Background(), testFiles(), testMetadata())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git-service request failed")
}

func TestListQuickstarts_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/list-quickstarts", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GitServiceListQuickstartsResponse{
			Quickstarts: []GitServiceQuickstartEntry{
				{Name: "getting-started", DisplayName: "Getting Started"},
				{Name: "cost-mgmt", DisplayName: "Cost Management"},
			},
		})
	}))
	defer server.Close()

	client := NewGitService(server.URL, "token")
	resp, err := client.ListQuickstarts(context.Background())
	require.NoError(t, err)
	assert.Len(t, resp.Quickstarts, 2)
	assert.Equal(t, "getting-started", resp.Quickstarts[0].Name)
	assert.Equal(t, "Getting Started", resp.Quickstarts[0].DisplayName)
}

func TestListQuickstarts_PSKHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "my-token", r.Header.Get("X-PSK-Token"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GitServiceListQuickstartsResponse{})
	}))
	defer server.Close()

	client := NewGitService(server.URL, "my-token")
	_, err := client.ListQuickstarts(context.Background())
	require.NoError(t, err)
}

func TestListQuickstarts_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(gitServiceError{Status: "error", Msg: "pull failed"})
	}))
	defer server.Close()

	client := NewGitService(server.URL, "")
	_, err := client.ListQuickstarts(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pull failed")
}

func TestGetQuickstartContent_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/quickstart-content/my-qs", r.URL.Path)
		assert.Equal(t, http.MethodGet, r.Method)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(GitServiceQuickstartContentResponse{
			Name: "my-qs",
			Files: []GitServiceFile{
				{Name: "metadata.yaml", Content: "kind: QuickStarts"},
				{Name: "my-qs.yml", Content: "spec:\n  displayName: My QS"},
			},
		})
	}))
	defer server.Close()

	client := NewGitService(server.URL, "")
	resp, err := client.GetQuickstartContent(context.Background(), "my-qs")
	require.NoError(t, err)
	assert.Equal(t, "my-qs", resp.Name)
	assert.Len(t, resp.Files, 2)
}

func TestGetQuickstartContent_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(gitServiceError{Status: "error", Msg: "quickstart not found"})
	}))
	defer server.Close()

	client := NewGitService(server.URL, "")
	_, err := client.GetQuickstartContent(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "quickstart not found")
}
