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

func TestListCreatorPRs_FiltersByBranchAcrossPages(t *testing.T) {
	created := openCreatorPRJSON(11, "qs-create-demo-1", "demo")
	created["title"] = "A title edited during review"
	unrelated := openCreatorPRJSON(12, "feature-demo", "demo")
	unrelated["labels"] = []map[string]string{{"name": "quickstarts-creator"}}
	closed := openCreatorPRJSON(13, "qs-create-closed-1", "closed")
	closed["state"] = "closed"
	missingSHA := openCreatorPRJSON(14, "qs-create-missing-sha-1", "missing-sha")
	missingSHA["head"].(map[string]interface{})["sha"] = ""
	missingHead := openCreatorPRJSON(15, "", "missing-head")
	missingHead["head"] = nil

	mux := http.NewServeMux()
	requests := 0
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		requests++
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "open", r.URL.Query().Get("state"))
		assert.Equal(t, "100", r.URL.Query().Get("per_page"))
		assert.Empty(t, r.URL.Query().Get("labels"))
		switch r.URL.Query().Get("page") {
		case "":
			w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/test-owner/test-repo/pulls?page=2>; rel="next"`, r.Host))
			json.NewEncoder(w).Encode([]map[string]interface{}{unrelated, closed, missingSHA, missingHead, created})
		case "2":
			json.NewEncoder(w).Encode([]map[string]interface{}{openCreatorPRJSON(16, "qs-update-existing-2", "existing")})
		default:
			t.Errorf("unexpected page: %s", r.URL.Query().Get("page"))
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	client := newTestClient(t, mux)
	prs, err := client.ListCreatorPRs(context.Background())
	require.NoError(t, err)
	require.Len(t, prs, 2)
	assert.Equal(t, 2, requests, "listing should not fetch each PR separately or call the Issues API")
	assert.Equal(t, 11, prs[0].Number)
	assert.Equal(t, "A title edited during review", prs[0].Title)
	assert.Equal(t, "https://github.com/test-owner/test-repo/pull/11", prs[0].HTMLURL)
	assert.Equal(t, "2026-09-29T12:00:00Z", prs[0].UpdatedAt.Format("2006-01-02T15:04:05Z"))
	assert.Equal(t, "qs-create-demo-1", prs[0].BranchName)
	assert.Equal(t, "demo", prs[0].Slug)
	assert.Equal(t, "deadbeef", prs[0].HeadSHA)
	assert.Equal(t, "fork-user", prs[0].HeadOwner)
	assert.Equal(t, "test-repo", prs[0].HeadRepo)
	assert.Equal(t, 16, prs[1].Number)
	assert.Equal(t, "qs-update-existing-2", prs[1].BranchName)
	assert.Equal(t, "existing", prs[1].Slug)
}

func TestListCreatorPRs_NoMatchesReturnsEmptySlice(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]map[string]interface{}{openCreatorPRJSON(11, "feature-demo", "demo")})
	})

	client := newTestClient(t, mux)
	prs, err := client.ListCreatorPRs(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, prs)
	assert.Empty(t, prs)
}

func TestListCreatorPRs_PageError(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("page") == "2" {
			w.WriteHeader(http.StatusForbidden)
			json.NewEncoder(w).Encode(map[string]string{"message": "Resource not accessible by personal access token"})
			return
		}
		w.Header().Set("Link", fmt.Sprintf(`<http://%s/repos/test-owner/test-repo/pulls?page=2>; rel="next"`, r.Host))
		json.NewEncoder(w).Encode([]map[string]interface{}{openCreatorPRJSON(11, "qs-create-demo-1", "demo")})
	})

	client := newTestClient(t, mux)
	prs, err := client.ListCreatorPRs(context.Background())
	require.ErrorContains(t, err, "failed to list pull requests")
	assert.True(t, isStatus(err, http.StatusForbidden))
	assert.Nil(t, prs, "a failed page must not return a partial list")
}

func TestGetCreatorPR_UnlabeledCreatorBranches(t *testing.T) {
	tests := []struct {
		ref    string
		branch string
	}{
		{ref: "qs-create-demo-1", branch: "qs-create-demo-1"},
		{ref: "qs-update-demo-1", branch: "qs-update-demo-1"},
		{ref: "refs/heads/qs-create-demo-1", branch: "qs-create-demo-1"},
		{ref: "fork-user:qs-update-demo-1", branch: "qs-update-demo-1"},
	}
	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/test-owner/test-repo/pulls/11", func(w http.ResponseWriter, r *http.Request) {
				pr := openCreatorPRJSON(11, tt.ref, "demo")
				pr["title"] = "Renamed by a reviewer"
				json.NewEncoder(w).Encode(pr)
			})

			client := newTestClient(t, mux)
			pr, err := client.GetCreatorPR(context.Background(), 11)
			require.NoError(t, err)
			assert.Equal(t, tt.branch, pr.BranchName)
			assert.Equal(t, "Renamed by a reviewer", pr.Title)
			assert.Equal(t, "demo", pr.Slug)
			assert.Equal(t, "deadbeef", pr.HeadSHA)
			assert.Equal(t, "fork-user", pr.HeadOwner)
			assert.Equal(t, "test-repo", pr.HeadRepo)
		})
	}
}

func TestGetCreatorPR_NonCreatorOrClosedIsNotFound(t *testing.T) {
	tests := []struct {
		name   string
		branch string
		state  string
	}{
		{name: "unrelated labeled PR", branch: "feature-demo", state: "open"},
		{name: "prefix in middle", branch: "feature/qs-create-demo-1", state: "open"},
		{name: "lookalike prefix", branch: "qs-creates-demo-1", state: "open"},
		{name: "closed creator PR", branch: "qs-create-demo-1", state: "closed"},
		{name: "closed update PR", branch: "qs-update-demo-1", state: "closed"},
		{name: "missing head", state: "open"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/test-owner/test-repo/pulls/11", func(w http.ResponseWriter, r *http.Request) {
				pr := openCreatorPRJSON(11, tt.branch, "demo")
				pr["state"] = tt.state
				pr["labels"] = []map[string]string{{"name": "quickstarts-creator"}}
				if tt.branch == "" {
					pr["head"] = nil
				}
				json.NewEncoder(w).Encode(pr)
			})

			client := newTestClient(t, mux)
			pr, err := client.GetCreatorPR(context.Background(), 11)
			assert.ErrorIs(t, err, ErrNotFound)
			assert.Nil(t, pr)
		})
	}
}

func TestGetCreatorPR_MissingSHA(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/test-owner/test-repo/pulls/11", func(w http.ResponseWriter, r *http.Request) {
		pr := openCreatorPRJSON(11, "qs-create-demo-1", "demo")
		pr["head"].(map[string]interface{})["sha"] = ""
		json.NewEncoder(w).Encode(pr)
	})

	client := newTestClient(t, mux)
	_, err := client.GetCreatorPR(context.Background(), 11)
	assert.EqualError(t, err, "pull request 11 is missing head SHA")
}

func TestGetCreatorPR_APIError(t *testing.T) {
	for _, status := range []int{http.StatusNotFound, http.StatusForbidden} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/test-owner/test-repo/pulls/11", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				json.NewEncoder(w).Encode(map[string]string{"message": http.StatusText(status)})
			})

			client := newTestClient(t, mux)
			_, err := client.GetCreatorPR(context.Background(), 11)
			if status == http.StatusNotFound {
				assert.ErrorIs(t, err, ErrNotFound)
			} else {
				require.ErrorContains(t, err, "failed to get pull request 11")
				assert.True(t, isStatus(err, status))
			}
		})
	}
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
	mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/demo", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
		writeDirectory(w, "docs/quickstarts/demo", "metadata.yml", "demo.yml")
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

func TestGetPRQuickstartFiles_IncludesUnchangedHeadFiles(t *testing.T) {
	tests := []struct {
		name    string
		changed string
		slug    string
	}{
		{name: "metadata-only update", changed: "metadata.yaml", slug: "demo"},
		{name: "content-only update", changed: "demo.yaml", slug: "demo"},
		{name: "slug inferred from diff", changed: "demo.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/test-owner/test-repo/pulls/11/files", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode([]map[string]string{
					{"filename": "docs/quickstarts/demo/" + tt.changed, "status": "modified"},
					{"filename": "docs/quickstarts/demo/old-name.yaml", "status": "removed"},
				})
			})
			mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/demo", func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
				json.NewEncoder(w).Encode([]map[string]string{
					{"type": "file", "path": "docs/quickstarts/demo/metadata.yaml"},
					{"type": "file", "path": "docs/quickstarts/demo/demo.yaml"},
					{"type": "dir", "path": "docs/quickstarts/demo/assets"},
					{"type": "symlink", "path": "docs/quickstarts/demo/link.yaml"},
					{"type": "file", "path": "docs/quickstarts/other/other.yaml"},
					{"type": "file", "path": "docs/quickstarts/demo/assets/nested.yaml"},
					{"type": "file", "path": "docs/quickstarts/demo/../other.yaml"},
				})
			})
			mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/demo/metadata.yaml", func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
				writeContent(w, "kind: QuickStarts\nname: demo\ntags: []\n")
			})
			mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/demo/demo.yaml", func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
				writeContent(w, "spec:\n  displayName: Demo\n")
			})

			client := newTestClient(t, mux)
			pr := &CreatorPR{Number: 11, Slug: tt.slug, HeadSHA: "deadbeef", HeadOwner: "fork-user", HeadRepo: "test-repo"}
			slug, files, err := client.GetPRQuickstartFiles(context.Background(), pr)
			require.NoError(t, err)
			assert.Equal(t, "demo", slug)
			assert.ElementsMatch(t, []File{
				{Name: "metadata.yaml", Content: "kind: QuickStarts\nname: demo\ntags: []\n"},
				{Name: "demo.yaml", Content: "spec:\n  displayName: Demo\n"},
			}, files, "resume needs both head files even when only one differs from main")
		})
	}
}

func TestGetPRQuickstartFiles_DirectoryErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   interface{}
		want   string
	}{
		{name: "missing directory", status: http.StatusNotFound, body: map[string]string{"message": "Not Found"}, want: "failed to list contents of docs/quickstarts/demo"},
		{name: "permission denied", status: http.StatusForbidden, body: map[string]string{"message": "Forbidden"}, want: "failed to list contents of docs/quickstarts/demo"},
		{name: "not a directory", status: http.StatusOK, body: map[string]string{"type": "file"}, want: "not a directory"},
		{name: "empty directory", status: http.StatusOK, body: []map[string]string{}, want: "no quickstart files found in directory docs/quickstarts/demo"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := http.NewServeMux()
			mux.HandleFunc("/repos/test-owner/test-repo/pulls/11/files", func(w http.ResponseWriter, r *http.Request) {
				json.NewEncoder(w).Encode([]map[string]string{
					{"filename": "docs/quickstarts/demo/demo.yaml", "status": "modified"},
				})
			})
			mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/demo", func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				json.NewEncoder(w).Encode(tt.body)
			})

			client := newTestClient(t, mux)
			pr := &CreatorPR{Number: 11, Slug: "demo", HeadSHA: "deadbeef", HeadOwner: "fork-user", HeadRepo: "test-repo"}
			_, files, err := client.GetPRQuickstartFiles(context.Background(), pr)
			require.ErrorContains(t, err, tt.want)
			assert.Nil(t, files)
		})
	}
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
	mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/new-name", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
		writeDirectory(w, "docs/quickstarts/new-name", "metadata.yaml", "new-name.yaml")
	})
	mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/new-name/metadata.yaml", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
		writeContent(w, "kind: QuickStarts\nname: new-name\n")
	})
	mux.HandleFunc("/repos/fork-user/test-repo/contents/docs/quickstarts/new-name/new-name.yaml", func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "deadbeef", r.URL.Query().Get("ref"))
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
		"labels":     []map[string]string{},
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

func writeDirectory(w http.ResponseWriter, dir string, names ...string) {
	entries := make([]map[string]string, 0, len(names))
	for _, name := range names {
		entries = append(entries, map[string]string{
			"type": "file",
			"name": name,
			"path": dir + "/" + name,
		})
	}
	json.NewEncoder(w).Encode(entries)
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
