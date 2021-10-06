package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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

type errorResponsePayload struct {
	Msg string `json:"msg"`
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(QuickstartEntityContext())
	r.GET("/", GetAllQuickstarts)
	r.GET("/:id", GetQuickstartById)
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
		var payload *errorResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 404, response.Code)
		assert.Equal(t, "record not found", payload.Msg)
	})

	t.Run("return 400 error response if bad request was sent", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/notanid", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		var payload *errorResponsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 400, response.Code)
		assert.Equal(t, "strconv.Atoi: parsing \"notanid\": invalid syntax", payload.Msg)
	})
}
