package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi"
	"github.com/stretchr/testify/assert"
)

func mockQuickstartProgress(id uint) *models.QuickstartProgress {
	var quickstartProgress models.QuickstartProgress

	quickstartProgress.ID = id
	quickstartProgress.Quickstart = 1234 + id
	quickstartProgress.AccountId = 4321

	database.DB.Create(&quickstartProgress)

	return &quickstartProgress
}

func setupQuickstartProgressRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", getAllQuickstartsProgress)
	r.Post("/", createQuickstartProgress)
	r.Delete("/{id}", deleteQuickstartProgress)
	return r
}

func TestGetAllQuickstartProgresses(t *testing.T) {
	router := setupQuickstartProgressRouter()

	qp1 := mockQuickstartProgress(1)
	qp2 := mockQuickstartProgress(2)

	t.Run("returns GET all quickstarts successfully", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		type responsePayload struct {
			Data []models.QuickstartProgress
		}

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
		assert.Equal(t, qp1.AccountId, payload.Data[0].AccountId)
		assert.Equal(t, qp1.Quickstart, payload.Data[0].Quickstart)
		assert.Equal(t, qp2.AccountId, payload.Data[1].AccountId)
		assert.Equal(t, qp2.Quickstart, payload.Data[1].Quickstart)
	})
}

func TestDeleteQuickstartProgress(t *testing.T) {
	router := setupQuickstartProgressRouter()

	mockQuickstartProgress(10)

	t.Run("deletes quickstart progress successfuly", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, "/10", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		type responsePayload struct {
			msg string
		}

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		var removedProgress models.QuickstartProgress
		err := database.DB.First(&removedProgress, 10).Error
		assert.Equal(t, "record not found", err.Error())
	})

	t.Run("return 404 if quickstart does not exists", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, "/666", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		type responsePayload struct {
			msg string
		}

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 404, response.Code)
	})

}
