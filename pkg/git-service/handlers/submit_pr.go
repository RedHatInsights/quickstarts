package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	gitops "github.com/RedHatInsights/quickstarts/pkg/git-service/git"
	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	"github.com/sirupsen/logrus"
)

type File struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type PRMetadata struct {
	BranchName    string `json:"branchName"`
	CommitMessage string `json:"commitMessage"`
	PRTitle       string `json:"prTitle"`
	PRBody        string `json:"prBody"`
	UserEmail     string `json:"userEmail"`
	IsUpdate      bool   `json:"isUpdate"`
	ExistingPath  string `json:"existingPath"`
}

type SubmitPRRequest struct {
	Files    []File     `json:"files"`
	Metadata PRMetadata `json:"metadata"`
}

type SubmitPRResponse struct {
	PRURL      string `json:"prUrl"`
	BranchName string `json:"branchName"`
	CommitSHA  string `json:"commitSha"`
	Status     string `json:"status"`
}

type Handler struct {
	repoMgr            gitops.RepoOperations
	gitHubClient       ghclient.PRCreator
	reviewersTeam      string
	quickstartsDirPath string
	mu                 sync.Mutex
}

func NewHandler(repoMgr gitops.RepoOperations, ghClient ghclient.PRCreator, reviewersTeam, quickstartsDirPath string) *Handler {
	return &Handler{
		repoMgr:            repoMgr,
		gitHubClient:       ghClient,
		reviewersTeam:      reviewersTeam,
		quickstartsDirPath: quickstartsDirPath,
	}
}

func (h *Handler) SubmitPR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req SubmitPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	if err := h.repoMgr.PullLatest(); err != nil {
		logrus.WithError(err).Error("Failed to pull latest")
		writeError(w, http.StatusInternalServerError, "failed to pull latest changes")
		return
	}

	if err := h.repoMgr.CreateBranch(req.Metadata.BranchName); err != nil {
		logrus.WithError(err).Error("Failed to create branch")
		writeError(w, http.StatusInternalServerError, "failed to create branch")
		return
	}

	dir := req.Metadata.ExistingPath
	if !req.Metadata.IsUpdate {
		dir = h.quickstartsDirPath + req.Metadata.BranchName + "/"
	}

	gitFiles := make([]gitops.File, len(req.Files))
	for i, f := range req.Files {
		gitFiles[i] = gitops.File{Name: f.Name, Content: f.Content}
	}

	if err := h.repoMgr.WriteFiles(dir, gitFiles); err != nil {
		logrus.WithError(err).Error("Failed to write files")
		h.cleanup(req.Metadata.BranchName)
		writeError(w, http.StatusInternalServerError, "failed to write files")
		return
	}

	sha, err := h.repoMgr.CommitChanges(req.Metadata.CommitMessage, "quickstarts-git-service", req.Metadata.UserEmail)
	if err != nil {
		logrus.WithError(err).Error("Failed to commit")
		h.cleanup(req.Metadata.BranchName)
		writeError(w, http.StatusInternalServerError, "failed to commit changes")
		return
	}

	if err := h.repoMgr.PushBranch(req.Metadata.BranchName); err != nil {
		logrus.WithError(err).Error("Failed to push")
		h.cleanup(req.Metadata.BranchName)
		writeError(w, http.StatusInternalServerError, "failed to push branch")
		return
	}

	body := req.Metadata.PRBody
	if req.Metadata.UserEmail != "" {
		body += fmt.Sprintf("\n\nSubmitted by: %s", req.Metadata.UserEmail)
	}

	prURL, prNumber, err := h.gitHubClient.CreatePullRequest(
		r.Context(),
		req.Metadata.PRTitle,
		body,
		req.Metadata.BranchName,
		h.repoMgr.GetBaseBranch(),
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to create PR")
		h.cleanup(req.Metadata.BranchName)
		writeError(w, http.StatusInternalServerError, "failed to create pull request")
		return
	}

	h.gitHubClient.AssignReviewers(r.Context(), prNumber, h.reviewersTeam)
	h.cleanup(req.Metadata.BranchName)

	status := "created"
	if req.Metadata.IsUpdate {
		status = "updated"
	}

	json.NewEncoder(w).Encode(SubmitPRResponse{
		PRURL:      prURL,
		BranchName: req.Metadata.BranchName,
		CommitSHA:  sha,
		Status:     status,
	})
}

func (h *Handler) cleanup(branch string) {
	if err := h.repoMgr.Cleanup(branch); err != nil {
		logrus.WithError(err).Warn("Branch cleanup failed")
	}
}

func validateRequest(req *SubmitPRRequest) error {
	if len(req.Files) == 0 {
		return fmt.Errorf("files are required")
	}
	m := req.Metadata
	if m.BranchName == "" {
		return fmt.Errorf("branchName is required")
	}
	if m.CommitMessage == "" {
		return fmt.Errorf("commitMessage is required")
	}
	if m.PRTitle == "" {
		return fmt.Errorf("prTitle is required")
	}
	if m.PRBody == "" {
		return fmt.Errorf("prBody is required")
	}
	if m.UserEmail == "" {
		return fmt.Errorf("userEmail is required")
	}
	if m.IsUpdate && m.ExistingPath == "" {
		return fmt.Errorf("existingPath is required when isUpdate is true")
	}
	return nil
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "error",
		"msg":    msg,
	})
}
