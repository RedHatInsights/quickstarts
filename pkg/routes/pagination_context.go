package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/quickstarts/pkg/models"
)

type Pagination struct {
	Limit  int
	Offset int
}

type key string

const (
	PaginationContextKey key = "pagination"
)

func PaginationContext(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pagination := Pagination{
			Limit:  50,
			Offset: 0,
		}

		limitQuery := r.URL.Query()["limit"]
		offsetQuery := r.URL.Query()["offset"]
		if len(limitQuery) > 0 {
			limit, err := strconv.Atoi(limitQuery[0])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				resp := models.BadRequest{
					Msg: err.Error(),
				}

				json.NewEncoder(w).Encode(resp)
				return
			}
			pagination.Limit = limit
		}

		if len(offsetQuery) > 0 {
			offset, err := strconv.Atoi(offsetQuery[0])
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				w.Header().Set("Content-Type", "application/json")
				resp := models.BadRequest{
					Msg: err.Error(),
				}

				json.NewEncoder(w).Encode(resp)
				return
			}

			pagination.Offset = offset
		}

		ctx := context.WithValue(r.Context(), PaginationContextKey, pagination)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
