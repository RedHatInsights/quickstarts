package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
)

var untaggedHelpTopic models.HelpTopic
var rhelHelpTopic models.HelpTopic
var rhelHelpTopicTag models.Tag

type helpTopicResponseBody struct {
	Id   uint   `json:"id"`
	Name string `json:"name"`
}

type helpTopicResponsePayload struct {
	Data []responseBody
}

type helpTopicSingleResponsePayload struct {
	Data responseBody
}

type helpTopicMessageResponsePayload struct {
	Msg string `json:"msg"`
}

func setupHelpTopicRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", GetAllHelpTopics)
	r.Route("/{name}", func(sub chi.Router) {
		sub.Use(HelpTopicEntityContext)
		sub.Get("/", GetHelpTopicByName)
	})
	return r
}

func setupHelpTopicTags() {
	rhelHelpTopicTag.Type = models.BundleTag
	rhelHelpTopicTag.Value = "rhel"
	database.DB.Create(&rhelHelpTopicTag)
}

func setupTaggedHelpTopics() {
	rhelHelpTopic.Name = "rhel-helpTopic"
	rhelHelpTopic.Content = []byte(`{"tags": "all-tags"}`)

	untaggedHelpTopic.Name = "untagged-helpTopic"
	untaggedHelpTopic.Content = []byte(`{"tags": "not-tags"}`)

	database.DB.Create(&rhelHelpTopic)
	database.DB.Model(&rhelHelpTopic).Association("Tags").Append(&rhelHelpTopicTag)
	database.DB.Save(&rhelHelpTopic)

	database.DB.Create(&untaggedHelpTopic)
}

func TestGetHelpTopic(t *testing.T) {
	router := setupHelpTopicRouter()
	setupHelpTopicTags()
	setupTaggedHelpTopics()

	t.Run("should get all helpTopics with 'rhel' bundle tag", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/?bundle[]=rhel", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 1, len(payload.Data))
		assert.Equal(t, "rhel-helpTopic", payload.Data[0].Name)
	})

	t.Run("should get all helpTopics if no tags given", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
	})

	t.Run("should get help topic by name", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/"+fmt.Sprint(rhelHelpTopic.Name), nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *singleResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, rhelHelpTopic.ID, payload.Data.Id)
		assert.Equal(t, rhelHelpTopic.Name, payload.Data.Name)
	})
}
