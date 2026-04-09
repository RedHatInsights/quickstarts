package routes

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
	"github.com/sirupsen/logrus"
)

// PostGitPullRequest handles POST /git/pull-request
func (s *ServerAdapter) PostGitPullRequest(w http.ResponseWriter, r *http.Request) {
	var reqBody generated.GitPullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if reqBody.Title == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "title is required")
		return
	}

	// Validate files
	if err := s.gitService.ValidateFiles(reqBody.Files); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Create the pull request
	result, err := s.gitService.CreatePullRequest(reqBody)
	if err != nil {
		logrus.Errorf("git proxy error: %v", err)
		utils.ErrorResponse(w, http.StatusInternalServerError, "failed to create pull request: "+err.Error())
		return
	}

	resp := generated.GitPullRequestResponse{
		PrUrl:  &result.PRUrl,
		Branch: &result.Branch,
	}

	utils.DataResponse(w, http.StatusCreated, resp)
}
