package services

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

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

	// Create a GIT_ASKPASS helper script to supply credentials without
	// embedding tokens in the clone URL (avoids exposure via process listing)
	askpassScript, err := createAskpassScript(cfg.GitAuthToken, tmpDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create askpass helper: %w", err)
	}
	defer os.Remove(askpassScript)

	// Clone the repository (shallow clone for speed)
	if err := runGitWithAskpass(tmpDir, askpassScript, "clone", "--depth", "1", "--branch", cfg.GitDefaultBranch, cfg.GitRepoURL, "repo"); err != nil {
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

	// Configure git user for the commit (configurable via env vars)
	if err := runGit(repoDir, "config", "user.email", cfg.GitUserEmail); err != nil {
		return nil, fmt.Errorf("failed to configure git email: %w", err)
	}
	if err := runGit(repoDir, "config", "user.name", cfg.GitUserName); err != nil {
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

	// Push the branch (using askpass for auth)
	if err := runGitWithAskpass(repoDir, askpassScript, "push", "origin", branch); err != nil {
		return nil, fmt.Errorf("failed to push branch: %w", err)
	}

	// Create the pull request using the GitHub REST API
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

// createAskpassScript creates a temporary script that supplies the token
// via GIT_ASKPASS, keeping credentials out of the process argument list.
func createAskpassScript(token, tmpDir string) (string, error) {
	script := fmt.Sprintf("#!/bin/sh\necho '%s'\n", token)
	f, err := os.CreateTemp(tmpDir, "git-askpass-*.sh")
	if err != nil {
		return "", err
	}
	if _, err := f.WriteString(script); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", err
	}
	f.Close()
	if err := os.Chmod(f.Name(), 0700); err != nil {
		os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// runGit executes a git command in the given directory
func runGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	logrus.Debugf("git %s (in %s)", strings.Join(args, " "), dir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		sanitized := redactToken(string(output))
		return fmt.Errorf("git %s failed: %s: %s", args[0], err, sanitized)
	}
	return nil
}

// runGitWithAskpass executes a git command with GIT_ASKPASS set for auth.
// This keeps tokens out of the process argument list (visible via ps/top).
func runGitWithAskpass(dir, askpassScript string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GIT_ASKPASS=%s", askpassScript),
		"GIT_TERMINAL_PROMPT=0",
	)
	logrus.Debugf("git %s (in %s)", strings.Join(args, " "), dir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		sanitized := redactToken(string(output))
		return fmt.Errorf("git %s failed: %s: %s", args[0], err, sanitized)
	}
	return nil
}

// redactToken removes any tokens from git output
func redactToken(s string) string {
	// Redact patterns like https://x-access-token:TOKEN@github.com
	re := regexp.MustCompile(`(https?://)x-access-token:[^@]+@`)
	return re.ReplaceAllString(s, "${1}***@")
}

// ghPullRequestRequest is the JSON body for creating a GitHub PR
type ghPullRequestRequest struct {
	Title string `json:"title"`
	Head  string `json:"head"`
	Base  string `json:"base"`
	Body  string `json:"body"`
}

// ghPullRequestResponse is the relevant part of the GitHub PR response
type ghPullRequestResponse struct {
	HTMLURL string `json:"html_url"`
	Message string `json:"message,omitempty"`
}

// createGitHubPR creates a pull request using Go's net/http client
func createGitHubPR(cfg *config.QuickstartsConfig, branch, title string, description *string) (string, error) {
	owner, repo, err := parseGitHubRepo(cfg.GitRepoURL)
	if err != nil {
		return "", err
	}

	body := ""
	if description != nil {
		body = *description
	}

	reqBody := ghPullRequestRequest{
		Title: title,
		Head:  branch,
		Base:  cfg.GitDefaultBranch,
		Body:  body,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal PR request: %w", err)
	}

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls", owner, repo)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/vnd.github+json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", cfg.GitAuthToken))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("GitHub API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read GitHub API response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var prResp ghPullRequestResponse
	if err := json.Unmarshal(respBody, &prResp); err != nil {
		return "", fmt.Errorf("failed to parse GitHub API response: %w", err)
	}

	if prResp.HTMLURL == "" {
		return "", fmt.Errorf("GitHub API response missing html_url: %s", string(respBody))
	}

	return prResp.HTMLURL, nil
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
