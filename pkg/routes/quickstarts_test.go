package routes

import (
	"encoding/json"
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

type responseBody struct {
	Title string `json:"Title"`
}

type responsePayload struct {
	Data []responseBody
}

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.GET("/", GetAllQuickstarts)
	return r
}

func TestGetAll(t *testing.T) {
	router := setupRouter()
	mockQuickstart()
	t.Run("returns GET all quickstarts successfully", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/", nil)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)

		var payload *responsePayload
		json.NewDecoder(response.Body).Decode(&payload)
		assert.Equal(t, 200, response.Code)
		assert.Equal(t, 1, len(payload.Data))
		assert.Equal(t, "test title", payload.Data[0].Title)
	})
}
