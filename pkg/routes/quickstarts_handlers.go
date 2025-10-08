package routes

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
)

// GetQuickstarts handles GET /quickstarts
func (s *ServerAdapter) GetQuickstarts(w http.ResponseWriter, r *http.Request, params generated.GetQuickstartsParams) {
	q := NewQuickstartsQuery(r, params)

	items, err := s.quickstartService.Find(
		q.TagTypes, q.TagValues,
		q.Name, q.DisplayName,
		q.Limit, q.Offset,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	resp := make([]generated.Quickstart, len(items))
	for i, it := range items {
		resp[i] = it.ToAPI()
	}
	utils.DataResponse(w, http.StatusOK, resp)
}

// GetQuickstartsId handles GET /quickstarts/{id}
func (s *ServerAdapter) GetQuickstartsId(w http.ResponseWriter, r *http.Request, id int) {
	// Find the quickstart by ID using service
	quickstart, err := s.quickstartService.FindById(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		w.Header().Set("Content-Type", "application/json")
		msg := err.Error()
		resp := generated.BadRequest{
			Msg: &msg,
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	// Convert to generated type and respond
	genQuickstart := quickstart.ToAPI()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := map[string]generated.Quickstart{"data": genQuickstart}
	json.NewEncoder(w).Encode(resp)
}

// GetQuickstartsFilters handles GET /quickstarts/filters
func (s *ServerAdapter) GetQuickstartsFilters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	// Use models directly since no generated filter types exist
	resp := map[string]models.FilterData{"data": models.FrontendFilters}
	json.NewEncoder(w).Encode(resp)
}