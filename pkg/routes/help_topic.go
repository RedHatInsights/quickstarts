package routes

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi"
)

func FindHelpTopicByName(name string) (models.HelpTopic, error) {
	var helpTopic models.HelpTopic
	err := database.DB.Where("name = ?", name).First(&helpTopic).Error
	return helpTopic, err
}

func findHelpTopicsByTags(tagTypes []models.TagType, tagValues []string) ([]models.HelpTopic, error) {
	var helpTopic []models.HelpTopic
	var tagsArray []models.Tag
	database.DB.Where("type IN ? AND value IN ?", tagTypes, tagValues).Find(&tagsArray)
	err := database.DB.Model(&tagsArray).Distinct("id, name, content").Association("HelpTopics").Find(&helpTopic)
	if err != nil {
		return helpTopic, err
	}

	return helpTopic, nil
}

func GetAllHelpTopics(w http.ResponseWriter, r *http.Request) {
	var helpTopic []models.HelpTopic
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

	var err error
	if len(bundleQueries) > 0 {
		tagTypes = append(tagTypes, models.BundleTag)
	}
	if len(applicationQueries) > 0 {
		tagTypes = append(tagTypes, models.ApplicationTag)
	}

	if len(tagTypes) > 0 {
		helpTopic, err = findHelpTopicsByTags(tagTypes, append(bundleQueries, applicationQueries...))
	} else {
		database.DB.Find(&helpTopic)
	}

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)

		resp["msg"] = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := make(map[string][]models.HelpTopic)
	resp["data"] = helpTopic
	json.NewEncoder(w).Encode(&resp)
}

func GetHelpTopicByName(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	resp := make(map[string]models.HelpTopic)
	resp["data"] = r.Context().Value("helpTopic").(models.HelpTopic)
	json.NewEncoder(w).Encode(resp)
}

func HelpTopicEntityContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if helpTopicName := chi.URLParam(r, "name"); helpTopicName != "" {
			helpTopicName, err := FindHelpTopicByName(helpTopicName)
			if err != nil {
				w.WriteHeader(http.StatusNotFound)
				w.Header().Set("Content-Type", "application/json")
				resp := make(map[string]string)
				resp["msg"] = err.Error()
				json.NewEncoder(w).Encode(resp)
				return
			}

			ctx := context.WithValue(r.Context(), "helpTopic", helpTopicName)
			next.ServeHTTP(w, r.WithContext(ctx))
		}

	})
}

// MakeHelpTopicsRouter creates a router handles for /helptopics group
func MakeHelpTopicsRouter(sub chi.Router) {
	sub.Get("/", GetAllHelpTopics)
	sub.Route("/{name}", func(r chi.Router) {
		r.Use(HelpTopicEntityContext)
		r.Get("/", GetHelpTopicByName)
	})
}
