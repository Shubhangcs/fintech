package store

import (
	"database/sql"
	"errors"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type PostgresTicketStore struct {
	db *sql.DB
}

func NewPostgresTicketStore(db *sql.DB) *PostgresTicketStore {
	return &PostgresTicketStore{db: db}
}

type TicketStore interface {
	CreateTicket(t *models.TicketModel) error
	UpdateTicket(t *models.TicketModel) error
	DeleteTicket(ticketID int64) error
	UpdateTicketClearStatus(t *models.TicketModel) error
	GetAllTickets(p utils.QueryParams) ([]models.TicketModel, error)
	GetTicketsByUserID(userID string, p utils.QueryParams) ([]models.TicketModel, error)
	GetAdminIDByUserID(userID string) (string, error)
}

// ticketSelectBase joins all user tables so every list query includes user_name and user_business_name.
const ticketSelectBase = `
SELECT
	t.ticket_id,
	t.admin_id,
	t.user_id,
	t.ticket_title,
	t.ticket_description,
	t.is_ticket_cleared,
	COALESCE(a.admin_name, md.master_distributor_name, d.distributor_name, r.retailer_name, '') AS user_name,
	COALESCE(md.master_distributor_business_name, d.distributor_business_name, r.retailer_business_name) AS user_business_name,
	t.created_at,
	t.updated_at
FROM ticket t
LEFT JOIN admins a              ON t.user_id = a.admin_id
LEFT JOIN master_distributors md ON t.user_id = md.master_distributor_id
LEFT JOIN distributors d        ON t.user_id = d.distributor_id
LEFT JOIN retailers r           ON t.user_id = r.retailer_id
`

// Create Ticket
func (ts *PostgresTicketStore) CreateTicket(t *models.TicketModel) error {
	query := `
	INSERT INTO ticket (admin_id, user_id, ticket_title, ticket_description)
	VALUES ($1, $2, $3, $4)
	RETURNING ticket_id, is_ticket_cleared, created_at, updated_at;
	`
	return ts.db.QueryRow(query, t.AdminID, t.UserID, t.TicketTitle, t.TicketDescription).
		Scan(&t.TicketID, &t.IsTicketCleared, &t.CreatedAT, &t.UpdatedAT)
}

// Update Ticket
func (ts *PostgresTicketStore) UpdateTicket(t *models.TicketModel) error {
	query := `
	UPDATE ticket
	SET ticket_title       = COALESCE(NULLIF($1, ''), ticket_title),
	    ticket_description = COALESCE(NULLIF($2, ''), ticket_description),
	    updated_at         = CURRENT_TIMESTAMP
	WHERE ticket_id = $3;
	`
	res, err := ts.db.Exec(query, t.TicketTitle, t.TicketDescription, t.TicketID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Delete Ticket
func (ts *PostgresTicketStore) DeleteTicket(ticketID int64) error {
	res, err := ts.db.Exec(`DELETE FROM ticket WHERE ticket_id = $1;`, ticketID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Update Ticket Clear Status
func (ts *PostgresTicketStore) UpdateTicketClearStatus(t *models.TicketModel) error {
	query := `
	UPDATE ticket
	SET is_ticket_cleared = $1,
	    updated_at        = CURRENT_TIMESTAMP
	WHERE ticket_id = $2;
	`
	res, err := ts.db.Exec(query, t.IsTicketCleared, t.TicketID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Get All Tickets
func (ts *PostgresTicketStore) GetAllTickets(p utils.QueryParams) ([]models.TicketModel, error) {
	query := ticketSelectBase + `
	WHERE t.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND   t.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY t.created_at DESC
	LIMIT $1 OFFSET $2;
	`
	return scanTickets(ts.db, query, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

// Get Tickets By User ID
func (ts *PostgresTicketStore) GetTicketsByUserID(userID string, p utils.QueryParams) ([]models.TicketModel, error) {
	query := ticketSelectBase + `
	WHERE t.user_id = $1
	AND   t.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND   t.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY t.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanTickets(ts.db, query, userID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

// GetAdminIDByUserID resolves the admin for any user in the hierarchy.
func (ts *PostgresTicketStore) GetAdminIDByUserID(userID string) (string, error) {
	query := `
	SELECT admin_id FROM (
		SELECT admin_id FROM admins WHERE admin_id = $1
		UNION ALL
		SELECT admin_id FROM master_distributors WHERE master_distributor_id = $1
		UNION ALL
		SELECT md.admin_id FROM distributors d
			JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
			WHERE d.distributor_id = $1
		UNION ALL
		SELECT md.admin_id FROM retailers r
			JOIN distributors d ON r.distributor_id = d.distributor_id
			JOIN master_distributors md ON d.master_distributor_id = md.master_distributor_id
			WHERE r.retailer_id = $1
	) AS hierarchy LIMIT 1;
	`
	var adminID string
	err := ts.db.QueryRow(query, userID).Scan(&adminID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("user not found")
		}
		return "", err
	}
	return adminID, nil
}

func scanTickets(db *sql.DB, query string, args ...any) ([]models.TicketModel, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tickets []models.TicketModel
	for rows.Next() {
		var t models.TicketModel
		if err := rows.Scan(
			&t.TicketID, &t.AdminID, &t.UserID,
			&t.TicketTitle, &t.TicketDescription, &t.IsTicketCleared,
			&t.UserName, &t.UserBusinessName,
			&t.CreatedAT, &t.UpdatedAT,
		); err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, rows.Err()
}
