package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	gitops "github.com/RedHatInsights/quickstarts/pkg/git-service/git"
	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	headSHA = "1111111111111111111111111111111111111111"
	baseSHA = "2222222222222222222222222222222222222222"
)

func batchCreateBody() string {
	return `{
		"metadata": {
			"branchName": "qs-update-batch-123",
			"baseSha": "` + baseSHA + `",
			"commitMessage": "docs: update Inventory documentation URLs",
			"prTitle": "Update Inventory documentation URLs",
			"userEmail": "user@example.com"
		},
		"resources": [
			{"name": "alpha", "files": [{"name": "alpha.yml", "content": "spec:\n  displayName: Alpha\n"}]},
			{"name": "beta", "files": [{"name": "metadata.yaml", "content": "name: beta\n"}]}
		]
	}`
}

func batchUpdateBody() string {
	return `{
		"metadata": {
			"prNumber": 42,
			"expectedHeadSha": "` + headSHA + `",
			"commitMessage": "docs: second pass"
		},
		"resources": [
			{"name": "alpha", "files": [{"name": "alpha.yml", "content": "spec: {}\n"}]}
		]
	}`
}

func batchRepo() *mockRepoManager {
	return &mockRepoManager{
		baseBranch: "main",
		commitSHA:  "3333333333333333333333333333333333333333",
		headSHA:    headSHA,
		files:      []string{"metadata.yaml", "metadata.yml", "alpha.yml"},
	}
}

func postBatch(h *Handler, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-batch-pr", bytes.NewBufferString(body))
	rr := httptest.NewRecorder()
	h.SubmitBatchPR(rr, req)
	return rr
}

func TestSubmitBatchPR_CreatesOnePRAcrossDirectories(t *testing.T) {
	repo := batchRepo()
	gh := &mockGitHubClient{createPRURL: "https://github.com/o/r/pull/7", createPRNumber: 7}
	h := NewHandler(repo, gh, "reviewers", "/docs/quickstarts/")

	rr := postBatch(h, batchCreateBody())
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp BatchPRResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "created", resp.Status)
	assert.Equal(t, 7, resp.PRNumber)
	assert.Equal(t, "qs-update-batch-123", resp.BranchName)
	assert.Equal(t, repo.commitSHA, resp.HeadSHA)
	assert.Equal(t, []string{"docs/quickstarts/alpha/alpha.yml", "docs/quickstarts/beta/metadata.yaml"}, resp.Paths)

	assert.Equal(t, baseSHA, repo.branchBase, "a new branch starts from the snapshot the files were read from")
	assert.Equal(t, resp.Paths, repo.committedPaths, "every directory goes into the one commit")
	assert.Equal(t, "docs: update Inventory documentation URLs", repo.commitMessage)
	assert.True(t, repo.forcePushed)
	assert.Equal(t, "qs-update-batch-123", repo.cleanedUp)
	assert.Nil(t, repo.removedFiles, "a batch never prunes files")

	assert.Equal(t, "Update Inventory documentation URLs", gh.createdTitle)
	assert.Contains(t, gh.createdBody, "- docs/quickstarts/alpha/")
	assert.Contains(t, gh.createdBody, "- docs/quickstarts/beta/")
	assert.NotContains(t, gh.createdBody, "Directory:", "the single-resource slug marker must not match a batch")
	assert.Contains(t, gh.createdBody, "Submitted by: user@example.com")
	assert.Equal(t, "reviewers", gh.assignedTeam)
}

func TestSubmitBatchPR_UpdateAppendsToExistingPR(t *testing.T) {
	repo := batchRepo()
	gh := &mockGitHubClient{getPR: &ghclient.CreatorPR{
		Number: 42, HTMLURL: "https://github.com/o/r/pull/42", BranchName: "qs-update-batch-1", HeadSHA: headSHA,
	}}
	h := NewHandler(repo, gh, "", "/docs/quickstarts/")

	rr := postBatch(h, batchUpdateBody())
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp BatchPRResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, "updated", resp.Status)
	assert.Equal(t, 42, resp.PRNumber)
	assert.Equal(t, "https://github.com/o/r/pull/42", resp.PRURL)
	assert.Equal(t, "qs-update-batch-1", resp.BranchName)
	assert.Equal(t, repo.commitSHA, resp.HeadSHA)
	assert.Equal(t, "qs-update-batch-1", repo.pushedBranch)
	assert.False(t, repo.forcePushed, "checkpoint commits must not rewrite the branch")
	assert.Empty(t, gh.createdTitle, "no second PR is opened")
}

func TestSubmitBatchPR_Conflicts(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*mockRepoManager, *mockGitHubClient)
		body  string
	}{
		{
			name: "PR head moved",
			setup: func(_ *mockRepoManager, gh *mockGitHubClient) {
				gh.getPR = &ghclient.CreatorPR{Number: 42, BranchName: "qs-update-batch-1", HeadSHA: "9999"}
			},
			body: batchUpdateBody(),
		},
		{
			name: "fetched branch moved after the PR lookup",
			setup: func(repo *mockRepoManager, gh *mockGitHubClient) {
				gh.getPR = &ghclient.CreatorPR{Number: 42, BranchName: "qs-update-batch-1", HeadSHA: headSHA}
				repo.headSHA = "9999"
			},
			body: batchUpdateBody(),
		},
		{
			name: "push raced another save",
			setup: func(repo *mockRepoManager, gh *mockGitHubClient) {
				gh.getPR = &ghclient.CreatorPR{Number: 42, BranchName: "qs-update-batch-1", HeadSHA: headSHA}
				repo.pushErr = gitops.ErrPushRejected
			},
			body: batchUpdateBody(),
		},
		{
			name: "first save already opened a PR",
			setup: func(_ *mockRepoManager, gh *mockGitHubClient) {
				gh.findPRNumber = 7
			},
			body: batchCreateBody(),
		},
		{
			name: "base commit unavailable",
			setup: func(repo *mockRepoManager, _ *mockGitHubClient) {
				repo.createBranchAtErr = gitops.ErrUnknownRevision
			},
			body: batchCreateBody(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := batchRepo()
			gh := &mockGitHubClient{}
			tt.setup(repo, gh)
			h := NewHandler(repo, gh, "", "/docs/quickstarts/")

			rr := postBatch(h, tt.body)
			assert.Equal(t, http.StatusConflict, rr.Code, rr.Body.String())
			assert.Empty(t, gh.createdTitle)
		})
	}
}

func TestSubmitBatchPR_ClosedPRIsNotReplaced(t *testing.T) {
	repo := batchRepo()
	gh := &mockGitHubClient{getPRErr: ghclient.ErrNotFound}
	h := NewHandler(repo, gh, "", "/docs/quickstarts/")

	rr := postBatch(h, batchUpdateBody())
	assert.Equal(t, http.StatusNotFound, rr.Code)
	assert.Empty(t, gh.createdTitle)
	assert.Empty(t, repo.pushedBranch)
}

func TestSubmitBatchPR_RejectsWritesOutsideExistingFiles(t *testing.T) {
	repo := batchRepo()
	repo.files = []string{"metadata.yaml"}
	h := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	rr := postBatch(h, batchCreateBody())
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "alpha.yml does not exist")
	assert.Nil(t, repo.committedPaths)
	assert.Empty(t, repo.pushedBranch)
	assert.Equal(t, "qs-update-batch-123", repo.cleanedUp, "a failed batch resets the whole worktree")
}

func TestSubmitBatchPR_MissingDirectory(t *testing.T) {
	repo := batchRepo()
	repo.listFilesErr = errors.New("no such directory")
	h := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	rr := postBatch(h, batchCreateBody())
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Contains(t, rr.Body.String(), "quickstart alpha does not exist")
}

func TestSubmitBatchPR_NoChanges(t *testing.T) {
	repo := batchRepo()
	repo.commitErr = gitops.ErrNoChanges
	h := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	rr := postBatch(h, batchCreateBody())
	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.Empty(t, repo.pushedBranch)
}

func TestSubmitBatchPR_PRCreationFailureIsRetryable(t *testing.T) {
	repo := batchRepo()
	gh := &mockGitHubClient{createPRErr: errors.New("boom")}
	h := NewHandler(repo, gh, "", "/docs/quickstarts/")

	rr := postBatch(h, batchCreateBody())
	assert.Equal(t, http.StatusBadGateway, rr.Code)
	assert.Contains(t, rr.Body.String(), "save again to retry")
	assert.Equal(t, "qs-update-batch-123", repo.pushedBranch)
}

func TestValidateBatchRequest(t *testing.T) {
	valid := func() BatchPRRequest {
		var req BatchPRRequest
		require.NoError(t, json.Unmarshal([]byte(batchCreateBody()), &req))
		return req
	}
	tests := []struct {
		name   string
		mutate func(*BatchPRRequest)
		want   string
	}{
		{"missing commit message", func(r *BatchPRRequest) { r.Metadata.CommitMessage = "" }, "commitMessage"},
		{"update without head", func(r *BatchPRRequest) { r.Metadata.PRNumber = 4 }, "expectedHeadSha"},
		{"create without title", func(r *BatchPRRequest) { r.Metadata.PRTitle = "" }, "prTitle"},
		{"unlisted branch prefix", func(r *BatchPRRequest) { r.Metadata.BranchName = "feature/x" }, "qs-update-"},
		{"branch traversal", func(r *BatchPRRequest) { r.Metadata.BranchName = "qs-update-../x" }, "valid branch"},
		{"no resources", func(r *BatchPRRequest) { r.Resources = nil }, "resources are required"},
		{"nested resource name", func(r *BatchPRRequest) { r.Resources[0].Name = "a/b" }, "invalid quickstart name"},
		{"traversal resource name", func(r *BatchPRRequest) { r.Resources[0].Name = ".." }, "invalid quickstart name"},
		{"duplicate resource", func(r *BatchPRRequest) { r.Resources[1].Name = "alpha" }, "more than once"},
		{"empty resource", func(r *BatchPRRequest) { r.Resources[0].Files = nil }, "has no files"},
		{"nested file", func(r *BatchPRRequest) { r.Resources[0].Files[0].Name = "x/alpha.yml" }, "invalid file name"},
		{"non-yaml file", func(r *BatchPRRequest) { r.Resources[0].Files[0].Name = "script.sh" }, "only YAML"},
		{"duplicate file", func(r *BatchPRRequest) {
			r.Resources[0].Files = append(r.Resources[0].Files, r.Resources[0].Files[0])
		}, "more than once"},
		{"invalid yaml reported per file", func(r *BatchPRRequest) {
			r.Resources[0].Files[0].Content = "a: [b"
			r.Resources[1].Files[0].Content = "c: : d"
		}, "alpha/alpha.yml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := valid()
			tt.mutate(&req)
			err := validateBatchRequest(&req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}

	req := valid()
	req.Resources[0].Files[0].Content = "a: [b"
	req.Resources[1].Files[0].Content = "c: : d"
	err := validateBatchRequest(&req)
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "beta/metadata.yaml"), "every invalid file is reported together")

	req = valid()
	assert.NoError(t, validateBatchRequest(&req))
}

func TestValidateBatchRequest_NewResource(t *testing.T) {
	complete := func() BatchResource {
		return BatchResource{Name: "fresh", Create: true, Files: []File{
			{Name: "metadata.yml", Content: "kind: QuickStarts\nname: fresh\n"},
			{Name: "fresh.yml", Content: "metadata:\n  name: fresh\nspec:\n  displayName: Fresh\n"},
		}}
	}
	tests := []struct {
		name   string
		mutate func(*BatchResource)
		want   string
	}{
		{"missing content file", func(r *BatchResource) { r.Files = r.Files[:1] }, "needs both"},
		{"extra file", func(r *BatchResource) {
			r.Files = append(r.Files, File{Name: "notes.yml", Content: "a: 1\n"})
		}, "may only contain"},
		{"content named for another resource", func(r *BatchResource) { r.Files[1].Name = "other.yml" }, "may only contain"},
		{"metadata names another resource", func(r *BatchResource) { r.Files[0].Content = "name: other\n" }, `name must be "fresh"`},
		{"content names another resource", func(r *BatchResource) {
			r.Files[1].Content = "metadata:\n  name: other\n"
		}, `metadata.name must be "fresh"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := complete()
			tt.mutate(&res)
			req := BatchPRRequest{Metadata: BatchPRMetadata{PRNumber: 1, ExpectedHeadSHA: headSHA, CommitMessage: "c"}, Resources: []BatchResource{res}}
			err := validateBatchRequest(&req)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}

	req := BatchPRRequest{Metadata: BatchPRMetadata{PRNumber: 1, ExpectedHeadSHA: headSHA, CommitMessage: "c"}, Resources: []BatchResource{complete()}}
	assert.NoError(t, validateBatchRequest(&req))
}

func TestSubmitBatchPR_InvalidBody(t *testing.T) {
	h := NewHandler(batchRepo(), &mockGitHubClient{}, "", "/docs/quickstarts/")
	rr := postBatch(h, "{")
	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func workspaceRequest(h http.HandlerFunc, pattern, target string) *httptest.ResponseRecorder {
	r := chi.NewRouter()
	r.Get(pattern, h)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, target, nil))
	return rr
}

func TestGetCreatorPRWorkspace(t *testing.T) {
	gh := &mockGitHubClient{
		getPR: &ghclient.CreatorPR{Number: 42, Title: "Update docs", HTMLURL: "u", BranchName: "qs-update-batch-1", HeadSHA: headSHA},
		workspace: &ghclient.Workspace{
			Resources: []ghclient.Resource{
				{Name: "alpha", Dir: "docs/quickstarts/alpha", Files: []ghclient.File{{Name: "metadata.yml", Content: "name: alpha\n"}}},
				{Name: "beta", Dir: "docs/quickstarts/beta", Files: []ghclient.File{{Name: "metadata.yml", Content: "name: beta\n"}}},
			},
			Missing: []string{"gamma"},
		},
	}
	h := NewHandler(batchRepo(), gh, "", "/docs/quickstarts/")

	rr := workspaceRequest(h.GetCreatorPRWorkspace, "/creator-prs/{number}/workspace", "/creator-prs/42/workspace")
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())

	var resp CreatorPRWorkspaceResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, headSHA, resp.HeadSHA)
	assert.Equal(t, "qs-update-batch-1", resp.BranchName)
	require.Len(t, resp.Resources, 2)
	assert.Equal(t, "docs/quickstarts/alpha", resp.Resources[0].Path)
	assert.Equal(t, "docs/quickstarts/beta/metadata.yml", resp.Resources[1].Files[0].Path, "same basename in two resources stays distinct")
	assert.Equal(t, []string{"gamma"}, resp.Missing)
}

func TestGetCreatorPRWorkspace_Errors(t *testing.T) {
	tests := []struct {
		name   string
		gh     *mockGitHubClient
		target string
		want   int
	}{
		{"bad number", &mockGitHubClient{}, "/creator-prs/abc/workspace", http.StatusBadRequest},
		{"not a creator PR", &mockGitHubClient{getPRErr: ghclient.ErrNotFound}, "/creator-prs/4/workspace", http.StatusNotFound},
		{"too many files", &mockGitHubClient{workspaceErr: ghclient.ErrIncompleteDiff}, "/creator-prs/4/workspace", http.StatusUnprocessableEntity},
		{"github failure", &mockGitHubClient{workspaceErr: errors.New("boom")}, "/creator-prs/4/workspace", http.StatusBadGateway},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewHandler(batchRepo(), tt.gh, "", "/docs/quickstarts/")
			rr := workspaceRequest(h.GetCreatorPRWorkspace, "/creator-prs/{number}/workspace", tt.target)
			assert.Equal(t, tt.want, rr.Code)
		})
	}
}

func TestGetCreatorPRResource(t *testing.T) {
	newGH := func() *mockGitHubClient {
		return &mockGitHubClient{
			getPR:    &ghclient.CreatorPR{Number: 42, HeadSHA: headSHA},
			resource: &ghclient.Resource{Name: "delta", Dir: "docs/quickstarts/delta", Files: []ghclient.File{{Name: "delta.yml", Content: "x: 1\n"}}},
		}
	}
	const pattern = "/creator-prs/{number}/resources/{name}"

	gh := newGH()
	h := NewHandler(batchRepo(), gh, "", "/docs/quickstarts/")
	rr := workspaceRequest(h.GetCreatorPRResource, pattern, "/creator-prs/42/resources/delta?headSha="+headSHA)
	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	var resp CreatorPRResourceResponse
	require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &resp))
	assert.Equal(t, headSHA, resp.HeadSHA)
	assert.Equal(t, "docs/quickstarts/delta/delta.yml", resp.Resource.Files[0].Path)
	assert.Equal(t, "delta", gh.resourceName)

	rr = workspaceRequest(h.GetCreatorPRResource, pattern, "/creator-prs/42/resources/delta?headSha=stale")
	assert.Equal(t, http.StatusConflict, rr.Code, "a resource from a newer head would mix two snapshots")

	rr = workspaceRequest(h.GetCreatorPRResource, pattern, "/creator-prs/42/resources/..")
	assert.Equal(t, http.StatusBadRequest, rr.Code)

	gh = newGH()
	gh.resourceErr = ghclient.ErrResource
	h = NewHandler(batchRepo(), gh, "", "/docs/quickstarts/")
	rr = workspaceRequest(h.GetCreatorPRResource, pattern, "/creator-prs/42/resources/delta")
	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func TestGetCreatorPR_MultiResourceIsConflict(t *testing.T) {
	gh := &mockGitHubClient{getPR: &ghclient.CreatorPR{Number: 42}, prFilesErr: ghclient.ErrMultiResource}
	h := NewHandler(batchRepo(), gh, "", "/docs/quickstarts/")
	rr := workspaceRequest(h.GetCreatorPR, "/creator-prs/{number}", "/creator-prs/42")
	assert.Equal(t, http.StatusConflict, rr.Code)
	assert.Contains(t, rr.Body.String(), "Edit multiple resources")
}
