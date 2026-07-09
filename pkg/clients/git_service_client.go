package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

	body, err := io.ReadAll(resp.Body)
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
