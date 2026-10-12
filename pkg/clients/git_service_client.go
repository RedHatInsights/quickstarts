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
}

type GitServiceQuickstartContentResponse struct {
	Name  string           `json:"name"`
	Files []GitServiceFile `json:"files"`
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

func (c *GitService) getJSON(ctx context.Context, path string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
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
