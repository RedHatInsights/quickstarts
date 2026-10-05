package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	gitops "github.com/RedHatInsights/quickstarts/pkg/git-service/git"
	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
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
	PRNumber      int    `json:"prNumber"`
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
			if req.Metadata.PRNumber > 0 {
				logrus.WithError(err).Error("Failed to checkout existing PR branch")
				writeError(w, http.StatusInternalServerError, "failed to checkout existing PR branch")
				return
			}
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
	} else if name := submittedQuickstartName(req.Files); name != "" {
		// A quickstart lives in docs/quickstarts/<name>/, so renaming one has to
		// move its directory. existingPath is the directory the branch already
		// has and cannot say where the quickstart belongs now, so the target
		// comes from the submitted metadata — the same name the content file is
		// named after. reconcileDirectory retires the old directory.
		dir = h.quickstartsDirPath + name + "/"
	}

	gitFiles := make([]gitops.File, len(req.Files))
	for i, f := range req.Files {
		gitFiles[i] = gitops.File{Name: f.Name, Content: f.Content}
	}

	if req.Metadata.IsUpdate {
		h.reconcileDirectory(req.Metadata.ExistingPath, dir, req.Files)
	}

	if err := h.repoMgr.WriteFiles(dir, gitFiles); err != nil {
		logrus.WithError(err).Error("Failed to write files")
		h.cleanup(req.Metadata.BranchName)
		writeError(w, http.StatusInternalServerError, "failed to write files")
		return
	}

	sha, err := h.repoMgr.CommitChanges(req.Metadata.CommitMessage, "nacho-bot", "crc-nachobot@redhat.com", dir, gitFiles)
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
			PRURL:      h.lookupPRURL(r.Context(), req.Metadata),
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

var (
	unsafeDirChars = regexp.MustCompile(`[^a-z0-9._-]`)
	repeatedDashes = regexp.MustCompile(`-{2,}`)
)

// submittedQuickstartName returns the directory name the submitted metadata
// asks for, or "" when the request does not carry a usable one.
//
// The creator derives the directory and the content file name from
// metadata.name, so that field is what the quickstart is actually called. It
// is read here rather than taken from the request metadata so a rename moves
// the directory for clients that still pin directoryName to the directory
// already on the branch.
func submittedQuickstartName(files []File) string {
	for _, f := range files {
		if f.Name != "metadata.yaml" && f.Name != "metadata.yml" {
			continue
		}
		var doc struct {
			Name string `yaml:"name"`
		}
		if err := yaml.Unmarshal([]byte(f.Content), &doc); err != nil {
			logrus.WithError(err).Info("Could not parse submitted metadata, keeping the existing directory")
			return ""
		}
		return sanitizeDirName(doc.Name)
	}
	return ""
}

// sanitizeDirName reduces a quickstart name to a single safe path segment,
// mirroring what the creator does before it sends one. Everything outside the
// allowed set collapses to a dash, which also means the result can never
// contain a separator or resolve to "." or "..".
func sanitizeDirName(name string) string {
	s := unsafeDirChars.ReplaceAllString(strings.ToLower(name), "-")
	s = repeatedDashes.ReplaceAllString(s, "-")
	return strings.Trim(s, "-.")
}

// reconcileDirectory retires whatever the update leaves behind in oldDir.
//
// Writing is purely additive, so without this a rename stacks the new state on
// top of the old: a renamed YAML file lands next to its predecessor, and a
// renamed quickstart leaves a whole abandoned directory. When newDir differs
// from oldDir the quickstart has moved and everything in oldDir is stale;
// otherwise only the files this update no longer sends are.
//
// Best effort: the submission carries the user's work, so a failure here is
// logged and the update proceeds. The worst case is the leftovers this exists
// to remove.
func (h *Handler) reconcileDirectory(oldDir, newDir string, files []File) {
	// existingPath comes from the request body and is otherwise only checked for
	// traversal segments, which "/docs/" satisfies. Deleting is destructive, so
	// confine it to a single quickstart directory.
	if !isQuickstartDir(oldDir, h.quickstartsDirPath) {
		logrus.WithField("dir", oldDir).Warn("Refusing to prune outside a quickstart directory")
		return
	}

	existing, err := h.repoMgr.ListFiles(oldDir)
	if err != nil {
		// Expected when an update targets a branch that does not have the
		// directory yet, so this is not treated as a failure.
		logrus.WithError(err).WithField("dir", oldDir).Info("Could not list existing files, skipping prune")
		return
	}

	stale := existing
	moved := !samePath(oldDir, newDir)
	if !moved {
		incoming := make(map[string]struct{}, len(files))
		for _, f := range files {
			incoming[f.Name] = struct{}{}
		}

		stale = nil
		for _, name := range existing {
			if _, ok := incoming[name]; !ok {
				stale = append(stale, name)
			}
		}
	}

	if len(stale) == 0 {
		return
	}

	if err := h.repoMgr.RemoveFiles(oldDir, stale); err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"dir":   oldDir,
			"files": stale,
		}).Warn("Failed to remove stale files, continuing")
		return
	}

	logrus.WithFields(logrus.Fields{
		"dir":    oldDir,
		"newDir": newDir,
		"files":  stale,
		"moved":  moved,
	}).Info("Pruned stale files from quickstart directory")
}

func (h *Handler) lookupPRURL(ctx context.Context, meta PRMetadata) string {
	if meta.PRNumber > 0 {
		pr, err := h.gitHubClient.GetCreatorPR(ctx, meta.PRNumber)
		if err == nil && pr != nil && pr.HTMLURL != "" {
			return pr.HTMLURL
		}
	}
	url, err := h.gitHubClient.FindPRURLByBranch(ctx, meta.BranchName)
	if err != nil {
		logrus.WithError(err).Warn("Failed to look up existing PR URL")
		return ""
	}
	return url
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
	// directoryName is concatenated onto the quickstarts directory to build the
	// write target, so it has to name exactly one directory beneath it.
	if m.DirectoryName != "" && (strings.ContainsAny(m.DirectoryName, `/\`) || m.DirectoryName == ".") {
		return fmt.Errorf("directoryName must be a single path segment")
	}
	for _, f := range req.Files {
		if containsTraversal(f.Name) {
			return fmt.Errorf("file name contains invalid path segment: %s", f.Name)
		}
	}
	return nil
}

// isQuickstartDir reports whether dir addresses one quickstart directory
// directly beneath base, tolerating leading and trailing slashes on either.
func isQuickstartDir(dir, base string) bool {
	b, d := trimSlashes(base), trimSlashes(dir)
	if b == "" || d == "" {
		return false
	}
	rest, ok := strings.CutPrefix(d, b+"/")
	return ok && rest != "" && !strings.Contains(rest, "/")
}

// samePath compares two repository paths that may differ only in their leading
// or trailing slashes, as existingPath and a path built from the configured
// quickstarts directory do.
func samePath(a, b string) bool {
	return trimSlashes(a) == trimSlashes(b)
}

func trimSlashes(s string) string {
	return strings.Trim(filepath.ToSlash(s), "/")
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
