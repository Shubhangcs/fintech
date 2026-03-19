package store

import (
	"database/sql"
	"errors"

	"github.com/levionstudio/fintech/internal/models"
)

var DefaultTransactionLimit = map[string]float64{
	"PAYOUT": 25000,
	"DMT":    25000,
	"AEPS":   25000,
}

type PostgresTransactionLimitStore struct {
	db *sql.DB
}

func NewPostgresTransactionLimitStore(db *sql.DB) *PostgresTransactionLimitStore {
	return &PostgresTransactionLimitStore{db: db}
}

type TransactionLimitStore interface {
	CreateTransactionLimit(t *models.TransactionLimitModel) error
	UpdateTransactionLimit(t *models.TransactionLimitModel) error
	DeleteTransactionLimit(id int64) error
	GetAllTransactionLimits(limit, offset int) ([]models.TransactionLimitModel, error)
	GetTransactionLimitByRetailerIDAndService(t *models.TransactionLimitModel) (float64, bool, error)
}

// Create Transaction Limit
func (ts *PostgresTransactionLimitStore) CreateTransactionLimit(t *models.TransactionLimitModel) error {
	query := `
	INSERT INTO transaction_limit (retailer_id, limit_amount, service)
	VALUES ($1, $2, $3)
	RETURNING limit_id, created_at, updated_at;
	`
	return ts.db.QueryRow(query, t.RetailerID, t.LimitAmount, t.Service).
		Scan(&t.LimitID, &t.CreatedAT, &t.UpdatedAT)
}

// Update Transaction Limit
func (ts *PostgresTransactionLimitStore) UpdateTransactionLimit(t *models.TransactionLimitModel) error {
	query := `
	UPDATE transaction_limit
	SET limit_amount = $1,
	    updated_at   = CURRENT_TIMESTAMP
	WHERE limit_id = $2;
	`
	res, err := ts.db.Exec(query, t.LimitAmount, t.LimitID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Delete Transaction Limit
func (ts *PostgresTransactionLimitStore) DeleteTransactionLimit(id int64) error {
	res, err := ts.db.Exec(`DELETE FROM transaction_limit WHERE limit_id = $1;`, id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

// Get All Transaction Limit
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

// Get Transaction Limit By Retailer ID and Service
func (ts *PostgresTransactionLimitStore) GetTransactionLimitByRetailerIDAndService(t *models.TransactionLimitModel) (float64, bool, error) {
	query := `
		SELECT
			limit_amount
		FROM transaction_limit
		WHERE retailer_id = $1
		AND service = $2;
	`
	var transactionLimit float64
	err := ts.db.QueryRow(
		query,
		t.RetailerID,
		t.Service,
	).Scan(
		&transactionLimit,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return DefaultTransactionLimit[t.Service], true, nil
		}
		return 0, false, err
	}

	return transactionLimit, false, nil
}
