package store

import (
	"database/sql"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresTransactionLimitStore struct {
	db *sql.DB
}

func NewPostgresTransactionLimitStore(db *sql.DB) *PostgresTransactionLimitStore {
	return &PostgresTransactionLimitStore{db: db}
}

type TransactionLimitStore interface {
	CreateTransactionLimit(t *models.TransactionLimitModel) error
	UpdateTransactionLimit(limitID int64, t *models.TransactionLimitModel) error
	DeleteTransactionLimit(limitID int64) error
	GetAllTransactionLimits(limit, offset int) ([]models.TransactionLimitModel, error)
}

func (ts *PostgresTransactionLimitStore) CreateTransactionLimit(t *models.TransactionLimitModel) error {
	query := `
	INSERT INTO transaction_limit (retailer_id, limit_amount, service)
	VALUES ($1, $2, $3)
	RETURNING limit_id, created_at, updated_at;
	`
	return ts.db.QueryRow(query, t.RetailerID, t.LimitAmount, t.Service).
		Scan(&t.LimitID, &t.CreatedAT, &t.UpdatedAT)
}

func (ts *PostgresTransactionLimitStore) UpdateTransactionLimit(limitID int64, t *models.TransactionLimitModel) error {
	query := `
	UPDATE transaction_limit
	SET limit_amount = $1,
	    updated_at   = CURRENT_TIMESTAMP
	WHERE limit_id = $2;
	`
	res, err := ts.db.Exec(query, t.LimitAmount, limitID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (ts *PostgresTransactionLimitStore) DeleteTransactionLimit(limitID int64) error {
	res, err := ts.db.Exec(`DELETE FROM transaction_limit WHERE limit_id = $1;`, limitID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (ts *PostgresTransactionLimitStore) GetAllTransactionLimits(limit, offset int) ([]models.TransactionLimitModel, error) {
	query := `
	SELECT limit_id, retailer_id, limit_amount, service, created_at, updated_at
	FROM transaction_limit
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2;
	`
	rows, err := ts.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var limits []models.TransactionLimitModel
	for rows.Next() {
		var t models.TransactionLimitModel
		if err := rows.Scan(
			&t.LimitID, &t.RetailerID, &t.LimitAmount, &t.Service,
			&t.CreatedAT, &t.UpdatedAT,
		); err != nil {
			return nil, err
		}
		limits = append(limits, t)
	}
	return limits, rows.Err()
}
