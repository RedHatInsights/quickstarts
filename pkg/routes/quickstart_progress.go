package routes

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi"
)

func getAllQuickstartsProgress(w http.ResponseWriter, r *http.Request) {
	var progress []models.QuickstartProgress
	database.DB.Find(&progress)

	resp := make(map[string][]models.QuickstartProgress)
	resp["data"] = progress
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func getQuickstartProgress(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()
	var accountId int
	var quickstart int
	accountId, _ = strconv.Atoi(queries.Get("account"))
	quickstart, _ = strconv.Atoi(queries.Get("quickstart"))

	if accountId != 0 && quickstart != 0 {
		var where models.QuickstartProgress
		var progresses []models.QuickstartProgress
		if accountId != 0 {
			where.AccountId = accountId
		}

		if quickstart != 0 {
			where.Quickstart = uint(quickstart)
		}
		database.DB.Where(where).Find(&progresses)
		resp := make(map[string][]models.QuickstartProgress)
		resp["data"] = progresses
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
		return
	} else {
		getAllQuickstartsProgress(w, r)
	}

}

func createQuickstartProgress(w http.ResponseWriter, r *http.Request) {
	var progress *models.QuickstartProgress
	if err := json.NewDecoder(r.Body).Decode(&progress); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)

		resp["msg"] = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	err := database.DB.Create(&progress).Error
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)

		resp["msg"] = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	resp := make(map[string]*models.QuickstartProgress)
	resp["data"] = progress
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func MakeQuickstartsProgressRouter(sub chi.Router) {
	sub.Get("/", getQuickstartProgress)
	sub.Post("/", createQuickstartProgress)
}
