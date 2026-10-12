package routes

import (
	"errors"
	"net/http"
	"time"

	"github.com/RedHatInsights/quickstarts/pkg/clients"
	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (s *ServerAdapter) GetCreatorPrs(w http.ResponseWriter, r *http.Request) {
	if !s.gitServiceEnabled {
		utils.ErrorResponse(w, http.StatusNotFound, "git-service is not available")
		return
	}

	result, err := s.gitServiceClient.ListCreatorPRs(r.Context())
	if err != nil {
		writeGitServiceError(w, err)
		return
	}

	prs := make([]generated.CreatorPrEntry, len(result.PullRequests))
	for i, pr := range result.PullRequests {
		number := pr.Number
		title := pr.Title
		htmlURL := pr.HTMLURL
		branch := pr.BranchName
		slug := pr.Slug
		entry := generated.CreatorPrEntry{
			Number:     &number,
			Title:      &title,
			HtmlUrl:    &htmlURL,
			BranchName: &branch,
			Slug:       &slug,
		}
		if pr.UpdatedAt != "" {
			if t, parseErr := time.Parse(time.RFC3339, pr.UpdatedAt); parseErr == nil {
				entry.UpdatedAt = &t
			}
		}
		prs[i] = entry
	}

	utils.DataResponse(w, http.StatusOK, generated.ListCreatorPrsResponse{
		PullRequests: &prs,
	})
}

func (s *ServerAdapter) GetCreatorPrsNumber(w http.ResponseWriter, r *http.Request, number int) {
	if !s.gitServiceEnabled {
		utils.ErrorResponse(w, http.StatusNotFound, "git-service is not available")
		return
	}

	result, err := s.gitServiceClient.GetCreatorPR(r.Context(), number)
	if err != nil {
		writeGitServiceError(w, err)
		return
	}

	files := make([]generated.SubmitPrFile, len(result.Files))
	for i, f := range result.Files {
		files[i] = generated.SubmitPrFile{
			Name:    f.Name,
			Content: f.Content,
		}
	}

	n := result.Number
	title := result.Title
	htmlURL := result.HTMLURL
	branch := result.BranchName
	slug := result.Slug
	utils.DataResponse(w, http.StatusOK, generated.CreatorPrContentResponse{
		Number:     &n,
		Title:      &title,
		HtmlUrl:    &htmlURL,
		BranchName: &branch,
		Slug:       &slug,
		Files:      &files,
	})
}

func writeGitServiceError(w http.ResponseWriter, err error) {
	var statusErr *clients.HTTPStatusError
	if errors.As(err, &statusErr) && statusErr.StatusCode == http.StatusNotFound {
		msg := statusErr.Msg
		if msg == "" {
			msg = "not found"
		}
		utils.ErrorResponse(w, http.StatusNotFound, msg)
		return
	}
	logrus.WithError(err).Error("git-service request failed")
	utils.ErrorResponse(w, http.StatusBadGateway, "git-service request failed")
}
