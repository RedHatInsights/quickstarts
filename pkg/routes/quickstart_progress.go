package routes

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi"
)

func getExistingProgress(name string, accountId int) (models.QuickstartProgress, error) {
	var progress models.QuickstartProgress
	var where models.QuickstartProgress
	where.QuickstartName = name
	where.AccountId = accountId
	err := database.DB.Where(where).First(&progress).Error
	return progress, err
}

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
	var quickstart string
	accountId, _ = strconv.Atoi(queries.Get("account"))
	quickstart = queries.Get("quickstart")

	if accountId != 0 || quickstart != "" {
		var where models.QuickstartProgress
		var progresses []models.QuickstartProgress
		if accountId != 0 {
			where.AccountId = accountId
		}

    if quickstart != "" {
      where.QuickstartName = quickstart
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

func updateQuickstartProgress(w http.ResponseWriter, r *http.Request) {
	var progress *models.QuickstartProgress
	var currentProgress models.QuickstartProgress
	if err := json.NewDecoder(r.Body).Decode(&progress); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)

		resp["msg"] = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	if progress.AccountId == 0 || progress.QuickstartName == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)

		resp["msg"] = "Bad request! Missing accountId or quickstartName."
		json.NewEncoder(w).Encode(resp)
		return
	}

	currentProgress, err := getExistingProgress(progress.QuickstartName, progress.AccountId)
	/**
	* if no progress is set for the name and account, create new
	* else update the exising progress
	 */
	if err != nil {
		err = database.DB.Create(&progress).Error
	} else {
		currentProgress.Progress = progress.Progress
		err = database.DB.Save(&currentProgress).Error
		progress = &currentProgress
	}
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

func deleteQuickstartProgress(w http.ResponseWriter, r *http.Request) {
	if progressId := chi.URLParam(r, "id"); progressId != "" {
		id, err := strconv.Atoi(progressId)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Header().Set("Content-Type", "application/json")
			resp := make(map[string]string)
			resp["msg"] = err.Error()
			json.NewEncoder(w).Encode(resp)
			return
		}
		var quickStartProgress models.QuickstartProgress
		err = database.DB.First(&quickStartProgress, id).Error
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			resp := make(map[string]string)
			resp["msg"] = err.Error()
			json.NewEncoder(w).Encode(resp)
			return
		}
		err = database.DB.Delete(&quickStartProgress).Error
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			w.Header().Set("Content-Type", "application/json")
			resp := make(map[string]string)
			resp["msg"] = err.Error()
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusOK)
		resp := make(map[string]string)
		resp["msg"] = "Quickstart progress successfully removed"
		json.NewEncoder(w).Encode(resp)
	}
}

func MakeQuickstartsProgressRouter(sub chi.Router) {
	sub.Get("/", getQuickstartProgress)
	sub.Post("/", updateQuickstartProgress)
	sub.Delete("/{id}", deleteQuickstartProgress)
}
