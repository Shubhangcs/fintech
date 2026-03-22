package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type PostgresRevertTransactionStore struct {
	db          *sql.DB
	walletStore WalletTransactionStore
}

func NewPostgresRevertTransactionStore(db *sql.DB, walletStore WalletTransactionStore) *PostgresRevertTransactionStore {
	return &PostgresRevertTransactionStore{db: db, walletStore: walletStore}
}

type RevertTransactionStore interface {
	CreateRevertTransaction(rt *models.RevertTransactionModel) error
	GetRevertTransactionsByRevertByID(revertByID string, p utils.QueryParams) ([]models.RevertTransactionModel, error)
	GetRevertTransactionsByRevertOnID(revertOnID string, p utils.QueryParams) ([]models.RevertTransactionModel, error)
	GetAllRevertTransactions(p utils.QueryParams) ([]models.RevertTransactionModel, error)
}

// Create Revert Transaction
func (rs *PostgresRevertTransactionStore) CreateRevertTransaction(rt *models.RevertTransactionModel) error {
	tx, err := rs.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	revertOnInfo, err := getUserTableInfo(rt.RevertOnID)
	if err != nil {
		return err
	}
	revertByInfo, err := getUserTableInfo(rt.RevertByID)
	if err != nil {
		return err
	}

	// Insert revert transaction record first to get reference ID for wallet entries
	err = tx.QueryRow(`
		INSERT INTO revert_transactions (revert_by_id, revert_on_id, amount, revert_status, remarks)
		VALUES ($1, $2, $3, 'SUCCESS', $4)
		RETURNING revert_transaction_id, revert_status, created_at
	`, rt.RevertByID, rt.RevertOnID, rt.Amount, rt.Remarks).Scan(
		&rt.RevertTransactionID, &rt.RevertStatus, &rt.CreatedAT,
	)
	if err != nil {
		return err
	}

	refID := fmt.Sprintf("%d", rt.RevertTransactionID)
	remarks := fmt.Sprintf("Revert transaction: %s", rt.Remarks)

	// Debit revert_on — atomically checks balance, also creates wallet transaction entry
	if err = debitTx(tx, transaction{
		UserID: rt.RevertOnID, ReferenceID: refID,
		Amount: rt.Amount, Reason: "REVERT", Remarks: remarks,
		userTableInfo: *revertOnInfo,
	}, rs.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return checkExistsTx(tx, revertOnInfo.TableName, revertOnInfo.IDColumnName, rt.RevertOnID, "revert_on user")
		}
		return err
	}

	// Credit revert_by — also creates wallet transaction entry
	if err = creditTx(tx, transaction{
		UserID: rt.RevertByID, ReferenceID: refID,
		Amount: rt.Amount, Reason: "REVERT", Remarks: remarks,
		userTableInfo: *revertByInfo,
	}, rs.walletStore); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("revert_by user not found")
		}
		return err
	}

	return tx.Commit()
}

// Get Revert Transaction Query
const revertTransactionSelectBase = `
SELECT
	rt.revert_transaction_id, rt.revert_by_id, rt.revert_on_id, rt.amount,
	rt.revert_status, rt.remarks, rt.created_at,
	COALESCE(b.name, '') AS revert_by_name,
	b.business_name       AS revert_by_business_name,
	COALESCE(o.name, '') AS revert_on_name,
	o.business_name       AS revert_on_business_name
FROM revert_transactions rt
LEFT JOIN LATERAL (
	SELECT name, business_name FROM (
		SELECT admin_name AS name,            NULL::TEXT AS business_name              FROM admins              WHERE admin_id              = rt.revert_by_id
		UNION ALL
		SELECT master_distributor_name,       master_distributor_business_name         FROM master_distributors WHERE master_distributor_id = rt.revert_by_id
		UNION ALL
		SELECT distributor_name,              distributor_business_name                FROM distributors        WHERE distributor_id       = rt.revert_by_id
		UNION ALL
		SELECT retailer_name,                 retailer_business_name                   FROM retailers           WHERE retailer_id          = rt.revert_by_id
	) u LIMIT 1
) b ON TRUE
LEFT JOIN LATERAL (
	SELECT name, business_name FROM (
		SELECT admin_name AS name,            NULL::TEXT AS business_name              FROM admins              WHERE admin_id              = rt.revert_on_id
		UNION ALL
		SELECT master_distributor_name,       master_distributor_business_name         FROM master_distributors WHERE master_distributor_id = rt.revert_on_id
		UNION ALL
		SELECT distributor_name,              distributor_business_name                FROM distributors        WHERE distributor_id       = rt.revert_on_id
		UNION ALL
		SELECT retailer_name,                 retailer_business_name                   FROM retailers           WHERE retailer_id          = rt.revert_on_id
	) u LIMIT 1
) o ON TRUE
`

// Get Revert Transaction By Revert By ID
func (rs *PostgresRevertTransactionStore) GetRevertTransactionsByRevertByID(revertByID string, p utils.QueryParams) ([]models.RevertTransactionModel, error) {
	q := revertTransactionSelectBase + `
	WHERE rt.revert_by_id = $1
	AND rt.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND rt.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY rt.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanRevertTransactions(rs.db, q, revertByID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

// Get Revert Transactions By Revert On ID
func (rs *PostgresRevertTransactionStore) GetRevertTransactionsByRevertOnID(revertOnID string, p utils.QueryParams) ([]models.RevertTransactionModel, error) {
	q := revertTransactionSelectBase + `
	WHERE rt.revert_on_id = $1
	AND rt.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND rt.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY rt.created_at DESC
	LIMIT $2 OFFSET $3;
	`
	return scanRevertTransactions(rs.db, q, revertOnID, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

// Get All Revert Transactions
func (rs *PostgresRevertTransactionStore) GetAllRevertTransactions(p utils.QueryParams) ([]models.RevertTransactionModel, error) {
	q := revertTransactionSelectBase + `
	WHERE rt.created_at >= COALESCE($3, '-infinity'::TIMESTAMPTZ)
	AND rt.created_at <= COALESCE($4, 'infinity'::TIMESTAMPTZ)
	ORDER BY rt.created_at DESC
	LIMIT $1 OFFSET $2;
	`
	return scanRevertTransactions(rs.db, q, p.Limit, p.Offset, p.StartDate, p.EndDate)
}

func scanRevertTransactions(db *sql.DB, query string, args ...any) ([]models.RevertTransactionModel, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []models.RevertTransactionModel{}
	for rows.Next() {
		var rt models.RevertTransactionModel
		if err = rows.Scan(
			&rt.RevertTransactionID, &rt.RevertByID, &rt.RevertOnID, &rt.Amount,
			&rt.RevertStatus, &rt.Remarks, &rt.CreatedAT,
			&rt.RevertByName, &rt.RevertByBusinessName,
			&rt.RevertOnName, &rt.RevertOnBusinessName,
		); err != nil {
			return nil, err
		}
		results = append(results, rt)
	}
	return results, rows.Err()
}
