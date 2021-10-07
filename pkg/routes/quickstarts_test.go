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
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

var quickstart models.Quickstart

func mockQuickstart() {
	quickstart.ID = 123
	quickstart.Title = "test title"
	database.DB.Create(&quickstart)
}

func mockBundleQuickstarts(bundles ...string) {
	quickstart.ID = 222
	quickstart.Title = "bundle title"
	quickstart.Bundles, _ = json.Marshal(bundles)
	database.DB.Create(quickstart)
	quickstart.ID = 333
	database.DB.Create(quickstart)
	fmt.Println(quickstart.Bundles)
}

type responseBody struct {
	Id    int    `json:"id"`
	Title string `json:"title"`
}

type responsePayload struct {
	Data []responseBody
}

type singleResponsePayload struct {
	Data responseBody
}

type messageResponsePayload struct {
	Msg string `json:"msg"`
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(QuickstartEntityContext())
	r.GET("/", GetAllQuickstarts)
	r.GET("/:id", GetQuickstartById)
	r.POST("/", CreateQuickstart)
	r.PATCH("/:id", UpdateQuickstartById)
	r.DELETE("/:id", DeleteQuickstartById)
	return r
}

func TestGetAll(t *testing.T) {
	router := setupRouter()
	mockQuickstart()
	mockBundleQuickstarts("foo", "bar")
	t.Run("returns GET all quickstarts successfully", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 3, len(payload.Data))
		assert.Equal(t, "test title", payload.Data[0].Title)
	})

	/**
	* Before we can query JSONS in SQLite, we need to use gorm build in JSON queries instead of RAW questions based on postgres
	 */
	t.Run("Should return an empty array when filtering non existing bundle", func(t *testing.T) {
		t.Skip()
		request, _ := http.NewRequest(http.MethodGet, "/?bundle=nonsense", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 0, len(payload.Data))
	})

	t.Run("Should an array of quickstarts within 'foo' bundle", func(t *testing.T) {
		t.Skip()
		request, _ := http.NewRequest(http.MethodGet, "/?bundle=foo", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 2, len(payload.Data))
	})
}

func TestGetOneById(t *testing.T) {
	router := setupRouter()
	mockQuickstart()
	t.Run("returns a quickstart object with ID 123", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/123", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var payload *singleResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 123, payload.Data.Id)
	})

	t.Run("returns 404 error response if record does not exists", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/999", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var payload *messageResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "record not found", payload.Msg)
	})

	t.Run("return 400 error response if bad request was sent", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/notanid", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var payload *messageResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 400, response.Code)
		assert.Equal(t, "strconv.Atoi: parsing \"notanid\": invalid syntax", payload.Msg)
	})
}

func TestCRUDRoutes(t *testing.T) {
	router := setupRouter()
	t.Run("should create new quickstart record", func(t *testing.T) {
		jsonParam := `{"title":"Create test title"}`
		request, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(string(jsonParam)))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *singleResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "Create test title", payload.Data.Title)

		var dbRecord models.Quickstart
		dbRecord, _ = FindQuickstartById(payload.Data.Id)
		assert.Equal(t, int(dbRecord.ID), payload.Data.Id)
	})

	t.Run("should return 400 response for invalid payload", func(t *testing.T) {
		jsonParam := `[]`
		request, _ := http.NewRequest(http.MethodPost, "/", strings.NewReader(string(jsonParam)))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *messageResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 400, response.Code)
		assert.Equal(t, "json: cannot unmarshal array into Go value of type models.Quickstart", payload.Msg)
	})

	t.Run("should delete existing record from database", func(t *testing.T) {
		var deleteQuickstart models.Quickstart
		deleteQuickstart.ID = 999999
		database.DB.Create(&deleteQuickstart)
		request, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/%d", deleteQuickstart.ID), nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *messageResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, "Quickstart successfully removed", payload.Msg)

		_, err := FindQuickstartById(999999)
		assert.Equal(t, "record not found", err.Error())
	})

	t.Run("should return 404 response when deleting non existing record", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/%d", 888), nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *messageResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "record not found", payload.Msg)
	})

	t.Run("should update existing record", func(t *testing.T) {
		var updatedQuickstart models.Quickstart
		updatedQuickstart.ID = 777
		database.DB.Create(&updatedQuickstart)
		jsonParam := `{"title":"Update test title"}`

		var dbRecord models.Quickstart
		dbRecord, _ = FindQuickstartById(777)
		assert.Equal(t, int(dbRecord.ID), 777)
		assert.Equal(t, dbRecord.Title, "")

		request, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/%d", updatedQuickstart.ID), strings.NewReader(string(jsonParam)))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *singleResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)

		dbRecord, _ = FindQuickstartById(777)
		assert.Equal(t, "Update test title", dbRecord.Title)
	})

	t.Run("should return 404 when updating non existing record", func(t *testing.T) {
		jsonParam := `{"title":"Update test title"}`

		request, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/%d", 123456), strings.NewReader(string(jsonParam)))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *messageResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "record not found", payload.Msg)
	})

	t.Run("should return 400 when updating record with invalid data", func(t *testing.T) {
		var updatedQuickstart models.Quickstart
		updatedQuickstart.ID = 777
		database.DB.Create(&updatedQuickstart)
		jsonParam := `[]`

		request, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("/%d", 777), strings.NewReader(string(jsonParam)))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *messageResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 400, response.Code)
		assert.Equal(t, "json: cannot unmarshal array into Go value of type models.Quickstart", payload.Msg)
	})
}
