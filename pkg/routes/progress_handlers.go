package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/RedHatInsights/quickstarts/pkg/securitylog"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
	"gorm.io/datatypes"
)

// GetProgress handles GET /progress
func (s *ServerAdapter) GetProgress(w http.ResponseWriter, r *http.Request, params generated.GetProgressParams) {
	var progresses []models.QuickstartProgress
	var err error

	// Convert account string to int with explicit error handling
	var accountId *int
	if params.Account != "" {
		if accountVal, parseErr := strconv.Atoi(params.Account); parseErr != nil {
			utils.ErrorResponse(w, http.StatusBadRequest, "Invalid account ID: must be an integer")
			return
		} else {
			accountId = &accountVal
		}
	}

	// If both account and quickstart filters are provided, or if neither are provided,
	// use the filtered search. If only one is provided, use it as a filter.
	if accountId != nil || params.Quickstart != nil {
		progresses, err = s.progressService.GetProgress(accountId, params.Quickstart)
	} else {
		progresses, err = s.progressService.GetAllProgress()
	}

	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert to generated types and respond
	genProgresses := make([]generated.QuickstartProgress, len(progresses))
	for i, progress := range progresses {
		genProgresses[i] = progress.ToAPI()
	}

	utils.DataResponse(w, http.StatusOK, genProgresses)
}

// PostProgress handles POST /progress
func (s *ServerAdapter) PostProgress(w http.ResponseWriter, r *http.Request) {
	// Parse request body — parse/validation failures are not security-relevant
	// (malformed client requests, not data access attempts) so no security log here.
	var reqBody generated.QuickstartProgressRequest
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Validate required fields
	if reqBody.AccountId == 0 || reqBody.QuickstartName == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Bad request! Missing accountId or quickstartName.")
		return
	}

	// Convert progress data to JSONB format
	var progressData *datatypes.JSON
	if reqBody.Progress != nil {
		progressJSON, err := json.Marshal(*reqBody.Progress)
		if err != nil {
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
		progressData = &datatypes.JSON{}
		if err := progressData.UnmarshalJSON(progressJSON); err != nil {
			utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
			return
		}
	}

	// Use service to update progress
	resourceID := fmt.Sprintf("%d/%s", reqBody.AccountId, reqBody.QuickstartName)
	progress, err := s.progressService.UpdateProgress(reqBody.AccountId, reqBody.QuickstartName, progressData)
	if err != nil {
		securitylog.LogWithReason(r.Context(), "UPDATE", "progress", resourceID, "failure", err.Error())
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	securitylog.Log(r.Context(), "UPDATE", "progress", resourceID, "success")

	// Convert to generated type and respond
	genProgress := progress.ToAPI()
	utils.DataResponse(w, http.StatusOK, genProgress)
}

// DeleteProgressId handles DELETE /progress/{id}
func (s *ServerAdapter) DeleteProgressId(w http.ResponseWriter, r *http.Request, id int) {
	resourceID := strconv.Itoa(id)

	// Use service to delete progress
	err := s.progressService.DeleteProgress(id)
	if err != nil {
		securitylog.LogWithReason(r.Context(), "DELETE", "progress", resourceID, "failure", err.Error())
		utils.NotFoundResponse(w, "Progress record")
		return
	}

	securitylog.Log(r.Context(), "DELETE", "progress", resourceID, "success")

	// Success response
	utils.MessageResponse(w, http.StatusOK, "Quickstart progress successfully removed")
}