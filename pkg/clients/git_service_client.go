package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type GitService struct {
	baseURL    string
	httpClient *http.Client
	pskToken   string
}

func NewGitService(baseURL, pskToken string) *GitService {
	return &GitService{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		pskToken: pskToken,
	}
}

type GitServiceFile struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type GitServiceMetadata struct {
	BranchName    string `json:"branchName"`
	CommitMessage string `json:"commitMessage"`
	PRTitle       string `json:"prTitle"`
	PRBody        string `json:"prBody"`
	UserEmail     string `json:"userEmail"`
	IsUpdate      bool   `json:"isUpdate"`
	ExistingPath  string `json:"existingPath,omitempty"`
	DirectoryName string `json:"directoryName,omitempty"`
	PRNumber      int    `json:"prNumber,omitempty"`
}

type GitServiceResponse struct {
	PRURL      string `json:"prUrl"`
	BranchName string `json:"branchName"`
	CommitSHA  string `json:"commitSha"`
	Status     string `json:"status"`
}

type gitServiceRequest struct {
	Files    []GitServiceFile   `json:"files"`
	Metadata GitServiceMetadata `json:"metadata"`
}

type gitServiceError struct {
	Status string `json:"status"`
	Msg    string `json:"msg"`
}

type GitServiceQuickstartEntry struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
}

type GitServiceListQuickstartsResponse struct {
	Quickstarts []GitServiceQuickstartEntry `json:"quickstarts"`
	HeadSHA     string                      `json:"headSha"`
}

type GitServiceQuickstartContentResponse struct {
	Name    string           `json:"name"`
	Files   []GitServiceFile `json:"files"`
	HeadSHA string           `json:"headSha"`
}

type GitServiceCreatorPREntry struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	HTMLURL    string `json:"htmlUrl"`
	UpdatedAt  string `json:"updatedAt"`
	BranchName string `json:"branchName"`
	Slug       string `json:"slug"`
}

type GitServiceListCreatorPRsResponse struct {
	PullRequests []GitServiceCreatorPREntry `json:"pullRequests"`
}

type GitServiceCreatorPRContentResponse struct {
	Number     int              `json:"number"`
	Title      string           `json:"title"`
	HTMLURL    string           `json:"htmlUrl"`
	BranchName string           `json:"branchName"`
	Slug       string           `json:"slug"`
	Files      []GitServiceFile `json:"files"`
}

type GitServiceBatchMetadata struct {
	PRNumber        int    `json:"prNumber,omitempty"`
	ExpectedHeadSHA string `json:"expectedHeadSha,omitempty"`
	BranchName      string `json:"branchName,omitempty"`
	BaseSHA         string `json:"baseSha,omitempty"`
	CommitMessage   string `json:"commitMessage"`
	PRTitle         string `json:"prTitle,omitempty"`
	PRBody          string `json:"prBody,omitempty"`
	UserEmail       string `json:"userEmail,omitempty"`
}

type GitServiceBatchResource struct {
	Name   string           `json:"name"`
	Create bool             `json:"create,omitempty"`
	Files  []GitServiceFile `json:"files"`
}

type GitServiceBatchRequest struct {
	Metadata  GitServiceBatchMetadata   `json:"metadata"`
	Resources []GitServiceBatchResource `json:"resources"`
}

type GitServiceBatchResponse struct {
	PRURL      string   `json:"prUrl"`
	PRNumber   int      `json:"prNumber"`
	BranchName string   `json:"branchName"`
	HeadSHA    string   `json:"headSha"`
	Status     string   `json:"status"`
	Paths      []string `json:"paths"`
}

type GitServiceResourceFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type GitServiceWorkspaceResource struct {
	Name  string                   `json:"name"`
	Path  string                   `json:"path"`
	Files []GitServiceResourceFile `json:"files"`
}

type GitServiceCreatorPRWorkspaceResponse struct {
	Number     int                           `json:"number"`
	Title      string                        `json:"title"`
	HTMLURL    string                        `json:"htmlUrl"`
	BranchName string                        `json:"branchName"`
	HeadSHA    string                        `json:"headSha"`
	Resources  []GitServiceWorkspaceResource `json:"resources"`
	Missing    []string                      `json:"missing"`
}

type GitServiceCreatorPRResourceResponse struct {
	HeadSHA  string                      `json:"headSha"`
	Resource GitServiceWorkspaceResource `json:"resource"`
}

type HTTPStatusError struct {
	StatusCode int
	Msg        string
}

func (e *HTTPStatusError) Error() string {
	if e.Msg != "" {
		return fmt.Sprintf("git-service error (%d): %s", e.StatusCode, e.Msg)
	}
	return fmt.Sprintf("git-service returned status %d", e.StatusCode)
}

func (c *GitService) ListQuickstarts(ctx context.Context) (*GitServiceListQuickstartsResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/list-quickstarts", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.pskToken != "" {
		req.Header.Set("X-PSK-Token", c.pskToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("git-service request failed: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseSize = 5 * 1024 * 1024 // 5MB
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err == nil && int64(len(body)) > maxResponseSize {
		return nil, fmt.Errorf("git-service response exceeded %d bytes", maxResponseSize)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read git-service response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp gitServiceError
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Msg != "" {
			return nil, fmt.Errorf("git-service error (%d): %s", resp.StatusCode, errResp.Msg)
		}
		return nil, fmt.Errorf("git-service returned status %d", resp.StatusCode)
	}

	var result GitServiceListQuickstartsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode git-service response: %w", err)
	}
	return &result, nil
}

func (c *GitService) GetQuickstartContent(ctx context.Context, name string) (*GitServiceQuickstartContentResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/quickstart-content/"+url.PathEscape(name), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if c.pskToken != "" {
		req.Header.Set("X-PSK-Token", c.pskToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("git-service request failed: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseSize = 5 * 1024 * 1024 // 5MB
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err == nil && int64(len(body)) > maxResponseSize {
		return nil, fmt.Errorf("git-service response exceeded %d bytes", maxResponseSize)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read git-service response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp gitServiceError
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Msg != "" {
			return nil, fmt.Errorf("git-service error (%d): %s", resp.StatusCode, errResp.Msg)
		}
		return nil, fmt.Errorf("git-service returned status %d", resp.StatusCode)
	}

	var result GitServiceQuickstartContentResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode git-service response: %w", err)
	}
	return &result, nil
}

func (c *GitService) SubmitPR(ctx context.Context, files []GitServiceFile, metadata GitServiceMetadata) (*GitServiceResponse, error) {
	reqBody := gitServiceRequest{
		Files:    files,
		Metadata: metadata,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/submit-pr", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.pskToken != "" {
		req.Header.Set("X-PSK-Token", c.pskToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("git-service request failed: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseSize = 5 * 1024 * 1024 // 5MB
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err == nil && int64(len(body)) > maxResponseSize {
		return nil, fmt.Errorf("git-service response exceeded %d bytes", maxResponseSize)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read git-service response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp gitServiceError
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Msg != "" {
			return nil, fmt.Errorf("git-service error (%d): %s", resp.StatusCode, errResp.Msg)
		}
		return nil, fmt.Errorf("git-service returned status %d", resp.StatusCode)
	}

	var result GitServiceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode git-service response: %w", err)
	}

	return &result, nil
}

func (c *GitService) ListCreatorPRs(ctx context.Context) (*GitServiceListCreatorPRsResponse, error) {
	var result GitServiceListCreatorPRsResponse
	if err := c.getJSON(ctx, "/api/v1/creator-prs", &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *GitService) GetCreatorPR(ctx context.Context, number int) (*GitServiceCreatorPRContentResponse, error) {
	var result GitServiceCreatorPRContentResponse
	if err := c.getJSON(ctx, fmt.Sprintf("/api/v1/creator-prs/%d", number), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *GitService) GetCreatorPRWorkspace(ctx context.Context, number int) (*GitServiceCreatorPRWorkspaceResponse, error) {
	var result GitServiceCreatorPRWorkspaceResponse
	if err := c.getJSON(ctx, fmt.Sprintf("/api/v1/creator-prs/%d/workspace", number), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *GitService) GetCreatorPRResource(ctx context.Context, number int, name, headSHA string) (*GitServiceCreatorPRResourceResponse, error) {
	path := fmt.Sprintf("/api/v1/creator-prs/%d/resources/%s", number, url.PathEscape(name))
	if headSHA != "" {
		path += "?headSha=" + url.QueryEscape(headSHA)
	}
	var result GitServiceCreatorPRResourceResponse
	if err := c.getJSON(ctx, path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SubmitBatchPR returns an *HTTPStatusError for git-service rejections so the
// caller can pass conflicts and validation errors through to the browser.
func (c *GitService) SubmitBatchPR(ctx context.Context, batch GitServiceBatchRequest) (*GitServiceBatchResponse, error) {
	jsonBody, err := json.Marshal(batch)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	var result GitServiceBatchResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/submit-batch-pr", bytes.NewReader(jsonBody), &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *GitService) getJSON(ctx context.Context, path string, dest interface{}) error {
	return c.doJSON(ctx, http.MethodGet, path, nil, dest)
}

func (c *GitService) doJSON(ctx context.Context, method, path string, reqBody io.Reader, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.pskToken != "" {
		req.Header.Set("X-PSK-Token", c.pskToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("git-service request failed: %w", err)
	}
	defer resp.Body.Close()

	const maxResponseSize = 5 * 1024 * 1024
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)
	body, err := io.ReadAll(limitedReader)
	if err == nil && int64(len(body)) > maxResponseSize {
		return fmt.Errorf("git-service response exceeded %d bytes", maxResponseSize)
	}
	if err != nil {
		return fmt.Errorf("failed to read git-service response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp gitServiceError
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Msg != "" {
			return &HTTPStatusError{StatusCode: resp.StatusCode, Msg: errResp.Msg}
		}
		return &HTTPStatusError{StatusCode: resp.StatusCode}
	}

	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("failed to decode git-service response: %w", err)
	}
	return nil
}
