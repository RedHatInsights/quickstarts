package routes

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi/v5"
)

func getFavoriteQuickstarts(accountId string) ([]string, error) {
	var favQuickstart []models.FavoriteQuickstart
	var quickstartNames []string
	database.DB.Find(&favQuickstart).Where("accountId = ? AND favorite = ?", accountId, true)

	for i := 1; i < len(favQuickstart); i++ {
		quickstartNames = append(quickstartNames, favQuickstart[i].QuickstartName)
	}

	return quickstartNames, nil
}

func getAllFavoriteQuickstarts(w http.ResponseWriter, r *http.Request) {
	var accountId = r.URL.Query().Get("account")
	var quickstarts, _ = getFavoriteQuickstarts(accountId)

	w.WriteHeader(http.StatusOK)
	resp := make(map[string][]string)
	resp["data"] = quickstarts
	json.NewEncoder(w).Encode(resp)

}

func switchFavorite(accountId string, quickstartName string, favorite bool) models.FavoriteQuickstart {
	var favQuickstart models.FavoriteQuickstart
	database.DB.Find(&favQuickstart).Where("accountId = ? AND quickstartName = ?", accountId, quickstartName)
	favQuickstart.Favorite = favorite

	// TODO: update quickstart table as well!
	database.DB.Save(&favQuickstart)
	return favQuickstart
}

func updateFavoriteQuickstart(w http.ResponseWriter, r *http.Request) {
	var accountId = r.URL.Query().Get("account")
	var favoriteQuickstart models.FavoriteQuickstart

	if err := json.NewDecoder(r.Body).Decode(&favoriteQuickstart); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)

		resp["msg"] = err.Error()
		json.NewEncoder(w).Encode(resp)
		return
	}

	var favQuickstart = switchFavorite(accountId, favoriteQuickstart.QuickstartName, favoriteQuickstart.Favorite)

	resp := make(map[string]models.FavoriteQuickstart)
	resp["data"] = favQuickstart
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func MakeFavoriteQuickstartsRouter(sub chi.Router) {
	sub.Get("/", getAllFavoriteQuickstarts)
	sub.Post("/", updateFavoriteQuickstart)
}
