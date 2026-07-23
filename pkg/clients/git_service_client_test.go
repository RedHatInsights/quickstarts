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
