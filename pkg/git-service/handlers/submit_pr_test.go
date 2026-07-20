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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepoManager struct {
	pullLatestErr          error
	createBranchErr        error
	checkoutExistingErr    error
	writeFilesErr          error
	commitSHA              string
	commitErr              error
	pushErr                error
	cleanupErr             error
	baseBranch             string

	writtenDir   string
	writtenFiles []gitops.File
	pushedBranch string
	forcePushed  bool
	cleanedUp    string

	directories    []string
	listDirsErr    error
	files          []string
	listFilesErr   error
	fileContents   map[string]string
	readFileErr    error
}

func (m *mockRepoManager) PullLatest() error                            { return m.pullLatestErr }
func (m *mockRepoManager) CreateBranch(name string) error               { return m.createBranchErr }
func (m *mockRepoManager) CheckoutExistingBranch(name string) error     { return m.checkoutExistingErr }
func (m *mockRepoManager) PushBranch(branch string) error               { m.pushedBranch = branch; return m.pushErr }
func (m *mockRepoManager) PushBranchForce(branch string) error          { m.pushedBranch = branch; m.forcePushed = true; return m.pushErr }
func (m *mockRepoManager) Cleanup(branch string) error                  { m.cleanedUp = branch; return m.cleanupErr }
func (m *mockRepoManager) GetBaseBranch() string                        { return m.baseBranch }
func (m *mockRepoManager) WriteFiles(dir string, files []gitops.File) error {
	m.writtenDir = dir
	m.writtenFiles = files
	return m.writeFilesErr
}
func (m *mockRepoManager) CommitChanges(message, authorName, authorEmail string) (string, error) {
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
	gh := &mockGitHubClient{createPRURL: "https://github.com/org/repo/pull/43", createPRNumber: 43}
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
			"existingPath": "/docs/quickstarts/existing-qs/"
		}
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/submit-pr", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()

	handler.SubmitPR(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp SubmitPRResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "updated", resp.Status)
	assert.Equal(t, "/docs/quickstarts/existing-qs/", repo.writtenDir)
	assert.True(t, repo.forcePushed, "update mode should force-push")
	assert.Empty(t, gh.createdTitle, "update mode should not create a new PR")
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
