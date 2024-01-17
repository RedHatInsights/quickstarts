package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

var accountTestId = "testAccountID"
var allTestIdFavorites []models.FavoriteQuickstart
var allFavorites []models.FavoriteQuickstart

func mockQuickstartFavoritable(qsName string) *models.FavoriteQuickstart {
	var qs models.Quickstart
	qs.Name = qsName

	favQuickstart := models.FavoriteQuickstart{
		AccountId:      accountTestId,
		QuickstartName: qs.Name,
		Favorite:       true,
	}
	qs.FavoriteQuickstart = append(qs.FavoriteQuickstart, favQuickstart)

	database.DB.Create(&qs)

	allFavorites = append(allFavorites, favQuickstart)
	allTestIdFavorites = append(allTestIdFavorites, favQuickstart)
	return &favQuickstart
}

func setupFavoriteQuickstartRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", GetAllFavoriteQuickstarts)
	r.Post("/", UpdateFavoriteQuickstart)
	return r
}

func TestGetAllFavoriteQuickstarts(t *testing.T) {
	router := setupFavoriteQuickstartRouter()
	mockQuickstartFavoritable("test-qs-1")
	mockQuickstartFavoritable("test-qs-2")

	type responseBody struct {
		QuickstartName string `json:"quickstartName"`
		Favorite       bool   `json:"favorite"`
	}

	type localResponsePayload struct {
		Data responseBody
	}

	type responsePayload struct {
		Data []responseBody
	}

	t.Run("should return all favorite quickstarts for specific AccountID", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/?account=%s", accountTestId), nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, len(allTestIdFavorites), len(payload.Data))
	})
	t.Run("should unfavorite exisitng favorite quickstart", func(t *testing.T) {
		jsonParams := `{"quickstartName": "test-qs-1", "favorite": false}`
		request, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/?account=%s", accountTestId), strings.NewReader(string(jsonParams)))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload localResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "test-qs-1", payload.Data.QuickstartName)
		assert.Equal(t, false, payload.Data.Favorite)
	})
	t.Run("should favorite the quickstart for the first time", func(t *testing.T) {
		var quickstart models.Quickstart
		quickstart.Name = "first-switch-qs"
		database.DB.Create(&quickstart)

		jsonParams := `{"quickstartName": "first-switch-qs", "favorite": true}`
		request, _ := http.NewRequest(http.MethodPost, fmt.Sprintf("/?account=%s", accountTestId), strings.NewReader(string(jsonParams)))
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		var payload localResponsePayload

		err := json.NewDecoder(response.Body).Decode(&payload)
		if err != nil {
			// Handle decoding error
			fmt.Println("Error decoding JSON:", err)
		}
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "first-switch-qs", payload.Data.QuickstartName)
		assert.Equal(t, true, payload.Data.Favorite)
	})
}
