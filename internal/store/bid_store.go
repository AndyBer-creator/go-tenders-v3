package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Bid struct {
	ID              string
	Name            string
	Description     string
	Status          string
	TenderID        string
	AuthorType      string
	AuthorID        string
	CreatorUsername string
	Version         int
	CreatedAt       time.Time
}

type CreateBidParams struct {
	Name            string
	Description     string
	Status          string
	TenderID        string
	OrganizationID  int
	CreatorUsername string
}

type ListBidParams struct {
	Limit  int
	Offset int
}

type UpdateBidParams struct {
	ID          string
	Name        *string
	Description *string
}

type BidStore struct {
	db *sql.DB
}

type BidReview struct {
	ID          string
	Description string
	CreatedAt   time.Time
}

func NewBidStore(db *sql.DB) *BidStore {
	return &BidStore{db: db}
}

func (s *BidStore) CreateBid(ctx context.Context, p CreateBidParams) (Bid, error) {
	authorType := "User"
	authorID := ""
	if p.OrganizationID > 0 {
		authorType = "Organization"
		authorID = fmt.Sprintf("%d", p.OrganizationID)
	}

	query := `
		INSERT INTO bid (name, description, status, tender_id, author_type, author_id, creator_username)
		VALUES ($1, $2, $3, $4::uuid, $5, $6, $7)
		RETURNING id::text, name, description, status::text, tender_id::text, author_type::text, author_id, creator_username, version, created_at
	`
	var b Bid
	err := s.db.QueryRowContext(ctx, query, p.Name, p.Description, p.Status, p.TenderID, authorType, authorID, p.CreatorUsername).Scan(
		&b.ID,
		&b.Name,
		&b.Description,
		&b.Status,
		&b.TenderID,
		&b.AuthorType,
		&b.AuthorID,
		&b.CreatorUsername,
		&b.Version,
		&b.CreatedAt,
	)
	if err != nil {
		return Bid{}, err
	}

	if err := s.insertBidVersion(ctx, b, p.CreatorUsername); err != nil {
		return Bid{}, err
	}
	return b, nil
}

func (s *BidStore) ListUserBids(ctx context.Context, username string, p ListBidParams) ([]Bid, error) {
	query := `
		SELECT id::text, name, description, status::text, tender_id::text, author_type::text, author_id, creator_username, version, created_at
		FROM bid
		WHERE creator_username = $1
		ORDER BY name ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, username, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBidList(rows)
}

func (s *BidStore) ListBidsForTender(ctx context.Context, tenderID string, p ListBidParams) ([]Bid, error) {
	query := `
		SELECT id::text, name, description, status::text, tender_id::text, author_type::text, author_id, creator_username, version, created_at
		FROM bid
		WHERE tender_id = $1::uuid
		ORDER BY name ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, tenderID, p.Limit, p.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanBidList(rows)
}

func (s *BidStore) GetBidByID(ctx context.Context, bidID string) (Bid, error) {
	query := `
		SELECT id::text, name, description, status::text, tender_id::text, author_type::text, author_id, creator_username, version, created_at
		FROM bid
		WHERE id = $1::uuid
	`
	var b Bid
	err := s.db.QueryRowContext(ctx, query, bidID).Scan(
		&b.ID,
		&b.Name,
		&b.Description,
		&b.Status,
		&b.TenderID,
		&b.AuthorType,
		&b.AuthorID,
		&b.CreatorUsername,
		&b.Version,
		&b.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Bid{}, ErrNotFound
	}
	if err != nil {
		return Bid{}, err
	}
	return b, nil
}

func (s *BidStore) UpdateBidStatus(ctx context.Context, bidID, status string) (Bid, error) {
	query := `
		UPDATE bid
		SET status = $2, version = version + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid
		RETURNING id::text, name, description, status::text, tender_id::text, author_type::text, author_id, creator_username, version, created_at, creator_username
	`
	var (
		b               Bid
		creatorUsername string
	)
	err := s.db.QueryRowContext(ctx, query, bidID, status).Scan(
		&b.ID, &b.Name, &b.Description, &b.Status, &b.TenderID, &b.AuthorType, &b.AuthorID, &b.CreatorUsername, &b.Version, &b.CreatedAt, &creatorUsername,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Bid{}, ErrNotFound
	}
	if err != nil {
		return Bid{}, err
	}
	if err := s.insertBidVersion(ctx, b, creatorUsername); err != nil {
		return Bid{}, err
	}
	return b, nil
}

func (s *BidStore) UpdateBid(ctx context.Context, p UpdateBidParams) (Bid, error) {
	setParts := []string{}
	args := []any{p.ID}
	if p.Name != nil {
		args = append(args, *p.Name)
		setParts = append(setParts, fmt.Sprintf("name = $%d", len(args)))
	}
	if p.Description != nil {
		args = append(args, *p.Description)
		setParts = append(setParts, fmt.Sprintf("description = $%d", len(args)))
	}
	if len(setParts) == 0 {
		return s.GetBidByID(ctx, p.ID)
	}
	setParts = append(setParts, "version = version + 1", "updated_at = CURRENT_TIMESTAMP")
	query := fmt.Sprintf(`
		UPDATE bid
		SET %s
		WHERE id = $1::uuid
		RETURNING id::text, name, description, status::text, tender_id::text, author_type::text, author_id, creator_username, version, created_at, creator_username
	`, strings.Join(setParts, ", "))

	var (
		b               Bid
		creatorUsername string
	)
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&b.ID, &b.Name, &b.Description, &b.Status, &b.TenderID, &b.AuthorType, &b.AuthorID, &b.CreatorUsername, &b.Version, &b.CreatedAt, &creatorUsername,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Bid{}, ErrNotFound
	}
	if err != nil {
		return Bid{}, err
	}
	if err := s.insertBidVersion(ctx, b, creatorUsername); err != nil {
		return Bid{}, err
	}
	return b, nil
}

func (s *BidStore) RollbackBid(ctx context.Context, bidID string, version int) (Bid, error) {
	var hist struct {
		Name            string
		Description     string
		Status          string
		TenderID        string
		AuthorType      string
		AuthorID        string
		CreatorUsername string
	}
	err := s.db.QueryRowContext(
		ctx,
		`SELECT name, description, status::text, tender_id::text, author_type::text, author_id, creator_username
		 FROM bid_version
		 WHERE bid_id = $1::uuid AND version = $2`,
		bidID, version,
	).Scan(&hist.Name, &hist.Description, &hist.Status, &hist.TenderID, &hist.AuthorType, &hist.AuthorID, &hist.CreatorUsername)
	if errors.Is(err, sql.ErrNoRows) {
		return Bid{}, ErrNotFound
	}
	if err != nil {
		return Bid{}, err
	}

	var b Bid
	err = s.db.QueryRowContext(
		ctx,
		`UPDATE bid
		 SET name = $2, description = $3, status = $4, tender_id = $5::uuid, author_type = $6, author_id = $7, creator_username = $8, version = version + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1::uuid
		 RETURNING id::text, name, description, status::text, tender_id::text, author_type::text, author_id, creator_username, version, created_at`,
		bidID, hist.Name, hist.Description, hist.Status, hist.TenderID, hist.AuthorType, hist.AuthorID, hist.CreatorUsername,
	).Scan(&b.ID, &b.Name, &b.Description, &b.Status, &b.TenderID, &b.AuthorType, &b.AuthorID, &b.CreatorUsername, &b.Version, &b.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Bid{}, ErrNotFound
	}
	if err != nil {
		return Bid{}, err
	}
	if err := s.insertBidVersion(ctx, b, hist.CreatorUsername); err != nil {
		return Bid{}, err
	}
	return b, nil
}

func (s *BidStore) insertBidVersion(ctx context.Context, b Bid, creatorUsername string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO bid_version (bid_id, version, name, description, status, tender_id, author_type, author_id, creator_username)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6::uuid, $7, $8, $9)`,
		b.ID, b.Version, b.Name, b.Description, b.Status, b.TenderID, b.AuthorType, b.AuthorID, creatorUsername,
	)
	return err
}

func scanBidList(rows *sql.Rows) ([]Bid, error) {
	out := make([]Bid, 0)
	for rows.Next() {
		var b Bid
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.Status, &b.TenderID, &b.AuthorType, &b.AuthorID, &b.CreatorUsername, &b.Version, &b.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *BidStore) UpsertDecision(ctx context.Context, bidID, username, decision string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO bid_decision (bid_id, username, decision)
		 VALUES ($1::uuid, $2, $3)
		 ON CONFLICT (bid_id, username)
		 DO UPDATE SET decision = EXCLUDED.decision, created_at = CURRENT_TIMESTAMP`,
		bidID, username, decision,
	)
	return err
}

func (s *BidStore) HasRejectedDecision(ctx context.Context, bidID string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM bid_decision WHERE bid_id = $1::uuid AND decision = 'Rejected')`,
		bidID,
	).Scan(&exists)
	return exists, err
}

func (s *BidStore) CountApprovedDecisions(ctx context.Context, bidID string) (int, error) {
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM bid_decision WHERE bid_id = $1::uuid AND decision = 'Approved'`,
		bidID,
	).Scan(&count)
	return count, err
}

func (s *BidStore) AddFeedback(ctx context.Context, bidID, description, authorUsername string) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO bid_feedback (bid_id, description, author_username) VALUES ($1::uuid, $2, $3)`,
		bidID, description, authorUsername,
	)
	return err
}

func (s *BidStore) GetBidByTenderAndAuthor(ctx context.Context, tenderID, authorUsername string) (Bid, error) {
	query := `
		SELECT id::text, name, description, status::text, tender_id::text, author_type::text, author_id, creator_username, version, created_at
		FROM bid
		WHERE tender_id = $1::uuid AND creator_username = $2
		ORDER BY created_at DESC
		LIMIT 1
	`
	var b Bid
	err := s.db.QueryRowContext(ctx, query, tenderID, authorUsername).Scan(
		&b.ID, &b.Name, &b.Description, &b.Status, &b.TenderID, &b.AuthorType, &b.AuthorID, &b.CreatorUsername, &b.Version, &b.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Bid{}, ErrNotFound
	}
	if err != nil {
		return Bid{}, err
	}
	return b, nil
}

func (s *BidStore) ListReviewsByAuthor(ctx context.Context, authorUsername string, limit, offset int) ([]BidReview, error) {
	query := `
		SELECT f.id::text, f.description, f.created_at
		FROM bid_feedback f
		JOIN bid b ON b.id = f.bid_id
		WHERE b.creator_username = $1
		ORDER BY f.created_at DESC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, authorUsername, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reviews := make([]BidReview, 0)
	for rows.Next() {
		var r BidReview
		if err := rows.Scan(&r.ID, &r.Description, &r.CreatedAt); err != nil {
			return nil, err
		}
		reviews = append(reviews, r)
	}
	return reviews, rows.Err()
}
