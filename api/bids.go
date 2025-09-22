package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"go-tenders-v3/model"
	"go-tenders-v3/storage"

	"github.com/gorilla/mux"
)

type BidHandler struct {
	storage *storage.Storage
}

func NewBidHandler(s *storage.Storage) *BidHandler {
	return &BidHandler{storage: s}
}

func (h *BidHandler) CreateBid(w http.ResponseWriter, r *http.Request) {
	var bid storage.Bid
	if err := json.NewDecoder(r.Body).Decode(&bid); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	if err := h.storage.CreateBid(&bid); err != nil {
		http.Error(w, "Failed to create bid", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(bid)
}

func (h *BidHandler) EditBid(w http.ResponseWriter, r *http.Request) {
	bidIdStr := mux.Vars(r)["bidId"]

	var bid model.Bid
	if err := json.NewDecoder(r.Body).Decode(&bid); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	bid.Id = bidIdStr

	if bid.Status == nil {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	// Преобразуем model.Bid в storage.Bid для передачи в storage слой
	storageBid := &storage.Bid{
		Id:          toInt(bid.Id),
		TenderId:    toInt(bid.TenderId),
		UserId:      toInt(bid.AuthorId), // если AuthorId - это UserId
		Amount:      0,                   // если есть, иначе убрать
		Description: bid.Description,
		Status:      string(*bid.Status),
	}

	if err := h.storage.EditBid(storageBid); err != nil {
		http.Error(w, "Failed to update bid", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Функция преобразования строки в int, с обработкой ошибок
func toInt(s string) int {
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func (h *BidHandler) GetBidReviews(w http.ResponseWriter, r *http.Request) {
	bidId := mux.Vars(r)["bidId"]

	reviews, err := h.storage.GetBidReviews(bidId)
	if err != nil {
		http.Error(w, "Failed to get bid reviews", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(reviews)
}

func (h *BidHandler) GetBidStatus(w http.ResponseWriter, r *http.Request) {
	bidId := mux.Vars(r)["bidId"]

	status, err := h.storage.GetBidStatus(bidId)
	if err != nil {
		http.Error(w, "Failed to get bid status", http.StatusInternalServerError)
		return
	}

	resp := struct {
		Status string `json:"status"`
	}{
		Status: status,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func (h *BidHandler) GetBidsForTender(w http.ResponseWriter, r *http.Request) {
	tenderId := mux.Vars(r)["tenderId"]

	bids, err := h.storage.GetBidsForTender(tenderId)
	if err != nil {
		http.Error(w, "Failed to get bids for tender", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bids)
}

func (h *BidHandler) GetUserBids(w http.ResponseWriter, r *http.Request) {
	userId := mux.Vars(r)["userId"]

	bids, err := h.storage.GetUserBids(userId)
	if err != nil {
		http.Error(w, "Failed to get user bids", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bids)
}

func (h *BidHandler) RollbackBid(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	bidId := vars["bidId"]
	versionStr := vars["version"]

	version, err := strconv.Atoi(versionStr)
	if err != nil {
		http.Error(w, "Invalid version number", http.StatusBadRequest)
		return
	}

	if err := h.storage.RollbackBid(bidId, version); err != nil {
		http.Error(w, "Failed to rollback bid", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BidHandler) SubmitBidDecision(w http.ResponseWriter, r *http.Request) {
	bidId := mux.Vars(r)["bidId"]

	var input struct {
		Decision model.BidDecision `json:"decision"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Проверка валидности решения, если нужно
	if input.Decision != model.BidDecisionAPPROVED && input.Decision != model.BidDecisionREJECTED {
		http.Error(w, "Invalid decision value", http.StatusBadRequest)
		return
	}

	if err := h.storage.SubmitBidDecision(bidId, input.Decision); err != nil {
		http.Error(w, "Failed to submit bid decision", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *BidHandler) SubmitBidFeedback(w http.ResponseWriter, r *http.Request) {
	bidId := mux.Vars(r)["bidId"]

	var feedback model.BidReview
	if err := json.NewDecoder(r.Body).Decode(&feedback); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}

	// Можно установить CreatedAt, если не передается клиентом
	if feedback.CreatedAt == "" {
		feedback.CreatedAt = time.Now().Format(time.RFC3339)
	}

	if err := h.storage.SubmitBidFeedback(bidId, feedback); err != nil {
		http.Error(w, "Failed to submit bid feedback", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *BidHandler) UpdateBidStatus(w http.ResponseWriter, r *http.Request) {
	bidId := mux.Vars(r)["bidId"]

	var input struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if input.Status == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	if err := h.storage.UpdateBidStatus(bidId, input.Status); err != nil {
		http.Error(w, "Failed to update bid status", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
