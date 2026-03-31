package store

import (
	"database/sql"

	"github.com/levionstudio/fintech/internal/models"
	"github.com/levionstudio/fintech/internal/utils"
)

type PostgresWalletTransactionStore struct {
	db *sql.DB
}

func NewPostgresWalletTransactionStore(db *sql.DB) *PostgresWalletTransactionStore {
	return &PostgresWalletTransactionStore{db: db}
}

type WalletTransactionStore interface {
	CreateWalletTransaction(wt *models.WalletTransactionModel) error
	CreateWalletTransactionTx(tx *sql.Tx, wt *models.WalletTransactionModel) error
	GetWalletTransactionsByUserID(userID string, p utils.QueryParams) ([]models.WalletTransactionModel, error)
}

// Create Wallet Transaction
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

func (ws *PostgresWalletTransactionStore) CreateWalletTransactionTx(tx *sql.Tx, wt *models.WalletTransactionModel) error {
	const query = `
	INSERT INTO wallet_transactions (
		user_id, reference_id, credit_amount, debit_amount,
		before_balance, after_balance, transaction_reason, remarks
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	RETURNING wallet_transaction_id, created_at;
	`
	return tx.QueryRow(query,
		wt.UserID, wt.ReferenceID, wt.CreditAmount, wt.DebitAmount,
		wt.BeforeBalance, wt.AfterBalance, wt.TransactionReason, wt.Remarks,
	).Scan(&wt.WalletTransactionID, &wt.CreatedAT)
}

// Get Wallet Transactions By User ID
func (ws *PostgresWalletTransactionStore) GetWalletTransactionsByUserID(userID string, p utils.QueryParams) ([]models.WalletTransactionModel, error) {
	query := `
	WITH user_info AS (
		SELECT admin_name AS user_name, NULL::TEXT AS user_business_name
		FROM admins WHERE admin_id = $1
		UNION ALL
		SELECT master_distributor_name, master_distributor_business_name
		FROM master_distributors WHERE master_distributor_id = $1
		UNION ALL
		SELECT distributor_name, distributor_business_name
		FROM distributors WHERE distributor_id = $1
		UNION ALL
		SELECT retailer_name, retailer_business_name
		FROM retailers WHERE retailer_id = $1
		LIMIT 1
	)
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
		COALESCE(ui.user_name, '') AS user_name,
		ui.user_business_name
	FROM wallet_transactions wt
	CROSS JOIN user_info ui
	WHERE wt.user_id = $1
	AND wt.created_at >= COALESCE($4, '-infinity'::TIMESTAMPTZ)
	AND wt.created_at <= COALESCE($5, 'infinity'::TIMESTAMPTZ)
	AND ($6::TEXT IS NULL OR (
		wt.reference_id ILIKE '%'||$6||'%' OR
		wt.transaction_reason ILIKE '%'||$6||'%'
	))
	ORDER BY wt.created_at DESC
	LIMIT $2 OFFSET $3;
	`

	rows, err := ws.db.Query(query, userID, p.Limit, p.Offset, p.StartDate, p.EndDate, p.Search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	transactions := []models.WalletTransactionModel{}
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
