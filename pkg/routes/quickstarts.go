package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi"
)

func FindQuickstartById(id int) (models.Quickstart, error) {
	var quickStart models.Quickstart
	err := database.DB.First(&quickStart, id).Error
	return quickStart, err
}

func findQuickstartsByTags(tagTypes []models.TagType, tagValues []string, pagination Pagination) ([]models.Quickstart, error) {
	var quickstarts []models.Quickstart
	var tagsArray []models.Tag
	database.DB.Where("type IN ? AND value IN ?", tagTypes, tagValues).Find(&tagsArray)
	err := database.DB.Model(&tagsArray).Limit(pagination.Limit).Offset(pagination.Offset).Distinct("id, name, content").Association("Quickstarts").Find(&quickstarts)
	if err != nil {
		return quickstarts, err
	}

	return quickstarts, nil
}

func GetAllQuickstarts(w http.ResponseWriter, r *http.Request) {
	var quickStarts []models.Quickstart
	var tagTypes []models.TagType

	// first try bundle query param
	bundleQueries := r.URL.Query()["bundle"]
	if len(bundleQueries) == 0 {
		// if empty try bundle[] queries
		bundleQueries = r.URL.Query()["bundle[]"]
	}

	applicationQueries := r.URL.Query()["application"]
	if len(applicationQueries) == 0 {
		applicationQueries = r.URL.Query()["application[]"]
	}

	if len(bundleQueries) > 0 {
		tagTypes = append(tagTypes, models.BundleTag)
	}
	if len(applicationQueries) > 0 {
		tagTypes = append(tagTypes, models.ApplicationTag)
	}

	pagination := r.Context().Value(PaginationContextKey).(Pagination)
	var err error

	if len(tagTypes) > 0 {
		quickStarts, err = findQuickstartsByTags(tagTypes, append(bundleQueries, applicationQueries...), pagination)
	} else {
		database.DB.Limit(pagination.Limit).Offset(pagination.Offset).Find(&quickStarts)
	}

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

// MakeQuickstartsRouter creates a router handles for /quickstarts group
func MakeQuickstartsRouter(sub chi.Router) {
	sub.Use(PaginationContext)
	sub.Get("/", GetAllQuickstarts)
	sub.Route("/{id}", func(r chi.Router) {
		r.Use(QuickstartEntityContext)
		r.Get("/", GetQuickstartById)
	})
}
