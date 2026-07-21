package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
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
	DirectoryName string `json:"directoryName"`
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

const maxRequestSize = 10 << 20 // 10 MB

func (h *Handler) SubmitPR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	var req SubmitPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
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

	branchExisted := false
	if req.Metadata.IsUpdate {
		if err := h.repoMgr.CheckoutExistingBranch(req.Metadata.BranchName); err != nil {
			logrus.WithError(err).Info("Existing branch not found, creating new branch for update")
			if err := h.repoMgr.CreateBranch(req.Metadata.BranchName); err != nil {
				logrus.WithError(err).Error("Failed to create branch")
				writeError(w, http.StatusInternalServerError, "failed to create branch")
				return
			}
		} else {
			branchExisted = true
		}
	} else {
		if err := h.repoMgr.CreateBranch(req.Metadata.BranchName); err != nil {
			logrus.WithError(err).Error("Failed to create branch")
			writeError(w, http.StatusInternalServerError, "failed to create branch")
			return
		}
	}

	dir := req.Metadata.ExistingPath
	if !req.Metadata.IsUpdate {
		dirName := req.Metadata.DirectoryName
		if dirName == "" {
			dirName = req.Metadata.BranchName
		}
		dir = h.quickstartsDirPath + dirName + "/"
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

	sha, err := h.repoMgr.CommitChanges(req.Metadata.CommitMessage, "nacho-bot", "crc-nachobot@redhat.com")
	if err != nil {
		logrus.WithError(err).Error("Failed to commit")
		h.cleanup(req.Metadata.BranchName)
		writeError(w, http.StatusInternalServerError, "failed to commit changes")
		return
	}

	if req.Metadata.IsUpdate && branchExisted {
		if err := h.repoMgr.PushBranchForce(req.Metadata.BranchName); err != nil {
			logrus.WithError(err).Error("Failed to push update")
			h.cleanup(req.Metadata.BranchName)
			writeError(w, http.StatusInternalServerError, "failed to push branch")
			return
		}

		h.cleanup(req.Metadata.BranchName)
		json.NewEncoder(w).Encode(SubmitPRResponse{
			BranchName: req.Metadata.BranchName,
			CommitSHA:  sha,
			Status:     "updated",
		})
	} else {
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

		json.NewEncoder(w).Encode(SubmitPRResponse{
			PRURL:      prURL,
			BranchName: req.Metadata.BranchName,
			CommitSHA:  sha,
			Status:     "created",
		})
	}
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
	if m.IsUpdate && m.ExistingPath == "" {
		return fmt.Errorf("existingPath is required when isUpdate is true")
	}
	if containsTraversal(m.BranchName) {
		return fmt.Errorf("branchName contains invalid path segment")
	}
	if m.ExistingPath != "" && containsTraversal(m.ExistingPath) {
		return fmt.Errorf("existingPath contains invalid path segment")
	}
	if m.DirectoryName != "" && containsTraversal(m.DirectoryName) {
		return fmt.Errorf("directoryName contains invalid path segment")
	}
	for _, f := range req.Files {
		if containsTraversal(f.Name) {
			return fmt.Errorf("file name contains invalid path segment: %s", f.Name)
		}
	}
	return nil
}

func containsTraversal(path string) bool {
	for _, part := range strings.Split(filepath.ToSlash(path), "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "error",
		"msg":    msg,
	})
}
