package store

import (
	"database/sql"
	"time"

	"github.com/levionstudio/fintech/internal/models"
)

type PostgresWalletTransactionStore struct {
	db *sql.DB
}

func NewPostgresWalletTransactionStore(db *sql.DB) *PostgresWalletTransactionStore {
	return &PostgresWalletTransactionStore{db: db}
}

type WalletTransactionStore interface {
	CreateWalletTransaction(wt *models.WalletTransactionModel) error
	GetWalletTransactionsByUserID(userID string, limit, offset int, startDate, endDate *time.Time) ([]models.WalletTransactionModel, error)
}

func (ws *PostgresWalletTransactionStore) CreateWalletTransaction(wt *models.WalletTransactionModel) error {
	query := `
	INSERT INTO wallet_transactions (
		user_id,
		reference_id,
		credit_amount,
		debit_amount,
		before_balance,
		after_balance,
		transaction_reason,
		remarks
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8
	)
	RETURNING wallet_transaction_id, created_at;
	`

	return ws.db.QueryRow(
		query,
		wt.UserID,
		wt.ReferenceID,
		wt.CreditAmount,
		wt.DebitAmount,
		wt.BeforeBalance,
		wt.AfterBalance,
		wt.TransactionReason,
		wt.Remarks,
	).Scan(
		&wt.WalletTransactionID,
		&wt.CreatedAT,
	)
}

func (ws *PostgresWalletTransactionStore) GetWalletTransactionsByUserID(userID string, limit, offset int, startDate, endDate *time.Time) ([]models.WalletTransactionModel, error) {
	query := `
	SELECT
		wt.wallet_transaction_id,
		wt.user_id,
		wt.reference_id,
		wt.credit_amount,
		wt.debit_amount,
		wt.before_balance,
		wt.after_balance,
		wt.transaction_reason,
		wt.remarks,
		wt.created_at,
		COALESCE(a.admin_name, md.master_distributor_name, d.distributor_name, r.retailer_name, '') AS user_name,
		COALESCE(md.master_distributor_business_name, d.distributor_business_name, r.retailer_business_name) AS user_business_name
	FROM wallet_transactions wt
	LEFT JOIN admins a               ON wt.user_id = a.admin_id
	LEFT JOIN master_distributors md ON wt.user_id = md.master_distributor_id
	LEFT JOIN distributors d         ON wt.user_id = d.distributor_id
	LEFT JOIN retailers r            ON wt.user_id = r.retailer_id
	WHERE wt.user_id = $1
	AND wt.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND wt.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	ORDER BY wt.created_at DESC
	LIMIT $2 OFFSET $3;
	`

	rows, err := ws.db.Query(query, userID, limit, offset, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.WalletTransactionModel
	for rows.Next() {
		var wt models.WalletTransactionModel
		err = rows.Scan(
			&wt.WalletTransactionID,
			&wt.UserID,
			&wt.ReferenceID,
			&wt.CreditAmount,
			&wt.DebitAmount,
			&wt.BeforeBalance,
			&wt.AfterBalance,
			&wt.TransactionReason,
			&wt.Remarks,
			&wt.CreatedAT,
			&wt.UserName,
			&wt.UserBusinessName,
		)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, wt)
	}

	return transactions, rows.Err()
}
