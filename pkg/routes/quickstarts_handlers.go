package routes

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/database"
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

// FuzzySearchResult represents a quickstart with its fuzzy search score
type FuzzySearchResult struct {
	models.Quickstart
	Distance int `json:"distance"`
}

// findQuickstartsByFuzzySearch performs fuzzy search using Levenshtein distance
func findQuickstartsByFuzzySearch(searchTerm string, maxDistance int, pagination Pagination) ([]FuzzySearchResult, error) {
	var results []FuzzySearchResult

	// Use raw SQL query with Levenshtein distance on spec.displayName only
	query := `
		SELECT id, created_at, updated_at, deleted_at, name, content, 
		       levenshtein(content->'spec'->>'displayName', $1) as distance
		FROM quickstarts 
		WHERE content->'spec'->>'displayName' IS NOT NULL 
		  AND levenshtein(content->'spec'->>'displayName', $2) <= $3
		ORDER BY distance ASC, content->'spec'->>'displayName' ASC
		LIMIT $4 OFFSET $5
	`

	type queryResult struct {
		ID        uint `gorm:"primarykey"`
		CreatedAt string
		UpdatedAt string
		DeletedAt *string
		Name      string
		Content   []byte
		Distance  int
	}

	var queryResults []queryResult
	err := database.DB.Raw(query, searchTerm, searchTerm, maxDistance, pagination.Limit, pagination.Offset).Scan(&queryResults).Error
	if err != nil {
		return results, err
	}

	for _, qr := range queryResults {
		result := FuzzySearchResult{
			Quickstart: models.Quickstart{
				BaseModel: models.BaseModel{
					ID: qr.ID,
				},
				Name:    qr.Name,
				Content: qr.Content,
			},
			Distance: qr.Distance,
		}
		results = append(results, result)
	}

	return results, nil
}

// GetQuickstartsFuzzySearch handles GET /quickstarts/fuzzy-search
func (s *ServerAdapter) GetQuickstartsFuzzySearch(w http.ResponseWriter, r *http.Request, params generated.GetQuickstartsFuzzySearchParams) {
	searchTerm := string(params.Q)

	// Validate search term
	if searchTerm == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Search term 'q' parameter is required")
		return
	}

	// Default max distance for typo tolerance
	maxDistance := 3
	if params.MaxDistance != nil {
		maxDistance = *params.MaxDistance
	}

	// Parse pagination parameters with defaults
	limit := 50
	offset := 0
	if params.Limit != nil {
		limit = *params.Limit
	}
	if params.Offset != nil {
		offset = *params.Offset
	}

	pagination := Pagination{
		Limit:  limit,
		Offset: offset,
	}

	results, err := findQuickstartsByFuzzySearch(searchTerm, maxDistance, pagination)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Header().Set("Content-Type", "application/json")
		msg := "Fuzzy search failed: " + err.Error()
		resp := generated.BadRequest{
			Msg: &msg,
		}
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := make(map[string]interface{})
	resp["data"] = results
	resp["search_term"] = searchTerm
	resp["max_distance"] = maxDistance
	json.NewEncoder(w).Encode(&resp)
}
