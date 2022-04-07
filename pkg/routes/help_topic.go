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

func findHelpTopics(tagTypes []models.TagType, tagValues []string, names []string) ([]models.HelpTopic, error) {
	var helpTopic []models.HelpTopic
	var tagsArray []models.Tag
	query := database.DB

	if len(tagTypes) > 0 {
		query.Where("type IN ? AND value IN ?", tagTypes, tagValues).Find(&tagsArray)
		query = database.DB.Model(&tagsArray).Distinct("id, name, content")
	}

	if len(names) > 0 {
		query = query.Where("name IN ?", names)
	}

	var err error
	if len(tagTypes) > 0 {
		query.Association("HelpTopics").Find(&helpTopic)
	} else {
		query.Find(&helpTopic)
	}

	if err != nil {
		return helpTopic, err
	}

	return helpTopic, nil
}

func concatAppendTags(slices [][]string) []string {
	var tmp []string
	for _, s := range slices {
		tmp = append(tmp, s...)
	}
	return tmp
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

	nameQueries := r.URL.Query()["name"]
	if len(nameQueries) == 0 {
		nameQueries = r.URL.Query()["name[]"]
	}

	var err error
	if len(bundleQueries) > 0 {
		tagTypes = append(tagTypes, models.BundleTag)
	}
	if len(applicationQueries) > 0 {
		tagTypes = append(tagTypes, models.ApplicationTag)
	}

	if len(tagTypes) > 0 || len(nameQueries) > 0 {
		/**
		 * future proofing more than 2 tag queries
		 */
		tagQueries := make([][]string, 2)
		tagQueries[0] = bundleQueries
		tagQueries[1] = applicationQueries
		helpTopic, err = findHelpTopics(tagTypes, concatAppendTags(tagQueries), nameQueries)
	} else {
		database.DB.Find(&helpTopic)
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
	type tResp struct {
		Data []models.HelpTopic `json:"data"`
	}
	var resp tResp

	resp.Data = helpTopic
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
				resp := models.NotFound{
					Msg: err.Error(),
				}
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
