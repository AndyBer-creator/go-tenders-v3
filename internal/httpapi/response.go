package httpapi

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Reason string `json:"reason"`
}

func writeError(w http.ResponseWriter, status int, reason string) {
	writeJSON(w, status, errorResponse{Reason: reason})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
