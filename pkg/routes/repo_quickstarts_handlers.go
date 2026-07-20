package routes

import (
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (s *ServerAdapter) GetRepoQuickstarts(w http.ResponseWriter, r *http.Request) {
	result, err := s.gitServiceClient.ListQuickstarts(r.Context())
	if err != nil {
		logrus.WithError(err).Error("git-service request failed")
		utils.ErrorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	quickstarts := make([]generated.RepoQuickstartEntry, len(result.Quickstarts))
	for i, qs := range result.Quickstarts {
		name := qs.Name
		displayName := qs.DisplayName
		quickstarts[i] = generated.RepoQuickstartEntry{
			Name:        &name,
			DisplayName: &displayName,
		}
	}

	utils.DataResponse(w, http.StatusOK, generated.ListRepoQuickstartsResponse{
		Quickstarts: &quickstarts,
	})
}

func (s *ServerAdapter) GetRepoQuickstartsName(w http.ResponseWriter, r *http.Request, name string) {
	result, err := s.gitServiceClient.GetQuickstartContent(r.Context(), name)
	if err != nil {
		logrus.WithError(err).Error("git-service request failed")
		utils.ErrorResponse(w, http.StatusBadGateway, err.Error())
		return
	}

	files := make([]generated.SubmitPrFile, len(result.Files))
	for i, f := range result.Files {
		files[i] = generated.SubmitPrFile{
			Name:    f.Name,
			Content: f.Content,
		}
	}

	qsName := result.Name
	utils.DataResponse(w, http.StatusOK, generated.QuickstartContentResponse{
		Name:  &qsName,
		Files: &files,
	})
}
