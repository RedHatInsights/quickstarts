package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	gitops "github.com/RedHatInsights/quickstarts/pkg/git-service/git"
	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepoManager struct {
	pullLatestErr       error
	createBranchErr     error
	checkoutExistingErr error
	writeFilesErr       error
	commitSHA           string
	commitErr           error
	pushErr             error
	cleanupErr          error
	baseBranch          string

	writtenDir   string
	writtenFiles []gitops.File
	removedDir   string
	removedFiles []string
	removeErr    error
	pushedBranch string
	forcePushed  bool
	cleanedUp    string

	directories     []string
	listDirsErr     error
	files           []string
	listFilesErr    error
	fileContents    map[string]string
	readFileErr     error
	pullLatestCount int
}

func (m *mockRepoManager) PullLatest() error {
	m.pullLatestCount++
	return m.pullLatestErr
}
func (m *mockRepoManager) CreateBranch(name string) error           { return m.createBranchErr }
func (m *mockRepoManager) CheckoutExistingBranch(name string) error { return m.checkoutExistingErr }
func (m *mockRepoManager) PushBranch(branch string) error           { m.pushedBranch = branch; return m.pushErr }
func (m *mockRepoManager) PushBranchForce(branch string) error {
	m.pushedBranch = branch
	m.forcePushed = true
	return m.pushErr
}
func (m *mockRepoManager) Cleanup(branch string) error { m.cleanedUp = branch; return m.cleanupErr }
func (m *mockRepoManager) GetBaseBranch() string       { return m.baseBranch }
func (m *mockRepoManager) WriteFiles(dir string, files []gitops.File) error {
	m.writtenDir = dir
	m.writtenFiles = files
	return m.writeFilesErr
}
func (m *mockRepoManager) RemoveFiles(dir string, names []string) error {
	m.removedDir = dir
	m.removedFiles = names
	return m.removeErr
}
func (m *mockRepoManager) CommitChanges(message, authorName, authorEmail, dir string, files []gitops.File) (string, error) {
	return m.commitSHA, m.commitErr
}
func (m *mockRepoManager) ListDirectories(basePath string) ([]string, error) {
	return m.directories, m.listDirsErr
}
func (m *mockRepoManager) ListFiles(basePath string) ([]string, error) {
	return m.files, m.listFilesErr
}
func (m *mockRepoManager) ReadFile(path string) (string, error) {
	if m.readFileErr != nil {
		return "", m.readFileErr
	}
	if m.fileContents != nil {
		if content, ok := m.fileContents[path]; ok {
			return content, nil
		}
	}
	return "", fmt.Errorf("file not found: %s", path)
}

type mockGitHubClient struct {
	createPRURL    string
	createPRNumber int
	createPRErr    error
	assignErr      error
	listPRs        []ghclient.CreatorPR
	listPRsErr     error
	getPR          *ghclient.CreatorPR
	getPRErr       error
	prFiles        []ghclient.File
	prFilesSlug    string
	prFilesErr     error
	findURL        string
	findURLErr     error

	createdTitle string
	createdBody  string
	createdHead  string
	createdBase  string
	assignedTeam string
}

func (m *mockGitHubClient) CreatePullRequest(ctx context.Context, title, body, head, base string) (string, int, error) {
	m.createdTitle = title
	m.createdBody = body
	m.createdHead = head
	m.createdBase = base
	return m.createPRURL, m.createPRNumber, m.createPRErr
}
func (m *mockGitHubClient) AssignReviewers(ctx context.Context, prNumber int, team string) error {
	m.assignedTeam = team
	return m.assignErr
}
func (m *mockGitHubClient) ListCreatorPRs(ctx context.Context) ([]ghclient.CreatorPR, error) {
	return m.listPRs, m.listPRsErr
}
func (m *mockGitHubClient) GetCreatorPR(ctx context.Context, prNumber int) (*ghclient.CreatorPR, error) {
	if m.getPRErr != nil {
		return nil, m.getPRErr
	}
	if m.getPR != nil {
		return m.getPR, nil
	}
	return &ghclient.CreatorPR{Number: prNumber, HTMLURL: m.createPRURL, BranchName: "branch"}, nil
}
func (m *mockGitHubClient) GetPRQuickstartFiles(ctx context.Context, pr *ghclient.CreatorPR) (string, []ghclient.File, error) {
	return m.prFilesSlug, m.prFiles, m.prFilesErr
}
func (m *mockGitHubClient) FindPRURLByBranch(ctx context.Context, branchName string) (string, error) {
	return m.findURL, m.findURLErr
}

func validRequestBody() string {
	return `{
		"files": [{"name": "metadata.yaml", "content": "name: test"}],
		"metadata": {
			"branchName": "quickstart/test-123",
			"commitMessage": "Add test quickstart",
			"prTitle": "Create test quickstart",
			"prBody": "Generated from creator",
			"userEmail": "user@example.com"
		}
	}`
}

func TestValidateRequest_MissingFiles(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{},
		Metadata: PRMetadata{
			BranchName:    "test",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "files are required")
}

func TestValidateRequest_MissingBranchName(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "f", Content: "c"}},
		Metadata: PRMetadata{
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "branchName is required")
}

func TestValidateRequest_MissingExistingPathOnUpdate(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "f", Content: "c"}},
		Metadata: PRMetadata{
			BranchName:    "test",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
			IsUpdate:      true,
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "existingPath is required when isUpdate is true")
}

func TestValidateRequest_Valid(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "f", Content: "c"}},
		Metadata: PRMetadata{
			BranchName:    "test",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
		},
	}
	err := validateRequest(req)
	assert.NoError(t, err)
}

func TestSubmitPR_InvalidJSON(t *testing.T) {
	handler := NewHandler(&mockRepoManager{}, &mockGitHubClient{}, "", "")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString("not json"))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]string
	json.NewDecoder(rec.Body).Decode(&resp)
	assert.Equal(t, "invalid request body", resp["msg"])
}

func TestSubmitPR_MissingFields(t *testing.T) {
	handler := NewHandler(&mockRepoManager{}, &mockGitHubClient{}, "", "")
	body := `{"files":[], "metadata":{"branchName":"test"}}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSubmitPR_Success(t *testing.T) {
	repo := &mockRepoManager{commitSHA: "abc123def456abc123def456abc123def456abcd", baseBranch: "main"}
	gh := &mockGitHubClient{createPRURL: "https://github.com/org/repo/pull/42", createPRNumber: 42}
	handler := NewHandler(repo, gh, "team-reviewers", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(validRequestBody()))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp SubmitPRResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "https://github.com/org/repo/pull/42", resp.PRURL)
	assert.Equal(t, "quickstart/test-123", resp.BranchName)
	assert.Equal(t, "abc123def456abc123def456abc123def456abcd", resp.CommitSHA)
	assert.Equal(t, "created", resp.Status)

	assert.Equal(t, "/docs/quickstarts/quickstart/test-123/", repo.writtenDir)
	assert.Len(t, repo.writtenFiles, 1)
	assert.Equal(t, "quickstart/test-123", repo.pushedBranch)
	assert.Equal(t, "quickstart/test-123", repo.cleanedUp)

	assert.Equal(t, "Create test quickstart", gh.createdTitle)
	assert.Contains(t, gh.createdBody, "Generated from creator")
	assert.Contains(t, gh.createdBody, "Submitted by: user@example.com")
	assert.Equal(t, "quickstart/test-123", gh.createdHead)
	assert.Equal(t, "main", gh.createdBase)
	assert.Equal(t, "team-reviewers", gh.assignedTeam)
}

func TestSubmitPR_DirectoryName(t *testing.T) {
	repo := &mockRepoManager{commitSHA: "abc123def456abc123def456abc123def456abcd", baseBranch: "main"}
	gh := &mockGitHubClient{createPRURL: "https://github.com/org/repo/pull/44", createPRNumber: 44}
	handler := NewHandler(repo, gh, "", "/docs/quickstarts/")

	body := `{
		"files": [{"name": "metadata.yml", "content": "name: test"}],
		"metadata": {
			"branchName": "quickstart/my-quickstart-1720000000",
			"commitMessage": "Add quickstart",
			"prTitle": "New quickstart",
			"prBody": "Adding new quickstart",
			"userEmail": "user@example.com",
			"directoryName": "my-quickstart"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/docs/quickstarts/my-quickstart/", repo.writtenDir)
}

func TestSubmitPR_UpdateMode(t *testing.T) {
	repo := &mockRepoManager{commitSHA: "abc123def456abc123def456abc123def456abcd", baseBranch: "main"}
	gh := &mockGitHubClient{
		getPR: &ghclient.CreatorPR{
			Number:  43,
			HTMLURL: "https://github.com/org/repo/pull/43",
		},
	}
	handler := NewHandler(repo, gh, "", "/docs/quickstarts/")

	body := `{
		"files": [{"name": "metadata.yaml", "content": "updated"}],
		"metadata": {
			"branchName": "quickstart/update-123",
			"commitMessage": "Update quickstart",
			"prTitle": "Update test quickstart",
			"prBody": "Updating existing",
			"userEmail": "user@example.com",
			"isUpdate": true,
			"existingPath": "/docs/quickstarts/existing-qs/",
			"prNumber": 43
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp SubmitPRResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "updated", resp.Status)
	assert.Equal(t, "https://github.com/org/repo/pull/43", resp.PRURL)
	assert.Equal(t, "/docs/quickstarts/existing-qs/", repo.writtenDir)
	assert.True(t, repo.forcePushed, "update mode should force-push")
	assert.Empty(t, gh.createdTitle, "update mode should not create a new PR")
}

func TestSubmitPR_UpdateMode_ContinueMissingBranch(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:           "abc123def456abc123def456abc123def456abcd",
		baseBranch:          "main",
		checkoutExistingErr: fmt.Errorf("remote branch not found"),
	}
	gh := &mockGitHubClient{}
	handler := NewHandler(repo, gh, "", "/docs/quickstarts/")

	body := `{
		"files": [{"name": "metadata.yaml", "content": "updated"}],
		"metadata": {
			"branchName": "qs-create-demo-1",
			"commitMessage": "Update quickstart",
			"prTitle": "Update test quickstart",
			"prBody": "Updating existing",
			"userEmail": "user@example.com",
			"isUpdate": true,
			"existingPath": "docs/quickstarts/demo/",
			"prNumber": 11
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to checkout existing PR branch")
	assert.Empty(t, gh.createdTitle)
}

func TestSubmitPR_UpdateMode_NewBranch(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:           "abc123def456abc123def456abc123def456abcd",
		baseBranch:          "main",
		checkoutExistingErr: fmt.Errorf("remote branch not found"),
	}
	gh := &mockGitHubClient{createPRURL: "https://github.com/org/repo/pull/45", createPRNumber: 45}
	handler := NewHandler(repo, gh, "team-reviewers", "/docs/quickstarts/")

	body := `{
		"files": [{"name": "metadata.yaml", "content": "updated"}],
		"metadata": {
			"branchName": "quickstart/update-123",
			"commitMessage": "Update quickstart",
			"prTitle": "Update test quickstart",
			"prBody": "Updating existing",
			"userEmail": "user@example.com",
			"isUpdate": true,
			"existingPath": "/docs/quickstarts/existing-qs/"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp SubmitPRResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "created", resp.Status, "first-time update should create a PR")
	assert.Equal(t, "https://github.com/org/repo/pull/45", resp.PRURL)
	assert.Equal(t, "/docs/quickstarts/existing-qs/", repo.writtenDir)
	assert.False(t, repo.forcePushed, "first-time update should not force-push")
	assert.Equal(t, "Update test quickstart", gh.createdTitle, "first-time update should create a PR")
}

func updateBodyWithFiles(files string) string {
	return updateBodyWithPath("/docs/quickstarts/existing-qs/", files)
}

func updateBodyWithPath(existingPath, files string) string {
	return `{
		"files": ` + files + `,
		"metadata": {
			"branchName": "quickstart/update-123",
			"commitMessage": "Update quickstart",
			"prTitle": "Update test quickstart",
			"prBody": "Updating existing",
			"isUpdate": true,
			"existingPath": "` + existingPath + `",
			"prNumber": 43
		}
	}`
}

// The deployed creator pins directoryName to the directory already on the PR
// head, so a rename has to be recognised from the submitted metadata alone.
func TestSubmitPR_UpdateMode_RenameInMetadataMovesDirectory(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:  "abc123def456abc123def456abc123def456abcd",
		baseBranch: "main",
		files:      []string{"metadata.yaml", "old-name.yaml"},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	body := updateBodyWithPath("/docs/quickstarts/old-name/", `[
		{"name": "metadata.yaml", "content": "kind: QuickStarts\nname: new-name"},
		{"name": "new-name.yaml", "content": "renamed"}
	]`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/docs/quickstarts/new-name/", repo.writtenDir, "files belong in the renamed directory")
	assert.Equal(t, "/docs/quickstarts/old-name/", repo.removedDir)
	assert.ElementsMatch(t, []string{"metadata.yaml", "old-name.yaml"}, repo.removedFiles,
		"the whole pre-rename directory should be retired, including files whose names did not change")
}

func TestSubmitPR_UpdateMode_SanitizesTheNameItReadsFromMetadata(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
	}{
		{"My Doc", "/docs/quickstarts/my-doc/"},
		{"Test-Open-PR-Editing", "/docs/quickstarts/test-open-pr-editing/"},
		{"../../etc", "/docs/quickstarts/etc/"},
		{"a//b", "/docs/quickstarts/a-b/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			repo := &mockRepoManager{commitSHA: "abc123def456abc123def456abc123def456abcd", baseBranch: "main"}
			handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

			body := updateBodyWithPath("/docs/quickstarts/existing-qs/", `[
				{"name": "metadata.yaml", "content": "name: `+tc.name+`"}
			]`)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
			rec := httptest.NewRecorder()
			handler.SubmitPR(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, tc.want, repo.writtenDir, "a name read from file content must stay one safe segment")
		})
	}
}

func TestSubmitPR_UpdateMode_UnusableMetadataKeepsExistingPath(t *testing.T) {
	for _, content := range []string{
		"kind: QuickStarts",   // no name at all
		"name: \"...\"",       // sanitizes away to nothing
		"- not: a mapping",    // parses, but not into the expected shape
		"a: b\n  c: ::broken", // does not parse
	} {
		t.Run(content, func(t *testing.T) {
			repo := &mockRepoManager{
				commitSHA:  "abc123def456abc123def456abc123def456abcd",
				baseBranch: "main",
				files:      []string{"metadata.yaml"},
			}
			handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

			payload, err := json.Marshal(content)
			require.NoError(t, err)
			body := updateBodyWithPath("/docs/quickstarts/existing-qs/", `[
				{"name": "metadata.yaml", "content": `+string(payload)+`}
			]`)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
			rec := httptest.NewRecorder()
			handler.SubmitPR(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			assert.Equal(t, "/docs/quickstarts/existing-qs/", repo.writtenDir,
				"an unusable name must leave the quickstart where it is, not move it somewhere odd")
			assert.Empty(t, repo.removedFiles, "and it must not retire the directory it is still writing to")
		})
	}
}

func TestSubmitPR_UpdateMode_KeepsFilesStillPresent(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:  "abc123def456abc123def456abc123def456abcd",
		baseBranch: "main",
		files:      []string{"metadata.yaml", "same-name.yaml"},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	body := updateBodyWithPath("/docs/quickstarts/same-name/", `[
		{"name": "metadata.yaml", "content": "kind: QuickStarts\nname: same-name"},
		{"name": "same-name.yaml", "content": "edited"}
	]`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/docs/quickstarts/same-name/", repo.writtenDir)
	assert.Empty(t, repo.removedFiles, "an edit without a rename should delete nothing")
}

// updateBodyWithRename is an update that also carries a directoryName, which
// older creator builds pin to the directory already on the PR head.
func updateBodyWithRename(existingPath, dirName, files string) string {
	return `{
		"files": ` + files + `,
		"metadata": {
			"branchName": "quickstart/update-123",
			"commitMessage": "Update quickstart",
			"prTitle": "Update test quickstart",
			"prBody": "Updating existing",
			"isUpdate": true,
			"existingPath": "` + existingPath + `",
			"directoryName": "` + dirName + `",
			"prNumber": 43
		}
	}`
}

func TestSubmitPR_UpdateMode_MetadataNameBeatsPinnedDirectoryName(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:  "abc123def456abc123def456abc123def456abcd",
		baseBranch: "main",
		files:      []string{"metadata.yaml", "old-name.yaml"},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	// directoryName still names the pre-rename directory, as the deployed
	// creator sends it. The metadata is what the user just renamed.
	body := updateBodyWithRename("/docs/quickstarts/old-name/", "old-name", `[
		{"name": "metadata.yaml", "content": "kind: QuickStarts\nname: new-name"},
		{"name": "new-name.yaml", "content": "renamed"}
	]`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/docs/quickstarts/new-name/", repo.writtenDir, "a stale directoryName must not pin the quickstart in place")
	assert.Equal(t, "/docs/quickstarts/old-name/", repo.removedDir)
	assert.ElementsMatch(t, []string{"metadata.yaml", "old-name.yaml"}, repo.removedFiles,
		"the whole pre-rename directory should be retired, including files whose names did not change")
}

func TestSubmitPR_UpdateMode_SameDirectoryOnlyPrunes(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:  "abc123def456abc123def456abc123def456abcd",
		baseBranch: "main",
		files:      []string{"metadata.yaml", "existing-qs.yaml", "leftover.yaml"},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	// existingPath has no leading slash but quickstartsDirPath does, so this
	// also covers the two spellings resolving to the same directory.
	body := updateBodyWithRename("docs/quickstarts/existing-qs/", "existing-qs", `[
		{"name": "metadata.yaml", "content": "kind: QuickStarts\nname: existing-qs"},
		{"name": "existing-qs.yaml", "content": "edited"}
	]`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "/docs/quickstarts/existing-qs/", repo.writtenDir)
	assert.Equal(t, []string{"leftover.yaml"}, repo.removedFiles,
		"staying put should drop only what the update no longer sends, never metadata.yaml")
}

func TestSubmitPR_UpdateMode_RenameOutOfBoundsDirIsNotPruned(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:  "abc123def456abc123def456abc123def456abcd",
		baseBranch: "main",
		files:      []string{"keep-me.yaml"},
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	body := updateBodyWithRename("/docs/", "new-name", `[
		{"name": "metadata.yaml", "content": "kind: QuickStarts\nname: new-name"}
	]`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, repo.removedFiles, "a move must not empty a directory outside docs/quickstarts/<name>/")
}

func TestValidateRequest_DirectoryNameMustBeOneSegment(t *testing.T) {
	for _, name := range []string{"nested/dir", `back\slash`, "."} {
		t.Run(name, func(t *testing.T) {
			err := validateRequest(&SubmitPRRequest{
				Files: []File{{Name: "metadata.yaml", Content: "x"}},
				Metadata: PRMetadata{
					BranchName:    "qs-create-1",
					CommitMessage: "Add",
					PRTitle:       "Add",
					PRBody:        "Add",
					DirectoryName: name,
				},
			})
			assert.Error(t, err)
		})
	}
}

func TestSubmitPR_CreateMode_NeverRemoves(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:  "abc123def456abc123def456abc123def456abcd",
		baseBranch: "main",
		files:      []string{"metadata.yaml", "unrelated.yaml"},
	}
	handler := NewHandler(repo, &mockGitHubClient{createPRURL: "https://github.com/org/repo/pull/1"}, "", "/docs/quickstarts/")

	body := `{
		"files": [{"name": "metadata.yaml", "content": "new"}],
		"metadata": {
			"branchName": "qs-create-demo-1",
			"commitMessage": "Add quickstart",
			"prTitle": "Add test quickstart",
			"prBody": "Adding new",
			"directoryName": "demo"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, repo.removedFiles, "create mode must not delete anything")
}

func TestSubmitPR_UpdateMode_ListFilesErrorDoesNotBlock(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:    "abc123def456abc123def456abc123def456abcd",
		baseBranch:   "main",
		listFilesErr: fmt.Errorf("directory does not exist"),
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	body := updateBodyWithFiles(`[{"name": "metadata.yaml", "content": "updated"}]`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "prune is best effort and must not fail the update")
	assert.Empty(t, repo.removedFiles)
}

func TestSubmitPR_UpdateMode_RemoveErrorDoesNotBlock(t *testing.T) {
	repo := &mockRepoManager{
		commitSHA:  "abc123def456abc123def456abc123def456abcd",
		baseBranch: "main",
		files:      []string{"metadata.yaml", "old-name.yaml"},
		removeErr:  fmt.Errorf("staging failed"),
	}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	body := updateBodyWithFiles(`[{"name": "metadata.yaml", "content": "updated"}]`)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code, "a failed delete must not lose the user's edit")
	assert.True(t, repo.forcePushed)
}

func TestSubmitPR_UpdateMode_RefusesToPruneOutsideQuickstarts(t *testing.T) {
	for _, existingPath := range []string{"/docs/", "/", "/docs/quickstarts/", "/docs/quickstarts/a/b/", "/other/place/"} {
		t.Run(existingPath, func(t *testing.T) {
			repo := &mockRepoManager{
				commitSHA:  "abc123def456abc123def456abc123def456abcd",
				baseBranch: "main",
				files:      []string{"README.md", "CONTRIBUTING.md"},
			}
			handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

			body := `{
				"files": [{"name": "metadata.yaml", "content": "updated"}],
				"metadata": {
					"branchName": "quickstart/update-123",
					"commitMessage": "Update quickstart",
					"prTitle": "Update test quickstart",
					"prBody": "Updating existing",
					"isUpdate": true,
					"existingPath": "` + existingPath + `",
					"prNumber": 43
				}
			}`

			req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
			rec := httptest.NewRecorder()
			handler.SubmitPR(rec, req)

			assert.Empty(t, repo.removedFiles, "prune must be confined to a single quickstart directory")
		})
	}
}

func TestIsQuickstartDir(t *testing.T) {
	base := "/docs/quickstarts/"
	assert.True(t, isQuickstartDir("/docs/quickstarts/demo/", base))
	assert.True(t, isQuickstartDir("docs/quickstarts/demo", base))
	assert.False(t, isQuickstartDir("/docs/quickstarts/", base), "the parent directory is not a quickstart")
	assert.False(t, isQuickstartDir("/docs/", base))
	assert.False(t, isQuickstartDir("/docs/quickstarts/a/b", base), "quickstarts are exactly one level deep")
	assert.False(t, isQuickstartDir("/other/demo/", base))
	assert.False(t, isQuickstartDir("", base))
	assert.False(t, isQuickstartDir("/docs/quickstarts/demo/", ""))
}

func TestSubmitPR_PullLatestError(t *testing.T) {
	repo := &mockRepoManager{pullLatestErr: fmt.Errorf("network error")}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(validRequestBody()))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to pull latest changes")
}

func TestSubmitPR_CreateBranchError(t *testing.T) {
	repo := &mockRepoManager{createBranchErr: fmt.Errorf("branch exists")}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(validRequestBody()))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to create branch")
}

func TestSubmitPR_WriteFilesError(t *testing.T) {
	repo := &mockRepoManager{writeFilesErr: fmt.Errorf("disk full")}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(validRequestBody()))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to write files")
	assert.Equal(t, "quickstart/test-123", repo.cleanedUp)
}

func TestSubmitPR_CommitError(t *testing.T) {
	repo := &mockRepoManager{commitErr: fmt.Errorf("nothing to commit")}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(validRequestBody()))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to commit changes")
	assert.Equal(t, "quickstart/test-123", repo.cleanedUp)
}

func TestSubmitPR_PushError(t *testing.T) {
	repo := &mockRepoManager{commitSHA: "abc123", pushErr: fmt.Errorf("auth failed")}
	handler := NewHandler(repo, &mockGitHubClient{}, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(validRequestBody()))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to push branch")
	assert.Equal(t, "quickstart/test-123", repo.cleanedUp)
}

func TestValidateRequest_TraversalInBranchName(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "f", Content: "c"}},
		Metadata: PRMetadata{
			BranchName:    "../../etc/passwd",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "branchName contains invalid path segment")
}

func TestValidateRequest_TraversalInExistingPath(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "f", Content: "c"}},
		Metadata: PRMetadata{
			BranchName:    "test",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
			IsUpdate:      true,
			ExistingPath:  "../../../etc/",
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "existingPath contains invalid path segment")
}

func TestValidateRequest_TraversalInFileName(t *testing.T) {
	req := &SubmitPRRequest{
		Files: []File{{Name: "../../etc/passwd", Content: "c"}},
		Metadata: PRMetadata{
			BranchName:    "test",
			CommitMessage: "msg",
			PRTitle:       "title",
			PRBody:        "body",
			UserEmail:     "test@test.com",
		},
	}
	err := validateRequest(req)
	assert.EqualError(t, err, "file name contains invalid path segment: ../../etc/passwd")
}

func TestSubmitPR_CreatePRError(t *testing.T) {
	repo := &mockRepoManager{commitSHA: "abc123", baseBranch: "main"}
	gh := &mockGitHubClient{createPRErr: fmt.Errorf("GitHub API error")}
	handler := NewHandler(repo, gh, "", "/docs/quickstarts/")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(validRequestBody()))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "failed to create pull request")
	assert.Equal(t, "quickstart/test-123", repo.cleanedUp)
}
