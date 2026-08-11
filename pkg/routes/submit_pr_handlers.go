package routes

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/clients"
	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
	"github.com/sirupsen/logrus"
)

// PostPullRequest handles POST /pull-request
func (s *ServerAdapter) PostPullRequest(w http.ResponseWriter, r *http.Request) {
	if !s.gitServiceEnabled {
		utils.ErrorResponse(w, http.StatusNotFound, "git-service is not available")
		return
	}

	const maxRequestSize = 5 * 1024 * 1024 // 5MB
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestSize)

	var reqBody generated.SubmitPrRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		if err.Error() == "http: request body too large" {
			utils.ErrorResponse(w, http.StatusRequestEntityTooLarge, "request body too large")
			return
		}
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if len(reqBody.Files) == 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "files are required")
		return
	}
	if reqBody.Metadata.BranchName == "" || reqBody.Metadata.CommitMessage == "" || reqBody.Metadata.PrTitle == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "branchName, commitMessage, and prTitle are required")
		return
	}

	files := make([]clients.GitServiceFile, len(reqBody.Files))
	for i, f := range reqBody.Files {
		files[i] = clients.GitServiceFile{
			Name:    f.Name,
			Content: f.Content,
		}
	}

	metadata := clients.GitServiceMetadata{
		BranchName:    reqBody.Metadata.BranchName,
		CommitMessage: reqBody.Metadata.CommitMessage,
		PRTitle:       reqBody.Metadata.PrTitle,
		PRBody:        derefString(reqBody.Metadata.PrBody),
		UserEmail:     derefString(reqBody.Metadata.UserEmail),
		IsUpdate:      derefBool(reqBody.Metadata.IsUpdate),
		ExistingPath:  derefString(reqBody.Metadata.ExistingPath),
		DirectoryName: derefString(reqBody.Metadata.DirectoryName),
	}

	result, err := s.gitServiceClient.SubmitPR(r.Context(), files, metadata)
	if err != nil {
		logrus.WithError(err).Error("git-service request failed")
		utils.ErrorResponse(w, http.StatusBadGateway, "git-service request failed")
		return
	}

	resp := generated.SubmitPrResponse{
		PrUrl:      &result.PRURL,
		BranchName: &result.BranchName,
		CommitSha:  &result.CommitSHA,
		Status:     &result.Status,
	}

	utils.DataResponse(w, http.StatusOK, resp)
}

func (s *ServerAdapter) GetRepoQuickstarts(w http.ResponseWriter, r *http.Request) {
	if !s.gitServiceEnabled {
		utils.ErrorResponse(w, http.StatusNotFound, "git-service is not available")
		return
	}
}

func (s *ServerAdapter) GetRepoQuickstartsName(w http.ResponseWriter, r *http.Request, name string) {
	if !s.gitServiceEnabled {
		utils.ErrorResponse(w, http.StatusNotFound, "git-service is not available")
		return
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}
