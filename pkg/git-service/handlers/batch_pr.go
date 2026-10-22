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

// BatchResource is the set of changed files for one quickstart. Files that are
// not sent are left exactly as they are on the branch.
//
// Create marks a quickstart that does not exist yet. Its directory must be
// absent from the branch and it must be sent complete: a metadata file and a
// content file, both naming it. Creation is explicit so that a typo in an
// existing name is reported instead of quietly becoming a new directory.
//
// Revert takes a quickstart back out of an existing pull request: its
// directory is restored to how it was where the branch started, or deleted if
// the pull request added it. A revert sends no files.
//
// Delete removes an existing quickstart from the repository: every file in its
// directory is deleted by the commit. Unlike a revert it works on the first
// save of a new pull request too, and on quickstarts the pull request has not
// otherwise changed. A deletion sends no files.
//
// RenameFrom moves an existing quickstart to the directory Name. A quickstart
// lives in a directory named after its metadata.name, so changing that name
// moves it: like a creation it is sent complete (metadata file and content
// file, both naming Name), its old YAML files are deleted, and anything else
// in the old directory moves along unchanged.
type BatchResource struct {
	Name       string `json:"name"`
	Create     bool   `json:"create,omitempty"`
	Revert     bool   `json:"revert,omitempty"`
	Delete     bool   `json:"delete,omitempty"`
	RenameFrom string `json:"renameFrom,omitempty"`
	Files      []File `json:"files"`
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
// Unlike SubmitPR nothing is pruned or moved implicitly: a batch is a set of
// edits to files that already exist, plus explicitly marked new, renamed,
// reverted and deleted quickstarts, and anything not sent stays untouched.
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
// returns their repository paths. An edit may only write files the quickstart
// already has, so it can neither add stray files nor be mistaken for a rename;
// a creation or rename may only write into a directory that does not exist yet.
func (h *Handler) writeBatch(resources []BatchResource) ([]string, error) {
	var paths []string
	for _, res := range resources {
		dir := h.quickstartsDirPath + res.Name + "/"
		if res.Revert {
			reverted, err := h.revertResource(res.Name, dir)
			if err != nil {
				return nil, err
			}
			paths = append(paths, reverted...)
			continue
		}
		if res.Delete {
			if err := h.deleteResource(res.Name, dir); err != nil {
				return nil, err
			}
			continue
		}
		if res.RenameFrom != "" {
			moved, err := h.renameResource(res, dir)
			if err != nil {
				return nil, err
			}
			paths = append(paths, moved...)
			continue
		}
		existing, err := h.repoMgr.ListFiles(dir)
		if res.Create && err == nil {
			return nil, batchFail(http.StatusConflict, "quickstart %s already exists on the branch; choose another name", res.Name)
		}
		if !res.Create && err != nil {
			return nil, batchFail(http.StatusBadRequest, "quickstart %s does not exist on the branch", res.Name)
		}

		files := make([]gitops.File, len(res.Files))
		for i, f := range res.Files {
			if !res.Create && !slices.Contains(existing, f.Name) {
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

// renameResource moves the quickstart in res.RenameFrom to dir: the sent files
// replace its YAML files, and every other file moves across byte for byte.
func (h *Handler) renameResource(res BatchResource, dir string) ([]string, error) {
	oldDir := h.quickstartsDirPath + res.RenameFrom + "/"
	existing, err := h.repoMgr.ListFiles(oldDir)
	if err != nil {
		return nil, batchFail(http.StatusBadRequest, "quickstart %s does not exist on the branch", res.RenameFrom)
	}
	if _, err := h.repoMgr.ListFiles(dir); err == nil {
		return nil, batchFail(http.StatusConflict, "quickstart %s already exists on the branch; choose another name", res.Name)
	}

	var carried []gitops.File
	for _, name := range existing {
		if ext := path.Ext(name); ext == ".yml" || ext == ".yaml" {
			continue
		}
		content, err := h.repoMgr.ReadFile(oldDir + name)
		if err != nil {
			logrus.WithError(err).WithField("dir", oldDir).Error("Failed to read file to move")
			return nil, batchFail(http.StatusInternalServerError, "failed to read %s%s", oldDir, name)
		}
		carried = append(carried, gitops.File{Name: name, Content: content})
	}
	// A quickstart is one metadata file and one content file. Any further YAML
	// would be dropped by the move, so leave such a directory to a human.
	if yamlCount := len(existing) - len(carried); yamlCount > 2 {
		return nil, batchFail(http.StatusBadRequest, "%s has %d YAML files; it cannot be renamed here", res.RenameFrom, yamlCount)
	}

	if err := h.repoMgr.RemoveFiles(oldDir, existing); err != nil {
		logrus.WithError(err).WithField("dir", oldDir).Error("Failed to remove files for rename")
		return nil, batchFail(http.StatusInternalServerError, "failed to remove files")
	}

	files := carried
	for _, f := range res.Files {
		files = append(files, gitops.File{Name: f.Name, Content: f.Content})
	}
	if err := h.repoMgr.WriteFiles(dir, files); err != nil {
		logrus.WithError(err).WithField("dir", dir).Error("Failed to write files for rename")
		return nil, batchFail(http.StatusInternalServerError, "failed to write files")
	}

	paths := make([]string, len(files))
	for i, f := range files {
		paths[i] = strings.TrimPrefix(dir, "/") + f.Name
	}
	return paths, nil
}

// deleteResource stages the removal of every file in dir. Removals are staged
// directly rather than returned as paths, so the commit picks them up from the
// index. A quickstart is a flat directory; one with subdirectories is left to a
// human rather than half deleted.
func (h *Handler) deleteResource(name, dir string) error {
	existing, err := h.repoMgr.ListFiles(dir)
	if err != nil {
		return batchFail(http.StatusBadRequest, "quickstart %s does not exist on the branch", name)
	}
	if subdirs, err := h.repoMgr.ListDirectories(dir); err == nil && len(subdirs) > 0 {
		return batchFail(http.StatusBadRequest, "%s has subdirectories; it cannot be deleted here", name)
	}
	if err := h.repoMgr.RemoveFiles(dir, existing); err != nil {
		logrus.WithError(err).WithField("dir", dir).Error("Failed to remove files for delete")
		return batchFail(http.StatusInternalServerError, "failed to remove files")
	}
	return nil
}

// revertResource restores dir to its merge-base contents, staging removals of
// files the branch added, and returns the paths it wrote for committing.
func (h *Handler) revertResource(name, dir string) ([]string, error) {
	existing, listErr := h.repoMgr.ListFiles(dir)
	base, err := h.repoMgr.BaseFiles(dir)
	if errors.Is(err, gitops.ErrPathNotFound) {
		base = nil
	} else if err != nil {
		logrus.WithError(err).WithField("dir", dir).Error("Failed to read base files")
		return nil, batchFail(http.StatusInternalServerError, "failed to read %s from the base branch", name)
	}
	// Absent from both: already out of the PR. This happens when the old
	// directory of a renamed quickstart was itself added by the PR.
	if listErr != nil && base == nil {
		return nil, nil
	}

	var stale []string
	for _, f := range existing {
		if _, kept := base[f]; !kept {
			stale = append(stale, f)
		}
	}
	if err := h.repoMgr.RemoveFiles(dir, stale); err != nil {
		logrus.WithError(err).WithField("dir", dir).Error("Failed to remove files for revert")
		return nil, batchFail(http.StatusInternalServerError, "failed to remove files")
	}
	if len(base) == 0 {
		return nil, nil
	}

	names := make([]string, 0, len(base))
	for f := range base {
		names = append(names, f)
	}
	slices.Sort(names)
	files := make([]gitops.File, len(names))
	paths := make([]string, len(names))
	for i, f := range names {
		files[i] = gitops.File{Name: f, Content: base[f]}
		paths[i] = strings.TrimPrefix(dir, "/") + f
	}
	if err := h.repoMgr.WriteFiles(dir, files); err != nil {
		logrus.WithError(err).WithField("dir", dir).Error("Failed to write files for revert")
		return nil, batchFail(http.StatusInternalServerError, "failed to write files")
	}
	return paths, nil
}

func batchPRBody(resources []BatchResource, quickstartsDir string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Updating %d quickstarts via the Quickstarts Creator tool.\n\nDirectories:\n", len(resources))
	for _, res := range resources {
		suffix := ""
		if res.Create {
			suffix = " (new)"
		} else if res.RenameFrom != "" {
			suffix = fmt.Sprintf(" (renamed from %s)", res.RenameFrom)
		} else if res.Revert {
			suffix = " (removed from this PR)"
		} else if res.Delete {
			suffix = " (deleted)"
		}
		fmt.Fprintf(&b, "- %s%s/%s\n", strings.TrimPrefix(quickstartsDir, "/"), res.Name, suffix)
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

		if res.RenameFrom != "" {
			if !isSingleSegment(res.RenameFrom) {
				return fmt.Errorf("invalid quickstart name %q", res.RenameFrom)
			}
			if res.RenameFrom == res.Name {
				return fmt.Errorf("quickstart %s cannot be renamed to itself", res.Name)
			}
			if res.Create || res.Revert || res.Delete {
				return fmt.Errorf("quickstart %s: a rename cannot also create, revert or delete", res.Name)
			}
			// The old directory is part of this change too, so nothing else in
			// the batch may touch it.
			if _, dup := seen[res.RenameFrom]; dup {
				return fmt.Errorf("quickstart %s is listed more than once", res.RenameFrom)
			}
			seen[res.RenameFrom] = struct{}{}
		}

		if res.Delete {
			if res.Create || res.Revert || len(res.Files) > 0 {
				return fmt.Errorf("quickstart %s: a deletion cannot also create, revert or send files", res.Name)
			}
			continue
		}
		if res.Revert {
			if res.Create || len(res.Files) > 0 {
				return fmt.Errorf("quickstart %s: a revert cannot also create or send files", res.Name)
			}
			if m.PRNumber == 0 {
				return fmt.Errorf("quickstart %s: only a quickstart in an existing pull request can be reverted", res.Name)
			}
			continue
		}
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
		if res.Create || res.RenameFrom != "" {
			if err := validateNewResource(res); err != nil {
				problems = append(problems, err.Error())
			}
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("invalid YAML: %s", strings.Join(problems, "; "))
	}
	return nil
}

// validateNewResource checks a quickstart being created or renamed is
// complete: exactly a metadata file and a content file named after it, both
// declaring that name, which is what the seeder needs to load it.
func validateNewResource(res BatchResource) error {
	var metadata, content *File
	for i, f := range res.Files {
		switch f.Name {
		case "metadata.yml", "metadata.yaml":
			metadata = &res.Files[i]
		case res.Name + ".yml", res.Name + ".yaml":
			content = &res.Files[i]
		default:
			return fmt.Errorf("%s/%s: a new or renamed quickstart may only contain metadata.yml and %s.yml", res.Name, f.Name, res.Name)
		}
	}
	if metadata == nil || content == nil {
		return fmt.Errorf("%s: a new or renamed quickstart needs both metadata.yml and %s.yml", res.Name, res.Name)
	}

	var meta struct {
		Name string `yaml:"name"`
	}
	if err := yaml.Unmarshal([]byte(metadata.Content), &meta); err != nil || meta.Name != res.Name {
		return fmt.Errorf("%s/%s: name must be %q", res.Name, metadata.Name, res.Name)
	}
	var doc struct {
		Metadata struct {
			Name string `yaml:"name"`
		} `yaml:"metadata"`
	}
	if err := yaml.Unmarshal([]byte(content.Content), &doc); err != nil || doc.Metadata.Name != res.Name {
		return fmt.Errorf("%s/%s: metadata.name must be %q", res.Name, content.Name, res.Name)
	}
	return nil
}

// isSingleSegment reports whether name is usable as exactly one path segment.
func isSingleSegment(name string) bool {
	return name != "" && name != "." && name != ".." &&
		!strings.ContainsAny(name, `/\`) && !strings.HasPrefix(name, ".")
}
