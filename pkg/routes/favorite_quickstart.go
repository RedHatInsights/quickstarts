package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi/v5"
)

func handleError(w http.ResponseWriter, statusCode int, errorMessage string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]string{"msg": errorMessage}
	json.NewEncoder(w).Encode(resp)
}

func makeResponse(w http.ResponseWriter, fav interface{}) {
	w.WriteHeader(http.StatusOK)

	switch v := fav.(type) {
	case models.FavoriteQuickstart:
		resp := map[string]models.FavoriteQuickstart{"data": v}
		json.NewEncoder(w).Encode(resp)
	case *[]models.FavoriteQuickstart:
		resp := map[string][]models.FavoriteQuickstart{"data": *v}
		json.NewEncoder(w).Encode(resp)
	default:
		handleError(w, http.StatusInternalServerError, "Unsupported type")
	}
}

func GetFavoriteQuickstarts(accountId string) ([]models.FavoriteQuickstart, error) {
	var favQuickstarts []models.FavoriteQuickstart
	result := database.DB.Where(&models.FavoriteQuickstart{AccountId: accountId, Favorite: true}).Find(&favQuickstarts)
	return favQuickstarts, result.Error
}

func GetAllFavoriteQuickstarts(w http.ResponseWriter, r *http.Request) {
	var accountId = r.URL.Query().Get("account")
	if accountId == "" {
		handleError(w, http.StatusBadRequest, "Account query param is required")
		return
	}

	var favorites, err = GetFavoriteQuickstarts(accountId)
	if err != nil {
		handleError(w, http.StatusBadRequest, err.Error())
		return
	}

	makeResponse(w, &favorites)
}

func SwitchFavorite(accountId string, quickstartName string, favorite bool) (models.FavoriteQuickstart, error) {
	var favQuickstart models.FavoriteQuickstart
	result := database.DB.Where(&models.FavoriteQuickstart{AccountId: accountId, QuickstartName: quickstartName}).Find(&favQuickstart).Update("Favorite", favorite)

	if result.Error != nil {
		fmt.Println("Error while quering database:", result.Error)
		return favQuickstart, result.Error
	} else {
		fmt.Printf("Retrieved Favorite Quickstart: %+v\n", favQuickstart)
	}

	// very first switch
	if result.RowsAffected == 0 {

		favQuickstart = models.FavoriteQuickstart{
			AccountId:      accountId,
			QuickstartName: quickstartName,
			Favorite:       favorite,
		}

		var qs models.Quickstart
		database.DB.Where("name = ?", quickstartName).Preload("FavoriteQuickstart").Find(&qs)
		qs.FavoriteQuickstart = append(qs.FavoriteQuickstart, favQuickstart)

		if err := database.DB.Save(&qs).Error; err != nil {
			fmt.Println("Error saving to database Quickstart:", err)
			return favQuickstart, err
		}
	}

	return favQuickstart, nil
}

type FavQuickstartPayload struct {
	QuickstartName string `json:"quickstartName"`
	Favorite       bool   `json:"favorite"`
}

func UpdateFavoriteQuickstart(w http.ResponseWriter, r *http.Request) {
	var accountId = r.URL.Query().Get("account")
	if accountId == "" {
		handleError(w, http.StatusBadRequest, "Account query param is required")
		return
	}

	var payload FavQuickstartPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		handleError(w, http.StatusBadRequest, err.Error())
		return
	}

	var favQuickstart, err = SwitchFavorite(accountId, payload.QuickstartName, payload.Favorite)
	if err != nil {
		handleError(w, http.StatusBadRequest, err.Error())
		return
	}

	makeResponse(w, favQuickstart)
}

func MakeFavoriteQuickstartsRouter(sub chi.Router) {
	sub.Get("/", GetAllFavoriteQuickstarts)
	sub.Post("/", UpdateFavoriteQuickstart)
}
