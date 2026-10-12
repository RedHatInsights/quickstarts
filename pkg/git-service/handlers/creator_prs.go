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

	raw := chi.URLParam(r, "number")
	number, err := strconv.Atoi(raw)
	if err != nil || number <= 0 {
		writeError(w, http.StatusBadRequest, "invalid pull request number")
		return
	}

	pr, err := h.gitHubClient.GetCreatorPR(r.Context(), number)
	if err != nil {
		if errors.Is(err, ghclient.ErrNotFound) {
			writeError(w, http.StatusNotFound, "creator pull request not found")
			return
		}
		logrus.WithError(err).WithField("pr", number).Error("Failed to get creator pull request")
		writeError(w, http.StatusBadGateway, "failed to get creator pull request")
		return
	}

	slug, files, err := h.gitHubClient.GetPRQuickstartFiles(r.Context(), pr)
	if err != nil {
		logrus.WithError(err).WithField("pr", number).Error("Failed to get creator pull request files")
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
