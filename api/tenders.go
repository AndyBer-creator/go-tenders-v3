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

type TenderHandler struct {
	storage *storage.Storage
}

func NewTenderHandler(s *storage.Storage) *TenderHandler {
	return &TenderHandler{storage: s}
}

func ModelToStorageTender(m *model.Tender) (*storage.Tender, error) {
	id, err := strconv.Atoi(m.Id)
	if err != nil {
		return nil, err
	}
	orgId, err := strconv.Atoi(m.OrganizationId)
	if err != nil {
		return nil, err
	}
	createdAt, err := time.Parse(time.RFC3339, m.CreatedAt)
	if err != nil {
		return nil, err
	}
	// Обратите внимание, что UpdatedAt нет в модели, убираем
	status := ""
	if m.Status != nil {
		status = string(*m.Status)
	}

	return &storage.Tender{
		Id:             id,
		OrganizationId: orgId,
		Title:          m.Name, // использован Name
		Description:    m.Description,
		Status:         status,
		CreatedAt:      createdAt,
		//	UpdatedAt:      time.Time{}, // или как-то иначе обрабатывать отсутствие
	}, nil
}

// Преобразование из storage.Tender в model.Tender
func StorageToModelTender(s *storage.Tender) *model.Tender {
	return &model.Tender{
		Id:             strconv.Itoa(s.Id),
		OrganizationId: strconv.Itoa(s.OrganizationId),
		Name:           s.Title,
		Description:    s.Description,
		Status:         NewTenderStatus(s.Status),
		CreatedAt:      s.CreatedAt.Format(time.RFC3339),
		//UpdatedAt:      s.UpdatedAt.Format(time.RFC3339),
	}
}

func NewTenderStatus(s string) *model.TenderStatus {
	ts := model.TenderStatus(s)
	return &ts
}

func (h *TenderHandler) CreateTender(w http.ResponseWriter, r *http.Request) {
	var t model.Tender
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	t.Status = NewTenderStatus("CREATED") // статус по умолчанию

	storageTender, err := ModelToStorageTender(&t)
	if err != nil {
		http.Error(w, "Invalid tender data", http.StatusBadRequest)
		return
	}

	if err := h.storage.CreateTender(storageTender); err != nil {
		http.Error(w, "Failed to create tender", http.StatusInternalServerError)
		return
	}

	respTender := StorageToModelTender(storageTender)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(respTender)
}

func (h *TenderHandler) EditTender(w http.ResponseWriter, r *http.Request) {
	tenderIdStr := mux.Vars(r)["tenderId"]
	tenderId, err := strconv.Atoi(tenderIdStr)
	if err != nil {
		http.Error(w, "Invalid tender ID", http.StatusBadRequest)
		return
	}

	var t model.Tender
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, "Invalid JSON body", http.StatusBadRequest)
		return
	}
	t.Id = strconv.Itoa(tenderId)

	if t.Status == nil {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	storageTender, err := ModelToStorageTender(&t)
	if err != nil {
		http.Error(w, "Invalid tender data", http.StatusBadRequest)
		return
	}

	if err := h.storage.EditTender(storageTender); err != nil {
		http.Error(w, "Failed to update tender", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TenderHandler) GetTenderStatus(w http.ResponseWriter, r *http.Request) {
	tenderIdStr := mux.Vars(r)["tenderId"]
	tenderId, err := strconv.Atoi(tenderIdStr)
	if err != nil {
		http.Error(w, "Invalid tender ID", http.StatusBadRequest)
		return
	}
	status, err := h.storage.GetTenderStatus(tenderId)
	if err != nil {
		http.Error(w, "Failed to get tender status", http.StatusInternalServerError)
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

func (h *TenderHandler) GetTenders(w http.ResponseWriter, r *http.Request) {
	tenders, err := h.storage.GetTenders()
	if err != nil {
		http.Error(w, "Failed to get tenders", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tenders)
}

func (h *TenderHandler) GetUserTenders(w http.ResponseWriter, r *http.Request) {
	userIdStr := mux.Vars(r)["userId"]

	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	tenders, err := h.storage.GetUserTenders(userId)
	if err != nil {
		http.Error(w, "Failed to get user tenders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tenders)
}

func (h *TenderHandler) RollbackTender(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tenderId := vars["tenderId"]
	prevStatus := vars["previousStatus"]
	if tenderId == "" || prevStatus == "" {
		http.Error(w, "Tender ID and previous status required", http.StatusBadRequest)
		return
	}
	if err := h.storage.RollbackTender(toInt(tenderId), prevStatus); err != nil {
		http.Error(w, "Failed to rollback tender", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *TenderHandler) UpdateTenderStatus(w http.ResponseWriter, r *http.Request) {
	tenderId := mux.Vars(r)["tenderId"]
	if tenderId == "" {
		http.Error(w, "Tender ID required", http.StatusBadRequest)
		return
	}
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
	if err := h.storage.UpdateTenderStatus(toInt(tenderId), input.Status); err != nil {
		http.Error(w, "Failed to update tender status", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
