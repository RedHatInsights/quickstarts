package handlers

import (
	"encoding/json"
	"net/http"
)

func SubmitPR(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "not_implemented",
		"msg":    "PR submission not yet implemented",
	})
}
