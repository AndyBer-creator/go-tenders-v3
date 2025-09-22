package storage

import (
	"go-tenders-v3/model"

	"github.com/jmoiron/sqlx"
)

type Storage struct {
	DB *sqlx.DB
}

type Bid struct {
	Id          int     `db:"id" json:"id"`
	TenderId    int     `db:"tender_id" json:"tenderId"`
	UserId      int     `db:"user_id" json:"userId"`
	Amount      float64 `db:"amount" json:"amount"`
	Description string  `db:"description" json:"description"`
	Status      string  `db:"status" json:"status"`
}

func NewStorage(db *sqlx.DB) *Storage {
	return &Storage{DB: db}
}

func (s *Storage) CreateBid(bid *Bid) error {
	query := `
        INSERT INTO bid (tender_id, user_id, amount, description, status)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id`
	return s.DB.QueryRow(query, bid.TenderId, bid.UserId, bid.Amount, bid.Description, bid.Status).Scan(&bid.Id)
}

func (s *Storage) EditBid(bid *Bid) error {
	query := `
        UPDATE bid
        SET tender_id = $1,
            description = $2,
            status = $3,
            amount = $4

        WHERE id = $5`
	_, err := s.DB.Exec(query, bid.TenderId, bid.Description, bid.Status, bid.Amount, bid.Id)
	return err
}

func (s *Storage) GetBidReviews(bidId string) ([]model.BidReview, error) {
	reviews := []model.BidReview{}
	query := `SELECT id, description, created_at FROM bid_reviews WHERE bid_id = $1 ORDER BY created_at DESC`
	err := s.DB.Select(&reviews, query, bidId)
	return reviews, err
}

func (s *Storage) GetBidStatus(bidId string) (string, error) {
	var status string
	query := `SELECT status FROM bid WHERE id = $1`
	err := s.DB.Get(&status, query, bidId)
	return status, err
}

func (s *Storage) GetBidsForTender(tenderId string) ([]Bid, error) {
	bids := []Bid{}
	query := `SELECT id, tender_id, user_id, amount, description, status FROM bid WHERE tender_id = $1`
	err := s.DB.Select(&bids, query, tenderId)
	return bids, err
}

func (s *Storage) GetUserBids(userId string) ([]Bid, error) {
	bids := []Bid{}
	query := `SELECT id, tender_id, user_id, amount, description, status FROM bid WHERE user_id = $1`
	err := s.DB.Select(&bids, query, userId)
	return bids, err
}

func (s *Storage) RollbackBid(bidId string, version int) (err error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()

	query := `
        UPDATE bid
        SET status = 'RolledBack', -- или нужный статус
            version = $1
        WHERE id = $2`
	_, err = tx.Exec(query, version, bidId)
	if err != nil {
		return err
	}

	// При необходимости добавить дополнительные операции в рамках транзакции

	return nil
}

func (s *Storage) SubmitBidDecision(bidId string, decision model.BidDecision) error {
	query := `
        UPDATE bid
        SET status = $1
        WHERE id = $2`
	// Решение напрямую меняет статус заявки
	_, err := s.DB.Exec(query, string(decision), bidId)
	return err
}

func (s *Storage) SubmitBidFeedback(bidId string, feedback model.BidReview) error {
	query := `
        INSERT INTO bid_reviews (bid_id, description, created_at)
        VALUES ($1, $2, $3)`
	_, err := s.DB.Exec(query, bidId, feedback.Description, feedback.CreatedAt)
	return err
}

func (s *Storage) UpdateBidStatus(bidId string, status string) error {
	query := `
        UPDATE bid
        SET status = $1
        WHERE id = $2`
	_, err := s.DB.Exec(query, status, bidId)
	return err
}
