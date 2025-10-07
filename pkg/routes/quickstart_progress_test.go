package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"gorm.io/datatypes"
)

func mockQuickstartProgress(id uint) *models.QuickstartProgress {
	var quickstartProgress models.QuickstartProgress

	quickstartProgress.ID = id
	quickstartProgress.QuickstartName = strconv.Itoa(1234 + int(id))
	quickstartProgress.AccountId = 4321

	database.DB.Create(&quickstartProgress)

	return &quickstartProgress
}

func mockQuickstartProgressWithSpecificName(id uint, qsName string) *models.QuickstartProgress {
	var quickstartProgress models.QuickstartProgress

	quickstartProgress.ID = id
	quickstartProgress.QuickstartName = qsName
	quickstartProgress.AccountId = 4321

	database.DB.Create(&quickstartProgress)

	return &quickstartProgress
}

func setupQuickstartProgressRouter() *chi.Mux {
	r := chi.NewRouter()

	adapter := NewServerAdapter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		params := generated.GetProgressParams{}

		// Parse query parameters
		query := r.URL.Query()
		if account := query.Get("account"); account != "" {
			params.Account = account
		}
		if quickstart := query.Get("quickstart"); quickstart != "" {
			params.Quickstart = &quickstart
		}

		adapter.GetProgress(w, r, params)
	})

	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		adapter.PostProgress(w, r)
	})

	r.Delete("/{id}", func(w http.ResponseWriter, r *http.Request) {
		idStr := chi.URLParam(r, "id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		adapter.DeleteProgressId(w, r, id)
	})

	return r
}

func TestGetAllQuickstartProgresses(t *testing.T) {
	router := setupQuickstartProgressRouter()

	type responseSinglePayload struct {
		Data models.QuickstartProgress
	}

	qp1 := mockQuickstartProgress(1)
	qp2 := mockQuickstartProgress(2)
	qp3 := mockQuickstartProgressWithSpecificName(3, "TestingQS")

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
		assert.Equal(t, 3, len(payload.Data))
		assert.Equal(t, qp1.AccountId, payload.Data[0].AccountId)
		assert.Equal(t, qp1.QuickstartName, payload.Data[0].QuickstartName)
		assert.Equal(t, qp2.AccountId, payload.Data[1].AccountId)
		assert.Equal(t, qp2.QuickstartName, payload.Data[1].QuickstartName)
	})

	t.Run("Returns all quickstart-progress for specific AccountID", func(t *testing.T) {
		qpActID := fmt.Sprint(qp1.AccountId)
		request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/?account=%s", qpActID), nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		type responsePayload struct {
			Data []models.QuickstartProgress
		}
		var payload *responsePayload
		fmt.Println("response.Body:", response.Body)

		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 3, len(payload.Data))
	})

	t.Run("Returns quickstart-progress for specific matching quickstart name", func(t *testing.T) {
		qsName := qp3.QuickstartName
		request, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/?quickstart=%s", qsName), nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responseSinglePayload

		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
	})
}

func TestUpdateQuickstartsProgress(t *testing.T) {
	router := setupQuickstartProgressRouter()
	qp1 := mockQuickstartProgress(11)
	type responsePayload struct {
		Data models.QuickstartProgress
	}

	t.Run("should return bad request if no accountId or quickstartName was provided", func(t *testing.T) {
		jsonParams := `{"progress": { "foo": "bar" }}`
		request, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(string(jsonParams)))
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		var payload *MessageResponsePayload

		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 400, response.Code)
		assert.Equal(t, "Bad request! Missing accountId or quickstartName.", payload.Msg)
	})

	t.Run("should create new entity", func(t *testing.T) {
		jsonParams := `{"accountId": 666, "quickstartName": "foo-bar", "progress": { "foo": "bar" }}`
		request, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(string(jsonParams)))
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		var payload *responsePayload

		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 666, payload.Data.AccountId)
		assert.Equal(t, "foo-bar", payload.Data.QuickstartName)

		err := database.DB.Where(&payload.Data).Error
		assert.Equal(t, err, nil)
	})

	t.Run("should update existing entity", func(t *testing.T) {
		var tempProgress *datatypes.JSON
		json.Unmarshal([]byte(`{"bar": "barz"}`), &tempProgress)

		qp1.Progress = tempProgress
		jsonParams := `{"accountId": 666, "quickstartName": "foo-bar", "progress": { "foo": "bar" }}`
		request, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(string(jsonParams)))
		response := httptest.NewRecorder()

		router.ServeHTTP(response, request)

		var payload *responsePayload

		dbLen := database.DB.Find(&models.QuickstartProgress{}).RowsAffected

		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 666, payload.Data.AccountId)
		assert.Equal(t, "foo-bar", payload.Data.QuickstartName)
		assert.Equal(t, dbLen, database.DB.Find(&models.QuickstartProgress{}).RowsAffected)

		err := database.DB.Where(&payload.Data).Error
		assert.Equal(t, err, nil)
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
