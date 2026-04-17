package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"go-tenders-v3-main/internal/store"
)

var allowedBidStatuses = map[string]struct{}{
	"Created":   {},
	"Published": {},
	"Canceled":  {},
	"Approved":  {},
	"Rejected":  {},
}

type createBidRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	TenderID        string `json:"tenderId"`
	OrganizationID  int    `json:"organizationId"`
	CreatorUsername string `json:"creatorUsername"`
}

type editBidRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type bidResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	TenderID    string `json:"tenderId"`
	AuthorType  string `json:"authorType"`
	AuthorID    string `json:"authorId"`
	Version     int    `json:"version"`
	CreatedAt   string `json:"createdAt"`
}

type bidReviewResponse struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
}

func (h *handler) handleCreateBid(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	var req createBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := validateCreateBidRequest(req); err != nil {
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

	t, err := h.tenders.GetTenderByID(ctx, req.TenderID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid tenderId")
		return
	}

	orgID, _ := strconv.Atoi(t.OrganizationID)
	if req.OrganizationID > 0 {
		allowed, err := h.tenders.IsOrganizationResponsible(ctx, req.OrganizationID, req.CreatorUsername)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to verify organization access")
			return
		}
		if !allowed {
			writeError(w, http.StatusForbidden, "insufficient permissions")
			return
		}
	}

	if t.Status != "Published" {
		allowed, err := h.tenders.IsOrganizationResponsible(ctx, orgID, req.CreatorUsername)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to verify organization access")
			return
		}
		if !allowed {
			writeError(w, http.StatusForbidden, "tender is not available")
			return
		}
	}

	var authorID string
	if req.OrganizationID <= 0 {
		eid, err := h.tenders.EmployeeIDByUsername(ctx, req.CreatorUsername)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				writeError(w, http.StatusUnauthorized, "user does not exist")
			} else {
				writeError(w, http.StatusInternalServerError, "failed to resolve author")
			}
			return
		}
		authorID = strconv.Itoa(eid)
	}

	b, err := h.bids.CreateBid(ctx, store.CreateBidParams{
		Name:            req.Name,
		Description:     req.Description,
		Status:          "Created",
		TenderID:        req.TenderID,
		OrganizationID:  req.OrganizationID,
		CreatorUsername: req.CreatorUsername,
		AuthorID:        authorID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to create bid")
		return
	}

	writeJSON(w, http.StatusOK, toBidResponse(b))
}

func (h *handler) handleUserBids(w http.ResponseWriter, r *http.Request) {
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

	bids, err := h.bids.ListUserBids(ctx, username, store.ListBidParams{Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch bids")
		return
	}
	resp := make([]bidResponse, 0, len(bids))
	for _, b := range bids {
		resp = append(resp, toBidResponse(b))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) handleBidPath(w http.ResponseWriter, r *http.Request) {
	clean := path.Clean(r.URL.Path)
	parts := strings.Split(strings.TrimPrefix(clean, "/api/bids/"), "/")
	if len(parts) < 2 {
		writeError(w, http.StatusNotFound, "endpoint not found")
		return
	}

	idPart := strings.TrimSpace(parts[0])
	action := parts[1]
	switch action {
	case "list":
		h.handleBidsForTender(w, r, idPart)
	case "status":
		h.handleBidStatus(w, r, idPart)
	case "edit":
		h.handleEditBid(w, r, idPart)
	case "rollback":
		if len(parts) != 3 {
			writeError(w, http.StatusBadRequest, "invalid rollback version")
			return
		}
		h.handleRollbackBid(w, r, idPart, parts[2])
	case "submit_decision":
		h.handleSubmitBidDecision(w, r, idPart)
	case "feedback":
		h.handleBidFeedback(w, r, idPart)
	case "reviews":
		h.handleBidReviews(w, r, idPart)
	default:
		writeError(w, http.StatusNotFound, "endpoint not found")
	}
}

func (h *handler) handleBidsForTender(w http.ResponseWriter, r *http.Request, tenderID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" || len(username) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}
	limit, offset, err := parsePagination(r)
	if err != nil {
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

	bids, err := h.bids.ListBidsForTender(ctx, tenderID, store.ListBidParams{Limit: limit, Offset: offset})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch bids")
		return
	}
	resp := make([]bidResponse, 0, len(bids))
	for _, b := range bids {
		resp = append(resp, toBidResponse(b))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) handleBidStatus(w http.ResponseWriter, r *http.Request, bidID string) {
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

	b, err := h.bids.GetBidByID(ctx, bidID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "bid not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid bidId")
		return
	}

	allowed, err := h.isBidVisibleForUser(ctx, b, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify access")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	if r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, b.Status)
		return
	}
	if r.Method != http.MethodPut {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}

	mutable, err := h.isBidMutableByUser(ctx, b, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify access")
		return
	}
	if !mutable {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if _, ok := allowedBidStatuses[status]; !ok {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	updated, err := h.bids.UpdateBidStatus(ctx, bidID, status)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "bid not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update status")
		return
	}
	writeJSON(w, http.StatusOK, toBidResponse(updated))
}

func (h *handler) handleEditBid(w http.ResponseWriter, r *http.Request, bidID string) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" || len(username) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}
	var req editBidRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if err := validateEditBidRequest(req); err != nil {
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
	b, err := h.bids.GetBidByID(ctx, bidID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "bid not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid bidId")
		return
	}
	mutable, err := h.isBidMutableByUser(ctx, b, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify access")
		return
	}
	if !mutable {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	updated, err := h.bids.UpdateBid(ctx, store.UpdateBidParams{ID: bidID, Name: req.Name, Description: req.Description})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "bid not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update bid")
		return
	}
	writeJSON(w, http.StatusOK, toBidResponse(updated))
}

func (h *handler) handleRollbackBid(w http.ResponseWriter, r *http.Request, bidID, versionRaw string) {
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
	b, err := h.bids.GetBidByID(ctx, bidID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "bid not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid bidId")
		return
	}
	mutable, err := h.isBidMutableByUser(ctx, b, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify access")
		return
	}
	if !mutable {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	updated, err := h.bids.RollbackBid(ctx, bidID, version)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "bid or version not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to rollback bid")
		return
	}
	writeJSON(w, http.StatusOK, toBidResponse(updated))
}

func (h *handler) handleSubmitBidDecision(w http.ResponseWriter, r *http.Request, bidID string) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	decision := strings.TrimSpace(r.URL.Query().Get("decision"))
	if decision != "Approved" && decision != "Rejected" {
		writeError(w, http.StatusBadRequest, "invalid decision")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" || len(username) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	b, t, ok := h.loadBidAndTenderWithUser(ctx, bidID, username, w)
	if !ok {
		return
	}

	tenderOrgID, _ := strconv.Atoi(t.OrganizationID)
	allowed, err := h.tenders.IsOrganizationResponsible(ctx, tenderOrgID, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify organization access")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	if err := h.bids.UpsertDecision(ctx, b.ID, username, decision); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save decision")
		return
	}

	shouldReject, err := h.bids.HasRejectedDecision(ctx, b.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to calculate decision")
		return
	}
	if shouldReject {
		updated, err := h.bids.UpdateBidStatus(ctx, b.ID, "Rejected")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update bid status")
			return
		}
		writeJSON(w, http.StatusOK, toBidResponse(updated))
		return
	}

	approvedCount, err := h.bids.CountApprovedDecisions(ctx, b.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to calculate approvals")
		return
	}
	responsibleCount, err := h.tenders.CountOrganizationResponsibles(ctx, tenderOrgID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to calculate quorum")
		return
	}
	quorum := responsibleCount
	if quorum > 3 {
		quorum = 3
	}
	if quorum < 1 {
		quorum = 1
	}

	if approvedCount >= quorum {
		updated, err := h.bids.UpdateBidStatus(ctx, b.ID, "Approved")
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to update bid status")
			return
		}
		_, _ = h.tenders.UpdateTenderStatus(ctx, t.ID, "Closed")
		writeJSON(w, http.StatusOK, toBidResponse(updated))
		return
	}

	refreshed, err := h.bids.GetBidByID(ctx, b.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch bid")
		return
	}
	writeJSON(w, http.StatusOK, toBidResponse(refreshed))
}

func (h *handler) handleBidFeedback(w http.ResponseWriter, r *http.Request, bidID string) {
	if r.Method != http.MethodPut {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	feedback := strings.TrimSpace(r.URL.Query().Get("bidFeedback"))
	if feedback == "" || len(feedback) > 1000 {
		writeError(w, http.StatusBadRequest, "invalid bidFeedback")
		return
	}
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	if username == "" || len(username) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	b, t, ok := h.loadBidAndTenderWithUser(ctx, bidID, username, w)
	if !ok {
		return
	}
	tenderOrgID, _ := strconv.Atoi(t.OrganizationID)
	allowed, err := h.tenders.IsOrganizationResponsible(ctx, tenderOrgID, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify organization access")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}
	if err := h.bids.AddFeedback(ctx, b.ID, feedback, username); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save feedback")
		return
	}
	updated, err := h.bids.GetBidByID(ctx, b.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch bid")
		return
	}
	writeJSON(w, http.StatusOK, toBidResponse(updated))
}

func (h *handler) handleBidReviews(w http.ResponseWriter, r *http.Request, tenderID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusBadRequest, "unsupported method")
		return
	}
	authorUsername := strings.TrimSpace(r.URL.Query().Get("authorUsername"))
	requesterUsername := strings.TrimSpace(r.URL.Query().Get("requesterUsername"))
	if authorUsername == "" || len(authorUsername) > 50 || requesterUsername == "" || len(requesterUsername) > 50 {
		writeError(w, http.StatusBadRequest, "invalid username")
		return
	}
	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	ctx, cancel := withTimeout(r.Context())
	defer cancel()

	t, err := h.tenders.GetTenderByID(ctx, tenderID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender not found")
			return
		}
		writeError(w, http.StatusBadRequest, "invalid tenderId")
		return
	}
	tenderOrgID, _ := strconv.Atoi(t.OrganizationID)
	allowed, err := h.tenders.IsOrganizationResponsible(ctx, tenderOrgID, requesterUsername)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify organization access")
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	if _, err := h.bids.GetBidByTenderAndAuthor(ctx, tenderID, authorUsername); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "author bid not found for tender")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to validate author participation")
		return
	}

	reviews, err := h.bids.ListReviewsByAuthor(ctx, authorUsername, limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to fetch reviews")
		return
	}
	resp := make([]bidReviewResponse, 0, len(reviews))
	for _, review := range reviews {
		resp = append(resp, bidReviewResponse{
			ID:          review.ID,
			Description: review.Description,
			CreatedAt:   review.CreatedAt.Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (h *handler) loadBidAndTenderWithUser(ctx context.Context, bidID, username string, w http.ResponseWriter) (store.Bid, store.Tender, bool) {
	exists, err := h.tenders.UserExists(ctx, username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to verify user")
		return store.Bid{}, store.Tender{}, false
	}
	if !exists {
		writeError(w, http.StatusUnauthorized, "user does not exist")
		return store.Bid{}, store.Tender{}, false
	}

	b, err := h.bids.GetBidByID(ctx, bidID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "bid not found")
		} else {
			writeError(w, http.StatusBadRequest, "invalid bidId")
		}
		return store.Bid{}, store.Tender{}, false
	}
	t, err := h.tenders.GetTenderByID(ctx, b.TenderID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "tender not found")
		} else {
			writeError(w, http.StatusInternalServerError, "failed to fetch tender")
		}
		return store.Bid{}, store.Tender{}, false
	}
	return b, t, true
}

func validateCreateBidRequest(req createBidRequest) error {
	req.Name = strings.TrimSpace(req.Name)
	req.Description = strings.TrimSpace(req.Description)
	req.CreatorUsername = strings.TrimSpace(req.CreatorUsername)

	if req.Name == "" || len(req.Name) > 100 {
		return errors.New("invalid name")
	}
	if req.Description == "" || len(req.Description) > 500 {
		return errors.New("invalid description")
	}
	req.Status = strings.TrimSpace(req.Status)
	if req.Status != "" && req.Status != "Created" {
		return errors.New("invalid status")
	}
	if req.TenderID == "" {
		return errors.New("invalid tenderId")
	}
	if req.OrganizationID < 0 {
		return errors.New("invalid organizationId")
	}
	if req.CreatorUsername == "" || len(req.CreatorUsername) > 50 {
		return errors.New("invalid creatorUsername")
	}
	return nil
}

func validateEditBidRequest(req editBidRequest) error {
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
	return nil
}

func (h *handler) isBidVisibleForUser(ctx context.Context, b store.Bid, username string) (bool, error) {
	if username == "" {
		return false, nil
	}
	if b.CreatorUsername == username {
		return true, nil
	}
	if b.AuthorType == "Organization" {
		orgID, err := strconv.Atoi(b.AuthorID)
		if err == nil {
			allowed, err := h.tenders.IsOrganizationResponsible(ctx, orgID, username)
			if err != nil {
				return false, err
			}
			if allowed {
				return true, nil
			}
		}
	}

	t, err := h.tenders.GetTenderByID(ctx, b.TenderID)
	if err != nil {
		return false, err
	}
	tenderOrgID, err := strconv.Atoi(t.OrganizationID)
	if err != nil {
		return false, nil
	}
	allowed, err := h.tenders.IsOrganizationResponsible(ctx, tenderOrgID, username)
	if err != nil {
		return false, err
	}
	if allowed {
		return true, nil
	}

	return false, nil
}

// isBidMutableByUser allows edit/rollback/status changes only for the bid author (creator) or org responsibles of the bid's authoring organization — not tender-side reviewers.
func (h *handler) isBidMutableByUser(ctx context.Context, b store.Bid, username string) (bool, error) {
	if username == "" {
		return false, nil
	}
	if b.CreatorUsername == username {
		return true, nil
	}
	if b.AuthorType == "Organization" {
		orgID, err := strconv.Atoi(b.AuthorID)
		if err != nil {
			return false, nil
		}
		return h.tenders.IsOrganizationResponsible(ctx, orgID, username)
	}
	return false, nil
}

func toBidResponse(b store.Bid) bidResponse {
	return bidResponse{
		ID:          b.ID,
		Name:        b.Name,
		Description: b.Description,
		Status:      b.Status,
		TenderID:    b.TenderID,
		AuthorType:  b.AuthorType,
		AuthorID:    b.AuthorID,
		Version:     b.Version,
		CreatedAt:   b.CreatedAt.Format(time.RFC3339),
	}
}
