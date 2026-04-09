package services

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/RedHatInsights/quickstarts/config"
	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/sirupsen/logrus"
)

// GitService handles git operations for the git proxy endpoint
type GitService struct{}

// NewGitService creates a new GitService
func NewGitService() *GitService {
	return &GitService{}
}

// PullRequestResult holds the result of a successful PR creation
type PullRequestResult struct {
	PRUrl  string
	Branch string
}

// CreatePullRequest performs the full git workflow:
// clone → branch → write files → commit → push → create PR
func (s *GitService) CreatePullRequest(req generated.GitPullRequestRequest) (*PullRequestResult, error) {
	cfg := config.Get()

	if cfg.GitRepoURL == "" {
		return nil, fmt.Errorf("GIT_REPO_URL is not configured")
	}
	if cfg.GitAuthToken == "" {
		return nil, fmt.Errorf("GIT_AUTH_TOKEN is not configured")
	}

	// Generate a unique branch name
	branch, err := generateBranchName(req.Title)
	if err != nil {
		return nil, fmt.Errorf("failed to generate branch name: %w", err)
	}

	// Create a temp directory for the clone
	tmpDir, err := os.MkdirTemp(cfg.GitTempDir, "quickstarts-git-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() {
		if removeErr := os.RemoveAll(tmpDir); removeErr != nil {
			logrus.Warnf("failed to clean up temp directory %s: %v", tmpDir, removeErr)
		}
	}()

	repoDir := filepath.Join(tmpDir, "repo")

	// Build authenticated clone URL
	cloneURL, err := buildAuthURL(cfg.GitRepoURL, cfg.GitAuthToken)
	if err != nil {
		return nil, fmt.Errorf("failed to build authenticated URL: %w", err)
	}

	// Clone the repository (shallow clone for speed)
	if err := runGit(tmpDir, "clone", "--depth", "1", "--branch", cfg.GitDefaultBranch, cloneURL, "repo"); err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Create and checkout the new branch
	if err := runGit(repoDir, "checkout", "-b", branch); err != nil {
		return nil, fmt.Errorf("failed to create branch: %w", err)
	}

	// Write each file
	for _, f := range req.Files {
		filePath := filepath.Join(repoDir, f.Path)

		// Ensure the parent directory exists
		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", f.Path, err)
		}

		if err := os.WriteFile(filePath, []byte(f.Content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", f.Path, err)
		}
	}

	// Stage all new/changed files
	if err := runGit(repoDir, "add", "."); err != nil {
		return nil, fmt.Errorf("failed to stage files: %w", err)
	}

	// Configure git user for the commit
	if err := runGit(repoDir, "config", "user.email", "nachobot@redhat.com"); err != nil {
		return nil, fmt.Errorf("failed to configure git email: %w", err)
	}
	if err := runGit(repoDir, "config", "user.name", "nachobot"); err != nil {
		return nil, fmt.Errorf("failed to configure git name: %w", err)
	}

	// Commit
	commitMsg := req.Title
	if req.Description != nil && *req.Description != "" {
		commitMsg = fmt.Sprintf("%s\n\n%s", req.Title, *req.Description)
	}
	if err := runGit(repoDir, "commit", "-m", commitMsg); err != nil {
		return nil, fmt.Errorf("failed to commit: %w", err)
	}

	// Push the branch
	if err := runGit(repoDir, "push", "origin", branch); err != nil {
		return nil, fmt.Errorf("failed to push branch: %w", err)
	}

	// Create the pull request using the GitHub API via git
	prURL, err := createGitHubPR(cfg, branch, req.Title, req.Description)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	return &PullRequestResult{
		PRUrl:  prURL,
		Branch: branch,
	}, nil
}

// ValidateFiles checks that the submitted files are valid
func (s *GitService) ValidateFiles(files []generated.GitFile) error {
	if len(files) == 0 {
		return fmt.Errorf("at least one file is required")
	}

	for _, f := range files {
		// Validate path is not empty
		if strings.TrimSpace(f.Path) == "" {
			return fmt.Errorf("file path cannot be empty")
		}

		// Prevent path traversal
		cleanPath := filepath.Clean(f.Path)
		if strings.HasPrefix(cleanPath, "..") || filepath.IsAbs(cleanPath) {
			return fmt.Errorf("invalid file path: %s (path traversal not allowed)", f.Path)
		}

		// Content must not be empty
		if strings.TrimSpace(f.Content) == "" {
			return fmt.Errorf("file content cannot be empty for %s", f.Path)
		}

		// Validate YAML extension
		ext := strings.ToLower(filepath.Ext(f.Path))
		if ext != ".yaml" && ext != ".yml" && ext != ".json" {
			return fmt.Errorf("unsupported file extension for %s: only .yaml, .yml, and .json are allowed", f.Path)
		}
	}

	return nil
}

// generateBranchName creates a unique branch name from the PR title
func generateBranchName(title string) (string, error) {
	// Create a slug from the title
	slug := slugify(title)
	if len(slug) > 40 {
		slug = slug[:40]
	}

	// Add a random suffix for uniqueness
	suffix, err := randomHex(4)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("quickstart/%s-%s", slug, suffix), nil
}

// slugify converts a string to a URL-safe slug
func slugify(s string) string {
	s = strings.ToLower(s)
	reg := regexp.MustCompile(`[^a-z0-9]+`)
	s = reg.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// randomHex generates a random hex string of n bytes
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// buildAuthURL injects a token into an HTTPS git URL for authentication
func buildAuthURL(repoURL, token string) (string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", fmt.Errorf("invalid repository URL: %w", err)
	}

	if u.Scheme != "https" {
		return "", fmt.Errorf("only HTTPS repository URLs are supported")
	}

	u.User = url.UserPassword("x-access-token", token)
	return u.String(), nil
}

// runGit executes a git command in the given directory
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	// Mask credentials from logs
	safeArgs := make([]string, len(args))
	copy(safeArgs, args)
	for i, arg := range safeArgs {
		if strings.Contains(arg, "x-access-token") {
			safeArgs[i] = "<REDACTED_URL>"
		}
	}
	logrus.Debugf("git %s (in %s)", strings.Join(safeArgs, " "), dir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// Redact any tokens from the error output
		sanitized := redactToken(string(output))
		return fmt.Errorf("git %s failed: %s: %s", safeArgs[0], err, sanitized)
	}
	return nil
}

// redactToken removes any tokens from git output
func redactToken(s string) string {
	// Redact patterns like https://x-access-token:TOKEN@github.com
	re := regexp.MustCompile(`(https?://)x-access-token:[^@]+@`)
	return re.ReplaceAllString(s, "${1}***@")
}

// createGitHubPR creates a pull request using the GitHub REST API
func createGitHubPR(cfg *config.QuickstartsConfig, branch, title string, description *string) (string, error) {
	// Parse owner/repo from the repo URL
	owner, repo, err := parseGitHubRepo(cfg.GitRepoURL)
	if err != nil {
		return "", err
	}

	body := ""
	if description != nil {
		body = *description
	}

	// Use curl to create the PR via GitHub API
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)
	jsonBody := fmt.Sprintf(
		`{"title":%q,"head":%q,"base":%q,"body":%q}`,
		title, branch, cfg.GitDefaultBranch, body,
	)

	cmd := exec.Command("curl", "-s", "-X", "POST",
		"-H", "Accept: application/vnd.github+json",
		"-H", fmt.Sprintf("Authorization: Bearer %s", cfg.GitAuthToken),
		"-H", "Content-Type: application/json",
		"-d", jsonBody,
		apiURL,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}

	// Parse the response to extract the PR URL
	prURL, err := extractPRUrl(output)
	if err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w: %s", err, redactToken(string(output)))
	}

	return prURL, nil
}

// parseGitHubRepo extracts owner and repo name from a GitHub URL
func parseGitHubRepo(repoURL string) (string, string, error) {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid repository URL: %w", err)
	}

	// Path should be /owner/repo or /owner/repo.git
	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	parts := strings.Split(path, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("cannot parse owner/repo from URL: %s", repoURL)
	}

	return parts[0], parts[1], nil
}

// extractPRUrl parses the GitHub API JSON response to extract the html_url
func extractPRUrl(response []byte) (string, error) {
	// Simple JSON parsing without importing encoding/json to avoid
	// a dependency on the exact response structure
	// Look for "html_url": "..." in the response
	s := string(response)

	// Check for error response
	if strings.Contains(s, `"message"`) && !strings.Contains(s, `"html_url"`) {
		return "", fmt.Errorf("GitHub API error: %s", s)
	}

	// Find the html_url field (first occurrence is the PR URL)
	marker := `"html_url":"`
	idx := strings.Index(s, marker)
	if idx == -1 {
		return "", fmt.Errorf("html_url not found in response")
	}

	start := idx + len(marker)
	end := strings.Index(s[start:], `"`)
	if end == -1 {
		return "", fmt.Errorf("malformed html_url in response")
	}

	return s[start : start+end], nil
}
