package httpapi

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Reason    string `json:"reason"`
	RequestID string `json:"request_id,omitempty"`
}

func writeError(w http.ResponseWriter, status int, reason string) {
	writeJSON(w, status, errorResponse{
		Reason:    reason,
		RequestID: w.Header().Get("X-Request-ID"),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
