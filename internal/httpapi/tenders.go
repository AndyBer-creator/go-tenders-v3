package httpapi

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"go-tenders-v3-main/internal/store"
)

var (
	allowedServiceTypes = map[string]struct{}{
		"Construction": {},
		"Delivery":     {},
		"Manufacture":  {},
	}
	allowedTenderStatuses = map[string]struct{}{
		"Created":   {},
		"Published": {},
		"Closed":    {},
	}
)

type handler struct {
	tenders *store.TenderStore
	bids    *store.BidStore
}

type createTenderRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	ServiceType     string `json:"serviceType"`
	Status          string `json:"status"`
	OrganizationID  int    `json:"organizationId"`
	CreatorUsername string `json:"creatorUsername"`
}

type tenderResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	ServiceType    string `json:"serviceType"`
	Status         string `json:"status"`
	OrganizationID string `json:"organizationId"`
	Version        int    `json:"version"`
	CreatedAt      string `json:"createdAt"`
}

type editTenderRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	ServiceType *string `json:"serviceType"`
}

func newHandler(db *sql.DB) *handler {
	return &handler{
		tenders: store.NewTenderStore(db),
		bids:    store.NewBidStore(db),
	}
}

func (h *handler) handleTenderByID(w http.ResponseWriter, r *http.Request) {
	clean := path.Clean(r.URL.Path)
	parts := strings.Split(strings.TrimPrefix(clean, "/api/tenders/"), "/")
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	tenderID := strings.TrimSpace(parts[0])
	action := parts[1]

	switch action {
	case "status":
		h.handleTenderStatus(w, r, tenderID)
	case "edit":
		h.handleEditTender(w, r, tenderID)
	case "rollback":
		if len(parts) != 3 {
			writeError(w, http.StatusBadRequest, "invalid rollback version")
			return
		}
		h.handleRollbackTender(w, r, tenderID, parts[2])
	default:
		writeError(w, http.StatusNotFound, "endpoint not found")
	}
}

func (h *handler) handleCreateTender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}

	var req createTenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := validateCreateTenderRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	exists, err := h.tenders.UserExists(ctx, req.CreatorUsername)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify user")
		return
	}
	if !exists {
		writeError(w, http.StatusUnauthorized, "user does not exist")
		return
	}

	allowed, err := h.tenders.IsOrganizationResponsible(ctx, req.OrganizationID, req.CreatorUsername)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify organization access")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	t, err := h.tenders.CreateTender(ctx, store.CreateTenderParams{
		Name:            req.Name,
		Description:     req.Description,
		ServiceType:     req.ServiceType,
		Status:          "Created",
		OrganizationID:  req.OrganizationID,
		CreatorUsername: req.CreatorUsername,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create tender")
		return
	}

	writeJSON(w, http.StatusOK, toTenderResponse(t))
}

func (h *handler) handleTenders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}

	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	serviceTypes := r.URL.Query()["service_type"]
	for _, st := range serviceTypes {
		if _, ok := allowedServiceTypes[st]; !ok {
			writeError(w, http.StatusBadRequest, "invalid service_type")
			return
		}
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	tenders, err := h.tenders.ListTenders(ctx, store.ListTenderParams{
		Limit:        limit,
		Offset:       offset,
		ServiceTypes: serviceTypes,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tenders")
		return
	}

	resp := make([]tenderResponse, 0, len(tenders))
	for _, t := range tenders {
		resp = append(resp, toTenderResponse(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) handleUserTenders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}

	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" || len(username) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	exists, err := h.tenders.UserExists(ctx, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify user")
		return
	}
	if !exists {
		writeError(w, http.StatusUnauthorized, "user does not exist")
		return
	}

	tenders, err := h.tenders.ListUserTenders(ctx, username, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch tenders")
		return
	}

	resp := make([]tenderResponse, 0, len(tenders))
	for _, t := range tenders {
		resp = append(resp, toTenderResponse(t))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) handleTenderStatus(w http.ResponseWriter, r *http.Request, tenderID string) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" || len(username) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	exists, err := h.tenders.UserExists(ctx, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify user")
		return
	}
	if !exists {
		writeError(w, http.StatusUnauthorized, "user does not exist")
		return
	}

	t, err := h.tenders.GetTenderByID(ctx, tenderID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid tenderId")
		return
	}

	orgID, _ := strconv.Atoi(t.OrganizationID)
	allowed, err := h.tenders.IsOrganizationResponsible(ctx, orgID, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify organization access")
		return
	}

	if r.Method == http.MethodGet {
		if !allowed && t.Status != "Published" {
			writeError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
		writeJSON(w, http.StatusOK, t.Status)
		return
	}

	if r.Method != http.MethodPut {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if _, ok := allowedTenderStatuses[status]; !ok {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}

	updated, err := h.tenders.UpdateTenderStatus(ctx, tenderID, status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update status")
		return
	}

	writeJSON(w, http.StatusOK, toTenderResponse(updated))
}

func (h *handler) handleEditTender(w http.ResponseWriter, r *http.Request, tenderID string) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" || len(username) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}

	var req editTenderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := validateEditTenderRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	exists, err := h.tenders.UserExists(ctx, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify user")
		return
	}
	if !exists {
		writeError(w, http.StatusUnauthorized, "user does not exist")
		return
	}

	t, err := h.tenders.GetTenderByID(ctx, tenderID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid tenderId")
		return
	}

	orgID, _ := strconv.Atoi(t.OrganizationID)
	allowed, err := h.tenders.IsOrganizationResponsible(ctx, orgID, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify organization access")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	updated, err := h.tenders.UpdateTender(ctx, store.UpdateTenderParams{
		ID:          tenderID,
		Name:        req.Name,
		Description: req.Description,
		ServiceType: req.ServiceType,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update tender")
		return
	}

	writeJSON(w, http.StatusOK, toTenderResponse(updated))
}

func (h *handler) handleRollbackTender(w http.ResponseWriter, r *http.Request, tenderID string, versionRaw string) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	version, err := strconv.Atoi(versionRaw)
	if err != nil || version <= 0 {
		writeError(w, http.StatusBadRequest, "invalid version")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" || len(username) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	exists, err := h.tenders.UserExists(ctx, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify user")
		return
	}
	if !exists {
		writeError(w, http.StatusUnauthorized, "user does not exist")
		return
	}

	t, err := h.tenders.GetTenderByID(ctx, tenderID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid tenderId")
		return
	}

	orgID, _ := strconv.Atoi(t.OrganizationID)
	allowed, err := h.tenders.IsOrganizationResponsible(ctx, orgID, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify organization access")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	updated, err := h.tenders.RollbackTender(ctx, tenderID, version)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender or version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to rollback tender")
		return
	}
	writeJSON(w, http.StatusOK, toTenderResponse(updated))
}

func validateCreateTenderRequest(req createTenderRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.CreatorUsername = strings.TrimSpace(req.CreatorUsername)

	if req.Name == "" || len(req.Name) > 100 {
		return errors.New("invalid name")
	}
	if req.Description == "" || len(req.Description) > 500 {
		return errors.New("invalid description")
	}
	if _, ok := allowedServiceTypes[req.ServiceType]; !ok {
		return errors.New("invalid serviceType")
	}
	req.Status = strings.TrimSpace(req.Status)
	if req.Status != "" && req.Status != "Created" {
		return errors.New("invalid status")
	}
	if req.OrganizationID <= 0 {
		return errors.New("invalid organizationId")
	}
	if req.CreatorUsername == "" || len(req.CreatorUsername) > 50 {
		return errors.New("invalid creatorUsername")
	}
	return nil
}

func validateEditTenderRequest(req editTenderRequest) error {
	if req.Name != nil {
		v := strings.TrimSpace(*req.Name)
		if v == "" || len(v) > 100 {
			return errors.New("invalid name")
		}
		*req.Name = v
	}
	if req.Description != nil {
		v := strings.TrimSpace(*req.Description)
		if v == "" || len(v) > 500 {
			return errors.New("invalid description")
		}
		*req.Description = v
	}
	if req.ServiceType != nil {
		v := strings.TrimSpace(*req.ServiceType)
		if _, ok := allowedServiceTypes[v]; !ok {
			return errors.New("invalid serviceType")
		}
		*req.ServiceType = v
	}
	return nil
}

func parsePagination(r *http.Request) (int, int, error) {
	limit := 5
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > 50 {
			return 0, 0, errors.New("invalid limit")
		}
		limit = n
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return 0, 0, errors.New("invalid offset")
		}
		offset = n
	}

	return limit, offset, nil
}

func toTenderResponse(t store.Tender) tenderResponse {
	return tenderResponse{
		ID:             t.ID,
		Name:           t.Name,
		Description:    t.Description,
		ServiceType:    t.ServiceType,
		Status:         t.Status,
		OrganizationID: t.OrganizationID,
		Version:        t.Version,
		CreatedAt:      t.CreatedAt.Format(time.RFC3339),
	}
}
