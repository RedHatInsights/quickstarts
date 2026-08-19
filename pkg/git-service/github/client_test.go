package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	gogithub "github.com/google/go-github/v66/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	ghClient := gogithub.NewClient(nil)
	baseURL, _ := url.Parse(server.URL + "/")
	ghClient.BaseURL = baseURL

	return &Client{
		gh:    ghClient,
		Owner: "test-owner",
		Repo:  "test-repo",
		Token: "test-token",
	}
}

func TestParseRepoURL_HTTPS(t *testing.T) {
	owner, repo, err := ParseRepoURL("https://github.com/RedHatInsights/quickstarts")
	require.NoError(t, err)
	assert.Equal(t, "RedHatInsights", owner)
	assert.Equal(t, "quickstarts", repo)
}

func TestParseRepoURL_WithGitSuffix(t *testing.T) {
	owner, repo, err := ParseRepoURL("https://github.com/RedHatInsights/quickstarts.git")
	require.NoError(t, err)
	assert.Equal(t, "RedHatInsights", owner)
	assert.Equal(t, "quickstarts", repo)
}

func TestParseRepoURL_SSH(t *testing.T) {
	owner, repo, err := ParseRepoURL("git@github.com:RedHatInsights/quickstarts.git")
	require.NoError(t, err)
	assert.Equal(t, "RedHatInsights", owner)
	assert.Equal(t, "quickstarts", repo)
}

func TestParseRepoURL_Invalid(t *testing.T) {
	_, _, err := ParseRepoURL("https://gitlab.com/some/repo")
	assert.Error(t, err)
}

func TestParseRepoURL_MissingRepo(t *testing.T) {
	_, _, err := ParseRepoURL("https://github.com/onlyowner")
	assert.Error(t, err)
}

func TestNewClient_Success(t *testing.T) {
	client, err := NewClient("my-token", "https://github.com/acme/widgets", "")
	require.NoError(t, err)
	assert.Equal(t, "acme", client.Owner)
	assert.Equal(t, "widgets", client.Repo)
	assert.Equal(t, "my-token", client.Token)
	assert.Empty(t, client.ForkOwner)
}

func TestNewClient_WithForkOwner(t *testing.T) {
	client, err := NewClient("my-token", "https://github.com/acme/widgets", "fork-user")
	require.NoError(t, err)
	assert.Equal(t, "fork-user", client.ForkOwner)
}

func TestCreatePullRequest_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "Test PR", req["title"])
		assert.Equal(t, "feature-branch", req["head"])
		assert.Equal(t, "main", req["base"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"html_url": "https://github.com/test-owner/test-repo/pull/42",
			"number":   42,
		})
	})

	client := newTestClient(t, mux)
	prURL, prNum, err := client.CreatePullRequest(context.Background(), "Test PR", "PR body", "feature-branch", "main")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/test-owner/test-repo/pull/42", prURL)
	assert.Equal(t, 42, prNum)
}

func TestCreatePullRequest_Error(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Validation Failed",
		})
	})

	client := newTestClient(t, mux)
	_, _, err := client.CreatePullRequest(context.Background(), "Test PR", "body", "branch", "main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create PR")
}

func TestAssignReviewers_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/42/requested_reviewers", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	})

	client := newTestClient(t, mux)
	err := client.AssignReviewers(context.Background(), 42, "my-team")
	assert.NoError(t, err)
}

func TestAssignReviewers_EmptyTeam(t *testing.T) {
	client := &Client{}
	err := client.AssignReviewers(context.Background(), 42, "")
	assert.NoError(t, err)
}

func TestCreatePullRequest_CrossRepo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		var req map[string]string
		json.NewDecoder(r.Body).Decode(&req)
		assert.Equal(t, "nacho-bot:feature-branch", req["head"])

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"html_url": "https://github.com/test-owner/test-repo/pull/99",
			"number":   99,
		})
	})

	client := newTestClient(t, mux)
	client.ForkOwner = "nacho-bot"
	prURL, prNum, err := client.CreatePullRequest(context.Background(), "Cross-repo PR", "body", "feature-branch", "main")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/test-owner/test-repo/pull/99", prURL)
	assert.Equal(t, 99, prNum)
}

func TestParseRepoURL_RejectsLookalikeHost(t *testing.T) {
	_, _, err := ParseRepoURL("https://github.com.evil.com/owner/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported host")
}

func TestParseRepoURL_RejectsHTTP(t *testing.T) {
	_, _, err := ParseRepoURL("http://github.com/owner/repo")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported scheme")
}
