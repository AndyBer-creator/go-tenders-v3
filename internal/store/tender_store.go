package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrNotFound = errors.New("not found")

type Tender struct {
	ID             string
	Name           string
	Description    string
	ServiceType    string
	Status         string
	OrganizationID string
	Version        int
	CreatedAt      time.Time
}

type CreateTenderParams struct {
	Name            string
	Description     string
	ServiceType     string
	Status          string
	OrganizationID  int
	CreatorUsername string
}

type ListTenderParams struct {
	Limit        int
	Offset       int
	ServiceTypes []string
}

type UpdateTenderParams struct {
	ID          string
	Name        *string
	Description *string
	ServiceType *string
}

type TenderStore struct {
	db *sql.DB
}

func NewTenderStore(db *sql.DB) *TenderStore {
	return &TenderStore{db: db}
}

func (s *TenderStore) UserExists(ctx context.Context, username string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM employee WHERE username = $1)`,
		username,
	).Scan(&exists)
	return exists, err
}

// EmployeeIDByUsername returns employee.id for a valid username (used as bid author_id for User bids).
func (s *TenderStore) EmployeeIDByUsername(ctx context.Context, username string) (int, error) {
	var id int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT id FROM employee WHERE username = $1`,
		username,
	).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrNotFound
	}
	return id, err
}

func (s *TenderStore) IsOrganizationResponsible(ctx context.Context, organizationID int, username string) (bool, error) {
	var exists bool
	err := s.db.QueryRowContext(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM organization_responsible r
			JOIN employee e ON e.id = r.user_id
			WHERE r.organization_id = $1 AND e.username = $2
		)`,
		organizationID,
		username,
	).Scan(&exists)
	return exists, err
}

func (s *TenderStore) CreateTender(ctx context.Context, p CreateTenderParams) (Tender, error) {
	query := `
		INSERT INTO tender (name, description, service_type, status, organization_id, creator_username)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text, name, description, service_type::text, status::text, organization_id::text, version, created_at
	`

	var t Tender
	err := s.db.QueryRowContext(ctx, query, p.Name, p.Description, p.ServiceType, p.Status, p.OrganizationID, p.CreatorUsername).Scan(
		&t.ID,
		&t.Name,
		&t.Description,
		&t.ServiceType,
		&t.Status,
		&t.OrganizationID,
		&t.Version,
		&t.CreatedAt,
	)
	if err != nil {
		return Tender{}, err
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO tender_version (tender_id, version, name, description, service_type, status, organization_id, creator_username)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Version, t.Name, t.Description, t.ServiceType, t.Status, p.OrganizationID, p.CreatorUsername,
	)
	if err != nil {
		return Tender{}, err
	}

	return t, nil
}

func (s *TenderStore) ListTenders(ctx context.Context, p ListTenderParams) ([]Tender, error) {
	base := `
		SELECT id::text, name, description, service_type::text, status::text, organization_id::text, version, created_at
		FROM tender
		WHERE status = 'Published'
	`
	args := []any{}

	if len(p.ServiceTypes) > 0 {
		placeholders := make([]string, 0, len(p.ServiceTypes))
		for _, st := range p.ServiceTypes {
			args = append(args, st)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
		}
		base += " AND service_type IN (" + strings.Join(placeholders, ",") + ")"
	}

	args = append(args, p.Limit, p.Offset)
	base += fmt.Sprintf(" ORDER BY name ASC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.db.QueryContext(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Tender, 0)
	for rows.Next() {
		var t Tender
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.ServiceType, &t.Status, &t.OrganizationID, &t.Version, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *TenderStore) ListUserTenders(ctx context.Context, username string, limit, offset int) ([]Tender, error) {
	query := `
		SELECT t.id::text, t.name, t.description, t.service_type::text, t.status::text, t.organization_id::text, t.version, t.created_at
		FROM tender t
		WHERE t.creator_username = $1
		ORDER BY t.name ASC
		LIMIT $2 OFFSET $3
	`
	rows, err := s.db.QueryContext(ctx, query, username, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Tender, 0)
	for rows.Next() {
		var t Tender
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.ServiceType, &t.Status, &t.OrganizationID, &t.Version, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *TenderStore) GetTenderByID(ctx context.Context, id string) (Tender, error) {
	query := `
		SELECT id::text, name, description, service_type::text, status::text, organization_id::text, version, created_at
		FROM tender
		WHERE id = $1::uuid
	`
	var t Tender
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&t.ID,
		&t.Name,
		&t.Description,
		&t.ServiceType,
		&t.Status,
		&t.OrganizationID,
		&t.Version,
		&t.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Tender{}, ErrNotFound
	}
	if err != nil {
		return Tender{}, err
	}
	return t, nil
}

func (s *TenderStore) UpdateTenderStatus(ctx context.Context, id string, status string) (Tender, error) {
	query := `
		UPDATE tender
		SET status = $2, version = version + 1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1::uuid
		RETURNING id::text, name, description, service_type::text, status::text, organization_id::text, version, created_at, creator_username, organization_id
	`
	var (
		t               Tender
		creatorUsername string
		organizationID  int
	)
	err := s.db.QueryRowContext(ctx, query, id, status).Scan(
		&t.ID, &t.Name, &t.Description, &t.ServiceType, &t.Status, &t.OrganizationID, &t.Version, &t.CreatedAt, &creatorUsername, &organizationID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Tender{}, ErrNotFound
	}
	if err != nil {
		return Tender{}, err
	}
	if err := s.insertTenderVersion(ctx, t, creatorUsername, organizationID); err != nil {
		return Tender{}, err
	}
	return t, nil
}

func (s *TenderStore) UpdateTender(ctx context.Context, p UpdateTenderParams) (Tender, error) {
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
	if p.ServiceType != nil {
		args = append(args, *p.ServiceType)
		setParts = append(setParts, fmt.Sprintf("service_type = $%d", len(args)))
	}
	if len(setParts) == 0 {
		return s.GetTenderByID(ctx, p.ID)
	}

	setParts = append(setParts, "version = version + 1", "updated_at = CURRENT_TIMESTAMP")
	query := fmt.Sprintf(`
		UPDATE tender
		SET %s
		WHERE id = $1::uuid
		RETURNING id::text, name, description, service_type::text, status::text, organization_id::text, version, created_at, creator_username, organization_id
	`, strings.Join(setParts, ", "))

	var (
		t               Tender
		creatorUsername string
		organizationID  int
	)
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&t.ID, &t.Name, &t.Description, &t.ServiceType, &t.Status, &t.OrganizationID, &t.Version, &t.CreatedAt, &creatorUsername, &organizationID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Tender{}, ErrNotFound
	}
	if err != nil {
		return Tender{}, err
	}
	if err := s.insertTenderVersion(ctx, t, creatorUsername, organizationID); err != nil {
		return Tender{}, err
	}
	return t, nil
}

func (s *TenderStore) RollbackTender(ctx context.Context, id string, version int) (Tender, error) {
	var historical struct {
		Name            string
		Description     string
		ServiceType     string
		Status          string
		OrganizationID  int
		CreatorUsername string
	}
	err := s.db.QueryRowContext(
		ctx,
		`SELECT name, description, service_type::text, status::text, organization_id, creator_username
		 FROM tender_version
		 WHERE tender_id = $1::uuid AND version = $2`,
		id, version,
	).Scan(
		&historical.Name,
		&historical.Description,
		&historical.ServiceType,
		&historical.Status,
		&historical.OrganizationID,
		&historical.CreatorUsername,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Tender{}, ErrNotFound
	}
	if err != nil {
		return Tender{}, err
	}

	var t Tender
	err = s.db.QueryRowContext(
		ctx,
		`UPDATE tender
		 SET name = $2, description = $3, service_type = $4, status = $5, organization_id = $6, creator_username = $7, version = version + 1, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1::uuid
		 RETURNING id::text, name, description, service_type::text, status::text, organization_id::text, version, created_at`,
		id,
		historical.Name,
		historical.Description,
		historical.ServiceType,
		historical.Status,
		historical.OrganizationID,
		historical.CreatorUsername,
	).Scan(&t.ID, &t.Name, &t.Description, &t.ServiceType, &t.Status, &t.OrganizationID, &t.Version, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Tender{}, ErrNotFound
	}
	if err != nil {
		return Tender{}, err
	}

	if err := s.insertTenderVersion(ctx, t, historical.CreatorUsername, historical.OrganizationID); err != nil {
		return Tender{}, err
	}
	return t, nil
}

func (s *TenderStore) insertTenderVersion(ctx context.Context, t Tender, creatorUsername string, organizationID int) error {
	_, err := s.db.ExecContext(
		ctx,
		`INSERT INTO tender_version (tender_id, version, name, description, service_type, status, organization_id, creator_username)
		 VALUES ($1::uuid, $2, $3, $4, $5, $6, $7, $8)`,
		t.ID, t.Version, t.Name, t.Description, t.ServiceType, t.Status, organizationID, creatorUsername,
	)
	return err
}

func (s *TenderStore) CountOrganizationResponsibles(ctx context.Context, organizationID int) (int, error) {
	var count int
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM organization_responsible WHERE organization_id = $1`,
		organizationID,
	).Scan(&count)
	return count, err
}
