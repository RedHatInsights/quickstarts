package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
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

func TestAddLabels_Success(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/issues/42/labels", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{{"name": CreatorPRLabel}})
	})

	client := newTestClient(t, mux)
	err := client.AddLabels(context.Background(), 42, []string{CreatorPRLabel})
	assert.NoError(t, err)
}

func TestAddLabels_CreatesMissingLabelThenRetries(t *testing.T) {
	mux := http.NewServeMux()
	addCalls := 0
	mux.HandleFunc("/repos/test-owner/test-repo/issues/7/labels", func(w http.ResponseWriter, r *http.Request) {
		addCalls++
		if addCalls == 1 {
			w.WriteHeader(http.StatusUnprocessableEntity)
			json.NewEncoder(w).Encode(map[string]string{"message": "Label does not exist"})
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{{"name": CreatorPRLabel}})
	})
	mux.HandleFunc("/repos/test-owner/test-repo/labels", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"name": CreatorPRLabel})
	})

	client := newTestClient(t, mux)
	err := client.AddLabels(context.Background(), 7, []string{CreatorPRLabel})
	assert.NoError(t, err)
	assert.Equal(t, 2, addCalls)
}

func TestListCreatorPRs_FiltersToLabeledOpenPRs(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/issues", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		assert.Equal(t, CreatorPRLabel, r.URL.Query().Get("labels"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{
				"number": 11,
				"title":  "feat(quickstarts): create demo",
				"pull_request": map[string]string{
					"url": "https://api.github.com/repos/test-owner/test-repo/pulls/11",
				},
			},
		})
	})
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/11", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(openCreatorPRJSON(11, "qs-create-demo-1", "demo"))
	})

	client := newTestClient(t, mux)
	prs, err := client.ListCreatorPRs(context.Background())
	require.NoError(t, err)
	require.Len(t, prs, 1)
	assert.Equal(t, 11, prs[0].Number)
	assert.Equal(t, "qs-create-demo-1", prs[0].BranchName)
	assert.Equal(t, "demo", prs[0].Slug)
	assert.Equal(t, "fork-user", prs[0].HeadOwner)
}

func TestGetCreatorPR_UnlabeledIsNotFound(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/5", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"number":   5,
			"state":    "open",
			"title":    "docs: something",
			"html_url": "https://github.com/test-owner/test-repo/pull/5",
			"body":     "no label",
			"labels":   []interface{}{},
			"head": map[string]interface{}{
				"ref": "branch",
				"sha": "abc",
				"repo": map[string]interface{}{
					"name":  "test-repo",
					"owner": map[string]string{"login": "test-owner"},
				},
			},
		})
	})

	client := newTestClient(t, mux)
	_, err := client.GetCreatorPR(context.Background(), 5)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestGetPRQuickstartFiles_FromHeadRepo(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/11/files", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"filename": "docs/quickstarts/demo/metadata.yml", "status": "added"},
			{"filename": "docs/quickstarts/demo/demo.yml", "status": "added"},
			{"filename": "README.md", "status": "modified"},
		})
	})
	mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/demo/metadata.yml", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
		writeContent(w, "kind: QuickStarts\n")
	})
	mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/demo/demo.yml", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
		writeContent(w, "spec:\n  displayName: Demo\n")
	})

	client := newTestClient(t, mux)
	pr := &CreatorPR{
		Number:    11,
		Slug:      "demo",
		HeadSHA:   "deadbeef",
		HeadOwner: "fork-user",
		HeadRepo:  "test-repo",
	}
	slug, files, err := client.GetPRQuickstartFiles(context.Background(), pr)
	require.NoError(t, err)
	assert.Equal(t, "demo", slug)
	require.Len(t, files, 2)
	assert.Equal(t, "metadata.yml", files[0].Name)
	assert.Contains(t, files[0].Content, "QuickStarts")
	assert.Equal(t, "demo.yml", files[1].Name)
}

func TestGetPRQuickstartFiles_StaleSlugFallsBackToBranchDirectory(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/12/files", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// A rename moved the quickstart, so the directory named in the PR body
		// only survives as removed entries.
		json.NewEncoder(w).Encode([]map[string]string{
			{"filename": "docs/quickstarts/old-name/old-name.yaml", "status": "removed"},
			{"filename": "docs/quickstarts/old-name/metadata.yaml", "status": "removed"},
			{"filename": "docs/quickstarts/new-name/metadata.yaml", "status": "added"},
			{"filename": "docs/quickstarts/new-name/new-name.yaml", "status": "added"},
		})
	})
	mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/new-name/metadata.yaml", func(w http.ResponseWriter, r *http.Request) {
		writeContent(w, "kind: QuickStarts\nname: new-name\n")
	})
	mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/new-name/new-name.yaml", func(w http.ResponseWriter, r *http.Request) {
		writeContent(w, "spec:\n  displayName: New\n")
	})

	client := newTestClient(t, mux)
	pr := &CreatorPR{
		Number:    12,
		Slug:      "old-name", // what the un-rewritten PR body still says
		HeadSHA:   "deadbeef",
		HeadOwner: "fork-user",
		HeadRepo:  "test-repo",
	}
	slug, files, err := client.GetPRQuickstartFiles(context.Background(), pr)
	require.NoError(t, err, "a renamed directory must not make the PR unresumable")
	assert.Equal(t, "new-name", slug, "the slug should follow the directory on the branch")
	require.Len(t, files, 2)
	assert.Equal(t, "metadata.yaml", files[0].Name)
	assert.Equal(t, "new-name.yaml", files[1].Name)
}

func TestGetPRQuickstartFiles_NoQuickstartFilesStillErrors(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/13/files", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]string{
			{"filename": "README.md", "status": "modified"},
		})
	})

	client := newTestClient(t, mux)
	pr := &CreatorPR{Number: 13, Slug: "demo", HeadSHA: "deadbeef", HeadOwner: "fork-user", HeadRepo: "test-repo"}
	_, _, err := client.GetPRQuickstartFiles(context.Background(), pr)
	assert.Error(t, err, "the fallback must not mask a PR that touches no quickstart")
}

func TestFindPRURLByBranch_UsesForkOwner(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "nacho-bot:qs-create-demo-1", r.URL.Query().Get("head"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]map[string]interface{}{
			{"html_url": "https://github.com/test-owner/test-repo/pull/11", "number": 11},
		})
	})

	client := newTestClient(t, mux)
	client.ForkOwner = "nacho-bot"
	url, err := client.FindPRURLByBranch(context.Background(), "qs-create-demo-1")
	require.NoError(t, err)
	assert.Equal(t, "https://github.com/test-owner/test-repo/pull/11", url)
}

func openCreatorPRJSON(number int, branch, slug string) map[string]interface{} {
	return map[string]interface{}{
		"number":     number,
		"state":      "open",
		"title":      "feat(quickstarts): create " + slug,
		"html_url":   fmt.Sprintf("https://github.com/test-owner/test-repo/pull/%d", number),
		"body":       "Adding new quickstart via the Quickstarts Creator tool.\n\nDirectory: docs/quickstarts/" + slug + "/",
		"updated_at": "2026-09-29T12:00:00Z",
		"labels":     []map[string]string{{"name": CreatorPRLabel}},
		"head": map[string]interface{}{
			"ref": branch,
			"sha": "deadbeef",
			"repo": map[string]interface{}{
				"name":  "test-repo",
				"owner": map[string]string{"login": "fork-user"},
			},
		},
	}
}

func writeContent(w http.ResponseWriter, content string) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"type":     "file",
		"encoding": "base64",
		"content":  stdb64(content),
	})
}

func stdb64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
