package httpapi

import (
	"database/sql"
	"net/http"
)

func NewRouter(db *sql.DB) http.Handler {
	mux := http.NewServeMux()
	h := newHandler(db)

	mux.HandleFunc("/api/ping", pingHandler)
	mux.HandleFunc("/api/tenders", h.handleTenders)
	mux.HandleFunc("/api/tenders/new", h.handleCreateTender)
	mux.HandleFunc("/api/tenders/my", h.handleUserTenders)
	mux.HandleFunc("/api/tenders/", h.handleTenderByID)
	mux.HandleFunc("/api/bids/new", h.handleCreateBid)
	mux.HandleFunc("/api/bids/my", h.handleUserBids)
	mux.HandleFunc("/api/bids/", h.handleBidPath)

	return requestLogger(mux)
}

func pingHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}
