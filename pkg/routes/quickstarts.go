package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi/v5"
)

func FindQuickstartById(id int) (models.Quickstart, error) {
	var quickStart models.Quickstart
	err := database.DB.First(&quickStart, id).Error
	return quickStart, err
}

func findQuickstartsByDisplayName(displayName string, pagination Pagination) ([]models.Quickstart, error) {
	var quickStarts []models.Quickstart
	err := database.DB.
		Limit(pagination.Limit).
		Offset(pagination.Offset).
		Where("content->'spec'->>'displayName' ILIKE ?", "%"+displayName+"%").
		Find(&quickStarts).Error

	return quickStarts, err
}

func findQuickstartsByTagsAndDisplayName(tagTypes []models.TagType, tagValues [][]string, displayName string, pagination Pagination) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart

	// Main query
	query := database.DB.Model(&models.Quickstart{})

	// Apply the filter for each tag type
	for i, tagType := range tagTypes {
		values := tagValues[i]

		// Subquery to find distinct quickstart_ids based on tag filters
		subquery := database.DB.Table("quickstart_tags qt").
			Select("quickstart_id").
			Joins("JOIN tags t ON qt.tag_id = t.id").
			Where("t.type = ? AND t.value IN (?)", tagType, values)

		// Ensure the main query only includes quickstarts matching the tag filters
		query = query.Where("quickstarts.id IN (?)", subquery)
	}

	if displayName != "" {
		query = query.Where("content->'spec'->>'displayName' ILIKE ?", "%"+displayName+"%")
	}

	// Add pagination
	err := query.Limit(pagination.Limit).Offset(pagination.Offset).Find(&quickstarts).Error

	return quickstarts, err
}

func findQuickstarts(tagTypes []models.TagType, tagValues [][]string, name string, displayName string, pagination Pagination) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart
	var err error

	if name != "" {
		err = database.DB.Where("name = ?", name).Find(&quickstarts).Error
	} else if len(tagTypes) > 0 {
		quickstarts, err = findQuickstartsByTagsAndDisplayName(tagTypes, tagValues, displayName, pagination)
	} else if displayName != "" {
		quickstarts, err = findQuickstartsByDisplayName(displayName, pagination)
	} else {
		err = database.DB.Limit(pagination.Limit).Offset(pagination.Offset).Find(&quickstarts).Error
	}

	return quickstarts, err
}

// Helper function to get query values
func getQueryValues(r *http.Request, key models.TagType) []string {
	keyStr := string(key)
	values := r.URL.Query()[keyStr]
	if len(values) == 0 {
		values = r.URL.Query()[keyStr+"[]"]
	}
	return values
}

func GetAllQuickstarts(w http.ResponseWriter, r *http.Request) {
	var quickStarts []models.Quickstart
	var tagTypes []models.TagType
	var allTagTypes []models.TagType
	var tagValues [][]string

	var queryTagMap []struct {
		queries []string
		tagType models.TagType
	}

	quickstartName := ""
	nameQuery := r.URL.Query()["name"]
	if len(nameQuery) > 0 {
		// array name query is not required and supported
		quickstartName = nameQuery[0]
	}

	quickstartDisplayName := ""
	displayNameQuery := r.URL.Query()["display-name"]
	if len(displayNameQuery) > 0 {
		// array name query is not required and supported
		quickstartDisplayName = displayNameQuery[0]
	}

	allTagTypes = models.TagType.GetAllTags("")

	// List of query parameters and corresponding tag types
	for _, tagType := range allTagTypes {
		queryTagMap = append(queryTagMap, struct {
			queries []string
			tagType models.TagType
		}{
			queries: getQueryValues(r, tagType),
			tagType: tagType,
		})
	}

	for _, tag := range queryTagMap {
		if len(tag.queries) > 0 {
			tagTypes = append(tagTypes, tag.tagType)
			tagValues = append(tagValues, tag.queries)
		}
	}

	pagination := r.Context().Value(PaginationContextKey).(Pagination)
	var err error

	quickStarts, err = findQuickstarts(tagTypes, tagValues, quickstartName, quickstartDisplayName, pagination)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := models.BadRequest{
			Msg: err.Error(),
		}

		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := make(map[string][]models.Quickstart)
	resp["data"] = quickStarts
	json.NewEncoder(w).Encode(&resp)
}

func GetQuickstartById(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := make(map[string]models.Quickstart)
	resp["data"] = r.Context().Value("quickstart").(models.Quickstart)
	json.NewEncoder(w).Encode(resp)
}

func QuickstartEntityContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if quickstartId := chi.URLParam(r, "id"); quickstartId != "" {
			id, err := strconv.Atoi(quickstartId)
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				resp := models.BadRequest{
					Msg: err.Error(),
				}
				json.NewEncoder(w).Encode(resp)
				return
			}
			quickstart, err := FindQuickstartById(id)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				w.Header().Set("Content-Type", "application/json")
				resp := models.NotFound{
					Msg: err.Error(),
				}
				json.NewEncoder(w).Encode(resp)
				return
			}

			ctx := context.WithValue(r.Context(), "quickstart", quickstart)
			next.ServeHTTP(w, r.WithContext(ctx))
		}

	})
}

func GetFilters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := make(map[string]models.FilterData)
	resp["data"] = models.FrontendFilters
	json.NewEncoder(w).Encode(resp)
}

// MakeQuickstartsRouter creates a router handles for /quickstarts group
func MakeQuickstartsRouter(sub chi.Router) {
	sub.Use(PaginationContext)
	sub.Get("/", GetAllQuickstarts)
	sub.Get("/filters", GetFilters)
	sub.Route("/{id}", func(r chi.Router) {
		r.Use(QuickstartEntityContext)
		r.Get("/", GetQuickstartById)
	})
}
