package storage

import (
	"time"
)

type Tender struct {
	Id             int       `db:"id" json:"id"`
	OrganizationId int       `db:"organization_id" json:"organizationId"`
	Title          string    `db:"title" json:"title"`
	Description    string    `db:"description" json:"description"`
	Status         string    `db:"status" json:"status"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}

func (s *Storage) CreateTender(t *Tender) error {
	query := `
        INSERT INTO tender (organization_id, title, description, status)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, updated_at`
	return s.DB.QueryRow(query, t.OrganizationId, t.Title, t.Description, t.Status).
		Scan(&t.Id, &t.CreatedAt, &t.UpdatedAt)
}

func (s *Storage) GetTenders() ([]Tender, error) {
	var tenders []Tender
	query := `SELECT * FROM tender ORDER BY created_at DESC`
	err := s.DB.Select(&tenders, query)
	return tenders, err
}

func (s *Storage) GetTenderById(id int) (*Tender, error) {
	var t Tender
	query := `SELECT * FROM tender WHERE id = $1`
	err := s.DB.Get(&t, query, id)
	return &t, err
}

func (s *Storage) UpdateTender(t *Tender) error {
	query := `
        UPDATE tender
        SET title = $1, description = $2, status = $3, updated_at = NOW()
        WHERE id = $4`
	_, err := s.DB.Exec(query, t.Title, t.Description, t.Status, t.Id)
	return err
}

func (s *Storage) EditTender(tender *Tender) error {
	query := `
        UPDATE tender
        SET title = $1,
            description = $2,
            status = $3,
            updated_at = NOW()
        WHERE id = $4`
	_, err := s.DB.Exec(query, tender.Title, tender.Description, tender.Status, tender.Id)
	return err
}

func (s *Storage) GetTenderStatus(tenderId int) (string, error) {
	var status string
	query := `SELECT status FROM tender WHERE id = $1`
	err := s.DB.Get(&status, query, tenderId)
	return status, err
}

func (s *Storage) GetUserTenders(userId int) ([]Tender, error) {
	var tenders []Tender
	query := `
        SELECT t.*
        FROM tender t
        JOIN organization o ON o.id = t.organization_id
        JOIN user_organization uo ON uo.organization_id = o.id
        WHERE uo.user_id = $1
        ORDER BY t.created_at DESC`

	err := s.DB.Select(&tenders, query, userId)
	return tenders, err
}

func (s *Storage) RollbackTender(tenderId int, previousStatus string) (err error) {
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
        UPDATE tender
        SET status = $1,
            updated_at = NOW()
        WHERE id = $2`
	_, err = tx.Exec(query, previousStatus, tenderId)
	return err
}

func (s *Storage) UpdateTenderStatus(tenderId int, status string) error {
	query := `
        UPDATE tender
        SET status = $1,
            updated_at = NOW()
        WHERE id = $2`
	_, err := s.DB.Exec(query, status, tenderId)
	return err
}
