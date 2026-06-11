package utils

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/quickstarts/pkg/generated"
)

// DataResponse creates a standardized JSON response with data
func DataResponse[T any](w http.ResponseWriter, statusCode int, data T) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := map[string]T{"data": data}
	json.NewEncoder(w).Encode(resp)
}

// MessageResponse creates a standardized JSON response with a message
func MessageResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := map[string]string{"msg": message}
	json.NewEncoder(w).Encode(resp)
}

// ErrorResponse creates a standardized JSON error response
func ErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	resp := generated.BadRequest{
		Msg: &message,
	}
	json.NewEncoder(w).Encode(resp)
}

// NotFoundResponse creates a standardized 404 response
func NotFoundResponse(w http.ResponseWriter, resource string) {
	message := resource + " not found"
	resp := generated.NotFound{
		Msg: &message,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(resp)
}