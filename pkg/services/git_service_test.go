package services

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/stretchr/testify/assert"
)

func TestValidateFiles_EmptyFiles(t *testing.T) {
	svc := NewGitService()
	err := svc.ValidateFiles([]generated.GitFile{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least one file is required")
}

func TestValidateFiles_EmptyPath(t *testing.T) {
	svc := NewGitService()
	err := svc.ValidateFiles([]generated.GitFile{
		{Path: "", Content: "some content"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file path cannot be empty")
}

func TestValidateFiles_PathTraversal(t *testing.T) {
	svc := NewGitService()

	tests := []struct {
		name string
		path string
	}{
		{"dot-dot prefix", "../etc/passwd"},
		{"nested dot-dot", "foo/../../etc/passwd"},
		{"absolute path", "/etc/passwd"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateFiles([]generated.GitFile{
				{Path: tt.path, Content: "content"},
			})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "path traversal not allowed")
		})
	}
}

func TestValidateFiles_EmptyContent(t *testing.T) {
	svc := NewGitService()
	err := svc.ValidateFiles([]generated.GitFile{
		{Path: "quickstarts/test.yaml", Content: "  "},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file content cannot be empty")
}

func TestValidateFiles_InvalidExtension(t *testing.T) {
	svc := NewGitService()
	err := svc.ValidateFiles([]generated.GitFile{
		{Path: "quickstarts/test.txt", Content: "content"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported file extension")
}

func TestValidateFiles_ValidFiles(t *testing.T) {
	svc := NewGitService()

	tests := []struct {
		name string
		path string
	}{
		{"yaml extension", "quickstarts/my-quickstart.yaml"},
		{"yml extension", "quickstarts/my-quickstart.yml"},
		{"json extension", "quickstarts/my-quickstart.json"},
		{"nested path", "data/quickstarts/getting-started/content.yaml"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := svc.ValidateFiles([]generated.GitFile{
				{Path: tt.path, Content: "name: test"},
			})
			assert.NoError(t, err)
		})
	}
}

func TestValidateFiles_MultipleFiles(t *testing.T) {
	svc := NewGitService()
	err := svc.ValidateFiles([]generated.GitFile{
		{Path: "quickstarts/qs1.yaml", Content: "name: qs1"},
		{Path: "quickstarts/qs2.yml", Content: "name: qs2"},
	})
	assert.NoError(t, err)
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Add New Quickstart: Getting Started", "add-new-quickstart-getting-started"},
		{"Fix  Multiple   Spaces", "fix-multiple-spaces"},
		{"special!@#chars$%^test", "special-chars-test"},
		{"---leading-trailing---", "leading-trailing"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := slugify(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGenerateBranchName(t *testing.T) {
	branch, err := generateBranchName("Add Getting Started Guide")
	assert.NoError(t, err)
	assert.Contains(t, branch, "quickstart/add-getting-started-guide-")
	// Should have the quickstart/ prefix and a hex suffix
	assert.Regexp(t, `^quickstart/[a-z0-9-]+-[0-9a-f]{8}$`, branch)
}

func TestGenerateBranchName_LongTitle(t *testing.T) {
	longTitle := "This is a very long title that should be truncated to keep the branch name reasonable in length"
	branch, err := generateBranchName(longTitle)
	assert.NoError(t, err)
	// Branch name prefix (quickstart/) + slug (max 40) + dash + hex (8) = max ~59 chars
	assert.Less(t, len(branch), 65)
}

func TestCreateAskpassScript(t *testing.T) {
	tmpDir := t.TempDir()
	script, err := createAskpassScript("test-token", tmpDir)
	assert.NoError(t, err)
	defer os.Remove(script)

	// Verify the script exists and is executable
	info, err := os.Stat(script)
	assert.NoError(t, err)
	assert.True(t, info.Mode()&0100 != 0, "script should be executable")

	// Verify the script content echoes the token
	content, err := os.ReadFile(script)
	assert.NoError(t, err)
	assert.Contains(t, string(content), "test-token")
}

func TestParseGitHubRepo(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "standard HTTPS URL",
			url:       "https://github.com/RedHatInsights/quickstarts.git",
			wantOwner: "RedHatInsights",
			wantRepo:  "quickstarts",
		},
		{
			name:      "URL without .git",
			url:       "https://github.com/RedHatInsights/quickstarts",
			wantOwner: "RedHatInsights",
			wantRepo:  "quickstarts",
		},
		{
			name:    "invalid URL - too few parts",
			url:     "https://github.com/RedHatInsights",
			wantErr: true,
		},
		{
			name:    "invalid URL - too many parts",
			url:     "https://github.com/a/b/c",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := parseGitHubRepo(tt.url)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantOwner, owner)
				assert.Equal(t, tt.wantRepo, repo)
			}
		})
	}
}

func TestRedactToken(t *testing.T) {
	input := "fatal: Authentication failed for 'https://x-access-token:ghp_secret123@github.com/org/repo.git'"
	result := redactToken(input)
	assert.NotContains(t, result, "ghp_secret123")
	assert.Contains(t, result, "***@github.com")
}

func TestGhPullRequestResponseParsing(t *testing.T) {
	// Test that the JSON struct correctly parses GitHub API responses
	response := `{"html_url":"https://github.com/org/repo/pull/42","number":42}`
	var prResp ghPullRequestResponse
	err := json.Unmarshal([]byte(response), &prResp)
	assert.NoError(t, err)
	assert.Equal(t, "https://github.com/org/repo/pull/42", prResp.HTMLURL)
}

func TestGhPullRequestResponseParsing_ErrorResponse(t *testing.T) {
	response := `{"message":"Validation Failed"}`
	var prResp ghPullRequestResponse
	err := json.Unmarshal([]byte(response), &prResp)
	assert.NoError(t, err)
	assert.Equal(t, "Validation Failed", prResp.Message)
	assert.Empty(t, prResp.HTMLURL)
}

func TestGhPullRequestResponseParsing_EmptyURL(t *testing.T) {
	response := `{"html_url":"","number":42}`
	var prResp ghPullRequestResponse
	err := json.Unmarshal([]byte(response), &prResp)
	assert.NoError(t, err)
	assert.Empty(t, prResp.HTMLURL)
}
