package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func getAllQuickstartsProgress(w http.ResponseWriter, r *http.Request) {
	var progress []models.QuickstartProgress
	database.DB.Find(&progress)

	resp := make(map[string][]models.QuickstartProgress)
	resp["data"] = progress
	json.NewEncoder(w).Encode(resp)
}

func getQuickstartProgress(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	var accountId int
	var quickstartId int
	accountId, _ = strconv.Atoi(queries.Get("account"))
	quickstartId, _ = strconv.Atoi(queries.Get("quickstart"))

	if accountId != 0 || quickstartId != 0 {
		var where models.QuickstartProgress
		var progresses []models.QuickstartProgress
		if accountId != 0 {
			where.AccountId = accountId
		}

		if quickstartId != 0 {
			where.QuickstartID = uint(quickstartId)
		}
		database.DB.Where(where).Find(&progresses)
		resp := make(map[string][]models.QuickstartProgress)
		resp["data"] = progresses
		json.NewEncoder(w).Encode(resp)
		return
	} else {
		getAllQuickstartsProgress(w, r)
	}

}

func createQuickstartProgress(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.Atoi(r.URL.Query().Get("quickstartId"))
	quickStart, err := FindQuickstartById(id)
	var progress *models.QuickstartProgress
	if errors.Is(err, gorm.ErrRecordNotFound) {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)
		resp["message"] = "Not found"

		jsonResp, err := json.Marshal(resp)
		if err != nil {
			logrus.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
		r.Body.Close()
		return
	}
	if err := json.NewDecoder(r.Body).Decode(&progress); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)
		resp["message"] = err.Error()

		jsonResp, err := json.Marshal(resp)
		if err != nil {
			logrus.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
		r.Body.Close()
		return
	}

	progress.Quickstart = &quickStart

	database.DB.Create(&progress)
	resp := make(map[string]*models.QuickstartProgress)
	resp["data"] = progress
	json.NewEncoder(w).Encode(resp)
}

func MakeQuickstartsProgressRouter(sub chi.Router) {
	sub.Get("", getQuickstartProgress)
	sub.Post("/:quickstartId", createQuickstartProgress)
}
