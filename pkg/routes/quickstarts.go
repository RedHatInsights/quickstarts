package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/RedHatInsights/quickstarts/pkg/database"
	"github.com/RedHatInsights/quickstarts/pkg/models"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

func FindQuickstartById(id int) (models.Quickstart, error) {
	var quickStart models.Quickstart
	err := database.DB.First(&quickStart, id).Error
	return quickStart, err
}

func GetAllQuickstarts(w http.ResponseWriter, r *http.Request) {
	var quickStarts []models.Quickstart
	var bundlesQuery = r.URL.Query().Get("[]bundles")
	var bundleQuery = r.URL.Query().Get("bundle")

	// Look for gorm supported APi instead of using RAW query
	// sample query /api/quickstarts/v1/quickstarts?[]bundles=settings&[]bundles=insights
	if bundlesQuery != "" {
		var conditions []string
		for _, s := range bundlesQuery {
			conditions = append(conditions, fmt.Sprintf("(bundles)::jsonb ? '%s'", s))
		}
		where := strings.Join(conditions, "OR ")
		err := database.DB.Raw(fmt.Sprintf("SELECT * FROM quickstarts WHERE %s", where)).Scan(&quickStarts).Error
		if err != nil {
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
	} else if bundleQuery != "" {
		err := database.DB.Raw(fmt.Sprintf("SELECT * FROM quickstarts WHERE (bundles)::jsonb ? '%s'", bundleQuery)).Scan(&quickStarts).Error
		if err != nil {
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
	} else {
		database.DB.Find(&quickStarts)
	}
	resp := make(map[string][]models.Quickstart)
	resp["data"] = quickStarts
	json.NewEncoder(w).Encode(resp)
}

func CreateQuickstart(w http.ResponseWriter, r *http.Request) {
	var quickStart *models.Quickstart
	if err := json.NewDecoder(r.Body).Decode(&quickStart); err != nil {
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

	database.DB.Create(&quickStart)
	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]*models.Quickstart)
	resp["data"] = quickStart
	json.NewEncoder(w).Encode(resp)
}

func GetQuickstartById(w http.ResponseWriter, r *http.Request) {
	quickStart := r.Context().Value("quickstart").(*models.Quickstart)
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]*models.Quickstart)
	resp["data"] = quickStart
	json.NewEncoder(w).Encode(resp)
}

func DeleteQuickstartById(w http.ResponseWriter, r *http.Request) {
	quickStart := r.Context().Value("quickstart").(*models.Quickstart)
	defer r.Body.Close()
	err := database.DB.Delete(quickStart).Error
	if err != nil {
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
	resp := make(map[string]string)
	resp["msg"] = "Quickstart successfully removed"
	json.NewEncoder(w).Encode(resp)
}

func UpdateQuickstartById(w http.ResponseWriter, r *http.Request) {
	quickStart := r.Context().Value("quickstart").(*models.Quickstart)
	if err := json.NewDecoder(r.Body).Decode(&quickStart); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Header().Set("Content-Type", "application/json")
		resp := make(map[string]string)
		resp["message"] = err.Error()

		err.Error()
		jsonResp, err := json.Marshal(resp)
		if err != nil {
			logrus.Fatalf("Error happened in JSON marshal. Err: %s", err)
		}
		w.Write(jsonResp)
		r.Body.Close()
		return
	}
	err := database.DB.Save(quickStart).Error
	if err != nil {
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

	w.Header().Set("Content-Type", "application/json")
	resp := make(map[string]*models.Quickstart)
	resp["data"] = quickStart
	json.NewEncoder(w).Encode(resp)
}

func QuickstartEntityContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if quickstartId := chi.URLParam(r, "id"); quickstartId != "" {
			id, err := strconv.Atoi(quickstartId)
			if err != nil {
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
			quickstart, err := FindQuickstartById(id)
			if err != nil {
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

			ctx := context.WithValue(r.Context(), "quickstart", &quickstart)
			next.ServeHTTP(w, r.WithContext(ctx))
		}

	})
}

// MakeQuickstartsRouter creates a router handles for /quickstarts group
func MakeQuickstartsRouter(sub chi.Router) {
	sub.Post("", CreateQuickstart)
	sub.Get("", GetAllQuickstarts)
	sub.Route("/:id", func(r chi.Router) {
		r.Use(QuickstartEntityContext)
		r.Get("", GetQuickstartById)
		r.Delete("", DeleteQuickstartById)
		r.Patch("", UpdateQuickstartById)
	})
}
