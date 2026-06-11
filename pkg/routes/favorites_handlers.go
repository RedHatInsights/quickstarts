package routes

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
	"github.com/RedHatInsights/quickstarts/pkg/utils"
)

// GetFavorites handles GET /favorites
func (s *ServerAdapter) GetFavorites(w http.ResponseWriter, r *http.Request, params generated.GetFavoritesParams) {
	// Validate account parameter
	if params.Account == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "missing account parameter")
		return
	}

	// Use service to get favorites for the account
	favorites, err := s.favoriteService.GetFavorites(params.Account)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert to generated types and respond
	genFavorites := make([]generated.FavoriteQuickstart, len(favorites))
	for i, favorite := range favorites {
		genFavorites[i] = favorite.ToAPI()
	}

	utils.DataResponse(w, http.StatusOK, genFavorites)
}

// PostFavorites handles POST /favorites
func (s *ServerAdapter) PostFavorites(w http.ResponseWriter, r *http.Request, params generated.PostFavoritesParams) {
	// Parse request body
	var reqBody generated.FavoriteQuickstart
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Extract quickstart name and favorite status from request body
	var quickstartName string
	var favorite bool
	if reqBody.QuickstartName != nil {
		quickstartName = *reqBody.QuickstartName
	}
	if reqBody.Favorite != nil {
		favorite = *reqBody.Favorite
	}

	// Use service to switch favorite status
	result, err := s.favoriteService.SwitchFavorite(params.Account, quickstartName, favorite)
	if err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, err.Error())
		return
	}

	// Convert to generated type and respond
	genFavorite := result.ToAPI()
	utils.DataResponse(w, http.StatusOK, genFavorite)
}