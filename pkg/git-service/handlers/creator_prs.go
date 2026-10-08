package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	ghclient "github.com/RedHatInsights/quickstarts/pkg/git-service/github"
	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

type CreatorPREntry struct {
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	HTMLURL    string    `json:"htmlUrl"`
	UpdatedAt  time.Time `json:"updatedAt"`
	BranchName string    `json:"branchName"`
	Slug       string    `json:"slug"`
}

type ListCreatorPRsResponse struct {
	PullRequests []CreatorPREntry `json:"pullRequests"`
}

type CreatorPRContentResponse struct {
	Number     int    `json:"number"`
	Title      string `json:"title"`
	HTMLURL    string `json:"htmlUrl"`
	BranchName string `json:"branchName"`
	Slug       string `json:"slug"`
	Files      []File `json:"files"`
}

// ResourceFile is one file of a quickstart, keyed by its repository path so
// files with the same name in different quickstarts stay distinct.
type ResourceFile struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Content string `json:"content"`
}

type WorkspaceResource struct {
	Name  string         `json:"name"`
	Path  string         `json:"path"`
	Files []ResourceFile `json:"files"`
}

type CreatorPRWorkspaceResponse struct {
	Number     int                 `json:"number"`
	Title      string              `json:"title"`
	HTMLURL    string              `json:"htmlUrl"`
	BranchName string              `json:"branchName"`
	HeadSHA    string              `json:"headSha"`
	Resources  []WorkspaceResource `json:"resources"`
	Missing    []string            `json:"missing"`
}

type CreatorPRResourceResponse struct {
	HeadSHA  string            `json:"headSha"`
	Resource WorkspaceResource `json:"resource"`
}

func (h *Handler) ListCreatorPRs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	prs, err := h.gitHubClient.ListCreatorPRs(r.Context())
	if err != nil {
		logrus.WithError(err).Error("Failed to list creator pull requests")
		writeError(w, http.StatusBadGateway, "failed to list creator pull requests")
		return
	}

	entries := make([]CreatorPREntry, 0, len(prs))
	for _, pr := range prs {
		entries = append(entries, toEntry(pr))
	}

	json.NewEncoder(w).Encode(ListCreatorPRsResponse{PullRequests: entries})
}

func (h *Handler) GetCreatorPR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pr, ok := h.loadCreatorPR(w, r)
	if !ok {
		return
	}

	slug, files, err := h.gitHubClient.GetPRQuickstartFiles(r.Context(), pr)
	if errors.Is(err, ghclient.ErrMultiResource) {
		writeError(w, http.StatusConflict, "pull request changes more than one quickstart; open it with Edit multiple resources")
		return
	}
	if err != nil {
		logrus.WithError(err).WithField("pr", pr.Number).Error("Failed to get creator pull request files")
		writeError(w, http.StatusBadGateway, "failed to get creator pull request files")
		return
	}

	out := make([]File, len(files))
	for i, f := range files {
		out[i] = File{Name: f.Name, Content: f.Content}
	}

	json.NewEncoder(w).Encode(CreatorPRContentResponse{
		Number:     pr.Number,
		Title:      pr.Title,
		HTMLURL:    pr.HTMLURL,
		BranchName: pr.BranchName,
		Slug:       slug,
		Files:      out,
	})
}

// GetCreatorPRWorkspace returns every quickstart a creator PR touches, read in
// full at its head SHA, so a multi-resource session can resume from the branch.
func (h *Handler) GetCreatorPRWorkspace(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	pr, ok := h.loadCreatorPR(w, r)
	if !ok {
		return
	}

	ws, err := h.gitHubClient.GetPRWorkspace(r.Context(), pr)
	if err != nil {
		if errors.Is(err, ghclient.ErrIncompleteDiff) {
			writeError(w, http.StatusUnprocessableEntity, "pull request changes too many files to load as a workspace")
			return
		}
		logrus.WithError(err).WithField("pr", pr.Number).Error("Failed to get creator pull request workspace")
		writeError(w, http.StatusBadGateway, "failed to get creator pull request workspace")
		return
	}

	resources := make([]WorkspaceResource, len(ws.Resources))
	for i, res := range ws.Resources {
		resources[i] = toWorkspaceResource(res)
	}

	json.NewEncoder(w).Encode(CreatorPRWorkspaceResponse{
		Number:     pr.Number,
		Title:      pr.Title,
		HTMLURL:    pr.HTMLURL,
		BranchName: pr.BranchName,
		HeadSHA:    pr.HeadSHA,
		Resources:  resources,
		Missing:    ws.Missing,
	})
}

// GetCreatorPRResource reads one more quickstart at a creator PR's head SHA so
// it can be added to a resumed workspace without mixing in base-branch content.
func (h *Handler) GetCreatorPRResource(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := chi.URLParam(r, "name")
	if !isSingleSegment(name) {
		writeError(w, http.StatusBadRequest, "invalid quickstart name")
		return
	}

	pr, ok := h.loadCreatorPR(w, r)
	if !ok {
		return
	}

	// A caller that loaded the workspace at one head must not silently get a
	// file from a newer one; it has to reload so both are consistent.
	if want := r.URL.Query().Get("headSha"); want != "" && want != pr.HeadSHA {
		writeError(w, http.StatusConflict, "pull request has new commits; reload the workspace")
		return
	}

	res, err := h.gitHubClient.GetPRResource(r.Context(), pr, name)
	if err != nil {
		if errors.Is(err, ghclient.ErrResource) {
			writeError(w, http.StatusNotFound, "quickstart not found on the pull request branch")
			return
		}
		logrus.WithError(err).WithFields(logrus.Fields{"pr": pr.Number, "name": name}).Error("Failed to get creator pull request resource")
		writeError(w, http.StatusBadGateway, "failed to get quickstart from pull request")
		return
	}

	json.NewEncoder(w).Encode(CreatorPRResourceResponse{
		HeadSHA:  pr.HeadSHA,
		Resource: toWorkspaceResource(*res),
	})
}

// loadCreatorPR resolves the {number} URL parameter to an open creator PR,
// writing the error response itself when it cannot.
func (h *Handler) loadCreatorPR(w http.ResponseWriter, r *http.Request) (*ghclient.CreatorPR, bool) {
	number, err := strconv.Atoi(chi.URLParam(r, "number"))
	if err != nil || number <= 0 {
		writeError(w, http.StatusBadRequest, "invalid pull request number")
		return nil, false
	}

	pr, err := h.gitHubClient.GetCreatorPR(r.Context(), number)
	if err != nil {
		if errors.Is(err, ghclient.ErrNotFound) {
			writeError(w, http.StatusNotFound, "creator pull request not found")
			return nil, false
		}
		logrus.WithError(err).WithField("pr", number).Error("Failed to get creator pull request")
		writeError(w, http.StatusBadGateway, "failed to get creator pull request")
		return nil, false
	}
	return pr, true
}

func toWorkspaceResource(res ghclient.Resource) WorkspaceResource {
	files := make([]ResourceFile, len(res.Files))
	for i, f := range res.Files {
		files[i] = ResourceFile{Name: f.Name, Path: res.Dir + "/" + f.Name, Content: f.Content}
	}
	return WorkspaceResource{Name: res.Name, Path: res.Dir, Files: files}
}

func toEntry(pr ghclient.CreatorPR) CreatorPREntry {
	return CreatorPREntry{
		Number:     pr.Number,
		Title:      pr.Title,
		HTMLURL:    pr.HTMLURL,
		UpdatedAt:  pr.UpdatedAt,
		BranchName: pr.BranchName,
		Slug:       pr.Slug,
	}
}
