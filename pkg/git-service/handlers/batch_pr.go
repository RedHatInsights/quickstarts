package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"slices"
	"strings"

	gitops "github.com/RedHatInsights/quickstarts/pkg/git-service/git"
	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// BatchResource is the set of changed files for one existing quickstart.
// Files that are not sent are left exactly as they are on the branch.
type BatchResource struct {
	Name  string `json:"name"`
	Files []File `json:"files"`
}

type BatchPRMetadata struct {
	// PRNumber selects an existing creator PR to add a commit to. Zero creates
	// a new PR on BranchName.
	PRNumber int `json:"prNumber"`
	// ExpectedHeadSHA is the PR head the caller's files were loaded from or
	// last saved to. An update is refused if the branch has moved since.
	ExpectedHeadSHA string `json:"expectedHeadSha"`
	BranchName      string `json:"branchName"`
	// BaseSHA is the base-branch commit the caller's files were read from. A
	// new branch starts there so later base-branch changes are not reverted.
	BaseSHA       string `json:"baseSha"`
	CommitMessage string `json:"commitMessage"`
	PRTitle       string `json:"prTitle"`
	PRBody        string `json:"prBody"`
	UserEmail     string `json:"userEmail"`
}

type BatchPRRequest struct {
	Metadata  BatchPRMetadata `json:"metadata"`
	Resources []BatchResource `json:"resources"`
}

type BatchPRResponse struct {
	PRURL      string   `json:"prUrl"`
	PRNumber   int      `json:"prNumber"`
	BranchName string   `json:"branchName"`
	HeadSHA    string   `json:"headSha"`
	Status     string   `json:"status"`
	Paths      []string `json:"paths"`
}

// batchError carries the HTTP status a failed batch step should be reported
// with, so the browser can tell a conflict it can resolve from a server fault.
type batchError struct {
	status int
	msg    string
}

func (e *batchError) Error() string { return e.msg }

func batchFail(status int, format string, args ...interface{}) *batchError {
	return &batchError{status: status, msg: fmt.Sprintf(format, args...)}
}

// SubmitBatchPR writes changed files across several quickstarts as one signed
// commit, either opening a new PR or appending to an existing one.
//
// Unlike SubmitPR this never prunes or moves directories: a batch is a set of
// edits to files that already exist, and anything not sent stays untouched.
func (h *Handler) SubmitBatchPR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)
	var req BatchPRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			writeError(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateBatchRequest(&req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	resp, err := h.submitBatch(r, &req)
	if err != nil {
		var be *batchError
		if errors.As(err, &be) {
			writeError(w, be.status, be.msg)
			return
		}
		logrus.WithError(err).Error("Batch submit failed")
		writeError(w, http.StatusInternalServerError, "failed to submit batch")
		return
	}

	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) submitBatch(r *http.Request, req *BatchPRRequest) (*BatchPRResponse, error) {
	ctx := r.Context()
	meta := req.Metadata

	if err := h.repoMgr.PullLatest(); err != nil {
		logrus.WithError(err).Error("Failed to pull latest")
		return nil, batchFail(http.StatusInternalServerError, "failed to pull latest changes")
	}

	isUpdate := meta.PRNumber > 0
	branch := meta.BranchName
	var pr *ghclient.CreatorPR

	if isUpdate {
		var err error
		pr, err = h.gitHubClient.GetCreatorPR(ctx, meta.PRNumber)
		if errors.Is(err, ghclient.ErrNotFound) {
			return nil, batchFail(http.StatusNotFound, "pull request #%d is not an open creator pull request", meta.PRNumber)
		}
		if err != nil {
			logrus.WithError(err).WithField("pr", meta.PRNumber).Error("Failed to get creator pull request")
			return nil, batchFail(http.StatusBadGateway, "failed to get pull request #%d", meta.PRNumber)
		}
		if pr.HeadSHA != meta.ExpectedHeadSHA {
			return nil, batchFail(http.StatusConflict, "pull request #%d has new commits; reload it before saving", meta.PRNumber)
		}
		branch = pr.BranchName
		if err := h.repoMgr.CheckoutExistingBranch(branch); err != nil {
			logrus.WithError(err).WithField("branch", branch).Error("Failed to checkout PR branch")
			return nil, batchFail(http.StatusInternalServerError, "failed to checkout pull request branch")
		}
	} else {
		// The branch name is the retry identity for a first save. If the push
		// went through but opening the PR did not, a retry with the same name
		// overwrites that orphaned branch and finishes the job; once a PR does
		// exist the caller has to continue it instead.
		_, existing, err := h.gitHubClient.FindPRByBranch(ctx, branch)
		if err == nil {
			return nil, batchFail(http.StatusConflict, "pull request #%d already exists for branch %s; continue it instead", existing, branch)
		}
		if !errors.Is(err, ghclient.ErrNotFound) {
			logrus.WithError(err).WithField("branch", branch).Error("Failed to look up existing PR")
			return nil, batchFail(http.StatusBadGateway, "failed to check for an existing pull request")
		}

		if meta.BaseSHA != "" {
			err = h.repoMgr.CreateBranchAt(branch, meta.BaseSHA)
			if errors.Is(err, gitops.ErrUnknownRevision) {
				return nil, batchFail(http.StatusConflict, "the content was loaded from a commit that is no longer available; reload it before saving")
			}
		} else {
			err = h.repoMgr.CreateBranch(branch)
		}
		if err != nil {
			logrus.WithError(err).WithField("branch", branch).Error("Failed to create branch")
			return nil, batchFail(http.StatusInternalServerError, "failed to create branch")
		}
	}
	defer h.cleanup(branch)

	// Re-check against the commit actually checked out: the PR lookup and the
	// fetch are separate calls, and another save can land in between.
	if isUpdate {
		head, err := h.repoMgr.HeadSHA()
		if err != nil {
			logrus.WithError(err).Error("Failed to resolve branch head")
			return nil, batchFail(http.StatusInternalServerError, "failed to resolve branch head")
		}
		if head != meta.ExpectedHeadSHA {
			return nil, batchFail(http.StatusConflict, "pull request #%d has new commits; reload it before saving", meta.PRNumber)
		}
	}

	paths, err := h.writeBatch(req.Resources)
	if err != nil {
		return nil, err
	}

	sha, err := h.repoMgr.CommitPaths(meta.CommitMessage, "nacho-bot", "crc-nachobot@redhat.com", paths)
	if errors.Is(err, gitops.ErrNoChanges) {
		return nil, batchFail(http.StatusBadRequest, "the submitted files match the branch; there is nothing to save")
	}
	if err != nil {
		logrus.WithError(err).Error("Failed to commit batch")
		return nil, batchFail(http.StatusInternalServerError, "failed to commit changes")
	}

	if isUpdate {
		if err := h.repoMgr.PushBranch(branch); err != nil {
			if errors.Is(err, gitops.ErrPushRejected) {
				return nil, batchFail(http.StatusConflict, "pull request #%d has new commits; reload it before saving", meta.PRNumber)
			}
			logrus.WithError(err).WithField("branch", branch).Error("Failed to push batch")
			return nil, batchFail(http.StatusBadGateway, "failed to push branch")
		}
		return &BatchPRResponse{
			PRURL:      pr.HTMLURL,
			PRNumber:   pr.Number,
			BranchName: branch,
			HeadSHA:    sha,
			Status:     "updated",
			Paths:      paths,
		}, nil
	}

	if err := h.repoMgr.PushBranchForce(branch); err != nil {
		logrus.WithError(err).WithField("branch", branch).Error("Failed to push batch")
		return nil, batchFail(http.StatusBadGateway, "failed to push branch")
	}

	body := meta.PRBody
	if body == "" {
		body = batchPRBody(req.Resources, h.quickstartsDirPath)
	}
	if meta.UserEmail != "" {
		body += fmt.Sprintf("\n\nSubmitted by: %s", meta.UserEmail)
	}

	prURL, prNumber, err := h.gitHubClient.CreatePullRequest(ctx, meta.PRTitle, body, branch, h.repoMgr.GetBaseBranch())
	if err != nil {
		logrus.WithError(err).WithField("branch", branch).Error("Failed to create PR")
		return nil, batchFail(http.StatusBadGateway, "changes were pushed to %s but the pull request could not be opened; save again to retry", branch)
	}
	h.gitHubClient.AssignReviewers(ctx, prNumber, h.reviewersTeam)

	return &BatchPRResponse{
		PRURL:      prURL,
		PRNumber:   prNumber,
		BranchName: branch,
		HeadSHA:    sha,
		Status:     "created",
		Paths:      paths,
	}, nil
}

// writeBatch writes every resource's files into the checked-out branch and
// returns their repository paths. Only files the quickstart already has may be
// written, so a batch can neither add stray files nor be mistaken for a rename.
func (h *Handler) writeBatch(resources []BatchResource) ([]string, error) {
	var paths []string
	for _, res := range resources {
		dir := h.quickstartsDirPath + res.Name + "/"
		existing, err := h.repoMgr.ListFiles(dir)
		if err != nil {
			return nil, batchFail(http.StatusBadRequest, "quickstart %s does not exist on the branch", res.Name)
		}

		files := make([]gitops.File, len(res.Files))
		for i, f := range res.Files {
			if !slices.Contains(existing, f.Name) {
				return nil, batchFail(http.StatusBadRequest, "%s%s does not exist on the branch; adding or renaming files is not supported", dir, f.Name)
			}
			files[i] = gitops.File{Name: f.Name, Content: f.Content}
			paths = append(paths, strings.TrimPrefix(dir, "/")+f.Name)
		}

		if err := h.repoMgr.WriteFiles(dir, files); err != nil {
			logrus.WithError(err).WithField("dir", dir).Error("Failed to write batch files")
			return nil, batchFail(http.StatusInternalServerError, "failed to write files")
		}
	}
	return paths, nil
}

func batchPRBody(resources []BatchResource, quickstartsDir string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Updating %d quickstarts via the Quickstarts Creator tool.\n\nDirectories:\n", len(resources))
	for _, res := range resources {
		fmt.Fprintf(&b, "- %s%s/\n", strings.TrimPrefix(quickstartsDir, "/"), res.Name)
	}
	return strings.TrimSuffix(b.String(), "\n")
}

// validateBatchRequest checks the whole batch before anything touches the
// worktree, and reports every invalid file at once rather than the first.
func validateBatchRequest(req *BatchPRRequest) error {
	m := req.Metadata
	if m.CommitMessage == "" {
		return fmt.Errorf("commitMessage is required")
	}
	if m.PRNumber < 0 {
		return fmt.Errorf("prNumber must not be negative")
	}
	if m.PRNumber > 0 {
		if m.ExpectedHeadSHA == "" {
			return fmt.Errorf("expectedHeadSha is required when prNumber is set")
		}
	} else {
		if m.PRTitle == "" {
			return fmt.Errorf("prTitle is required to open a pull request")
		}
		// The creator PR list recognises branches by these prefixes.
		if !strings.HasPrefix(m.BranchName, "qs-update-") && !strings.HasPrefix(m.BranchName, "qs-create-") {
			return fmt.Errorf("branchName must start with qs-update- or qs-create-")
		}
		if strings.Contains(m.BranchName, "..") || strings.ContainsAny(m.BranchName, " ~^:?*[\\") {
			return fmt.Errorf("branchName is not a valid branch name")
		}
	}

	if len(req.Resources) == 0 {
		return fmt.Errorf("resources are required")
	}

	var problems []string
	seen := map[string]struct{}{}
	for _, res := range req.Resources {
		if !isSingleSegment(res.Name) {
			return fmt.Errorf("invalid quickstart name %q", res.Name)
		}
		if _, dup := seen[res.Name]; dup {
			return fmt.Errorf("quickstart %s is listed more than once", res.Name)
		}
		seen[res.Name] = struct{}{}

		if len(res.Files) == 0 {
			return fmt.Errorf("quickstart %s has no files", res.Name)
		}
		names := map[string]struct{}{}
		for _, f := range res.Files {
			if !isSingleSegment(f.Name) {
				return fmt.Errorf("invalid file name %q in %s", f.Name, res.Name)
			}
			if ext := path.Ext(f.Name); ext != ".yml" && ext != ".yaml" {
				return fmt.Errorf("%s/%s: only YAML files can be edited", res.Name, f.Name)
			}
			if _, dup := names[f.Name]; dup {
				return fmt.Errorf("%s/%s is listed more than once", res.Name, f.Name)
			}
			names[f.Name] = struct{}{}

			var doc interface{}
			if err := yaml.Unmarshal([]byte(f.Content), &doc); err != nil {
				problems = append(problems, fmt.Sprintf("%s/%s: %v", res.Name, f.Name, err))
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid YAML: %s", strings.Join(problems, "; "))
	}
	return nil
}

// isSingleSegment reports whether name is usable as exactly one path segment.
func isSingleSegment(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, `/\`) && !strings.HasPrefix(name, ".")
}
